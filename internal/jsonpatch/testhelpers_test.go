package jsonpatch

import "reflect"

// resultName maps an operation result to its error type name (or nil),
// mirroring the reference test suite's resultName helper.
func resultName(err error) any {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *MissingError:
		return "MissingError"
	case *TestError:
		return "TestError"
	case *InvalidOperationError:
		return "InvalidOperationError"
	}
	return err.Error()
}

func resultNames(results []error) []any {
	names := make([]any, len(results))
	for i, err := range results {
		names[i] = resultName(err)
	}
	return names
}

func allNil(results []error) bool {
	for _, err := range results {
		if err != nil {
			return false
		}
	}
	return true
}

func requireEqual(t interface{ Fatalf(string, ...any) }, actual, expected any, message string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s: got %#v, want %#v", message, actual, expected)
	}
}
