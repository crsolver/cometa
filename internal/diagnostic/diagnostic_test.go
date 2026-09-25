package diagnostic_test

import (
	"errors"
	"hacha/internal/ast"
	"hacha/internal/diagnostic"
	"hacha/internal/lexer"
	"hacha/internal/sema"
	"testing"
)

func TestAggregationPreservesDistinctErrorsAndTypes(t *testing.T) {
	a := &lexer.Error{Filename: "a.hacha", Pos: ast.Pos{Line: 2, Column: 1}, Message: "léxico"}
	b := &sema.Error{Filename: "a.hacha", Pos: a.Pos, Message: "semántico"}
	c := &sema.Error{Filename: "b.hacha", Pos: ast.Pos{Line: 1, Column: 1}, Message: "otro"}
	list := diagnostic.List{c, b, a, a}
	err := list.Err()
	flat := diagnostic.Flatten(err)
	if len(flat) != 3 || flat[0] != a || flat[1] != b || flat[2] != c {
		t.Fatalf("order/dedup: %v", err)
	}
	var lexical *lexer.Error
	var semantic *sema.Error
	if !errors.As(err, &lexical) || !errors.As(err, &semantic) {
		t.Fatalf("lost typed errors: %v", err)
	}
	if single := (diagnostic.List{a}).Err(); single != a {
		t.Fatalf("single error changed: %v", single)
	}
}

func TestLimitSurvivesNestedAggregation(t *testing.T) {
	var list diagnostic.List
	for line := 120; line > 0; line-- {
		list.Add(&sema.Error{Filename: "a.hacha", Pos: ast.Pos{Line: line, Column: 1}, Message: "error"})
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
		if e.(*sema.Error).Pos.Line != i+1 {
			t.Fatalf("lost source error %d: %v", i+1, e)
		}
	}
}
