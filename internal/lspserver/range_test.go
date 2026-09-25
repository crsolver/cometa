package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestRangeHover(t *testing.T) {
	source := "fn inicio()\n\tvar fin = 5\n\trepetir (0..fin) |i, indice| imprimir(i)\n"
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///rangos.hacha")
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if d := waitForDiagnostics(t, h, uri); len(d) != 0 {
		t.Fatal(d)
	}
	for _, col := range []int{13, 19, 22, 39} {
		hover, err := h.Hover(uri, 2, col)
		if err != nil || hover == nil || !strings.Contains(hover.Contents.Value(), "entero") {
			t.Fatalf("hover at %d = %+v, %v; want entero", col, hover, err)
		}
	}
}
