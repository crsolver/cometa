package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestQualifiedLibraryConstants(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///constantes-biblioteca.cometa")
	source := "usar std/pincel/retro\n\nfn inicio()\n\tvar flecha = retro.Icono.Arriba\n\tretro.Icono.\n"
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, h, uri)
	list, err := h.Completion(uri, 4, len("\tretro.Icono."))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list.Items {
		found = found || item.Label == "Arriba"
	}
	if !found {
		t.Fatalf("falta Arriba en %+v", list.Items)
	}
	hover, err := h.Hover(uri, 3, len("\tvar flecha = retro.Icono.A"))
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil || !strings.Contains(hover.Contents.Value(), "retro.Icono.Arriba") {
		t.Fatalf("hover inesperado: %+v", hover)
	}
}
