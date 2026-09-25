package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"strings"
	"testing"
)

const embeddingDeclarations = `tipo Caja<T>
	valor T
	// Devuelve el contenido.
	fn obtener() T @valor
tipo Otra
	valor cadena
	fn obtener() cadena @valor
tipo Derivada
	Caja<entero>
tipo Ambigua
	Caja<entero>
	Otra
`

func TestEmbeddingCompletion(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		promoted   bool
	}{
		{"variable", "fn f(v Derivada)\n\tv.§\n", true},
		{"receiver", "tipo Receptor\n\tDerivada\n\tfn f()\n\t\t@§\n", true},
		{"ambiguous", "fn f(v Ambigua)\n\tv.§\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := embeddingDeclarations + tc.body
			before := source[:strings.Index(source, "§")]
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///embedding.cometa")
			if err := h.DidOpen(uri, "cometa", strings.Replace(source, "§", "", 1)); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, h, uri)
			list, err := h.Completion(uri, strings.Count(before, "\n"), utf16Length(before[strings.LastIndex(before, "\n")+1:]))
			if err != nil {
				t.Fatal(err)
			}
			found := map[string]string{}
			for _, item := range list.Items {
				found[item.Label] = item.Detail
			}
			if tc.promoted {
				if found["valor"] != "entero" || found["obtener"] != "fn obtener() entero" {
					t.Fatalf("items: %+v", found)
				}
			} else if found["valor"] != "" || found["obtener"] != "" || found["Caja"] != "Caja<entero>" || found["Otra"] != "Otra" {
				t.Fatalf("items: %+v", found)
			}
		})
	}
}

func TestEmbeddingHoverAndSymbols(t *testing.T) {
	source := embeddingDeclarations + "tipo Receptor\n\tDerivada\n\tfn f() entero @valor\nfn f(v Derivada)\n\timprimir(v.valor)\n\timprimir(v.obtener())\n"
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///embedding.cometa")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, h, uri); len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	for _, tc := range []struct{ needle, want string }{{"valor)", "var valor entero"}, {"valor\nfn f", "var valor entero"}, {"obtener())", "fn obtener() entero"}} {
		before := source[:strings.Index(source, tc.needle)]
		hover, err := h.Hover(uri, strings.Count(before, "\n"), utf16Length(before[strings.LastIndex(before, "\n")+1:]))
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil || !strings.Contains(hover.Contents.Value(), tc.want) {
			t.Fatalf("hover %s: %+v", tc.needle, hover)
		}
		if tc.needle == "obtener())" && !strings.Contains(hover.Contents.Value(), "Devuelve el contenido.") {
			t.Fatalf("lost method documentation: %+v", hover)
		}
	}
	symbols, err := h.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	field := symbols[2].Children[0]
	if field.Name != "Caja" || field.Detail != "Caja<entero>" || field.Kind != lsp.SymbolKindField {
		t.Fatalf("symbol: %+v", field)
	}
}
