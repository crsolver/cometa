package parser

import (
	"hacha/internal/ast"
	"hacha/internal/lexer"
	"testing"
)

func TestEmbeddedFields(t *testing.T) {
	tokens, err := lexer.Lex("embedding.hacha", "tipo Empleado<T>\n\tPersona // anónimo\n\tCaja<T>\n\tpuesto cadena\n\terror !\n")
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("embedding.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	fields := program.Decls[0].(*ast.TypeDecl).Fields
	if len(fields) != 4 || !fields[0].Embedded || fields[0].Name != "Persona" || !fields[1].Embedded || fields[1].Name != "Caja" || fields[1].Type.Args[0].Name != "T" || fields[2].Embedded || fields[3].Embedded || fields[3].Name != "error" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	for _, source := range []string{"tipo A\n\tCaja<>\n", "tipo A\n\tCaja<num> extra\n", "tipo A\n\tPersona Persona extra\n"} {
		tokens, err := lexer.Lex("bad.hacha", source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse("bad.hacha", tokens); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
}
