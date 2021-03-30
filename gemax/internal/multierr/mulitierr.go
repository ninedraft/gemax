// Package multierr provides utilities for error chaining.
package multierr

import (
	"errors"
	"strings"
)

// Combine merges to provided non-nil errors. Every argumetn can be nil.
func Combine(a, b error) error {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	return &pair{a: a, b: b}
}

const errSeparator = "; "

type pair struct {
	a, b error
}

func (p *pair) Error() string {
	var a, b = p.a.Error(), p.b.Error()

	var str = &strings.Builder{}
	str.Grow(len(a) + len(b) + len(errSeparator))
	_, _ = str.WriteString(a)
	_, _ = str.WriteString(errSeparator)
	_, _ = str.WriteString(b)
	return str.String()
}

func (p *pair) Errors() []error {
	return []error{p.a, p.b}
}

func (p *pair) Is(err error) bool {
	return errors.Is(p.a, err) || errors.Is(p.b, err)
}

func (p *pair) As(err interface{}) bool {
	return errors.As(p.a, err) || errors.As(p.b, err)
}
