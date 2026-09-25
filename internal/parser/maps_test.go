package parser

import (
	"testing"

	"cometa/internal/ast"
	"cometa/internal/lexer"
)

func TestMapSyntax(t *testing.T) {
	for _, literal := range []string{`[:]`, `["a": [1, 2],]`, "[\n\t// entrada\n\t\"a\": [1, 2],\n]", "[\n\t:\n]"} {
		tokens, err := lexer.Lex("mapas.cometa", "var m [cadena: [entero]] = "+literal+"\n")
		if err != nil {
			t.Fatal(err)
		}
		p, err := Parse("mapas.cometa", tokens)
		if err != nil {
			t.Fatalf("%s: %v", literal, err)
		}
		g := p.Decls[0].(*ast.GlobalDecl)
		if g.Type.Key.Name != "cadena" || !g.Type.Element.IsSlice() || g.Type.IsSlice() {
			t.Fatalf("bad type: %+v", g.Type)
		}
		if _, ok := g.Value.(*ast.MapLiteralExpr); !ok {
			t.Fatalf("bad literal: %T", g.Value)
		}
	}
}

func TestMalformedMapsRecover(t *testing.T) {
	for _, literal := range []string{`["a":]`, `["a": 1, 2]`, `[1, "a": 2]`, `[::]`} {
		tokens, err := lexer.Lex("mapas.cometa", "var m = "+literal+"\nfn siguiente() entero 4\n")
		if err != nil {
			t.Fatal(err)
		}
		p, err := Parse("mapas.cometa", tokens)
		if err == nil {
			t.Fatalf("accepted %s", literal)
		}
		found := false
		for _, d := range p.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Name == "siguiente" {
				found = true
			}
		}
		if !found {
			t.Fatalf("did not recover after %s", literal)
		}
	}
}
