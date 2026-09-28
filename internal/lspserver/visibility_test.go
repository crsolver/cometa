package lspserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

const privateLibrary = `fn secreto() entero 1
pub fn crear() Caja Caja {}
pub tipo Caja
	privado entero
	pub visible entero
	fn oculto() entero 2
	pub fn leer() entero @privado
pub tipo Cerrada
	Caja
`

func TestVisibilityCompletionAndNavigation(t *testing.T) {
	uri, depURI := moduleURIs(t)
	path, _ := pathFromURI(depURI)
	if err := os.WriteFile(path, []byte(privateLibrary), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, source    string
		present, absent []string
	}{
		{"namespace", "usar modelos como m\nfn inicio()\n\tm.§\n", []string{"crear()", "Caja"}, []string{"secreto"}},
		{"members", "usar modelos como m\nfn f(c m.Caja)\n\tc.§\n", []string{"visible", "leer()"}, []string{"privado", "oculto"}},
		{"promotion", "usar modelos como m\nfn f(c m.Cerrada)\n\tc.§\n", nil, []string{"Caja", "visible", "leer", "privado", "oculto"}},
		{"own file", "usar std/mate\n" + privateLibrary + "fn f(c Caja)\n\tc.§\n", []string{"privado", "oculto()", "visible", "leer()"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, pos := markerPosition(tc.source)
			h := NewHandler()
			_, _ = h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}})
			list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}})
			if err != nil {
				t.Fatal(err)
			}
			labels := map[string]bool{}
			for _, item := range list.Items {
				labels[item.Label] = true
			}
			for _, name := range tc.present {
				if !labels[name] {
					t.Fatalf("missing %s: %v", name, labels)
				}
			}
			for _, name := range tc.absent {
				if labels[name] {
					t.Fatalf("private %s: %v", name, labels)
				}
			}
		})
	}
	for _, body := range []string{"imprimir(m.§secreto())", "var c = m.crear()\n\timprimir(c.§oculto())", "var c = m.crear()\n\timprimir(c.§privado)"} {
		source, pos := markerPosition("usar modelos como m\nfn inicio()\n\t" + body + "\n")
		h := NewHandler()
		_, _ = h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}})
		params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}
		hover, err := h.Hover(context.Background(), &lsp.HoverParams{TextDocumentPositionParams: params})
		if err != nil || hover != nil {
			t.Fatalf("private hover: %+v %v", hover, err)
		}
		locations, err := h.Definition(context.Background(), &lsp.DefinitionParams{TextDocumentPositionParams: params})
		if err != nil || len(locations) != 0 {
			t.Fatalf("private definition: %+v %v", locations, err)
		}
	}
}

func TestVisibilityOverlayUpdates(t *testing.T) {
	uri, depURI := moduleURIs(t)
	path, _ := pathFromURI(depURI)
	if err := os.WriteFile(filepath.Clean(path), []byte("fn f() entero 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	h := servertest.New(t, NewHandler())
	if err := h.DidOpen(uri, "cometa", "usar modelos como m\nfn inicio() imprimir(m.f())\n"); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 1 || !strings.Contains(ds[0].Message, "privada") {
		t.Fatal(ds)
	}
	h.ClearDiagnostics()
	if err := h.DidOpen(depURI, "cometa", "pub fn f() entero 1\n"); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 0 {
		t.Fatal(ds)
	}
	h.ClearDiagnostics()
	if err := h.DidChange(depURI, 2, "fn f() entero 1\n"); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 1 || !strings.Contains(ds[0].Message, "privada") {
		t.Fatal(ds)
	}
}
