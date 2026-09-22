package jsonpatch

import (
	"testing"

	"github.com/firestartorg/rfc6902-go/internal/jsonutil"
)

// TestEmptyStringKeyPointer covers JSON Pointer paths whose token is the empty
// string: "/" addresses the "" member of the root object, and "/foo/" the ""
// member of the "foo" object.
func TestEmptyStringKeyPointer(t *testing.T) {
	root := map[string]any{"": "foo"}
	pointer, err := PointerFromJSON("/")
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, pointer.Get(root), "foo", "root empty-string key")

	nested := map[string]any{"foo": map[string]any{"": "bar"}}
	nestedPointer, err := PointerFromJSON("/foo/")
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, nestedPointer.Get(nested), "bar", "nested empty-string key")
}

// TestDiffEmptyStringKey checks that a changed "" member produces a
// trailing-slash path and round-trips.
func TestDiffEmptyStringKey(t *testing.T) {
	input := map[string]any{"foo": map[string]any{"a": 1, "": 2}}
	output := map[string]any{"foo": map[string]any{"a": 1, "": 3}}
	patch := CreatePatch(input, output, nil)
	requireEqual(t, patch, Patch{ReplaceOperation{Path: "/foo/", Value: 3}}, "replace empty-string key")

	result, results, err := ApplyPatch(jsonutil.Clone(input), patch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operations should succeed, got %v", results)
	}
	requireEqual(t, result, output, "round-trip empty-string key replace")
}

// TestAddEmptyStringKey checks that adding a "" member produces a
// trailing-slash path and round-trips.
func TestAddEmptyStringKey(t *testing.T) {
	input := map[string]any{"foo": map[string]any{}}
	output := map[string]any{"foo": map[string]any{"": 1}}
	patch := CreatePatch(input, output, nil)
	requireEqual(t, patch, Patch{AddOperation{Path: "/foo/", Value: 1}}, "add empty-string key")

	result, results, err := ApplyPatch(jsonutil.Clone(input), patch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operations should succeed, got %v", results)
	}
	requireEqual(t, result, output, "round-trip empty-string key add")
}
