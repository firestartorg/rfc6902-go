package jsonpatch

import (
	"fmt"

	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
)

// Options customizes patch application behavior.
type Options struct {
	// ImplicitArrayCreation, when true, makes "add" operations whose path ends
	// in "/-" implicitly create an empty array where possible. Only the leaf
	// array is inferred; missing parent objects still produce errors.
	ImplicitArrayCreation bool
}

// MissingError indicates that a value required at a path is absent.
type MissingError struct {
	Path string
}

func (e *MissingError) Error() string {
	return "Value required at path: " + e.Path
}

// TestError indicates that a "test" operation failed.
type TestError struct {
	Actual   any
	Expected any
}

func (e *TestError) Error() string {
	return fmt.Sprintf("Test failed: %v != %v", e.Actual, e.Expected)
}

// InvalidOperationError indicates a malformed or unrecognized operation.
type InvalidOperationError struct {
	Operation Operation
}

func (e *InvalidOperationError) Error() string {
	return "Invalid operation: " + e.Operation.OpName()
}

// setAt stores value at the location named by ptr, returning a possibly-new
// root document. Because a []any cannot be grown through a copied slice header,
// inserting into or removing from an array requires reassigning that array in
// its parent; this walks back up the path to do so.
func setAt(root any, ptr *Pointer, value any) any {
	return setAtTokens(root, ptr.Tokens, value)
}

func setAtTokens(node any, tokens []string, value any) any {
	if len(tokens) <= 1 {
		return value
	}
	key := tokens[1]
	if len(tokens) == 2 {
		switch container := node.(type) {
		case map[string]any:
			container[key] = value
		case []any:
			if index, ok := arrayIndex(key); ok && index < len(container) {
				container[index] = value
			}
		}
		return node
	}
	switch container := node.(type) {
	case map[string]any:
		child, ok := container[key]
		if !ok {
			return node
		}
		container[key] = setAtTokens(child, tokens[1:], value)
	case []any:
		index, ok := arrayIndex(key)
		if !ok || index >= len(container) {
			return node
		}
		container[index] = setAtTokens(container[index], tokens[1:], value)
	}
	return node
}

// arrayInsert inserts value into a at key. key is either "-" (append) or a
// canonical index. It returns the resulting slice.
func arrayInsert(a []any, key string, value any) []any {
	if key == "-" {
		return append(a, value)
	}
	index, _ := arrayIndex(key)
	a = append(a, nil)
	copy(a[index+1:], a[index:])
	a[index] = value
	return a
}

// addAt performs the raw add used by move/copy (no RFC 6902 validation beyond
// the existence of the parent), returning a possibly-new root document.
func addAt(object any, pointer *Pointer, value any) (any, error, error) {
	endpoint := pointer.Evaluate(object)
	if endpoint.Parent == nil || jsonutil.IsUndefined(endpoint.Parent) {
		return object, &MissingError{Path: pointer.String()}, nil
	}
	switch parent := endpoint.Parent.(type) {
	case map[string]any:
		parent[endpoint.Key] = value
		return object, nil, nil
	case []any:
		newArray := arrayInsert(parent, endpoint.Key, value)
		return setAt(object, pointer.Parent(), newArray), nil, nil
	}
	return object, &MissingError{Path: pointer.String()}, nil
}

// removeAt performs the raw remove used by move, returning a possibly-new root.
func removeAt(object any, pointer *Pointer) (any, error, error) {
	endpoint := pointer.Evaluate(object)
	if endpoint.Parent == nil || jsonutil.IsUndefined(endpoint.Value) {
		return object, &MissingError{Path: pointer.String()}, nil
	}
	switch parent := endpoint.Parent.(type) {
	case map[string]any:
		delete(parent, endpoint.Key)
		return object, nil, nil
	case []any:
		index, _ := arrayIndex(endpoint.Key)
		newArray := append(parent[:index], parent[index+1:]...)
		return setAt(object, pointer.Parent(), newArray), nil, nil
	}
	return object, &MissingError{Path: pointer.String()}, nil
}

// add implements the RFC 6902 "add" operation.
func add(object any, operation AddOperation, options Options) (any, error, error) {
	// RFC 6902 §4: every operation object MUST contain a "path" member, and
	// add/replace MUST contain a "value" member. JSON null is a legitimate
	// value; only an absent member is invalid.
	if operation.pathAbsent || operation.valueAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	// Handle implicit array creation for terminal "/-" paths.
	if options.ImplicitArrayCreation && pointer.Tokens[len(pointer.Tokens)-1] == "-" {
		parentEndpoint := pointer.Parent().Evaluate(object)
		if jsonutil.IsUndefined(parentEndpoint.Value) && jsonutil.ObjectType(parentEndpoint.Parent) == "object" {
			parentEndpoint.Parent.(map[string]any)[parentEndpoint.Key] = []any{}
		}
	}
	endpoint := pointer.Evaluate(object)
	// it's not exactly a "MissingError" in the same way that remove is -- more
	// like a MissingParent.
	if endpoint.Parent == nil || jsonutil.IsUndefined(endpoint.Parent) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	// When using implicitArrayCreation, validate that "/-" targets are arrays.
	if options.ImplicitArrayCreation && endpoint.Key == "-" {
		if _, ok := endpoint.Parent.([]any); !ok {
			return object, &MissingError{Path: operation.Path}, nil
		}
	}
	// For arrays, the key must strictly follow RFC 6901 (digits only, no
	// leading zeros) and must not exceed the array's length.
	if array, ok := endpoint.Parent.([]any); ok && endpoint.Key != "-" {
		index, valid := arrayIndex(endpoint.Key)
		if !valid || index > len(array) {
			return object, &MissingError{Path: operation.Path}, nil
		}
	}
	value := jsonutil.Clone(operation.Value)
	switch parent := endpoint.Parent.(type) {
	case map[string]any:
		parent[endpoint.Key] = value
		return object, nil, nil
	case []any:
		newArray := arrayInsert(parent, endpoint.Key, value)
		return setAt(object, pointer.Parent(), newArray), nil, nil
	}
	return object, &MissingError{Path: operation.Path}, nil
}

// remove implements the RFC 6902 "remove" operation.
func remove(object any, operation RemoveOperation, options Options) (any, error, error) {
	if operation.pathAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	endpoint := pointer.Evaluate(object)
	// the root pointer "" has no parent to remove from (mirrors replace)
	if endpoint.Parent == nil {
		return object, &MissingError{Path: operation.Path}, nil
	}
	if jsonutil.IsUndefined(endpoint.Value) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	switch parent := endpoint.Parent.(type) {
	case map[string]any:
		delete(parent, endpoint.Key)
		return object, nil, nil
	case []any:
		index, _ := arrayIndex(endpoint.Key)
		newArray := append(parent[:index], parent[index+1:]...)
		return setAt(object, pointer.Parent(), newArray), nil, nil
	}
	return object, &MissingError{Path: operation.Path}, nil
}

// replace implements the RFC 6902 "replace" operation.
func replace(object any, operation ReplaceOperation, options Options) (any, error, error) {
	if operation.pathAbsent || operation.valueAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	endpoint := pointer.Evaluate(object)
	if endpoint.Parent == nil {
		return object, &MissingError{Path: operation.Path}, nil
	}
	if array, ok := endpoint.Parent.([]any); ok {
		index, valid := arrayIndex(endpoint.Key)
		if !valid || index >= len(array) {
			return object, &MissingError{Path: operation.Path}, nil
		}
	} else if jsonutil.IsUndefined(endpoint.Value) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	setInPlace(endpoint.Parent, endpoint.Key, jsonutil.Clone(operation.Value))
	return object, nil, nil
}

// isProperPrefix reports whether from is a proper prefix of path, i.e. a
// location cannot be moved into one of its own children.
func isProperPrefix(from, path []string) bool {
	if len(from) >= len(path) {
		return false
	}
	for i, token := range from {
		if token != path[i] {
			return false
		}
	}
	return true
}

// move implements the RFC 6902 "move" operation.
func move(object any, operation MoveOperation, options Options) (any, error, error) {
	if operation.pathAbsent || operation.fromAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	fromPointer, err := PointerFromJSON(operation.From)
	if err != nil {
		return object, nil, err
	}
	fromEndpoint := fromPointer.Evaluate(object)
	if jsonutil.IsUndefined(fromEndpoint.Value) {
		return object, &MissingError{Path: operation.From}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	// A location cannot be moved into one of its own children; doing so would
	// detach the destination along with the value and silently lose data.
	if isProperPrefix(fromPointer.Tokens, pointer.Tokens) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	endpoint := pointer.Evaluate(object)
	if endpoint.Parent == nil || jsonutil.IsUndefined(endpoint.Parent) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	value := fromEndpoint.Value
	root, result, fatal := removeAt(object, fromPointer)
	if fatal != nil {
		return object, nil, fatal
	}
	if result != nil {
		return object, result, nil
	}
	return addAt(root, pointer, value)
}

// copy implements the RFC 6902 "copy" operation.
func copyOperation(object any, operation CopyOperation, options Options) (any, error, error) {
	if operation.pathAbsent || operation.fromAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	fromPointer, err := PointerFromJSON(operation.From)
	if err != nil {
		return object, nil, err
	}
	fromEndpoint := fromPointer.Evaluate(object)
	if jsonutil.IsUndefined(fromEndpoint.Value) {
		return object, &MissingError{Path: operation.From}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	endpoint := pointer.Evaluate(object)
	if endpoint.Parent == nil || jsonutil.IsUndefined(endpoint.Parent) {
		return object, &MissingError{Path: operation.Path}, nil
	}
	return addAt(object, pointer, jsonutil.Clone(fromEndpoint.Value))
}

// test implements the RFC 6902 "test" operation.
func test(object any, operation TestOperation, options Options) (any, error, error) {
	if operation.pathAbsent {
		return object, &InvalidOperationError{Operation: operation}, nil
	}
	pointer, err := PointerFromJSON(operation.Path)
	if err != nil {
		return object, nil, err
	}
	endpoint := pointer.Evaluate(object)
	expected := operation.Value
	if operation.valueAbsent {
		expected = jsonutil.Undefined
	}
	if len(defaultDiff(endpoint.Value, expected, NewPointer())) != 0 {
		return object, &TestError{Actual: endpoint.Value, Expected: expected}, nil
	}
	return object, nil, nil
}

// applyOne applies a single operation, returning the (possibly-new) document,
// a per-operation result error, and a fatal error that aborts the whole patch
// call (mirroring the reference implementation's uncaught throw on malformed
// pointer syntax).
func applyOne(object any, operation Operation, options Options) (any, error, error) {
	switch op := operation.(type) {
	case AddOperation:
		return add(object, op, options)
	case *AddOperation:
		return add(object, *op, options)
	case RemoveOperation:
		return remove(object, op, options)
	case *RemoveOperation:
		return remove(object, *op, options)
	case ReplaceOperation:
		return replace(object, op, options)
	case *ReplaceOperation:
		return replace(object, *op, options)
	case MoveOperation:
		return move(object, op, options)
	case *MoveOperation:
		return move(object, *op, options)
	case CopyOperation:
		return copyOperation(object, op, options)
	case *CopyOperation:
		return copyOperation(object, *op, options)
	case TestOperation:
		return test(object, op, options)
	case *TestOperation:
		return test(object, *op, options)
	}
	return object, &InvalidOperationError{Operation: operation}, nil
}
