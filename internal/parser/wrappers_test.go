package parser

import (
	"cometa/internal/ast"
	"cometa/internal/lexer"
	"testing"
)

func TestWrapperTypeGroupingAndRecoveryPrecedence(t *testing.T) {
	source := "fn f(a entero?!, b (entero!)?, c entero!(bool?), d [entero?], e !bool) entero! a o b o c\n"
	tokens, err := lexer.Lex("types.cometa", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("types.cometa", tokens)
	if err != nil {
		t.Fatal(err)
	}
	f := program.Decls[0].(*ast.FuncDecl)
	if f.Params[0].Type.Wrapper != "!" || f.Params[0].Type.Payload.Wrapper != "?" {
		t.Fatal("entero?! must be result(optional(entero))")
	}
	if f.Params[1].Type.Wrapper != "?" || f.Params[1].Type.Payload.Wrapper != "!" {
		t.Fatal("(entero!)? must be optional(result(entero))")
	}
	if f.Params[2].Type.ErrorType.Wrapper != "?" {
		t.Fatal("parenthesized error type lost")
	}
	if f.Params[3].Type.Element.Wrapper != "?" {
		t.Fatal("optional list element lost")
	}
	if f.Params[4].Type.Payload.Name != "$unidad" {
		t.Fatal("bare result needs unit success")
	}
	if f.ReturnType.ErrorType.Name != "cadena" {
		t.Fatal("whitespace must distinguish the inline body from a typed error")
	}
	outer := f.Body[0].(*ast.ExprStmt).Expr.(*ast.RecoverExpr)
	if _, ok := outer.Body[0].(*ast.ExprStmt).Expr.(*ast.RecoverExpr); !ok {
		t.Fatal("fallback must associate to the right")
	}
}

func TestTryCallPrecedence(t *testing.T) {
	tokens, err := lexer.Lex("try.cometa", "fn f() entero! intentar g(1) + 2\n")
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("try.cometa", tokens)
	if err != nil {
		t.Fatal(err)
	}
	e := program.Decls[0].(*ast.FuncDecl).Body[0].(*ast.ExprStmt).Expr.(*ast.BinaryExpr)
	if _, ok := e.Left.(*ast.TryExpr).Value.(*ast.CallExpr); !ok {
		t.Fatal("try must include call postfix and exclude addition")
	}
}
