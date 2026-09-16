package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"strings"
	"testing"
)

func TestGenericInterfaceCompletion(t *testing.T) {
	const decl = `interfaz Fuente<T>
	fn obtener() T
tipo Caja<T>
	valor T
	fn obtener() T @valor
enum E<T>
	Dato T
	Vacio
`
	for _, tt := range []struct{ name, body, want string }{
		{"interface", "fn f(v Fuente<num>)\n\tv.§\n", "fn obtener() num"},
		{"constraint", "fn f<T Fuente<cadena>>(v T)\n\tv.§\n", "fn obtener() cadena"},
		{"instance", "fn f(v Caja<num>)\n\tv.§\n", "fn obtener() num"},
		{"field", "fn f(v Caja<cadena>)\n\tv.§\n", "cadena"},
		{"enum", "fn inicio()\n\tvar v = E<num>.§\n", "E<num>.Dato(num)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := decl + tt.body
			mark := strings.Index(source, "§")
			before := source[:mark]
			line := strings.Count(before, "\n")
			col := utf16Length(before[strings.LastIndex(before, "\n")+1:])
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///generic-completion.hacha")
			if err := h.DidOpen(uri, "hacha", strings.Replace(source, "§", "", 1)); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, h, uri)
			list, err := h.Completion(uri, line, col)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Items {
				if item.Detail == tt.want {
					return
				}
			}
			t.Fatalf("missing %q in %+v", tt.want, list.Items)
		})
	}
}

func TestGenericHoverAndInterfaceSymbols(t *testing.T) {
	source := `interfaz Fuente<T>
	fn obtener() T
tipo Caja<T>
	valor T
	fn obtener() T @valor
fn identidad<T>(v T) T v
fn inicio()
	var c = Caja<num> {valor: 1}
	imprimir(c.obtener())
	imprimir(identidad(2))
`
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///generic-hover.hacha")
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	diagnostics := waitForDiagnostics(t, h, uri)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	for _, tt := range []struct{ needle, want string }{
		{"c.obtener", "var c Caja<num>"},
		{"obtener())", "fn obtener() num"},
		{"identidad(2)", "fn identidad<num>(v num) num"},
	} {
		at := strings.Index(source, tt.needle)
		before := source[:at]
		hover, err := h.Hover(uri, strings.Count(before, "\n"), utf16Length(before[strings.LastIndex(before, "\n")+1:]))
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil || !strings.Contains(hover.Contents.Value(), tt.want) {
			t.Fatalf("hover %s: %+v", tt.needle, hover)
		}
	}
	symbols, err := h.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 4 || symbols[0].Kind != lsp.SymbolKindInterface || len(symbols[0].Children) != 1 || symbols[0].Detail != "interfaz<T>" {
		t.Fatalf("symbols: %+v", symbols)
	}
}
