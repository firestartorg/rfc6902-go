package jsonpatch

import (
	"fmt"
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"os"
	"testing"
)

// See https://github.com/json-patch/json-patch-tests#test-format
type jsonPatchTestCase struct {
	Comment  string `json:"comment"`
	Doc      any    `json:"doc"`
	Patch    Patch  `json:"patch"`
	Expected any    `json:"expected"`
	Error    string `json:"error"`
	Disabled bool   `json:"disabled"`
}

// skipComments lists test cases that are infeasible or intentionally divergent,
// with the same two documented reasons as the reference implementation:
//
//   - Replacing the root document's type (e.g., object -> array) is infeasible
//     for this library, since ApplyPatch mutates the given document in place
//     rather than returning a new one; a mutated document cannot change its own
//     type. (Array root add/remove now works via ApplyPatch's result return, but
//     object<->array root type changes remain infeasible.)
//   - v5 behavior is to abort the whole call on an invalid JSON Pointer path;
//     this is preserved rather than "fixed".
var skipComments = map[string]string{
	"replace object document with array document?":            "infeasible: cannot change a root document's type in place",
	"replace array document with object document?":            "infeasible: cannot change a root document's type in place",
	"replace whole document":                                  "infeasible: cannot change a root document's type in place",
	"replacing the root of the document is possible with add": "infeasible: cannot change a root document's type in place",
	"invalid JSON Pointer token":                              "divergence: invalid pointer syntax aborts the call (as in v5)",
}

func TestJSONPatchTestSuite(t *testing.T) {
	runJSONPatchSuite(t, "tests", "testdata/json-patch-test-suite/tests.json")
	runJSONPatchSuite(t, "spec_tests", "testdata/json-patch-test-suite/spec_tests.json")
}

func runJSONPatchSuite(t *testing.T, suiteName, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []jsonPatchTestCase
	if err := unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for index, testCase := range cases {
		title := fmt.Sprintf("%s[%d]: %s", suiteName, index, testCase.Comment)
		t.Run(title, func(t *testing.T) {
			if testCase.Disabled {
				t.Skip("disabled in the json-patch-tests suite")
			}
			if reason, ok := skipComments[testCase.Comment]; ok {
				t.Skip(reason)
			}
			result, results, fatal := ApplyPatch(jsonutil.Clone(testCase.Doc), testCase.Patch, Options{})
			if testCase.Error != "" {
				if fatal == nil && allNil(results) {
					t.Fatalf("should produce an error: %s", testCase.Error)
				}
				return
			}
			if fatal != nil {
				t.Fatalf("unexpected fatal error: %v", fatal)
			}
			requireEqual(t, result, testCase.Expected, "should equal expected output after applying patches")
		})
	}
}
