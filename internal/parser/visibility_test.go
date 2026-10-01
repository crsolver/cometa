package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/lexer"
	"testing"
)

func TestPublicDeclarations(t *testing.T) {
	source := "pub tipo T\n\tpub valor entero\n\tpub fn leer() entero @valor\n\tpub Otra\ntipo Otra\n\tx entero\npub fn f() entero 1\npub var v = 1\npub const c = 2\npub enum E\n\tUno\npub interfaz I\n\tfn leer() entero\n"
	tokens, err := lexer.Lex("pub.cometa", source)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("pub.cometa", tokens)
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range p.Decls {
		if ast.IsPublic(d) != (i != 1) {
			t.Fatalf("visibility: %#v", d)
		}
	}
	d := p.Decls[0].(*ast.TypeDecl)
	if !d.Fields[0].Public || !d.Fields[1].Public || !d.Methods[0].Public || d.NamePos.Column != 10 || d.Fields[0].Pos.Column != 6 {
		t.Fatalf("positions/visibility: %#v", d)
	}
}

func TestInvalidPublicModifierRecovery(t *testing.T) {
	for _, source := range []string{
		"pub pub fn f() imprimir(0)\n",
		"pub usar biblioteca\n",
		"fn f()\n\tpub var x = 1\n",
		"fn f(pub x entero) imprimir(x)\n",
		"enum E\n\tpub Uno\n",
		"interfaz I\n\tpub fn f()\n",
		"pub fn inicio() imprimir(0)\n",
		"pub fn pub() imprimir(0)\n",
		"tipo T\n\tpub pub x entero\n",
	} {
		t.Run(source, func(t *testing.T) {
			tokens, _ := lexer.Lex("pub.cometa", source+"pub fn valido() imprimir(0)\n")
			p, err := Parse("pub.cometa", tokens)
			if err == nil {
				t.Fatal("accepted invalid pub")
			}
			last, ok := p.Decls[len(p.Decls)-1].(*ast.FuncDecl)
			if !ok || last.Name != "valido" || !last.Public {
				t.Fatalf("recovery: %#v", p)
			}
		})
	}
}
