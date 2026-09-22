package jsonpatch

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"slices"
	"strconv"
)

// unmarshal decodes JSON into out, preserving numeric precision by decoding
// numbers as json.Number. This matches the byte-level fidelity of the reference
// implementation's JSON parsers and avoids corrupting integers beyond 2^53.
// The v1 decoder also allows duplicate object member names (last name wins),
// matching JavaScript's JSON.parse.
func unmarshal(data []byte, out any) error {
	decoder := jsonv1.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(out)
}

// All diff* functions return a list of operations, often empty.
//
// Each operation is one of the concrete types below, all of which implement
// Operation. This is the idiomatic Go analog of the TypeScript union type.

// Operation is any RFC 6902 operation.
type Operation interface {
	OpName() string
}

// Patch is a list of operations, as accepted/returned by the JSON Patch API.
type Patch []Operation

// AddOperation adds a value at Path.
type AddOperation struct {
	Path  string
	Value any

	pathAbsent  bool
	valueAbsent bool
}

func (o AddOperation) OpName() string { return "add" }

// RemoveOperation removes the value at Path.
type RemoveOperation struct {
	Path string

	pathAbsent bool
}

func (o RemoveOperation) OpName() string { return "remove" }

// ReplaceOperation replaces the value at Path.
type ReplaceOperation struct {
	Path  string
	Value any

	pathAbsent  bool
	valueAbsent bool
}

func (o ReplaceOperation) OpName() string { return "replace" }

// MoveOperation moves the value at From to Path.
type MoveOperation struct {
	Path string
	From string

	pathAbsent bool
	fromAbsent bool
}

func (o MoveOperation) OpName() string { return "move" }

// CopyOperation copies the value at From to Path.
type CopyOperation struct {
	Path string
	From string

	pathAbsent bool
	fromAbsent bool
}

func (o CopyOperation) OpName() string { return "copy" }

// TestOperation tests that the value at Path equals Value.
type TestOperation struct {
	Path  string
	Value any

	pathAbsent  bool
	valueAbsent bool
}

func (o TestOperation) OpName() string { return "test" }

// unknownOperation carries an unrecognized "op" value so that patch application
// can report an InvalidOperationError, mirroring the reference implementation.
type unknownOperation struct {
	name string
}

func (o unknownOperation) OpName() string { return o.name }

func (o unknownOperation) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"op": o.name})
}

// stringField reads a string member, reporting absence for a missing member or
// a member that is not a string (JSON null included).
func stringField(m map[string]any, key string) (string, bool) {
	value, ok := m[key]
	if !ok {
		return "", true
	}
	s, ok := value.(string)
	if !ok {
		return "", true
	}
	return s, false
}

func (o *AddOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
	o.Value, o.valueAbsent = memberValue(m, "value")
}

func (o AddOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("add", o.Path, "", o.Value, o.valueAbsent)
}

func (o *AddOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func (o *RemoveOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
}

func (o RemoveOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("remove", o.Path, "", nil, true)
}

func (o *RemoveOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func (o *ReplaceOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
	o.Value, o.valueAbsent = memberValue(m, "value")
}

func (o ReplaceOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("replace", o.Path, "", o.Value, o.valueAbsent)
}

func (o *ReplaceOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func (o *MoveOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
	o.From, o.fromAbsent = stringField(m, "from")
}

func (o MoveOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("move", o.Path, o.From, nil, true)
}

func (o *MoveOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func (o *CopyOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
	o.From, o.fromAbsent = stringField(m, "from")
}

func (o CopyOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("copy", o.Path, o.From, nil, true)
}

func (o *CopyOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func (o *TestOperation) populate(m map[string]any) {
	o.Path, o.pathAbsent = stringField(m, "path")
	o.Value, o.valueAbsent = memberValue(m, "value")
}

func (o TestOperation) MarshalJSON() ([]byte, error) {
	return marshalOperation("test", o.Path, "", o.Value, o.valueAbsent)
}

func (o *TestOperation) UnmarshalJSON(data []byte) error {
	m, err := objectMembers(data)
	if err != nil {
		return err
	}
	o.populate(m)
	return nil
}

func objectMembers(data []byte) (map[string]any, error) {
	var m map[string]any
	if err := unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// memberValue returns the value of a member and whether the member is present.
// A present member explicitly set to JSON null yields (nil, false).
func memberValue(m map[string]any, key string) (any, bool) {
	if value, ok := m[key]; ok {
		return value, false
	}
	return nil, true
}

// marshalOperation encodes a single operation with the RFC 6902 member names.
func marshalOperation(op, path, from string, value any, omitValue bool) ([]byte, error) {
	m := map[string]any{"op": op, "path": path}
	if op == "move" || op == "copy" {
		m["from"] = from
	}
	if !omitValue && !jsonutil.IsUndefined(value) {
		m["value"] = value
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes an array of operations, dispatching on each "op".
func (p *Patch) UnmarshalJSON(data []byte) error {
	var raw []map[string]any
	if err := unmarshal(data, &raw); err != nil {
		return err
	}
	operations := make(Patch, len(raw))
	for i, m := range raw {
		name, _ := m["op"].(string)
		switch name {
		case "add":
			var op AddOperation
			op.populate(m)
			operations[i] = op
		case "remove":
			var op RemoveOperation
			op.populate(m)
			operations[i] = op
		case "replace":
			var op ReplaceOperation
			op.populate(m)
			operations[i] = op
		case "move":
			var op MoveOperation
			op.populate(m)
			operations[i] = op
		case "copy":
			var op CopyOperation
			op.populate(m)
			operations[i] = op
		case "test":
			var op TestOperation
			op.populate(m)
			operations[i] = op
		default:
			operations[i] = unknownOperation{name: name}
		}
	}
	*p = operations
	return nil
}

// MarshalJSON encodes the patch as a JSON array of operations.
func (p Patch) MarshalJSON() ([]byte, error) {
	operations := make([]Operation, len(p))
	copy(operations, p)
	return json.Marshal(operations)
}

// isDestructive mirrors diff.isDestructive.
func isDestructive(operation Operation) bool {
	switch operation.OpName() {
	case "remove", "replace", "copy", "move":
		return true
	}
	return false
}

// Diff is a user-supplied diff function. Returning ok == false means "no
// opinion; fall back to the default diffAny behavior" (replacing the reference
// implementation's VoidableDiff returning undefined). The final diff argument
// is provided for callers that want to recurse using the same custom diff.
type Diff func(input, output any, ptr *Pointer, diff Diff) (ops []Operation, ok bool)

// diffFunc is the internal recursive diff signature (always returns a patch).
type diffFunc func(input, output any, ptr *Pointer) []Operation

// defaultDiff is the internal recursion entry point for the built-in behavior.
func defaultDiff(input, output any, ptr *Pointer) []Operation {
	return diffAny(input, output, ptr, defaultDiff)
}

// asDiffFunc adapts a user-supplied Diff to the internal diffFunc, falling back
// to diffAny whenever the user expresses "no opinion" (ok == false). The Diff
// handed to the user as its final argument recurses with the same fallback.
func asDiffFunc(diff Diff) diffFunc {
	var recurse diffFunc
	var wrapped Diff
	wrapped = func(input, output any, ptr *Pointer, _ Diff) ([]Operation, bool) {
		return recurse(input, output, ptr), true
	}
	recurse = func(input, output any, ptr *Pointer) []Operation {
		operations, ok := diff(input, output, ptr, wrapped)
		if ok {
			return operations
		}
		return diffAny(input, output, ptr, recurse)
	}
	return recurse
}

// subtract lists the keys in minuend that are not present in subtrahend. A key
// is present only if its value is not undefined, matching JSON semantics.
// The result is sorted for deterministic output.
func subtract(minuend, subtrahend map[string]any) []string {
	keys := []string{}
	for key, value := range minuend {
		if jsonutil.IsUndefined(value) {
			continue
		}
		if !hasValue(subtrahend, key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func hasValue(object map[string]any, key string) bool {
	value, ok := object[key]
	return ok && !jsonutil.IsUndefined(value)
}

// intersection lists the keys shared by all objects. The result is sorted for
// deterministic output.
func intersection(objects []map[string]any) []string {
	counter := map[string]int{}
	for _, object := range objects {
		for key, value := range object {
			if !jsonutil.IsUndefined(value) {
				counter[key]++
			}
		}
	}
	keys := []string{}
	for key, count := range counter {
		if count >= len(objects) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// intersection2 lists the keys shared by a and b. The result is sorted for
// deterministic output.
func intersection2(a, b map[string]any) []string {
	keys := []string{}
	for key, value := range a {
		if jsonutil.IsUndefined(value) {
			continue
		}
		if hasValue(b, key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// arrayOperation is the internal (non-JSON) form of an array edit. index
// indicates the actual target position, never end-of-array.
type arrayOperation struct {
	op       string
	index    int
	value    any
	original any
}

type dynamicAlternative struct {
	operations []arrayOperation
	cost       int
}

func (base dynamicAlternative) append(operation arrayOperation) dynamicAlternative {
	operations := make([]arrayOperation, len(base.operations)+1)
	copy(operations, base.operations)
	operations[len(base.operations)] = operation
	return dynamicAlternative{operations: operations, cost: base.cost + 1}
}

// diffArrays computes the shortest sequence of operations to get from input to
// output, using a dynamic programming implementation of the Levenshtein
// distance algorithm.
func diffArrays(input, output []any, ptr *Pointer, diff diffFunc) []Operation {
	inputLength := len(input)
	outputLength := len(output)
	memo := make([][]*dynamicAlternative, inputLength+1)
	for i := range memo {
		memo[i] = make([]*dynamicAlternative, outputLength+1)
	}
	// The origin (0, 0) is the empty alternative: no operations, no cost.
	origin := dynamicAlternative{operations: []arrayOperation{}, cost: 0}
	memo[0][0] = &origin

	var dist func(i, j int) dynamicAlternative
	dist = func(i, j int) dynamicAlternative {
		if memoized := memo[i][j]; memoized != nil {
			return *memoized
		}
		var result dynamicAlternative
		if i > 0 && j > 0 && len(diff(input[i-1], output[j-1], ptr.Add(strconv.Itoa(i-1)))) == 0 {
			// equal (no operations => no cost)
			result = dist(i-1, j-1)
		} else {
			alternatives := []dynamicAlternative{}
			if i > 0 {
				// NOT topmost row
				removeBase := dist(i-1, j)
				alternatives = append(alternatives, removeBase.append(arrayOperation{op: "remove", index: i - 1}))
			}
			if j > 0 {
				// NOT leftmost column
				addBase := dist(i, j-1)
				alternatives = append(alternatives, addBase.append(arrayOperation{op: "add", index: i - 1, value: output[j-1]}))
			}
			if i > 0 && j > 0 {
				// TABLE MIDDLE
				replaceBase := dist(i-1, j-1)
				alternatives = append(alternatives, replaceBase.append(arrayOperation{
					op:       "replace",
					index:    i - 1,
					original: input[i-1],
					value:    output[j-1],
				}))
			}
			// the meat of the algorithm: pick the cheapest alternative,
			// stable in the order remove, add, replace.
			best := alternatives[0]
			for _, alternative := range alternatives[1:] {
				if alternative.cost < best.cost {
					best = alternative
				}
			}
			result = best
		}
		memo[i][j] = &result
		return result
	}

	operations := []Operation{}
	padding := 0
	for _, arrayOperation := range dist(inputLength, outputLength).operations {
		switch arrayOperation.op {
		case "add":
			paddedIndex := arrayOperation.index + 1 + padding
			indexToken := "-"
			if paddedIndex < inputLength+padding {
				indexToken = strconv.Itoa(paddedIndex)
			}
			operations = append(operations, AddOperation{Path: ptr.Add(indexToken).String(), Value: arrayOperation.value})
			padding++
		case "remove":
			operations = append(operations, RemoveOperation{Path: ptr.Add(strconv.Itoa(arrayOperation.index + padding)).String()})
			padding--
		default: // replace
			replacePointer := ptr.Add(strconv.Itoa(arrayOperation.index + padding))
			operations = append(operations, diff(arrayOperation.original, arrayOperation.value, replacePointer)...)
		}
	}
	return operations
}

// diffObjects compares two JSON objects member by member.
func diffObjects(input, output map[string]any, ptr *Pointer, diff diffFunc) []Operation {
	operations := []Operation{}
	// if a key is in input but not output -> remove it
	for _, key := range subtract(input, output) {
		operations = append(operations, RemoveOperation{Path: ptr.Add(key).String()})
	}
	// if a key is in output but not input -> add it
	for _, key := range subtract(output, input) {
		operations = append(operations, AddOperation{Path: ptr.Add(key).String(), Value: output[key]})
	}
	// if a key is in both, diff it recursively
	for _, key := range intersection2(input, output) {
		operations = append(operations, diff(input[key], output[key], ptr.Add(key))...)
	}
	return operations
}

// diffAny returns an empty slice if input and output are materially equal;
// otherwise it produces operations that transform input into output.
func diffAny(input, output any, ptr *Pointer, diff diffFunc) []Operation {
	if inputMap, ok := input.(map[string]any); ok {
		if outputMap, ok := output.(map[string]any); ok {
			return diffObjects(inputMap, outputMap, ptr, diff)
		}
	}
	if inputArray, ok := input.([]any); ok {
		if outputArray, ok := output.([]any); ok {
			return diffArrays(inputArray, outputArray, ptr, diff)
		}
	}
	if jsonutil.ScalarEqual(input, output) {
		return []Operation{}
	}
	// At this point input and output are materially different and cannot be
	// split into multiple patches: output replaces input wholesale.
	return []Operation{ReplaceOperation{Path: ptr.String(), Value: output}}
}
