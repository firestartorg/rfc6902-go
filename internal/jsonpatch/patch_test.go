package jsonpatch

import (
	"reflect"
	"testing"
)

func TestBrokenAdd(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{AddOperation{Path: "/a/b", Value: 1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestAddToRootPointer(t *testing.T) {
	user := map[string]any{"a": 1}
	result, results, err := ApplyPatch(user, Patch{AddOperation{Path: "", Value: map[string]any{"b": 2}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"a": 1}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestRemoveAtRootPointer(t *testing.T) {
	user := map[string]any{"a": 1}
	result, results, err := ApplyPatch(user, Patch{RemoveOperation{Path: ""}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"a": 1}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestMoveToRootPointer(t *testing.T) {
	user := map[string]any{"a": 1}
	result, results, err := ApplyPatch(user, Patch{MoveOperation{From: "/a", Path: ""}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"a": 1}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestCopyToRootPointer(t *testing.T) {
	user := map[string]any{"a": 1}
	result, results, err := ApplyPatch(user, Patch{CopyOperation{From: "/a", Path: ""}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"a": 1}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenAddArrayDoesNotExist(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{AddOperation{Path: "/tag/-", Value: 1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestWorkingAddArrayExists(t *testing.T) {
	user := map[string]any{"id": "chbrown", "tag": []any{}}
	result, results, err := ApplyPatch(user, Patch{AddOperation{Path: "/tag/-", Value: 123}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, results, []error{nil}, "should succeed")
	requireEqual(t, result, map[string]any{"id": "chbrown", "tag": []any{123}}, "should add tag")
}

func TestWorkingAddArrayExistsNonEmpty(t *testing.T) {
	user := map[string]any{"id": "chbrown", "tag": []any{999}}
	result, results, err := ApplyPatch(user, Patch{AddOperation{Path: "/tag/-", Value: 123}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, results, []error{nil}, "should succeed")
	requireEqual(t, result, map[string]any{"id": "chbrown", "tag": []any{999, 123}}, "should add tag")
}

func TestBrokenRemove(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{RemoveOperation{Path: "/name"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenReplace(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{ReplaceOperation{Path: "/name", Value: 1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenReplaceArray(t *testing.T) {
	users := []any{map[string]any{"id": "chbrown"}}
	result, results, err := ApplyPatch(users, Patch{ReplaceOperation{Path: "/1", Value: map[string]any{"id": "chbrown2"}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, []any{map[string]any{"id": "chbrown"}}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenMoveFrom(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{MoveOperation{From: "/name", Path: "/id"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenMovePath(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{MoveOperation{From: "/id", Path: "/a/b"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestMoveIntoOwnChild(t *testing.T) {
	// RFC 6902 4.4: the "from" location MUST NOT be a proper prefix of "path".
	doc := map[string]any{"a": map[string]any{"b": map[string]any{}}}
	result, results, err := ApplyPatch(doc, Patch{MoveOperation{From: "/a", Path: "/a/b/c"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"a": map[string]any{"b": map[string]any{}}}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenCopyFrom(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{CopyOperation{From: "/name", Path: "/id"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestBrokenCopyPath(t *testing.T) {
	user := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(user, Patch{CopyOperation{From: "/id", Path: "/a/b"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown"}, "should change nothing")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

// Option: ImplicitArrayCreation

func TestImplicitCreatesMissingTopLevelArray(t *testing.T) {
	object := map[string]any{}
	result, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/identifier/-", Value: map[string]any{"system": "mrn"}}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, results, []error{nil}, "should succeed")
	requireEqual(t, result, map[string]any{"identifier": []any{map[string]any{"system": "mrn"}}}, "should create array")
}

func TestImplicitCreatesMissingNestedArray(t *testing.T) {
	object := map[string]any{"meta": map[string]any{}}
	result, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/meta/tag/-", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, results, []error{nil}, "should succeed")
	requireEqual(t, result, map[string]any{"meta": map[string]any{"tag": []any{"x"}}}, "should create nested array")
}

func TestImplicitMultipleAppends(t *testing.T) {
	object := map[string]any{"meta": map[string]any{}}
	result, results, err := ApplyPatch(object, Patch{
		AddOperation{Path: "/meta/tag/-", Value: "a"},
		AddOperation{Path: "/meta/tag/-", Value: "b"},
	}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, results, []error{nil, nil}, "should succeed")
	requireEqual(t, result, map[string]any{"meta": map[string]any{"tag": []any{"a", "b"}}}, "should append twice")
}

func TestImplicitFailsWhenParentMissing(t *testing.T) {
	object := map[string]any{}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/meta/tag/-", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsWhenParentIsValue(t *testing.T) {
	object := map[string]any{"meta": true}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/meta/tag/-", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsWhenParentIsArray(t *testing.T) {
	object := map[string]any{"meta": []any{}}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/meta/tag/-", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsWhenParentIsNull(t *testing.T) {
	object := map[string]any{"meta": nil}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/meta/tag/-", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsOnExistingNonArrayValue(t *testing.T) {
	object := map[string]any{"a": 1}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/a/-", Value: 2}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsOnExistingNonArrayObject(t *testing.T) {
	object := map[string]any{"a": map[string]any{"b": 2}}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/a/-", Value: 2}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

func TestImplicitFailsOnNonTerminalSlashDash(t *testing.T) {
	object := map[string]any{}
	_, results, err := ApplyPatch(object, Patch{AddOperation{Path: "/list/-/name", Value: "x"}}, Options{ImplicitArrayCreation: true})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should fail")
}

// A small guard that the root-array result return actually works, per deviation #1.
func TestApplyPatchRootArrayReturnsResult(t *testing.T) {
	input := []any{"A", "B"}
	result, results, err := ApplyPatch(input, Patch{
		AddOperation{Path: "/0", Value: "B"},
		RemoveOperation{Path: "/2"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("all operations should succeed, got %v", results)
	}
	requireEqual(t, result, []any{"B", "A"}, "returned result should reflect root array mutation")
	if !reflect.DeepEqual(input, []any{"A", "B"}) {
		t.Fatalf("original slice header should be unchanged, got %#v", input)
	}
}
