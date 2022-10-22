package scope

import (
	"errors"
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type SuggestVarNameFunc func(input string) string

func defaultSuggestVarNameFunc(input string) (ret string) {
	defer func() {
		if ret == "_" {
			ret = "v"
		}
		ret = defaultSafeNameFunc(ret)
	}()

	r, size := utf8.DecodeRuneInString(input)

	var buf strings.Builder
	buf.WriteRune(unicode.ToLower(r))

	words := true
	u0 := unicode.IsUpper(r)
	for slice := input[size:]; slice != ""; slice = slice[size:] {
		r, size = utf8.DecodeRuneInString(slice)
		if r == utf8.RuneError {
			words = false
			break
		}
		u1 := unicode.IsUpper(r)

		if u0 && u1 {
			words = false
			break
		}

		if u1 {
			buf.WriteRune(unicode.ToLower(r))
		}

		u0 = u1
	}

	if words {
		return buf.String()
	}

	slice := strings.TrimLeftFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r)
	})

	r, _ = utf8.DecodeRuneInString(slice)
	switch r {
	case utf8.RuneError:
		return "v"
	default:
		return fmt.Sprintf("%c", unicode.ToLower(r))
	}
}

type SafeNameFunc func(name string) string

func defaultSafeNameFunc(name string) string {
	name = strings.Map(onlyValidIdentChars, name)
	if name == "" {
		return "v"
	}

	if token.IsKeyword(name) || predeclared(name) {
		name = "_" + name
	}

	r, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsLetter(r) && r != '_' {
		name = "_" + name
	}

	return name
}

type UniqueNameFunc func(s *Scope, name string, recursive bool) string

func defaultUniqueNameFunc(s *Scope, name string, recursive bool) string {
	safeName := s.safeName(name)

	const maxAttempts = 999
	for ndx := 0; ndx < maxAttempts; ndx++ {
		nameN := fmt.Sprintf("%s%d", safeName, ndx)

		if !s.defined(nameN, recursive) {
			return nameN
		}
	}

	panic(fmt.Sprintf("scope: failed to find safe, unique, name for root %q after %d attempts", name, maxAttempts))
}

type Scope struct {
	parent             *Scope
	defs               map[string]bool
	safeNameFunc       SafeNameFunc
	uniqueNameFunc     UniqueNameFunc
	suggestVarNameFunc SuggestVarNameFunc
}

type OptFunc func(*Scope)

func New(opts ...OptFunc) (ret *Scope) {
	ret = new(Scope)

	for _, f := range opts {
		f(ret)
	}

	return ret
}

func WithSafeNameFunc(f SafeNameFunc) OptFunc {
	return func(s *Scope) {
		s.safeNameFunc = f
	}
}

func WithUniqueNameFunc(f UniqueNameFunc) OptFunc {
	return func(s *Scope) {
		s.uniqueNameFunc = f
	}
}

func WithSuggestVarNameFunc(f SuggestVarNameFunc) OptFunc {
	return func(s *Scope) {
		s.suggestVarNameFunc = f
	}
}

func withParent(parent *Scope) OptFunc {
	return func(s *Scope) {
		s.parent = parent
	}
}

func (s *Scope) defined(name string, recursive bool) bool {
	switch {
	case s.defs != nil && s.defs[name]:
		return true
	case !recursive:
		return false
	case s.parent == nil:
		return false
	default:
		return s.parent.defined(name, true)
	}
}

func (s *Scope) Claim(name string) string {
	return s.define(s.safeName(name), false)
}

func (s *Scope) ClaimGlobal(name string) string {
	return s.define(s.safeName(name), true)
}

func (s *Scope) Suggest(input string) string {
	return s.suggestVarName(input)
}

// New returns a new Scope with s set as the parent of ret
func (s *Scope) New(opts ...OptFunc) (ret *Scope) {
	return New(append(opts, withParent(s))...)
}

func (s *Scope) define(safeName string, recursive bool) string {
	if s.defs == nil {
		s.defs = make(map[string]bool)
	}

	if !s.defined(safeName, recursive) {
		s.defs[safeName] = true
		if recursive && s.parent != nil {
			s.parent.define(safeName, recursive)
		}
		return safeName
	}

	return s.define(s.uniqueName(safeName, recursive), recursive)
}

func (s *Scope) safeName(name string) string {
	if s.safeNameFunc != nil {
		return s.safeNameFunc(name)
	}

	if s.parent != nil {
		return s.parent.safeName(name)
	}

	return defaultSafeNameFunc(name)
}

func (s *Scope) uniqueName(name string, recursive bool) string {
	if s.uniqueNameFunc != nil {
		return s.uniqueNameFunc(s, name, recursive)
	}

	if s.parent != nil {
		return s.parent.uniqueName(name, recursive)
	}

	return defaultUniqueNameFunc(s, name, recursive)
}

func (s *Scope) suggestVarName(input string) string {
	if s.suggestVarNameFunc != nil {
		return s.suggestVarNameFunc(input)
	}

	if s.parent != nil {
		return s.parent.suggestVarName(input)
	}

	return defaultSuggestVarNameFunc(input)
}

func Unexported(name string) string {
	if !token.IsIdentifier(name) {
		panic(fmt.Errorf("scope: failed to create unexported identifier for name %q: %q is not a valid identifier", name, name))
	}

	return unexported(name)
}

func unexported(name string) string {
	if !token.IsExported(name) {
		return name
	}

	n, size := utf8.DecodeRuneInString(name)
	return fmt.Sprintf("%c%s", unicode.ToLower(n), name[size:])
}

func Exported(name string) string {
	if !token.IsIdentifier(name) {
		panic(fmt.Errorf("scope: failed to create exported identifier for name %q: %q is not a valid identifier", name, name))
	}

	ret, err := exported(name)
	if err != nil {
		panic(fmt.Errorf("scope: failed to create exported identifier for name %q: %w", name, err))
	}

	return ret
}

func exported(name string) (string, error) {
	if name == "" {
		return "", errors.New("input does not contain any uppercase letters")
	}

	if token.IsExported(name) {
		return name, nil
	}

	n, size := utf8.DecodeRuneInString(name)
	l := unicode.ToUpper(n)

	name = fmt.Sprintf("%c%s", l, name[size:])
	if token.IsExported(name) {
		return name, nil
	}

	return exported(name[size:])
}

func onlyValidIdentChars(in rune) rune {
	switch {
	case in == '_':
		return in
	case unicode.In(in, unicode.Letter, unicode.Number):
		return in
	default:
		return -1
	}
}

func predeclared(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "true", "false", "iota", "nil", "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover", "any":
		return true
	default:
		return false
	}
}
