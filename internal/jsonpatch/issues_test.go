package jsonpatch

import (
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"testing"
	"time"
)

func checkRoundtrip(t *testing.T, input, output any, expected Patch, diff Diff) {
	t.Helper()
	actual := CreatePatch(input, output, diff)
	checkRoundtripPatch(t, input, output, expected, actual)
}

func checkRoundtripPatch(t *testing.T, input, output any, expected, actual Patch) {
	t.Helper()
	requireEqual(t, actual, expected, "should produce patch equal to expectation")
	result, results, err := ApplyPatch(jsonutil.Clone(input), actual, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, output, "should apply patch to arrive at output")
	if len(results) != len(actual) {
		t.Fatalf("should apply all patches: got %d results for %d operations", len(results), len(actual))
	}
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
}

func TestIssue3(t *testing.T) {
	input := map[string]any{"arr": []any{"1", "2", "2"}}
	output := map[string]any{"arr": []any{"1"}}
	expected := Patch{
		RemoveOperation{Path: "/arr/1"},
		RemoveOperation{Path: "/arr/1"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue4(t *testing.T) {
	input := []any{"A", "B"}
	output := []any{"B", "A"}
	expected := Patch{
		AddOperation{Path: "/0", Value: "B"},
		RemoveOperation{Path: "/2"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue5(t *testing.T) {
	input := []any{}
	output := []any{"A", "B"}
	expected := Patch{
		AddOperation{Path: "/-", Value: "A"},
		AddOperation{Path: "/-", Value: "B"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue9(t *testing.T) {
	input := []any{
		map[string]any{"A": 1, "B": 2},
		map[string]any{"C": 3},
	}
	output := []any{
		map[string]any{"A": 1, "B": 20},
		map[string]any{"C": 3},
	}
	expected := Patch{ReplaceOperation{Path: "/0/B", Value: 20}}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue12(t *testing.T) {
	input := map[string]any{"name": "ABC", "repositories": []any{"a", "e"}}
	output := map[string]any{"name": "ABC", "repositories": []any{"a", "b", "c", "d", "e"}}
	expected := Patch{
		AddOperation{Path: "/repositories/1", Value: "b"},
		AddOperation{Path: "/repositories/2", Value: "c"},
		AddOperation{Path: "/repositories/3", Value: "d"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}

func dateDiff(input, output any, ptr *Pointer, diff Diff) ([]Operation, bool) {
	in, inOK := input.(time.Time)
	out, outOK := output.(time.Time)
	if inOK && outOK && !in.Equal(out) {
		return []Operation{ReplaceOperation{Path: ptr.String(), Value: out}}, true
	}
	return nil, false
}

func TestIssue15(t *testing.T) {
	input := map[string]any{"date": time.Unix(0, 0)}
	output := map[string]any{"date": time.Unix(1, 0)}
	expected := Patch{ReplaceOperation{Path: "/date", Value: time.Unix(1, 0)}}
	checkRoundtrip(t, input, output, expected, dateDiff)
}

func TestIssue15Array(t *testing.T) {
	input := []any{time.Unix(0, 0)}
	output := []any{time.Unix(1, 0)}
	expected := Patch{ReplaceOperation{Path: "/0", Value: time.Unix(1, 0)}}
	checkRoundtrip(t, input, output, expected, dateDiff)
}

// customDiff29 short-circuits recursion when the last token in the current
// pointer is the key "stop_recursing", so that key's values are compared as
// primitives rather than objects/arrays.
func customDiff29(input, output any, ptr *Pointer, diff Diff) ([]Operation, bool) {
	if ptr.Tokens[len(ptr.Tokens)-1] == "stop_recursing" {
		return []Operation{ReplaceOperation{Path: ptr.String(), Value: output}}, true
	}
	return nil, false
}

func TestIssue29(t *testing.T) {
	input := map[string]any{
		"normal":         []any{"a", "b"},
		"stop_recursing": []any{"a", "b"},
	}
	output := map[string]any{
		"normal":         []any{"a"},
		"stop_recursing": []any{"a"},
	}
	expected := Patch{
		RemoveOperation{Path: "/normal/1"},
		ReplaceOperation{Path: "/stop_recursing", Value: []any{"a"}},
	}
	actual := CreatePatch(input, output, customDiff29)
	checkRoundtripPatch(t, input, output, expected, actual)

	nestedInput := map[string]any{"root": input}
	nestedOutput := map[string]any{"root": output}
	nestedExpected := Patch{
		RemoveOperation{Path: "/root/normal/1"},
		ReplaceOperation{Path: "/root/stop_recursing", Value: []any{"a"}},
	}
	nestedActual := CreatePatch(nestedInput, nestedOutput, customDiff29)
	checkRoundtripPatch(t, nestedInput, nestedOutput, nestedExpected, nestedActual)
}

func TestIssue32(t *testing.T) {
	input := "a"
	output := "b"
	expected := Patch{ReplaceOperation{Path: "", Value: "b"}}
	actual := CreatePatch(input, output, nil)
	requireEqual(t, actual, expected, "should produce patch equal to expectation")
	result, results, err := ApplyPatch(input, actual, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, "a", "should not change input")
	requireEqual(t, resultNames(results), []any{"MissingError"}, "should result in MissingError")
}

func TestIssue33(t *testing.T) {
	object := map[string]any{"root": map[string]any{"0": 4}}
	array := map[string]any{"root": []any{4}}
	checkRoundtrip(t, object, array, Patch{
		ReplaceOperation{Path: "/root", Value: []any{4}},
	}, nil)
	checkRoundtrip(t, array, object, Patch{
		ReplaceOperation{Path: "/root", Value: map[string]any{"0": 4}},
	}, nil)
}

func TestIssue34(t *testing.T) {
	// JS: `const output = [3, 4]; delete output[0]` yields a sparse array whose
	// element 0 is undefined. Go has no sparse arrays, so the hole is modeled
	// as nil (which diffArrays treats as a wholesale replacement, as in JS).
	input := []any{3, 4}
	output := []any{nil, 4}
	expected := Patch{ReplaceOperation{Path: "/0", Value: nil}}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue35(t *testing.T) {
	// JS input has `image: undefined`, which JSON semantics treat as an absent
	// key; in Go we simply omit it. `cat: null` is a genuine null.
	input := map[string]any{"name": "bob", "cat": nil}
	output := map[string]any{"name": "bob", "image": "foo.jpg", "cat": "nikko"}
	expected := Patch{
		AddOperation{Path: "/image", Value: "foo.jpg"},
		ReplaceOperation{Path: "/cat", Value: "nikko"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue36(t *testing.T) {
	// JS: `const input = [undefined, 'B']`
	input := []any{nil, "B"}
	output := []any{"A", "B"}
	expected := Patch{ReplaceOperation{Path: "/0", Value: "A"}}
	checkRoundtrip(t, input, output, expected, nil)
}

func TestIssue37(t *testing.T) {
	value := map[string]any{"id": "chbrown"}
	result, results, err := ApplyPatch(value, Patch{CopyOperation{From: "/id", Path: "/name"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"id": "chbrown", "name": "chbrown"}, "should apply patch to arrive at output")
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
}

func TestIssue38(t *testing.T) {
	value := map[string]any{
		"current": map[string]any{"timestamp": 23},
		"history": []any{},
	}
	result, results, err := ApplyPatch(value, Patch{
		CopyOperation{From: "/current", Path: "/history/-"},
		ReplaceOperation{Path: "/current/timestamp", Value: 24},
		CopyOperation{From: "/current", Path: "/history/-"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{
		"current": map[string]any{"timestamp": 24},
		"history": []any{
			map[string]any{"timestamp": 23},
			map[string]any{"timestamp": 24},
		},
	}
	requireEqual(t, result, expected, "should apply patch to arrive at output")
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
}

func TestIssue44(t *testing.T) {
	value := map[string]any{}
	author := map[string]any{"firstName": "Chris"}
	result, results, err := ApplyPatch(value, Patch{
		AddOperation{Path: "/author", Value: author},
		AddOperation{Path: "/author/lastName", Value: "Brown"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, result, map[string]any{"author": map[string]any{"firstName": "Chris", "lastName": "Brown"}}, "should apply patch to arrive at output")
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
	requireEqual(t, author, map[string]any{"firstName": "Chris"}, "patch reference should not be changed")
}

// Issues/76 (prototype pollution via "/__proto__/polluted") is deliberately not
// ported: it is a JavaScript-specific security regression test for the
// prototype chain, which does not exist in Go. See the note in pointer.go.

func TestIssue78(t *testing.T) {
	user := map[string]any{"firstName": "Chris"}
	result, results, err := ApplyPatch(user, Patch{
		AddOperation{Path: "/createdAt", Value: time.Date(2010, time.August, 10, 22, 10, 48, 0, time.UTC)},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
	createdAt, ok := result.(map[string]any)["createdAt"].(time.Time)
	if !ok {
		t.Fatalf("createdAt should be a time.Time, got %#v", result)
	}
	if createdAt.UnixMilli() != 1281478248000 {
		t.Fatalf("should add Date recoverably, got %d", createdAt.UnixMilli())
	}
}

func TestIssue97(t *testing.T) {
	commits := []any{}
	user := map[string]any{"firstName": "Chris", "commits": []any{"80f1243"}}
	result, results, err := ApplyPatch(user, Patch{
		ReplaceOperation{Path: "/commits", Value: commits},
		AddOperation{Path: "/commits/-", Value: "5d565c8"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !allNil(results) {
		t.Fatalf("should apply patch successfully: %v", results)
	}
	requireEqual(t, result, map[string]any{"firstName": "Chris", "commits": []any{"5d565c8"}}, "should alter user correctly")
	if len(commits) != 0 {
		t.Fatalf("original array should not be modified, len=%d", len(commits))
	}
}

func TestMinimalArrayDiff(t *testing.T) {
	// diffArrays is documented to compute the shortest sequence of operations;
	// a colliding memoization key made the reference implementation return a
	// sub-optimal patch until it was fixed. Removing the leading 'A' and
	// appending 'C' is two operations, not three.
	input := []any{"A", "B", "A"}
	output := []any{"B", "A", "C"}
	expected := Patch{
		RemoveOperation{Path: "/0"},
		AddOperation{Path: "/-", Value: "C"},
	}
	checkRoundtrip(t, input, output, expected, nil)
}
