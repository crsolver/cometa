package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestGlobalSymbolsAndHover(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///globales.hacha")
	source := "const limite = 10\nvar contador num = limite\nfn inicio()\n\timprimir(contador)\n"
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	symbols, err := harness.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 3 || symbols[0].Kind != lsp.SymbolKindConstant || symbols[1].Kind != lsp.SymbolKindVariable {
		t.Fatalf("global symbols = %+v", symbols)
	}
	hover, err := harness.Hover(uri, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil || !strings.Contains(hover.Contents.Value(), "const limite num") {
		t.Fatalf("global hover = %+v", hover)
	}
}
