package jsonpatch

import (
	"fmt"
	jsonutil "github.com/firestartorg/rfc6902-go/internal/jsonutil"
	"regexp"
	"strconv"
	"strings"
)

func unescapeToken(token string) string {
	// ~1 is unescaped before ~0 (see RFC 6901 §4 and the reference implementation).
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
}

func escapeToken(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

// PointerEvaluation is the result of evaluating a Pointer against a document.
//
// In the special case that the Pointer's path is "" (root), Parent is nil,
// Key is "", and Value is the whole document. Otherwise Parent and Key are
// such that Parent[Key] == Value (conceptually).
type PointerEvaluation struct {
	Parent any
	Key    string
	Value  any
}

// Pointer is a JSON Pointer (RFC 6901) representation.
type Pointer struct {
	Tokens []string
}

// NewPointer returns the root pointer, whose sole token is "".
func NewPointer() *Pointer {
	return &Pointer{Tokens: []string{""}}
}

// PointerFromJSON parses a properly escaped JSON Pointer string.
func PointerFromJSON(path string) (*Pointer, error) {
	tokens := strings.Split(path, "/")
	for i, token := range tokens {
		tokens[i] = unescapeToken(token)
	}
	if len(tokens) == 0 || tokens[0] != "" {
		return nil, fmt.Errorf("Invalid JSON Pointer: %s", path)
	}
	return &Pointer{Tokens: tokens}, nil
}

func (p *Pointer) String() string {
	parts := make([]string, len(p.Tokens))
	for i, token := range p.Tokens {
		parts[i] = escapeToken(token)
	}
	return strings.Join(parts, "/")
}

// arrayIndex parses a canonical RFC 6901 array index (no sign, decimal point,
// exponent, or leading zeros). Non-canonical tokens are not array indices.
func arrayIndex(key string) (int, bool) {
	if !arrayIndexPattern.MatchString(key) {
		return 0, false
	}
	n, err := strconv.Atoi(key)
	if err != nil {
		return 0, false
	}
	return n, true
}

var arrayIndexPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// resolveToken returns the member of parent named by key, or jsonutil.Undefined if
// parent is not a container or the member does not exist. Note that a member
// explicitly set to null resolves to nil, which is distinct from jsonutil.Undefined.
func resolveToken(parent any, key string) any {
	switch container := parent.(type) {
	case map[string]any:
		if value, ok := container[key]; ok {
			return value
		}
	case []any:
		if index, ok := arrayIndex(key); ok && index < len(container) {
			return container[index]
		}
	}
	return jsonutil.Undefined
}

// Evaluate returns an object with Parent, Key, and Value properties.
//
// Note: the reference implementation skips the tokens "__proto__",
// "constructor", and "prototype" to avoid prototype pollution. Go has no
// prototype chain, so that guard is deliberately omitted here as it would be
// dead and misleading code.
func (p *Pointer) Evaluate(object any) PointerEvaluation {
	var parent any = nil
	key := ""
	value := object
	for i := 1; i < len(p.Tokens); i++ {
		parent = value
		key = p.Tokens[i]
		value = resolveToken(parent, key)
	}
	return PointerEvaluation{Parent: parent, Key: key, Value: value}
}

// Get evaluates the pointer and returns the resolved value (or nil/undefined
// if the path cannot be resolved).
func (p *Pointer) Get(object any) any {
	return p.Evaluate(object).Value
}

// Set evaluates the pointer and writes value in place. Like the reference
// implementation, a root pointer (parent nil) does nothing. Unlike JavaScript,
// a []any cannot be grown through a copied slice header, so assigning beyond
// the end of an array is a no-op (see setInPlace).
func (p *Pointer) Set(object, value any) {
	endpoint := p.Evaluate(object)
	setInPlace(endpoint.Parent, endpoint.Key, value)
}

// setInPlace writes value into an existing member of parent. It cannot grow a
// slice, so out-of-range indices are ignored.
func setInPlace(parent any, key string, value any) {
	switch container := parent.(type) {
	case map[string]any:
		container[key] = value
	case []any:
		if index, ok := arrayIndex(key); ok && index < len(container) {
			container[index] = value
		}
	}
}

// Push appends a token, mutating the pointer.
func (p *Pointer) Push(token string) {
	p.Tokens = append(p.Tokens, token)
}

// Add returns a new (shallowly immutable) Pointer with token appended.
func (p *Pointer) Add(token string) *Pointer {
	tokens := make([]string, len(p.Tokens)+1)
	copy(tokens, p.Tokens)
	tokens[len(p.Tokens)] = token
	return &Pointer{Tokens: tokens}
}

// Parent returns a new Pointer representing this pointer's parent. The parent
// of the empty pointer is the empty pointer.
func (p *Pointer) Parent() *Pointer {
	if len(p.Tokens) > 1 {
		return &Pointer{Tokens: p.Tokens[:len(p.Tokens)-1]}
	}
	return &Pointer{Tokens: []string{""}}
}
