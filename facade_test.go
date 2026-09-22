package rfc6902_test

import (
	"errors"
	"reflect"
	"testing"

	rfc6902 "github.com/firestartorg/rfc6902-go"
)

func TestFacadePatchRoundtrip(t *testing.T) {
	input := map[string]any{"foo": []any{"bar", "baz"}}
	output := map[string]any{"foo": []any{"bar", "qux", "baz"}, "extra": true}
	patch := rfc6902.CreatePatch(input, output, nil)

	result, results, err := rfc6902.ApplyPatch(input, patch, rfc6902.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for i, opResult := range results {
		if opResult != nil {
			t.Fatalf("operation %d failed: %v", i, opResult)
		}
	}
	if !reflect.DeepEqual(result, output) {
		t.Fatalf("got %#v, want %#v", result, output)
	}
}

func TestFacadePointer(t *testing.T) {
	pointer, err := rfc6902.PointerFromJSON("/foo/1")
	if err != nil {
		t.Fatal(err)
	}
	got := pointer.Get(map[string]any{"foo": []any{"bar", "baz"}})
	if got != "baz" {
		t.Fatalf("got %#v, want %q", got, "baz")
	}
}

func TestFacadeErrorTypes(t *testing.T) {
	_, results, err := rfc6902.ApplyPatch(
		map[string]any{},
		rfc6902.Patch{rfc6902.RemoveOperation{Path: "/missing"}},
		rfc6902.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	var missing *rfc6902.MissingError
	if len(results) != 1 || !errors.As(results[0], &missing) {
		t.Fatalf("expected a *MissingError, got %#v", results)
	}
	if missing.Path != "/missing" {
		t.Fatalf("got path %q, want %q", missing.Path, "/missing")
	}
}
