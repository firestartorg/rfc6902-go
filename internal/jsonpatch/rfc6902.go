// Package jsonpatch contains the implementation of RFC 6902 (JSON Patch) and
// RFC 6901 (JSON Pointer). The public API is re-exported by the root package
// github.com/firestartorg/rfc6902-go; import that instead.
package jsonpatch

// ApplyPatch applies a JSON Patch to object.
//
// This method mutates the target document in place wherever possible. Because a
// top-level []any cannot be grown or shrunk through a copied slice value, the
// resulting document is also returned: callers whose document is a top-level
// array must use the returned result rather than assuming object was mutated.
//
// It returns one result per operation: nil on success, otherwise a
// *MissingError, *InvalidOperationError, or *TestError. A malformed JSON
// Pointer path aborts the whole call (mirroring the reference implementation's
// uncaught throw) and is returned as err.
func ApplyPatch(object any, patch Patch, options Options) (result any, results []error, err error) {
	result = object
	results = make([]error, 0, len(patch))
	for _, operation := range patch {
		newRoot, opResult, fatal := applyOne(result, operation, options)
		if fatal != nil {
			return result, results, fatal
		}
		result = newRoot
		results = append(results, opResult)
	}
	return result, results, nil
}

// CreatePatch produces a JSON Patch that transforms input into output.
//
// diff, if non-nil, is called on each pair of comparable non-primitive nodes,
// producing nested patches; returning ok == false falls back to the default
// behavior.
func CreatePatch(input, output any, diff Diff) Patch {
	ptr := NewPointer()
	if diff == nil {
		return Patch(diffAny(input, output, ptr, defaultDiff))
	}
	return Patch(asDiffFunc(diff)(input, output, ptr))
}

// createTest builds a test operation from input's current evaluation of path.
func createTest(input any, path string) (TestOperation, bool) {
	pointer, err := PointerFromJSON(path)
	if err != nil {
		return TestOperation{}, false
	}
	endpoint := pointer.Evaluate(input)
	return TestOperation{Path: path, Value: endpoint.Value}, true
}

// CreateTests produces a list of test operations that verify existing values in
// input against their captured state, for each destructive operation in patch.
func CreateTests(input any, patch Patch) []TestOperation {
	tests := []TestOperation{}
	for _, operation := range patch {
		if !isDestructive(operation) {
			continue
		}
		if pathTest, ok := createTest(input, operationPath(operation)); ok {
			tests = append(tests, pathTest)
		}
		if from, ok := operationFrom(operation); ok {
			if fromTest, ok := createTest(input, from); ok {
				tests = append(tests, fromTest)
			}
		}
	}
	return tests
}

func operationPath(operation Operation) string {
	switch op := operation.(type) {
	case AddOperation:
		return op.Path
	case *AddOperation:
		return op.Path
	case RemoveOperation:
		return op.Path
	case *RemoveOperation:
		return op.Path
	case ReplaceOperation:
		return op.Path
	case *ReplaceOperation:
		return op.Path
	case MoveOperation:
		return op.Path
	case *MoveOperation:
		return op.Path
	case CopyOperation:
		return op.Path
	case *CopyOperation:
		return op.Path
	case TestOperation:
		return op.Path
	case *TestOperation:
		return op.Path
	}
	return ""
}

func operationFrom(operation Operation) (string, bool) {
	switch op := operation.(type) {
	case MoveOperation:
		return op.From, true
	case *MoveOperation:
		return op.From, true
	case CopyOperation:
		return op.From, true
	case *CopyOperation:
		return op.From, true
	}
	return "", false
}
