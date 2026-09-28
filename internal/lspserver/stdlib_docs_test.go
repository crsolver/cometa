package lspserver

import (
	"context"
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
)

// TestStdlibDocsShowInHoverAndCompletion checks that the Spanish doc
// comments written into stdlib.FunctionDocs/MethodDocs (internal/stdlib/docs.go)
// actually reach the editor: as documentation text under a function's hover,
// and in the Documentation panel of its completion item.
func TestStdlibDocsShowInHoverAndCompletion(t *testing.T) {
	h := NewHandler()
	uri := lsp.DocumentURI("file:///stdlib-docs.cometa")
	source, pos := markerPosition("usar std/mate\nfn inicio()\n\timprimir(mate.§absoluto(-5))\n")
	if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
		t.Fatal(err)
	}
	params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}

	hover, err := h.Hover(context.Background(), &lsp.HoverParams{TextDocumentPositionParams: params})
	if err != nil || hover == nil || !strings.Contains(hover.Contents.Value(), "Valor absoluto") {
		t.Fatalf("hover missing stdlib doc: %+v %v", hover, err)
	}

	list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: params})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list.Items {
		if item.Label == "absoluto(...)" {
			if item.Documentation == nil || !strings.Contains(item.Documentation.Value, "Valor absoluto") {
				t.Fatalf("completion documentation missing stdlib doc: %+v", item)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("missing absoluto completion: %+v", list.Items)
	}
}

// TestStdlibMethodDocsShowInCompletion checks that native struct methods
// (Vec2, Rect) also carry their doc comment into the completion panel.
func TestStdlibMethodDocsShowInCompletion(t *testing.T) {
	h := NewHandler()
	uri := lsp.DocumentURI("file:///stdlib-method-docs.cometa")
	source, pos := markerPosition("usar std/mate\nfn inicio()\n\tvar v = mate.Vec2 {}\n\timprimir(v.§normalizado())\n")
	if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
		t.Fatal(err)
	}
	params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}
	list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: params})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		if item.Label == "normalizado()" {
			if item.Documentation == nil || !strings.Contains(item.Documentation.Value, "longitud 1") {
				t.Fatalf("method completion documentation missing stdlib doc: %+v", item)
			}
			return
		}
	}
	t.Fatalf("missing normalizado completion: %+v", list.Items)
}
