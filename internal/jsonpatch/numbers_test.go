package jsonpatch

import (
	"encoding/json/v2"
	"testing"
)

func decodePatch(t *testing.T, data string) Patch {
	t.Helper()
	var patch Patch
	if err := unmarshal([]byte(data), &patch); err != nil {
		t.Fatal(err)
	}
	return patch
}

// TestLargeIntegerPrecision ensures numbers are decoded losslessly (as
// json.Number) so integers beyond 2^53 survive a patch round-trip.
func TestLargeIntegerPrecision(t *testing.T) {
	const id = "9007199254740993" // 2^53 + 1

	patch := decodePatch(t, `[{"op":"add","path":"/id","value":`+id+`}]`)
	result, results, err := ApplyPatch(map[string]any{}, patch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operation should succeed, got %v", results)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, string(encoded), `{"id":`+id+`}`, "large integer should be preserved exactly")
}

// TestLargeIntegerEquality ensures two numerically equal large integers compare
// equal while adjacent ones do not (no float64 conflation).
func TestLargeIntegerEquality(t *testing.T) {
	var doc any
	if err := unmarshal([]byte(`{"id":9007199254740993}`), &doc); err != nil {
		t.Fatal(err)
	}

	equalPatch := decodePatch(t, `[{"op":"test","path":"/id","value":9007199254740993}]`)
	adjacentPatch := decodePatch(t, `[{"op":"test","path":"/id","value":9007199254740992}]`)

	_, results, err := ApplyPatch(doc, equalPatch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("equal large integers should pass test, got %v", results)
	}

	_, results, err = ApplyPatch(doc, adjacentPatch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if allNil(results) {
		t.Fatal("adjacent large integers should not compare equal")
	}
}
