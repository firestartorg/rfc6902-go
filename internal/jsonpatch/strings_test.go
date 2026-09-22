package jsonpatch

import (
	"encoding/json/v2"
	"testing"
)

// TestPointerTokenWithQuote verifies that a double quote needs no escaping in a
// JSON Pointer token (RFC 6901 only escapes '~' and '/').
func TestPointerTokenWithQuote(t *testing.T) {
	doc := map[string]any{`quote"key`: 1}
	pointer, err := PointerFromJSON(`/quote"key`)
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, pointer.Get(doc), 1, "pointer token containing a quote")
}

// TestPatchValueWithQuote verifies that a JSON Patch value containing escaped
// quotes round-trips through decoding and application unchanged.
func TestPatchValueWithQuote(t *testing.T) {
	patchJSON := []byte(`[{"op":"add","path":"/msg","value":"he said \"hi\""}]`)
	var patch Patch
	if err := unmarshal(patchJSON, &patch); err != nil {
		t.Fatal(err)
	}
	result, results, err := ApplyPatch(map[string]any{}, patch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operation should succeed, got %v", results)
	}
	requireEqual(t, result, map[string]any{"msg": `he said "hi"`}, "quoted patch value")
}

// TestPatchValueWithBackslash verifies escaped backslashes behave too.
func TestPatchValueWithBackslash(t *testing.T) {
	patchJSON := []byte(`[{"op":"add","path":"/msg","value":"back\\slash"}]`)
	var patch Patch
	if err := unmarshal(patchJSON, &patch); err != nil {
		t.Fatal(err)
	}
	result, results, err := ApplyPatch(map[string]any{}, patch, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operation should succeed, got %v", results)
	}
	requireEqual(t, result, map[string]any{"msg": `back\slash`}, "backslashed patch value")
}

// TestDiffWithQuotesRoundtripsThroughJSON checks that a generated patch whose
// key and value contain quotes survives JSON marshal/unmarshal and still applies.
func TestDiffWithQuotesRoundtripsThroughJSON(t *testing.T) {
	output := map[string]any{`a"b`: `x"y`}
	patch := CreatePatch(map[string]any{}, output, nil)

	encoded, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Patch
	if err := unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	result, results, err := ApplyPatch(map[string]any{}, decoded, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("operations should succeed, got %v", results)
	}
	requireEqual(t, result, output, "quotes survive JSON round-trip")
}
