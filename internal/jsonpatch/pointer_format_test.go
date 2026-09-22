package jsonpatch

import (
	"os"
	"testing"
)

// jsonPointerFormatSuite mirrors the structure of the JSON Schema Test Suite's
// optional "format": "json-pointer" fixture.
type jsonPointerFormatSuite struct {
	Description string `json:"description"`
	Tests       []struct {
		Description string `json:"description"`
		Data        any    `json:"data"`
		Valid       bool   `json:"valid"`
	} `json:"tests"`
}

// malformedEscapeDivergences lists the RFC 6901 syntax vectors that the
// reference TypeScript implementation (and this port, which preserves its
// behavior) accepts even though they are invalid: PointerFromJSON only checks
// that a pointer is empty or begins with "/", and does not reject a "~" that is
// not followed by "0" or "1".
var malformedEscapeDivergences = map[string]bool{
	"/foo/bar~": true,
	"/~0~":      true,
	"/~0/~":     true,
	"/~2":       true,
	"/~-1":      true,
	"/~~":       true,
}

// TestPointerSyntaxConformance checks PointerFromJSON against the JSON Schema
// Test Suite's string-form JSON Pointer syntax vectors (derived from RFC 6901).
// Non-string data is ignored, since this API only accepts strings.
func TestPointerSyntaxConformance(t *testing.T) {
	data, err := os.ReadFile("testdata/json-pointer-format.json")
	if err != nil {
		t.Fatal(err)
	}
	var suites []jsonPointerFormatSuite
	if err := unmarshal(data, &suites); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, suite := range suites {
		for _, testCase := range suite.Tests {
			pointer, ok := testCase.Data.(string)
			if !ok {
				continue // "all string formats ignore <non-string>"
			}
			checked++
			t.Run(testCase.Description, func(t *testing.T) {
				if !testCase.Valid && malformedEscapeDivergences[pointer] {
					t.Skip("reference implementation accepts malformed '~' escapes; RFC 6901 rejects them")
				}
				_, err := PointerFromJSON(pointer)
				if testCase.Valid && err != nil {
					t.Errorf("%q should be accepted, got %v", pointer, err)
				}
				if !testCase.Valid && err == nil {
					t.Errorf("%q should be rejected", pointer)
				}
			})
		}
	}
	if checked == 0 {
		t.Fatal("no string-form JSON Pointer vectors were checked")
	}
}
