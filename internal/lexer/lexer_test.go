package lexer

import (
	"strings"
	"testing"

	"hacha/internal/token"
)

func TestRangeAndDecimalTokens(t *testing.T) {
	tokens, err := Lex("rango.hacha", "0..5 0.5..2.5 1... 1.25 .Variante")
	if err != nil {
		t.Fatal(err)
	}
	want := []token.Kind{token.Number, token.Range, token.Number, token.Number, token.Range, token.Number, token.Number, token.Ellipsis, token.Number, token.Dot, token.Ident, token.Newline, token.EOF}
	if len(tokens) != len(want) {
		t.Fatalf("tokens = %v", tokens)
	}
	for i, kind := range want {
		if tokens[i].Kind != kind {
			t.Fatalf("token %d = %v, want %v", i, tokens[i], kind)
		}
	}
}

func TestIndentationAndComments(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\n\t// comentario ignorado\n\tfn valor() entero\n\t\t1\nfn inicio()\n\timprimir(\"ok\")\n"
	tokens, err := Lex("prueba.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, tok := range tokens {
		kinds = append(kinds, string(tok.Kind))
	}
	joined := strings.Join(kinds, " ")
	for _, expected := range []string{"tipo IDENT NEWLINE INDENT", "fn IDENT ( ) entero NEWLINE INDENT", "NUMBER NEWLINE DEDENT DEDENT fn"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("token stream does not contain %q:\n%s", expected, joined)
		}
	}
}

func TestLiteralsAndOperators(t *testing.T) {
	tokens, err := Lex("prueba.hacha", "fn elegir(a entero) bool si (a <= 4.5) verdadero sino falso\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[token.Kind]bool{token.Number: false, token.LessEq: false, token.True: false, token.False: false}
	for _, tok := range tokens {
		if _, exists := want[tok.Kind]; exists {
			want[tok.Kind] = true
		}
	}
	for kind, found := range want {
		if !found {
			t.Errorf("missing token %s", kind)
		}
	}
}

func TestVariableAndCompositeLiteralTokens(t *testing.T) {
	tokens, err := Lex("variables.hacha", "fn inicio()\n\tvar usuario Usuario = {nombre: \"Ana\"}\n\tusuario.activar(verdadero)\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[token.Kind]bool{token.Var: false, token.LBrace: false, token.Colon: false, token.RBrace: false, token.Dot: false}
	for _, tok := range tokens {
		if _, exists := want[tok.Kind]; exists {
			want[tok.Kind] = true
		}
	}
	for kind, found := range want {
		if !found {
			t.Errorf("token %s was not produced", kind)
		}
	}
}

func TestRepeatAndLoopControlTokens(t *testing.T) {
	tokens, err := Lex("ciclos.hacha", "fn inicio()\n\trepetir ([1, 2]) |valor, indice|\n\t\tsi (indice == 0) continuar\n\t\tromper\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[token.Kind]bool{
		token.Repetir:   false,
		token.Pipe:      false,
		token.Continuar: false,
		token.Romper:    false,
	}
	for _, tok := range tokens {
		if _, exists := want[tok.Kind]; exists {
			want[tok.Kind] = true
		}
	}
	for kind, found := range want {
		if !found {
			t.Errorf("token %s was not produced", kind)
		}
	}
}

func TestRejectsSpaceIndentation(t *testing.T) {
	_, err := Lex("malo.hacha", "fn inicio()\n    imprimir(\"no\")\n")
	if err == nil || !strings.Contains(err.Error(), "malo.hacha:2:1") {
		t.Fatalf("expected positioned indentation error, got %v", err)
	}
}

func TestIgnoresWhitespaceOnBlankAndCommentLines(t *testing.T) {
	_, err := Lex("bien.hacha", "fn inicio()\n   \n  // comentario\n\timprimir(\"sí\")\n")
	if err != nil {
		t.Fatal(err)
	}
}
