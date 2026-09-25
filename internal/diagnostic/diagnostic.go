// Package diagnostic collects recoverable frontend failures without losing their types.
package diagnostic

import (
	"fmt"
	"hacha/internal/ast"
	"sort"
	"strings"
)

const Limit = 100

type Located interface {
	error
	Diagnostic() (string, ast.Pos, string, string)
}

type List []error

func (l List) Error() string {
	lines := make([]string, len(l))
	for i, err := range l {
		lines[i] = err.Error()
	}
	return strings.Join(lines, "\n")
}
func (l List) Unwrap() []error { return []error(l) }

func Flatten(err error) []error {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var out []error
		for _, e := range many.Unwrap() {
			out = append(out, Flatten(e)...)
		}
		return out
	}
	return []error{err}
}

func (l *List) Add(err error) { *l = append(*l, Flatten(err)...) }
func (l List) Err() error {
	var out List
	truncated := false
	seen := map[string]bool{}
	for _, e := range l {
		for _, item := range Flatten(e) {
			if _, ok := item.(limitError); ok {
				truncated = true
				continue
			}
			key := item.Error()
			if d, ok := item.(Located); ok {
				_, _, stage, _ := d.Diagnostic()
				key = stage + ":" + key
			}
			if !seen[key] {
				seen[key] = true
				out = append(out, item)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, aok := out[i].(Located)
		b, bok := out[j].(Located)
		if !aok || !bok {
			return out[i].Error() < out[j].Error()
		}
		af, ap, as, am := a.Diagnostic()
		bf, bp, bs, bm := b.Diagnostic()
		if af != bf {
			return af < bf
		}
		if ap.Line != bp.Line {
			return ap.Line < bp.Line
		}
		if ap.Column != bp.Column {
			return ap.Column < bp.Column
		}
		if as != bs {
			return as < bs
		}
		return am < bm
	})
	if len(out) > Limit {
		out = out[:Limit]
		truncated = true
	}
	if truncated && len(out) > 0 {
		out = append(out, limitError{out[len(out)-1]})
	}
	if len(out) == 0 {
		return nil
	}
	if len(out) == 1 {
		return out[0]
	}
	return out
}

type limitError struct{ at error }

func (e limitError) Error() string {
	file, pos, _, message := e.Diagnostic()
	return fmt.Sprintf("%s:%d:%d: %s", file, pos.Line, pos.Column, message)
}
func (e limitError) Diagnostic() (string, ast.Pos, string, string) {
	file, pos := "", ast.Pos{Line: 1, Column: 1}
	if d, ok := e.at.(Located); ok {
		file, pos, _, _ = d.Diagnostic()
	}
	return file, pos, "frontend", "se alcanzó el límite de 100 errores; corrija los errores anteriores"
}
