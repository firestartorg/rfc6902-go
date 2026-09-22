// Package rfc6902 implements RFC 6902 (JSON Patch) and RFC 6901 (JSON Pointer).
//
// It is a Go port of the TypeScript library https://github.com/chbrown/rfc6902,
// preserving its public API surface and behavior.
//
// The implementation lives in the internal package internal/jsonpatch; this
// package re-exports its public API as type aliases and thin wrappers.
package rfc6902

import jsonpatch "github.com/firestartorg/rfc6902-go/internal/jsonpatch"

// Pointer and its associated values mirror pointer.ts / the JSON Pointer API.
type (
	// Pointer is a JSON Pointer (RFC 6901) representation.
	Pointer = jsonpatch.Pointer
	// PointerEvaluation is the result of evaluating a Pointer.
	PointerEvaluation = jsonpatch.PointerEvaluation
)

// NewPointer returns the root pointer, whose only token is "".
func NewPointer() *Pointer {
	return (*Pointer)(jsonpatch.NewPointer())
}

// PointerFromJSON parses a properly escaped JSON Pointer string.
func PointerFromJSON(path string) (*Pointer, error) {
	pointer, err := jsonpatch.PointerFromJSON(path)
	if err != nil {
		return nil, err
	}
	return (*Pointer)(pointer), nil
}

// Operations and patches mirror diff.ts / the JSON Patch document model.
type (
	// Operation is any RFC 6902 operation.
	Operation = jsonpatch.Operation
	// Patch is a list of operations.
	Patch = jsonpatch.Patch
	// AddOperation adds a value at Path.
	AddOperation = jsonpatch.AddOperation
	// RemoveOperation removes the value at Path.
	RemoveOperation = jsonpatch.RemoveOperation
	// ReplaceOperation replaces the value at Path.
	ReplaceOperation = jsonpatch.ReplaceOperation
	// MoveOperation moves the value at From to Path.
	MoveOperation = jsonpatch.MoveOperation
	// CopyOperation copies the value at From to Path.
	CopyOperation = jsonpatch.CopyOperation
	// TestOperation tests that the value at Path equals Value.
	TestOperation = jsonpatch.TestOperation
	// Diff is a user-supplied diff function.
	Diff = jsonpatch.Diff
)

// Options and the error types mirror patch.ts.
type (
	// Options customizes patch application behavior.
	Options = jsonpatch.Options
	// MissingError indicates that a value required at a path is absent.
	MissingError = jsonpatch.MissingError
	// TestError indicates that a "test" operation failed.
	TestError = jsonpatch.TestError
	// InvalidOperationError indicates a malformed or unrecognized operation.
	InvalidOperationError = jsonpatch.InvalidOperationError
)

// ApplyPatch applies a JSON Patch to object.
//
// This method mutates the target document in place wherever possible. Because a
// top-level []any cannot be grown or shrunk through a copied slice value, the
// resulting document is also returned: callers whose document is a top-level
// array must use the returned result rather than assuming object was mutated.
//
// It returns one result per operation: nil on success, otherwise a
// *MissingError, *InvalidOperationError, or *TestError. A malformed JSON
// Pointer path aborts the whole call and is returned as err.
func ApplyPatch(object any, patch Patch, options Options) (result any, results []error, err error) {
	return jsonpatch.ApplyPatch(object, jsonpatch.Patch(patch), jsonpatch.Options(options))
}

// CreatePatch produces a JSON Patch that transforms input into output.
//
// diff, if non-nil, is called on each pair of comparable non-primitive nodes,
// producing nested patches; returning ok == false falls back to the default
// behavior.
func CreatePatch(input, output any, diff Diff) Patch {
	return Patch(jsonpatch.CreatePatch(input, output, jsonpatch.Diff(diff)))
}

// CreateTests produces a list of test operations that verify existing values in
// input against their captured state, for each destructive operation in patch.
func CreateTests(input any, patch Patch) []TestOperation {
	return []TestOperation(jsonpatch.CreateTests(input, jsonpatch.Patch(patch)))
}
