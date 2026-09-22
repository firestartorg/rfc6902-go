# rfc6902

[![CI](https://github.com/firestartorg/rfc6902-go/actions/workflows/ci.yml/badge.svg)](https://github.com/firestartorg/rfc6902-go/actions/workflows/ci.yml)

Go implementation of [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902) (JSON Patch)
and [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901) (JSON Pointer).

This is an idiomatic Go port of the TypeScript library
[`chbrown/rfc6902`](https://github.com/chbrown/rfc6902) (v5.3.0), preserving its
public behavior and semantics.

## Install

```sh
go get github.com/firestartorg/rfc6902-go
```

```go
import rfc6902 "github.com/firestartorg/rfc6902-go"
```

**Requirements:** Go 1.27 or newer (the implementation uses `encoding/json/v2`).

## Usage

### Apply a patch

```go
doc := map[string]any{"foo": []any{"bar", "baz"}}
patch := rfc6902.Patch{
	rfc6902.AddOperation{Path: "/foo/1", Value: "qux"},
	rfc6902.RemoveOperation{Path: "/foo/3"},
}

result, results, err := rfc6902.ApplyPatch(doc, patch, rfc6902.Options{})
if err != nil {
	log.Fatal(err) // fatal: malformed JSON Pointer syntax
}
for i, opErr := range results {
	if opErr != nil {
		log.Printf("operation %d failed: %v", i, opErr)
	}
}
fmt.Println(result) // map[foo:[bar qux baz]]
```

`ApplyPatch` mutates the document in place wherever possible **and** returns the
resulting document. Because a top-level Go slice cannot be grown through a
copied header, callers whose document is a top-level array must use the returned
`result` rather than assuming `doc` was mutated.

### Create a patch (diff)

```go
input := map[string]any{"a": 1, "b": 2}
output := map[string]any{"a": 1, "b": 3, "c": 4}

patch := rfc6902.CreatePatch(input, output, nil)
// []rfc6902.Operation{...}
```

A custom diff function may be supplied as the third argument (pass `nil` for the
default). Return `ok == false` to fall back to the default behavior.

### Decode an `application/json-patch+json` document

```go
var patch rfc6902.Patch
if err := json.Unmarshal(body, &patch); err != nil {
	log.Fatal(err)
}
```

`Patch` and each operation type implement `MarshalJSON`/`UnmarshalJSON`.

### Create test operations

```go
tests := rfc6902.CreateTests(doc, patch)
// one {"op":"test"} operation per destructive operation, capturing current values
```

### JSON Pointer

```go
pointer, err := rfc6902.PointerFromJSON("/foo/0")
if err != nil {
	log.Fatal(err)
}
value := pointer.Get(doc) // "bar"
```

## Error handling

`ApplyPatch` returns three values:

| Return | Description |
| --- | --- |
| `result any` | The resulting document (see note above about top-level arrays). |
| `results []error` | One entry per operation: `nil` on success, otherwise `*MissingError`, `*TestError`, or `*InvalidOperationError`. |
| `err error` | Fatal error that aborts the whole call, reserved for malformed JSON Pointer syntax. |

```go
var missing *rfc6902.MissingError
if errors.As(results[0], &missing) {
	// missing.Path
}
```

## Options

```go
rfc6902.Options{ImplicitArrayCreation: true}
```

When enabled, `add` operations whose path ends in `/-` create an empty array
where possible (only the leaf array is inferred; missing parent objects still
error).

## Behavior and limitations

- **Root document type changes are not supported.** A `replace`/`add` at the
  root pointer (`""`) returns `*MissingError`; a mutated document cannot change
  its own top-level type. (Adding to / removing from a *top-level array* works
  via the returned `result`.)
- **Malformed JSON Pointer syntax aborts the whole patch call** via the fatal
  `err` return, mirroring the reference implementation.
- **`Pointer.Set` is a no-op for out-of-range array indices**, since a Go slice
  cannot be grown through a copied header (JavaScript grows the array instead).
- **Object keys are diffed in sorted order**, so patches are deterministic;
  operation ordering may differ from the JavaScript implementation, which
  follows insertion order.
- **JS `undefined` semantics are preserved internally**: an absent object member
  or operation `value` is distinct from an explicit JSON `null`.

## Testing

```sh
go test ./...
```

The suite combines the ported upstream tests with vendored external fixtures:

- [`json-patch/json-patch-tests`](https://github.com/json-patch/json-patch-tests)
  (`tests.json` + `spec_tests.json`, pinned commit)
- The JSON Schema Test Suite's `format: json-pointer` syntax vectors

A small number of cases are skipped with documented reasons (root type changes
and invalid-pointer-token behavior inherited from the reference implementation).

## License

[MIT](LICENSE). This project is a port of
[`chbrown/rfc6902`](https://github.com/chbrown/rfc6902), © Christopher Brown,
which is distributed under the MIT license; the original copyright notice is
retained in [`LICENSE`](LICENSE).
