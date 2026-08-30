package input

import (
	"fmt"
	"strings"
	"unicode"
)

// ContextEntry is a single identifier or key-value pair within a KeyContext.
type ContextEntry struct {
	Key   string
	Value string // empty if boolean identifier
}

// KeyContext holds the identifiers and key-value properties active at a node
// in the focus / element hierarchy.
type KeyContext struct {
	entries []ContextEntry
}

// NewKeyContext returns an empty KeyContext.
func NewKeyContext() KeyContext {
	return KeyContext{}
}

// ParseKeyContext parses a context string containing identifiers and/or
// key=value pairs separated by whitespace (such as "Editor mode=full vim=normal").
func ParseKeyContext(source string) (KeyContext, error) {
	var ctx KeyContext
	tokens := strings.Fields(source)
	for _, tok := range tokens {
		if idx := strings.IndexByte(tok, '='); idx >= 0 {
			k := strings.TrimSpace(tok[:idx])
			v := strings.TrimSpace(tok[idx+1:])
			if k == "" {
				return KeyContext{}, fmt.Errorf("parse key context %q: empty key in %q", source, tok)
			}
			ctx.Set(k, v)
		} else {
			id := strings.TrimSpace(tok)
			if id != "" {
				ctx.Add(id)
			}
		}
	}
	return ctx, nil
}

// Add adds an identifier to the context if not already present.
func (c *KeyContext) Add(identifier string) {
	if !c.Contains(identifier) {
		c.entries = append(c.entries, ContextEntry{Key: identifier})
	}
}

// Set sets a key-value property in the context, replacing an existing entry
// for key if one exists.
func (c *KeyContext) Set(key, value string) {
	for i := range c.entries {
		if c.entries[i].Key == key {
			c.entries[i].Value = value
			return
		}
	}
	c.entries = append(c.entries, ContextEntry{Key: key, Value: value})
}

// Contains reports whether the context contains the given identifier or key.
func (c KeyContext) Contains(key string) bool {
	for _, e := range c.entries {
		if e.Key == key {
			return true
		}
	}
	return false
}

// Get returns the value for the given key and true, or empty string and false
// if the key is not set or has no value.
func (c KeyContext) Get(key string) (string, bool) {
	for _, e := range c.entries {
		if e.Key == key && e.Value != "" {
			return e.Value, true
		}
	}
	return "", false
}

// Entries returns a copy of all context entries.
func (c KeyContext) Entries() []ContextEntry {
	if len(c.entries) == 0 {
		return nil
	}
	res := make([]ContextEntry, len(c.entries))
	copy(res, c.entries)
	return res
}

// IsEmpty reports whether the context has no entries.
func (c KeyContext) IsEmpty() bool {
	return len(c.entries) == 0
}

// String returns a space-separated string representation of the context entries.
func (c KeyContext) String() string {
	var parts []string
	for _, e := range c.entries {
		if e.Value == "" {
			parts = append(parts, e.Key)
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", e.Key, e.Value))
		}
	}
	return strings.Join(parts, " ")
}

// ContextPredicate is an expression evaluated against an active context stack
// to determine whether a keybinding is enabled.
type ContextPredicate interface {
	evalInner(contexts []KeyContext, allContexts []KeyContext) bool
	Match(contexts []KeyContext) (depth int, matched bool)
	IsSuperset(other ContextPredicate) bool
	String() string
}

// IdentifierPredicate matches if the innermost context contains the identifier.
type IdentifierPredicate struct {
	Identifier string
}

func (p IdentifierPredicate) evalInner(contexts []KeyContext, _ []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	return contexts[len(contexts)-1].Contains(p.Identifier)
}

func (p IdentifierPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p IdentifierPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p IdentifierPredicate) String() string {
	return p.Identifier
}

// EqualPredicate matches if the innermost context has the key equal to value.
type EqualPredicate struct {
	Key   string
	Value string
}

func (p EqualPredicate) evalInner(contexts []KeyContext, _ []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	val, ok := contexts[len(contexts)-1].Get(p.Key)
	return ok && val == p.Value
}

func (p EqualPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p EqualPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p EqualPredicate) String() string {
	return fmt.Sprintf("%s == %s", p.Key, p.Value)
}

// NotEqualPredicate matches if the innermost context does not have key equal to value.
type NotEqualPredicate struct {
	Key   string
	Value string
}

func (p NotEqualPredicate) evalInner(contexts []KeyContext, _ []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	val, ok := contexts[len(contexts)-1].Get(p.Key)
	if !ok {
		return true
	}
	return val != p.Value
}

func (p NotEqualPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p NotEqualPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p NotEqualPredicate) String() string {
	return fmt.Sprintf("%s != %s", p.Key, p.Value)
}

// DescendantPredicate matches if Child matches at a deeper level than Parent.
type DescendantPredicate struct {
	Parent ContextPredicate
	Child  ContextPredicate
}

func (p DescendantPredicate) evalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) < 2 {
		return false
	}
	for i := 0; i < len(contexts)-1; i++ {
		if p.Parent.evalInner(contexts[:i+1], allContexts) {
			if p.Child.evalInner(contexts[i+1:], contexts[i+1:]) {
				return true
			}
		}
	}
	return false
}

func (p DescendantPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p DescendantPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p DescendantPredicate) String() string {
	return fmt.Sprintf("%s > %s", p.Parent.String(), p.Child.String())
}

// NotPredicate inverts its child predicate across the entire context stack.
type NotPredicate struct {
	Child ContextPredicate
}

func (p NotPredicate) evalInner(_ []KeyContext, allContexts []KeyContext) bool {
	for i := 0; i < len(allContexts); i++ {
		if p.Child.evalInner(allContexts[:i+1], allContexts) {
			return false
		}
	}
	return true
}

func (p NotPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p NotPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p NotPredicate) String() string {
	return fmt.Sprintf("!(%s)", p.Child.String())
}

// AndPredicate matches if both Left and Right match.
type AndPredicate struct {
	Left  ContextPredicate
	Right ContextPredicate
}

func (p AndPredicate) evalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	return p.Left.evalInner(contexts, allContexts) && p.Right.evalInner(contexts, allContexts)
}

func (p AndPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p AndPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p AndPredicate) String() string {
	return fmt.Sprintf("%s && %s", p.Left.String(), p.Right.String())
}

// OrPredicate matches if either Left or Right matches.
type OrPredicate struct {
	Left  ContextPredicate
	Right ContextPredicate
}

func (p OrPredicate) evalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	return p.Left.evalInner(contexts, allContexts) || p.Right.evalInner(contexts, allContexts)
}

func (p OrPredicate) Match(contexts []KeyContext) (int, bool) {
	return matchPredicate(p, contexts)
}

func (p OrPredicate) IsSuperset(other ContextPredicate) bool {
	return isSuperset(p, other)
}

func (p OrPredicate) String() string {
	return fmt.Sprintf("(%s || %s)", p.Left.String(), p.Right.String())
}

func matchPredicate(p ContextPredicate, contexts []KeyContext) (int, bool) {
	for depth := len(contexts); depth >= 0; depth-- {
		if p.evalInner(contexts[:depth], contexts) {
			return depth, true
		}
	}
	return 0, false
}

func isSuperset(self, other ContextPredicate) bool {
	if self.String() == other.String() {
		return true
	}
	if orPred, ok := self.(OrPredicate); ok {
		return isSuperset(orPred.Left, other) || isSuperset(orPred.Right, other)
	}
	switch o := other.(type) {
	case DescendantPredicate:
		return isSuperset(self, o.Child)
	case AndPredicate:
		return isSuperset(self, o.Left) || isSuperset(self, o.Right)
	default:
		return false
	}
}

// ParseContextPredicate parses a context predicate expression such as:
//
//   - "Editor"
//   - "mode == full"
//   - "mode != full"
//   - "!Editor"
//   - "Editor && mode == full"
//   - "Editor || Terminal"
//   - "Workspace > Editor"
//   - "(Workspace || Modal) > Editor && mode == full"
func ParseContextPredicate(source string) (ContextPredicate, error) {
	p := contextParser{source: strings.TrimSpace(source)}
	pred, err := p.parseExpr(0)
	if err != nil {
		return nil, fmt.Errorf("parse context predicate %q: %w", source, err)
	}
	p.skipWhitespace()
	if p.pos < len(p.source) {
		return nil, fmt.Errorf("parse context predicate %q: unexpected trailing characters %q", source, p.source[p.pos:])
	}
	return pred, nil
}

type contextParser struct {
	source string
	pos    int
}

const (
	precChild = 1
	precOr    = 2
	precAnd   = 3
	precEq    = 4
	precNot   = 5
)

func (p *contextParser) parseExpr(minPrec int) (ContextPredicate, error) {
	lhs, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.source) {
			break
		}

		op, prec, length := p.peekOperator()
		if op == "" || prec < minPrec {
			break
		}

		p.pos += length
		p.skipWhitespace()

		rhs, err := p.parseExpr(prec + 1)
		if err != nil {
			return nil, err
		}

		switch op {
		case ">":
			lhs = DescendantPredicate{Parent: lhs, Child: rhs}
		case "||":
			lhs = OrPredicate{Left: lhs, Right: rhs}
		case "&&":
			lhs = AndPredicate{Left: lhs, Right: rhs}
		case "==":
			idLeft, okLeft := lhs.(IdentifierPredicate)
			idRight, okRight := rhs.(IdentifierPredicate)
			if !okLeft || !okRight {
				return nil, fmt.Errorf("operands of == must be identifiers")
			}
			lhs = EqualPredicate{Key: idLeft.Identifier, Value: idRight.Identifier}
		case "!=":
			idLeft, okLeft := lhs.(IdentifierPredicate)
			idRight, okRight := rhs.(IdentifierPredicate)
			if !okLeft || !okRight {
				return nil, fmt.Errorf("operands of != must be identifiers")
			}
			lhs = NotEqualPredicate{Key: idLeft.Identifier, Value: idRight.Identifier}
		default:
			return nil, fmt.Errorf("unknown operator %q", op)
		}
	}

	return lhs, nil
}

func (p *contextParser) parsePrimary() (ContextPredicate, error) {
	p.skipWhitespace()
	if p.pos >= len(p.source) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	ch := p.source[p.pos]
	if ch == '(' {
		p.pos++
		expr, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.source) || p.source[p.pos] != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		p.pos++
		return expr, nil
	}

	if ch == '!' {
		p.pos++
		child, err := p.parseExpr(precNot)
		if err != nil {
			return nil, err
		}
		return NotPredicate{Child: child}, nil
	}

	id := p.parseIdentifier()
	if id == "" {
		return nil, fmt.Errorf("expected identifier at index %d", p.pos)
	}
	return IdentifierPredicate{Identifier: id}, nil
}

func (p *contextParser) parseIdentifier() string {
	start := p.pos
	for p.pos < len(p.source) {
		r := rune(p.source[p.pos])
		if isIdentChar(r) {
			p.pos++
		} else {
			break
		}
	}
	return p.source[start:p.pos]
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func (p *contextParser) peekOperator() (op string, prec int, length int) {
	rest := p.source[p.pos:]
	if strings.HasPrefix(rest, "==") {
		return "==", precEq, 2
	}
	if strings.HasPrefix(rest, "!=") {
		return "!=", precEq, 2
	}
	if strings.HasPrefix(rest, "&&") {
		return "&&", precAnd, 2
	}
	if strings.HasPrefix(rest, "||") {
		return "||", precOr, 2
	}
	if strings.HasPrefix(rest, ">") {
		return ">", precChild, 1
	}
	return "", 0, 0
}

func (p *contextParser) skipWhitespace() {
	for p.pos < len(p.source) && unicode.IsSpace(rune(p.source[p.pos])) {
		p.pos++
	}
}
