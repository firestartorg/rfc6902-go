package jsonpatch

import (
	"testing"

	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
)

func TestArrayDiffPatchRoundtrip(t *testing.T) {
	pairs := []struct {
		name   string
		input  []any
		output []any
	}{
		{"A Z Z to A", []any{"A", "Z", "Z"}, []any{"A"}},
		{"A B to B A", []any{"A", "B"}, []any{"B", "A"}},
		{"empty to A B", []any{}, []any{"A", "B"}},
		{"B A M to M A A", []any{"B", "A", "M"}, []any{"M", "A", "A"}},
		{"A A R to empty", []any{"A", "A", "R"}, []any{}},
		{"A B C to B C D", []any{"A", "B", "C"}, []any{"B", "C", "D"}},
		{"A C to A B C", []any{"A", "C"}, []any{"A", "B", "C"}},
		{"A B C to A Z", []any{"A", "B", "C"}, []any{"A", "Z"}},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			patch := CreatePatch(pair.input, pair.output, nil)
			result, results, err := ApplyPatch(jsonutil.Clone(pair.input), patch, Options{})
			if err != nil {
				t.Fatalf("apply patch: %v", err)
			}
			if !allNil(results) {
				t.Fatalf("all operations should succeed, got %v", results)
			}
			requireEqual(t, result, pair.output, "should apply produced patch to arrive at output")
		})
	}
}
