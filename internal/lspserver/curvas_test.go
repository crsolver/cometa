package lspserver

import (
	"context"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/crsolver/cometa/internal/stdlib"
	"strings"
	"testing"
)

func TestCurvesTooling(t *testing.T) {
	for _, alias := range []string{"curvas", "c"} {
		imp := "usar std/mate/curvas"
		if alias != "curvas" {
			imp += " como " + alias
		}
		h := NewHandler()
		uri := lsp.DocumentURI("file:///curvas.cometa")
		source, pos := markerPosition(imp + "\nfn inicio() imprimir(" + alias + ".§cubica_entrada(0.5))\n")
		if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
			t.Fatal(err)
		}
		params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}
		list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: params})
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Items) != 31 {
			t.Fatalf("expected 31 curves: %+v", list)
		}
		found := false
		for _, item := range list.Items {
			text := completionText(item)
			if item.Label == "cubica_entrada(...)" && strings.Contains(text, "progreso decimal") && strings.Contains(text, ") decimal") {
				found = true
			}
		}
		if !found {
			t.Fatal("missing curve completion signature")
		}
		hover, err := h.Hover(context.Background(), &lsp.HoverParams{TextDocumentPositionParams: params})
		if err != nil || hover == nil {
			t.Fatalf("hover: %+v %v", hover, err)
		}
		hoverText := hover.Contents.Value()
		if !strings.Contains(hoverText, "cubica_entrada(") || !strings.Contains(hoverText, "progreso decimal") || !strings.Contains(hoverText, ") decimal") {
			t.Fatalf("hover: %+v %v", hover, err)
		}
		locations, err := h.Definition(context.Background(), &lsp.DefinitionParams{TextDocumentPositionParams: params})
		if err != nil || len(locations) != 1 {
			t.Fatalf("definition: %+v %v", locations, err)
		}
		if locations[0].URI != "cometa-std:///std/mate/curvas.cometa" {
			t.Fatalf("wrong URI: %+v", locations)
		}
		reference, ok := stdlib.Source("std/mate/curvas")
		if !ok || !strings.Contains(strings.Split(reference, "\n")[locations[0].Range.Start.Line], "fn cubica_entrada(") {
			t.Fatal("wrong declaration location")
		}
	}
}
