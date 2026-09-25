package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"testing"
)

func TestMultipleDiagnosticsPartialCorrection(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///multiples.cometa")
	source := "fn inicio()\n\timprimir(\"😀\" + ausente)\n\tvar b bool = 1\n\tvar c =\n"
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	ds := waitForDiagnostics(t, h, uri)
	if len(ds) != 3 {
		t.Fatalf("diagnostics: %+v", ds)
	}
	if ds[0].Range.Start.Line != 1 || ds[0].Range.Start.Character != 17 {
		t.Fatalf("UTF-16: %+v", ds[0])
	}
	h.ClearDiagnostics()
	if err := h.Notify("textDocument/didChange", map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 2},
		"contentChanges": []any{
			map[string]any{"range": lsp.Range{Start: lsp.Position{Line: 1, Character: 14}, End: lsp.Position{Line: 1, Character: 24}}, "text": ""},
			map[string]any{"range": lsp.Range{Start: lsp.Position{Line: 3, Character: 8}, End: lsp.Position{Line: 3, Character: 8}}, "text": " 2"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	ds = waitForDiagnostics(t, h, uri)
	if len(ds) != 1 || ds[0].Range.Start.Line != 2 {
		t.Fatalf("remaining diagnostics: %+v", ds)
	}
	h.ClearDiagnostics()
	if err := h.DidChange(uri, 3, "fn inicio()\n\timprimir(\"😀\")\n\tvar b bool = verdadero\n\tvar c = 2\n"); err != nil {
		t.Fatal(err)
	}
	if ds = waitForDiagnostics(t, h, uri); len(ds) != 0 {
		t.Fatalf("not cleared: %+v", ds)
	}
}

func TestMultipleDependencyDiagnostics(t *testing.T) {
	uri, depURI := moduleURIs(t)
	h := servertest.New(t, NewHandler())
	if err := h.DidOpen(uri, "cometa", "usar modelos como m\nfn inicio() imprimir(m.identidad(1))\n"); err != nil {
		t.Fatal(err)
	}
	waitForDiagnostics(t, h, uri)
	h.ClearDiagnostics()
	if err := h.DidOpen(depURI, "cometa", "fn identidad(valor entero) entero\n\tvar a bool = 1\n\tvar b =\n\tvalor\n"); err != nil {
		t.Fatal(err)
	}
	ds := waitForDiagnostics(t, h, depURI)
	if len(ds) != 2 {
		t.Fatalf("dependency diagnostics duplicated or missing: %+v", ds)
	}
	h.ClearDiagnostics()
	if err := h.DidChange(depURI, 2, "fn identidad(valor entero) entero\n\tvar a bool = verdadero\n\tvar b =\n\tvalor\n"); err != nil {
		t.Fatal(err)
	}
	if ds = waitForDiagnostics(t, h, depURI); len(ds) != 1 || ds[0].Range.Start.Line != 2 {
		t.Fatalf("partial dependency correction: %+v", ds)
	}
	h.ClearDiagnostics()
	if err := h.DidClose(depURI); err != nil {
		t.Fatal(err)
	}
	if ds = waitForDiagnostics(t, h, depURI); len(ds) != 0 {
		t.Fatalf("closed overlay diagnostics: %+v", ds)
	}
}

func TestDistinctDiagnosticsAtSameRange(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///campos.cometa")
	if err := h.DidOpen(uri, "cometa", "enum E\n\tA\ntipo T\n\ta E\n\tb E\nfn inicio()\n\tvar t T = {}\n"); err != nil {
		t.Fatal(err)
	}
	ds := waitForDiagnostics(t, h, uri)
	if len(ds) != 2 || ds[0].Range != ds[1].Range || ds[0].Message == ds[1].Message {
		t.Fatalf("distinct messages at same range: %+v", ds)
	}
}
