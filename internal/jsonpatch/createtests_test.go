package jsonpatch

import (
	"testing"

	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
)

func TestCreateTestsSimplePatch(t *testing.T) {
	obj := map[string]any{"itemCodes": []any{"123", "456", "789"}}
	patch := Patch{RemoveOperation{Path: "/itemCodes/1"}}
	expected := []TestOperation{{Path: "/itemCodes/1", Value: "456"}}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate the expected test")

	_, results, err := ApplyPatch(obj, Patch{actual[0]}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("tests should apply without errors, got %v", results)
	}
}

func TestCreateTestsComplexPatch(t *testing.T) {
	obj := map[string]any{
		"items": []any{
			map[string]any{"code": "123", "description": "item # 123", "componentCodes": []any{"456", "789"}},
			map[string]any{"code": "456", "description": "item # 456", "componentCodes": []any{"789"}},
			map[string]any{"code": "789", "description": "item # 789", "componentCodes": []any{}},
		},
	}
	patch := Patch{RemoveOperation{Path: "/items/1"}}
	expected := []TestOperation{{
		Path: "/items/1",
		Value: map[string]any{
			"code": "456", "description": "item # 456", "componentCodes": []any{"789"},
		},
	}}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate the expected test")

	_, results, err := ApplyPatch(obj, Patch{actual[0]}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("tests should apply without errors, got %v", results)
	}
}

func TestCreateTestsSimplePatchWithAdd(t *testing.T) {
	obj := map[string]any{"itemCodes": []any{"123", "456", "789"}}
	patch := Patch{AddOperation{Path: "/itemCodes/-", Value: "abc"}}
	expected := []TestOperation{}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate no tests")
}

func TestCreateTestsSimplePatchWithMove(t *testing.T) {
	obj := map[string]any{"itemCodes": []any{"123", "456", "789"}, "alternateItemCodes": []any{"abc"}}
	patch := Patch{MoveOperation{From: "/itemCodes/1", Path: "/alternateItemCodes/0"}}
	expected := []TestOperation{
		{Path: "/alternateItemCodes/0", Value: "abc"},
		{Path: "/itemCodes/1", Value: "456"},
	}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate the expected tests")

	_, results, err := ApplyPatch(obj, Patch{actual[0], actual[1]}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("tests should apply without errors, got %v", results)
	}
}

func TestCreateTestsSimplePatchWithCopy(t *testing.T) {
	obj := map[string]any{"itemCodes": []any{"123", "456", "789"}, "alternateItemCodes": []any{}}
	patch := Patch{CopyOperation{From: "/itemCodes/1", Path: "/alternateItemCodes/0"}}
	expected := []TestOperation{
		{Path: "/alternateItemCodes/0", Value: jsonutil.Undefined},
		{Path: "/itemCodes/1", Value: "456"},
	}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate the expected tests")

	_, results, err := ApplyPatch(obj, Patch{actual[0], actual[1]}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("tests should apply without errors, got %v", results)
	}
}

func TestCreateTestsSimplePatchWithReplace(t *testing.T) {
	obj := map[string]any{"itemCodes": []any{"123", "456", "789"}}
	patch := Patch{ReplaceOperation{Path: "/itemCodes/1", Value: "abc"}}
	expected := []TestOperation{{Path: "/itemCodes/1", Value: "456"}}

	actual := CreateTests(obj, patch)
	requireEqual(t, actual, expected, "patch should generate the expected test")

	_, results, err := ApplyPatch(obj, Patch{actual[0]}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("tests should apply without errors, got %v", results)
	}
}
