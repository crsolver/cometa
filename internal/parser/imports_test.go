package parser

import (
	"cometa/internal/lexer"
	"testing"
)

func TestImportGrammar(t *testing.T) {
	for _, source := range []string{"usar herramientas\n", "usar modelos como m\n", "usar interno/base_de_datos como bd\n", "usar ../../compartido/fechas\n", "usar ./modelos\n"} {
		tokens, err := lexer.Lex("test.cometa", source)
		if err != nil {
			t.Fatal(err)
		}
		p, err := Parse("test.cometa", tokens)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Imports) != 1 {
			t.Fatal(p)
		}
	}
	for _, source := range []string{"usar \"a\"\n", "usar a.cometa\n", "usar /a\n", "usar a /b\n", "usar a/ b\n", "usar a como _\n", "usar a/\n", "fn f() imprimir(1)\nusar a\n", "fn f()\n\tusar a\n"} {
		tokens, err := lexer.Lex("test.cometa", source)
		if err == nil {
			_, err = Parse("test.cometa", tokens)
		}
		if err == nil {
			t.Fatalf("accepted %q", source)
		}
	}
}
