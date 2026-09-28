package lspserver

import (
	"context"
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"cometa/internal/stdlib"
)

func TestNoiseTooling(t *testing.T) {
	for _, alias := range []string{"ruido", "r"} {
		imp := "usar std/mate/ruido"
		if alias != "ruido" {
			imp += " como " + alias
		}
		for _, name := range []string{"suave", "fractal"} {
			h := NewHandler()
			uri := lsp.DocumentURI("file:///ruido.cometa")
			source, pos := markerPosition(imp + "\nfn inicio() imprimir(" + alias + ".§" + name + "(0, 0))\n")
			if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
				t.Fatal(err)
			}
			params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}
			list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: params})
			if err != nil {
				t.Fatal(err)
			}
			if len(list.Items) != 2 {
				t.Fatalf("expected two noise functions: %+v", list)
			}
			found := false
			for _, item := range list.Items {
				text := completionText(item)
				if item.Label == name+"(...)" && strings.Contains(text, "x decimal") && strings.Contains(text, "y decimal") && strings.Contains(text, "semilla entero = 0") {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing noise completion signature: %+v", list.Items)
			}
			hover, err := h.Hover(context.Background(), &lsp.HoverParams{TextDocumentPositionParams: params})
			if err != nil || hover == nil {
				t.Fatalf("hover: %+v %v", hover, err)
			}
			hoverText := hover.Contents.Value()
			if !strings.Contains(hoverText, name+"(") || !strings.Contains(hoverText, "x decimal") || !strings.Contains(hoverText, "semilla entero = 0") {
				t.Fatalf("hover: %+v %v", hover, err)
			}
			locations, err := h.Definition(context.Background(), &lsp.DefinitionParams{TextDocumentPositionParams: params})
			if err != nil || len(locations) != 1 {
				t.Fatalf("definition: %+v %v", locations, err)
			}
			if locations[0].URI != "cometa-std:///std/mate/ruido.cometa" {
				t.Fatalf("wrong URI: %+v", locations)
			}
			reference, ok := stdlib.Source("std/mate/ruido")
			if !ok || !strings.Contains(strings.Split(reference, "\n")[locations[0].Range.Start.Line], "fn "+name+"(") {
				t.Fatal("wrong declaration location")
			}
		}
	}
}
