// Package jsonutil provides low-level JSON value helpers shared by the
// rfc6902 implementation. It is internal: importers should use the root
// rfc6902 package instead.
package jsonutil

import (
	jsonv1 "encoding/json"
	"math/big"
	"reflect"
)

// undefinedType models JavaScript's `undefined`, which has no JSON
// representation. It exists solely to preserve the reference implementation's
// distinction between "no value at all" (a missing object member or an
// unresolvable JSON Pointer path) and JSON `null` (which Go models with nil).
// The concrete type is unexported; callers work with the exported Undefined
// sentinel value alongside the natural Go types (map[string]any, []any, nil).
type undefinedType struct{}

// Undefined is the canonical value of type undefinedType.
var Undefined any = undefinedType{}

// IsUndefined reports whether v is the JS-undefined sentinel.
func IsUndefined(v any) bool {
	_, ok := v.(undefinedType)
	return ok
}

// ObjectType mirrors util.objectType() from the TypeScript implementation.
// Note that, like the reference implementation, nil (JSON null) reports "null"
// rather than the JS `typeof null === 'object'`.
func ObjectType(object any) string {
	switch object.(type) {
	case nil:
		return "null"
	case undefinedType:
		return "undefined"
	case bool:
		return "boolean"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	if isNumber(object) {
		return "number"
	}
	return "object"
}

// isNumber reports whether v is any Go numeric type, including json.Number.
func isNumber(v any) bool {
	if v == nil {
		return false
	}
	if _, ok := v.(jsonv1.Number); ok {
		return true
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// numberValue converts any Go numeric value to float64, matching JavaScript's
// single number type.
func numberValue(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	case jsonv1.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

// jsonNumbersEqual compares two json.Number values exactly, using big.Rat so
// that integers beyond float64's 2^53 precision are not conflated.
func jsonNumbersEqual(a, b jsonv1.Number) bool {
	aRat, aOK := new(big.Rat).SetString(string(a))
	bRat, bOK := new(big.Rat).SetString(string(b))
	if aOK && bOK {
		return aRat.Cmp(bRat) == 0
	}
	return string(a) == string(b)
}

// ScalarEqual mirrors JavaScript's `===` for the values that reach it: it
// treats all numeric types as numerically comparable (so an int 3 and a
// float64 3 are equal, as they would be in JS), and nil/undefined specially.
// It falls back to reflect.DeepEqual for any other (non-container) value.
func ScalarEqual(a, b any) bool {
	if IsUndefined(a) || IsUndefined(b) {
		return IsUndefined(a) && IsUndefined(b)
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if aNumber, ok := a.(jsonv1.Number); ok {
		if bNumber, ok := b.(jsonv1.Number); ok {
			return jsonNumbersEqual(aNumber, bNumber)
		}
	}
	if isNumber(a) && isNumber(b) {
		return numberValue(a) == numberValue(b)
	}
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	}
	return reflect.DeepEqual(a, b)
}

// DeepEqual reports whether a and b are equal under JSON Patch semantics:
// objects compare by their defined members (undefined members are absent),
// arrays compare element-wise, and scalars compare with ScalarEqual. This is
// equivalent to the reference implementation's `diffAny(a, b, ...) == []`.
func DeepEqual(a, b any) bool {
	if aMap, ok := a.(map[string]any); ok {
		bMap, ok := b.(map[string]any)
		if !ok {
			return false
		}
		aCount := 0
		for key, aValue := range aMap {
			if IsUndefined(aValue) {
				continue
			}
			aCount++
			bValue, ok := bMap[key]
			if !ok || IsUndefined(bValue) {
				return false
			}
			if !DeepEqual(aValue, bValue) {
				return false
			}
		}
		bCount := 0
		for _, bValue := range bMap {
			if !IsUndefined(bValue) {
				bCount++
			}
		}
		return aCount == bCount
	}
	if aArray, ok := a.([]any); ok {
		bArray, ok := b.([]any)
		if !ok {
			return false
		}
		if len(aArray) != len(bArray) {
			return false
		}
		for i := range aArray {
			if !DeepEqual(aArray[i], bArray[i]) {
				return false
			}
		}
		return true
	}
	return ScalarEqual(a, b)
}

// Clone recursively copies a value. Arrays and objects are reconstructed from
// their constituent elements; everything else (primitives, and any other Go
// value such as time.Time) is returned as-is, which is a value copy.
func Clone(source any) any {
	switch s := source.(type) {
	case map[string]any:
		target := make(map[string]any, len(s))
		for key, value := range s {
			target[key] = Clone(value)
		}
		return target
	case []any:
		target := make([]any, len(s))
		for i, value := range s {
			target[i] = Clone(value)
		}
		return target
	default:
		return source
	}
}
