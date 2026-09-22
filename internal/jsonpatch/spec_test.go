package jsonpatch

import (
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"os"
	"reflect"
	"slices"
	"testing"
)

type specCase struct {
	Comment  string `json:"comment"`
	Name     string `json:"name"`
	Input    any    `json:"input"`
	Patch    Patch  `json:"patch"`
	Output   any    `json:"output"`
	Results  []any  `json:"results"`
	Diffable bool   `json:"diffable"`
}

func loadSpec(t *testing.T) ([]specCase, []map[string]any) {
	t.Helper()
	data, err := os.ReadFile("testdata/spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []specCase
	if err := unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var raw []map[string]any
	if err := unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	return cases, raw
}

func TestJSONPointerRFCExamples(t *testing.T) {
	obj := map[string]any{
		"foo":  []any{"bar", "baz"},
		"":     0,
		"a/b":  1,
		"c%d":  2,
		"e^f":  3,
		"g|h":  4,
		"i\\j": 5,
		"k'l":  6,
		" ":    7,
		"m~n":  8,
	}
	pointers := []struct {
		path     string
		expected any
	}{
		{"", obj},
		{"/foo", []any{"bar", "baz"}},
		{"/foo/0", "bar"},
		{"/", 0},
		{"/a~1b", 1},
		{"/c%d", 2},
		{"/e^f", 3},
		{"/g|h", 4},
		{"/i\\j", 5},
		{"/k'l", 6},
		{"/ ", 7},
		{"/m~0n", 8},
	}
	for _, pointer := range pointers {
		p, err := PointerFromJSON(pointer.path)
		if err != nil {
			t.Fatalf("pointer %q: %v", pointer.path, err)
		}
		requireEqual(t, p.Evaluate(obj).Value, pointer.expected, "pointer \""+pointer.path+"\" should evaluate to expected output")
	}
}

func TestJSONPointerPackageExample(t *testing.T) {
	obj := map[string]any{
		"first": "chris",
		"last":  "brown",
		"github": map[string]any{
			"account": map[string]any{
				"id":     "chbrown",
				"handle": "@chbrown",
			},
			"repos": []any{"amulet", "twilight", "rfc6902"},
			"stars": []any{
				map[string]any{"owner": "raspberrypi", "repo": "userland"},
				map[string]any{"owner": "angular", "repo": "angular.js"},
			},
		},
		"github/account": "deprecated",
	}
	pointers := []struct {
		path     string
		expected any
	}{
		{"/first", "chris"},
		{"/github~1account", "deprecated"},
		{"/github/account/handle", "@chbrown"},
		{"/github/repos", []any{"amulet", "twilight", "rfc6902"}},
		{"/github/repos/2", "rfc6902"},
		{"/github/stars/0/repo", "userland"},
	}
	for _, pointer := range pointers {
		p, err := PointerFromJSON(pointer.path)
		if err != nil {
			t.Fatalf("pointer %q: %v", pointer.path, err)
		}
		requireEqual(t, p.Evaluate(obj).Value, pointer.expected, "pointer \""+pointer.path+"\" should evaluate to expected output")
	}
}

func TestSpecificationFormat(t *testing.T) {
	_, raw := loadSpec(t)
	if len(raw) != 19 {
		t.Fatalf("should have 19 items, got %d", len(raw))
	}
	props := []string{"diffable", "input", "name", "output", "patch", "results"}
	for _, spec := range raw {
		keys := []string{}
		for key := range spec {
			if key != "comment" {
				keys = append(keys, key)
			}
		}
		slices.Sort(keys)
		name, _ := spec["name"].(string)
		requireEqual(t, keys, props, "\""+name+"\" should have items with specific properties")
	}
}

func TestSpecPatches(t *testing.T) {
	cases, _ := loadSpec(t)
	for _, spec := range cases {
		t.Run("patch "+spec.Name, func(t *testing.T) {
			result, results, err := ApplyPatch(jsonutil.Clone(spec.Input), spec.Patch, Options{})
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}
			requireEqual(t, result, spec.Output, "should equal expected output after applying patches")
			requireEqual(t, resultNames(results), spec.Results, "should produce expected results")
		})
	}
}

func TestSpecDiffs(t *testing.T) {
	cases, _ := loadSpec(t)
	for _, spec := range cases {
		if !spec.Diffable {
			continue
		}
		t.Run("diff "+spec.Name, func(t *testing.T) {
			actual := CreatePatch(spec.Input, spec.Output, nil)
			expected := Patch{}
			for _, operation := range spec.Patch {
				if operation.OpName() != "test" {
					expected = append(expected, operation)
				}
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("should produce diff equal to spec patch: got %#v, want %#v", actual, expected)
			}
		})
	}
}
