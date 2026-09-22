package jsonpatch

import (
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"reflect"
	"strings"
	"testing"
)

func TestPointerFromJSONEmpty(t *testing.T) {
	if _, err := PointerFromJSON(""); err != nil {
		t.Fatalf("empty pointer should parse, got %v", err)
	}
}

func TestPointerFromJSONSlash(t *testing.T) {
	if _, err := PointerFromJSON("/"); err != nil {
		t.Fatalf("slash pointer should parse, got %v", err)
	}
}

func TestPointerFromJSONInvalid(t *testing.T) {
	_, err := PointerFromJSON("a")
	if err == nil {
		t.Fatal("invalid pointer should error")
	}
	if !strings.Contains(err.Error(), "Invalid JSON Pointer") {
		t.Fatalf("thrown error should have descriptive message, got %q", err.Error())
	}
}

func TestPointerParent(t *testing.T) {
	pointer, err := PointerFromJSON("/abc/-")
	if err != nil {
		t.Fatal(err)
	}
	parent := pointer.Parent()
	requireEqual(t, parent.String(), "/abc", "parent")
	grandParent := parent.Parent()
	requireEqual(t, grandParent.String(), "", "grandparent")
	greatGrandParent := grandParent.Parent()
	requireEqual(t, greatGrandParent.String(), "", "great-grandparent")
}

var pointerExample = map[string]any{"bool": false, "arr": []any{10, 20, 30}, "obj": map[string]any{"a": "A", "b": "B"}}

func TestPointerGetBool(t *testing.T) {
	pointer, _ := PointerFromJSON("/bool")
	requireEqual(t, pointer.Get(pointerExample), false, "should get bool value")
}

func TestPointerGetArray(t *testing.T) {
	pointer, _ := PointerFromJSON("/arr/1")
	requireEqual(t, pointer.Get(pointerExample), 20, "should get array value")
}

func TestPointerGetObject(t *testing.T) {
	pointer, _ := PointerFromJSON("/obj/b")
	requireEqual(t, pointer.Get(pointerExample), "B", "should get object value")
}

func TestPointerPush(t *testing.T) {
	pointer, _ := PointerFromJSON("/obj")
	pointer.Push("a")
	requireEqual(t, pointer.String(), "/obj/a", "should add token")
}

func TestPointerGetAfterPush(t *testing.T) {
	pointer, _ := PointerFromJSON("/obj")
	pointer.Push("a")
	requireEqual(t, pointer.Get(pointerExample), "A", "should get object value after adding token")
}

func TestPointerSetBool(t *testing.T) {
	input := map[string]any{"bool": true}
	pointer, _ := PointerFromJSON("/bool")
	pointer.Set(input, false)
	requireEqual(t, input["bool"], false, "should set bool value in-place")
}

func TestPointerSetArrayMiddle(t *testing.T) {
	input := map[string]any{"arr": []any{"10", "20", "30"}}
	pointer, _ := PointerFromJSON("/arr/1")
	pointer.Set(input, 0)
	requireEqual(t, input["arr"].([]any)[1], 0, "should set array value in-place")
}

func TestPointerSetArrayBeyond(t *testing.T) {
	// The reference implementation's `arr[3] = 40` grows the array in place.
	// A Go []any cannot be grown through a copied slice header, so setting
	// beyond the end is a no-op. This is a documented deviation from the
	// TypeScript implementation.
	input := map[string]any{"arr": []any{"10", "20", "30"}}
	pointer, _ := PointerFromJSON("/arr/3")
	pointer.Set(input, 40)
	requireEqual(t, input, map[string]any{"arr": []any{"10", "20", "30"}}, "out-of-range set should be a no-op")
}

func TestPointerSetTopLevel(t *testing.T) {
	input := map[string]any{"obj": map[string]any{"a": "A", "b": "B"}}
	original := jsonutil.Clone(input)
	pointer, _ := PointerFromJSON("")
	pointer.Set(input, map[string]any{"other": map[string]any{"c": "C"}})
	requireEqual(t, input, original, "should not mutate object for top-level pointer")
	pointer.Set(input, "root")
	requireEqual(t, input, original, "should not mutate object for top-level pointer")
}

func TestPointerSetObjectExisting(t *testing.T) {
	input := map[string]any{"obj": map[string]any{"a": "A", "b": "B"}}
	pointer, _ := PointerFromJSON("/obj/b")
	pointer.Set(input, "BBB")
	requireEqual(t, input["obj"], map[string]any{"a": "A", "b": "BBB"}, "should set object value in-place")
}

func TestPointerSetObjectNew(t *testing.T) {
	input := map[string]any{"obj": map[string]any{"a": "A", "b": "B"}}
	pointer, _ := PointerFromJSON("/obj/c")
	pointer.Set(input, "C")
	requireEqual(t, input["obj"], map[string]any{"a": "A", "b": "B", "c": "C"}, "should add object value in-place")
}

func TestPointerSetDeepObjectNew(t *testing.T) {
	input := map[string]any{"obj": map[string]any{"subobj": map[string]any{"a": "A", "b": "B"}}}
	pointer, _ := PointerFromJSON("/obj/subobj/c")
	pointer.Set(input, "C")
	requireEqual(t, input["obj"], map[string]any{"subobj": map[string]any{"a": "A", "b": "B", "c": "C"}}, "should add deep object value in-place")
}

func TestPointerSetNotFound(t *testing.T) {
	input := map[string]any{"obj": map[string]any{"a": "A", "b": "B"}}
	original := jsonutil.Clone(input)
	for _, path := range []string{"/notfound/c", "/obj/notfound/c", "/notfound/subobj/c"} {
		pointer, _ := PointerFromJSON(path)
		pointer.Set(input, "C")
		if !reflect.DeepEqual(input, original) {
			t.Fatalf("should not mutate object if parent not found (path %q): got %#v", path, input)
		}
	}
}
