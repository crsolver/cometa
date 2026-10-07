package diagnostic_test

import (
	"errors"
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"testing"
)

func TestAggregationPreservesDistinctErrorsAndTypes(t *testing.T) {
	a := &diagnostic.Error{Stage: diagnostic.Lexer, Filename: "a.cometa", Pos: ast.Pos{Line: 2, Column: 1}, Message: "léxico"}
	b := &diagnostic.Error{Stage: diagnostic.Sema, Filename: "a.cometa", Pos: a.Pos, Message: "semántico"}
	c := &diagnostic.Error{Stage: diagnostic.Sema, Filename: "b.cometa", Pos: ast.Pos{Line: 1, Column: 1}, Message: "otro"}
	list := diagnostic.List{c, b, a, a}
	err := list.Err()
	flat := diagnostic.Flatten(err)
	if len(flat) != 3 || flat[0] != a || flat[1] != b || flat[2] != c {
		t.Fatalf("order/dedup: %v", err)
	}
	var typed *diagnostic.Error
	if !errors.As(err, &typed) || typed != a {
		t.Fatalf("lost typed errors: %v", err)
	}
	if _, _, stage, _ := flat[1].(diagnostic.Located).Diagnostic(); stage != diagnostic.Sema {
		t.Fatalf("lost the stage: %q", stage)
	}
	if single := (diagnostic.List{a}).Err(); single != a {
		t.Fatalf("single error changed: %v", single)
	}
}

func TestLimitSurvivesNestedAggregation(t *testing.T) {
	var list diagnostic.List
	for line := 120; line > 0; line-- {
		list.Add(&diagnostic.Error{Stage: diagnostic.Sema, Filename: "a.cometa", Pos: ast.Pos{Line: line, Column: 1}, Message: "error"})
	}
	err := list.Err()
	for i := 0; i < 3; i++ {
		var outer diagnostic.List
		outer.Add(err)
		err = outer.Err()
	}
	flat := diagnostic.Flatten(err)
	if len(flat) != 101 {
		t.Fatalf("limit: %d", len(flat))
	}
	for i, e := range flat[:100] {
		if e.(*diagnostic.Error).Pos.Line != i+1 {
			t.Fatalf("lost source error %d: %v", i+1, e)
		}
	}
}

// Diagnostics with a position come first, by position; the rest follow by
// text, whatever order they were added in.
func TestOrderIsTheSameForAnyInputOrder(t *testing.T) {
	located := func(line int, message string) error {
		return &diagnostic.Error{Stage: diagnostic.Sema, Filename: "a.cometa", Pos: ast.Pos{Line: line, Column: 1}, Message: message}
	}
	a, b, c := located(1, "zeta"), located(2, "alfa"), located(3, "beta")
	x, y := errors.New("a.cometa: sin posición"), errors.New("zz sin posición")
	want := []error{a, b, c, x, y}
	for _, input := range [][]error{{a, b, c, x, y}, {y, c, x, b, a}, {x, a, y, c, b}, {c, y, b, x, a}} {
		got := diagnostic.Flatten(diagnostic.List(input).Err())
		if len(got) != len(want) {
			t.Fatalf("se perdieron errores: %v", got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("orden de %v: %v", input, got)
			}
		}
	}
}
