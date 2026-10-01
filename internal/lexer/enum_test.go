package lexer

import (
	"github.com/crsolver/cometa/internal/token"
	"testing"
)

func TestEnumAndMatchTokens(t *testing.T) {
	tokens, err := Lex("enum.cometa", "enum E\n\tA\n\t// comment\nfn inicio()\n\tcasos E.A |e|\n\t\t_ => imprimir(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[token.Kind]int{}
	for _, tok := range tokens {
		counts[tok.Kind]++
	}
	for _, kind := range []token.Kind{token.Enum, token.Casos, token.Arrow} {
		if counts[kind] != 1 {
			t.Errorf("%s count = %d", kind, counts[kind])
		}
	}
	if counts[token.Indent] != 3 || counts[token.Dedent] != 3 {
		t.Fatalf("unbalanced match indentation: %v", counts)
	}
}
