package parser

import (
	"hacha/internal/ast"
	"hacha/internal/lexer"
	"testing"
)

func TestEnumAndMatchAST(t *testing.T) {
	source := "enum E\n\tA\n\tB [num]\nfn inicio()\n\tvar e = E.A\n\tvar n = casos e |p|\n\t\t.A => 1\n\t\t.B =>\n\t\t\timprimir(p)\n\t\t\t2\n\timprimir(n)\n\tcasos e\n\t\t_ => imprimir(3)\n"
	tokens, err := lexer.Lex("enum.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("enum.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	e := program.Decls[0].(*ast.EnumDecl)
	if len(e.Variants) != 2 || e.Variants[0].Payload != nil || !e.Variants[1].Payload.IsSlice() {
		t.Fatalf("bad variants: %#v", e)
	}
	f := program.Decls[1].(*ast.FuncDecl)
	if len(f.Body) != 4 {
		t.Fatalf("match consumed a following statement: %#v", f.Body)
	}
	m := f.Body[1].(*ast.VarDeclStmt).Value.(*ast.MatchExpr)
	if m.Binding != "p" || len(m.Arms[0].Body) != 1 || len(m.Arms[1].Body) != 2 {
		t.Fatalf("bad match: %#v", m)
	}
	if _, ok := f.Body[3].(*ast.MatchStmt); !ok {
		t.Fatalf("bad statement: %T", f.Body[3])
	}
}

func TestRejectsInvalidEnumAndMatchSyntax(t *testing.T) {
	for _, source := range []string{
		"fn inicio()\n\tcasos e\n\t\tE._ => 1\n",
		"fn inicio()\n\tcasos e\n\t\t.A(1) => 1\n",
		"enum E\nfn inicio() imprimir(1)\n",
		"enum E\nA\n",
		"enum E\n\tA num bool\n",
		"fn inicio()\n\tcasos e\n\t.A => imprimir(1)\n",
		"fn inicio()\n\tcasos e\n\t\tA imprimir(1)\n",
		"fn inicio()\n\tcasos e\n\t\t.A =>\n\t\t.B => imprimir(1)\n",
		"fn inicio()\n\tcasos e\n\t\t.A => imprimir(1) imprimir(2)\n",
		"fn inicio()\n\tcasos e\n\t\t.A => imprimir(1)\n\t\t\timprimir(2)\n",
		"fn inicio() => imprimir(1)\n",
	} {
		tokens, err := lexer.Lex("enum.hacha", source)
		if err != nil {
			continue
		}
		if _, err := Parse("enum.hacha", tokens); err == nil {
			t.Fatalf("accepted invalid syntax:\n%s", source)
		}
	}
}
