package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"strings"
	"testing"
)

func TestWrapSignatureBreaksLongParameterLists(t *testing.T) {
	long := "pub fn fractal(x decimal, y decimal, semilla entero = 0, octavas entero = 4, persistencia decimal = 0.5, lacunaridad decimal = 2.0) decimal"
	got := wrapSignature(long)
	want := "pub fn fractal(\n" +
		"\tx decimal,\n" +
		"\ty decimal,\n" +
		"\tsemilla entero = 0,\n" +
		"\toctavas entero = 4,\n" +
		"\tpersistencia decimal = 0.5,\n" +
		"\tlacunaridad decimal = 2.0,\n" +
		") decimal"
	if got != want {
		t.Fatalf("wrapSignature = %q, want %q", got, want)
	}
}

func TestWrapSignatureLeavesShortSignaturesAlone(t *testing.T) {
	short := "fn f(a entero) entero"
	if got := wrapSignature(short); got != short {
		t.Fatalf("wrapSignature = %q, want unchanged %q", got, short)
	}
}

// A single long parameter still needs wrapping — splitting a comma-joined
// list into one part is not a reason to skip it.
func TestWrapSignatureBreaksSingleLongParameter(t *testing.T) {
	long := "pub fn captura_raton(contexto ui.ContextoDeEntradaMuyLargoParaProbar) bool"
	got := wrapSignature(long)
	want := "pub fn captura_raton(\n" +
		"\tcontexto ui.ContextoDeEntradaMuyLargoParaProbar,\n" +
		") bool"
	if got != want {
		t.Fatalf("wrapSignature = %q, want %q", got, want)
	}
}

func TestGenericInterfaceCompletion(t *testing.T) {
	const decl = `interfaz Proveedor<T>
	fn obtener() T
tipo Caja<T>
	valor T
	fn obtener() T @valor
enum E<T>
	Dato T
	Vacio
`
	for _, tt := range []struct{ name, body, want string }{
		{"interface", "fn f(v Proveedor<entero>)\n\tv.§\n", "fn obtener() entero"},
		{"constraint", "fn f<T Proveedor<cadena>>(v T)\n\tv.§\n", "fn obtener() cadena"},
		{"instance", "fn f(v Caja<entero>)\n\tv.§\n", "fn obtener() entero"},
		{"field", "fn f(v Caja<cadena>)\n\tv.§\n", "cadena"},
		{"enum", "fn inicio()\n\tvar v = E<entero>.§\n", "E<entero>.Dato(entero)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := decl + tt.body
			mark := strings.Index(source, "§")
			before := source[:mark]
			line := strings.Count(before, "\n")
			col := utf16Length(before[strings.LastIndex(before, "\n")+1:])
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///generic-completion.cometa")
			if err := h.DidOpen(uri, "cometa", strings.Replace(source, "§", "", 1)); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, h, uri)
			list, err := h.Completion(uri, line, col)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Items {
				if completionText(item) == tt.want {
					return
				}
			}
			t.Fatalf("missing %q in %+v", tt.want, list.Items)
		})
	}
}

func TestGenericHoverAndInterfaceSymbols(t *testing.T) {
	source := `interfaz Proveedor<T>
	fn obtener() T
tipo Caja<T>
	valor T
	fn obtener() T @valor
fn identidad<T>(v T) T v
fn inicio()
	var c = Caja<entero> {valor: 1}
	imprimir(c.obtener())
	imprimir(identidad(2))
`
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///generic-hover.cometa")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	diagnostics := waitForDiagnostics(t, h, uri)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	for _, tt := range []struct{ needle, want string }{
		{"c.obtener", "var c Caja<entero>"},
		{"obtener())", "fn obtener() entero"},
		{"identidad(2)", "fn identidad<entero>(v entero) entero"},
	} {
		at := strings.Index(source, tt.needle)
		before := source[:at]
		hover, err := h.Hover(uri, strings.Count(before, "\n"), utf16Length(before[strings.LastIndex(before, "\n")+1:]))
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil || !strings.Contains(flattenWrapped(hover.Contents.Value()), tt.want) {
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
