package lspserver

import (
	"context"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"cometa/internal/compiler"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const moduleLibrary = `// Devuelve el argumento.
pub fn identidad<T>(valor T) T valor
pub tipo Usuario
	pub nombre cadena
	pub fn describir() cadena @nombre
pub tipo Caja<T>
	pub valor T
	pub fn obtener() T @valor
pub enum Evento<T>
	Dato T
	Vacio
pub interfaz Proveedor<T>
	fn obtener() T
pub const limite = 10
pub var contador = 0
`

func moduleURIs(t *testing.T) (lsp.DocumentURI, lsp.DocumentURI) {
	t.Helper()
	dir, err := compiler.CanonicalPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "modelos.cometa"), []byte(moduleLibrary), 0600); err != nil {
		t.Fatal(err)
	}
	return fileURI(filepath.Join(dir, "main.cometa")), fileURI(filepath.Join(dir, "modelos.cometa"))
}

func markerPosition(source string) (string, lsp.Position) {
	i := strings.Index(source, "§")
	before := source[:i]
	return strings.Replace(source, "§", "", 1), lsp.Position{Line: strings.Count(before, "\n"), Character: utf16Length(before[strings.LastIndex(before, "\n")+1:])}
}

func TestModuleCompletion(t *testing.T) {
	for _, tc := range []struct{ name, body, label, detail string }{
		{"namespace", "fn inicio()\n\tm.§\n", "Usuario", "pub tipo Usuario"},
		{"qualified type", "pub fn f(valor m.§)\n\timprimir(valor)\n", "Usuario", "pub tipo Usuario"},
		{"generic enum", "fn inicio()\n\tvar e = m.Evento<entero>.§\n", "Dato", "modelos.Evento<entero>.Dato(entero)"},
		{"imported variable", "pub fn f(valor m.Usuario)\n\tvalor.§\n", "nombre", "cadena"},
		{"generic variable", "pub fn f(valor m.Caja<entero>)\n\tvalor.§\n", "obtener()", "pub fn obtener() entero"},
		{"interface", "pub fn f(valor m.Proveedor<cadena>)\n\tvalor.§\n", "obtener()", "fn obtener() cadena"},
		{"constraint", "pub fn f<T m.Proveedor<entero>>(valor T)\n\tvalor.§\n", "obtener()", "fn obtener() entero"},
		{"contextual", "pub fn f() m.Evento<entero>\n\t.§\n", "Dato", "modelos.Evento<entero>.Dato(entero)"},
		{"shadow", "pub fn f(m m.Usuario)\n\tm.§\n", "nombre", "cadena"},
		{"alias", "fn inicio()\n\tm§\n", "m", "módulo m"},
		{"root type argument", "pub tipo Propio\n\tpub x entero\nfn inicio()\n\tvar e = m.Evento<Propio>.§\n", "Dato", "modelos.Evento<Propio>.Dato(Propio)"},
		{"global constant", "fn inicio()\n\timprimir(m.§)\n", "limite", "pub const limite entero"},
		{"global variable", "fn inicio()\n\timprimir(m.§)\n", "contador", "pub var contador entero"},
		{"list variable", "fn inicio()\n\tvar valores = [1]\n\tvalores.§\n", "agregar(...)", "fn agregar(valor entero)"},
		{"map variable", "fn inicio()\n\tvar valores = [\"a\": 1]\n\tvalores.§\n", "obtener(...)", "fn obtener(clave cadena) entero?"},
		{"string variable", "fn inicio()\n\tvar valor = \"a\"\n\tvalor.§\n", "longitud()", "fn longitud() entero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uri, _ := moduleURIs(t)
			source, pos := markerPosition("usar modelos como m\n" + tc.body)
			h := NewHandler()
			_, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}})
			if err != nil {
				t.Fatal(err)
			}
			list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}})
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Items {
				if item.Label == tc.label && completionText(item) == tc.detail {
					return
				}
			}
			t.Fatalf("missing %s / %s: %+v", tc.label, tc.detail, list)
		})
	}
}

func TestUsarPathCompletion(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"empty", "usar §\n", "std/mate"},
		{"partial", "usar std/pin§\n", "std/pincel"},
		{"submodule", "usar std/pincel/gra§\n", "std/pincel/graficos"},
		{"curvas", "usar std/mate/cu§\n", "std/mate/curvas"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, pos := markerPosition(tc.source)
			uri := fileURI(filepath.Join(t.TempDir(), "main.cometa"))
			h := NewHandler()
			if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
				t.Fatal(err)
			}
			list, err := h.Completion(context.Background(), &lsp.CompletionParams{TextDocumentPositionParams: lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}})
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Items {
				if item.Label == tc.want {
					if item.TextEdit == nil || item.TextEdit.TextEdit.NewText != tc.want {
						t.Fatalf("missing text edit for %s: %+v", tc.want, item)
					}
					return
				}
			}
			t.Fatalf("missing %s: %+v", tc.want, list.Items)
		})
	}
}

func TestModuleHoverAndDefinition(t *testing.T) {
	uri, depURI := moduleURIs(t)
	for _, tc := range []struct {
		body, want string
		line       int
	}{
		{"fn inicio() imprimir(m.§identidad(1))\n", "identidad<entero>(valor entero) entero", 1},
		{"pub fn f(valor m.§Usuario)\n\timprimir(valor)\n", "pub tipo Usuario", 2},
		{"pub fn f(valores [cadena: m.§Usuario])\n\timprimir(valores.longitud())\n", "pub tipo Usuario", 2},
		{"fn inicio()\n\tvar valores = [\"a\": m.§identidad(1)]\n", "identidad<entero>(valor entero) entero", 1},
		{"fn inicio()\n\tvar x = m.Evento<entero>.§Dato(1)\n\timprimir(x)\n", "Evento<entero>.Dato(entero)", 9},
		{"pub fn f(valor m.Usuario)\n\timprimir(valor.§nombre)\n", "cadena", 3},
		{"pub fn f(valor m.Usuario)\n\timprimir(valor.§describir())\n", "pub fn describir() cadena", 4},
		{"fn inicio() imprimir(m.§identidad<entero>(1))\n", "identidad<entero>(valor entero) entero", 1},
		{"fn inicio() imprimir(m.§limite)\n", "pub const limite entero", 13},
		{"fn inicio() imprimir(m.§contador)\n", "pub var contador entero", 14},
	} {
		source, pos := markerPosition("usar modelos como m\n" + tc.body)
		h := NewHandler()
		_, _ = h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}})
		params := lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}
		hover, err := h.Hover(context.Background(), &lsp.HoverParams{TextDocumentPositionParams: params})
		if err != nil || hover == nil || !strings.Contains(flattenWrapped(hover.Contents.Value()), tc.want) {
			t.Fatalf("hover %s: %+v %v", tc.want, hover, err)
		}
		locations, err := h.Definition(context.Background(), &lsp.DefinitionParams{TextDocumentPositionParams: params})
		if err != nil || len(locations) != 1 || locations[0].URI != depURI || locations[0].Range.Start.Line != tc.line {
			t.Fatalf("definition %s: %+v %v", tc.want, locations, err)
		}
	}
}

func TestModuleOverlayDiagnosticsAndClose(t *testing.T) {
	uri, depURI := moduleURIs(t)
	h := servertest.New(t, NewHandler())
	if err := h.DidOpen(uri, "cometa", "usar modelos como m\nfn inicio() imprimir(m.identidad(1))\n"); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 0 {
		t.Fatal(ds)
	}
	bad := "pub fn identidad(valor entero) entero\n\t\"😀\" + ausente\n"
	h.ClearDiagnostics()
	if err := h.DidOpen(depURI, "cometa", bad); err != nil {
		t.Fatal(err)
	}
	ds := waitForDiagnostics(t, h, depURI)
	if len(ds) != 1 || ds[0].Range.Start.Line != 1 || ds[0].Range.Start.Character != 8 {
		t.Fatalf("dependency diagnostics: %+v", ds)
	}
	h.ClearDiagnostics()
	if err := h.DidChange(depURI, 2, "pub fn identidad(valor cadena) cadena valor\n"); err != nil {
		t.Fatal(err)
	}
	// Drain previously published root reports until the changed signature appears.
	for i := 0; i < 3; i++ {
		ds = waitForDiagnostics(t, h, uri)
		if len(ds) != 0 {
			break
		}
	}
	if len(ds) != 1 {
		t.Fatalf("dependent not rechecked: %+v", ds)
	}
	h.ClearDiagnostics()
	if err := h.DidClose(depURI); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		ds = waitForDiagnostics(t, h, uri)
		if len(ds) == 0 {
			break
		}
	}
	if len(ds) != 0 {
		t.Fatalf("disk fallback: %+v", ds)
	}
}

func TestModuleWatchedCreationDeletion(t *testing.T) {
	uri, _ := moduleURIs(t)
	rootPath, _ := pathFromURI(uri)
	missing := filepath.Join(filepath.Dir(rootPath), "nuevo.cometa")
	h := servertest.New(t, NewHandler())
	if err := h.DidOpen(uri, "cometa", "usar nuevo\nfn inicio() nuevo.f()\n"); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 1 {
		t.Fatal(ds)
	}
	if err := os.WriteFile(missing, []byte("pub fn f() imprimir(1)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	h.ClearDiagnostics()
	if err := h.DidChangeWatchedFiles(&lsp.DidChangeWatchedFilesParams{Changes: []lsp.FileEvent{{URI: fileURI(missing), Type: lsp.FileCreated}}}); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 0 {
		t.Fatal(ds)
	}
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	h.ClearDiagnostics()
	if err := h.DidChangeWatchedFiles(&lsp.DidChangeWatchedFilesParams{Changes: []lsp.FileEvent{{URI: fileURI(missing), Type: lsp.FileDeleted}}}); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 1 {
		t.Fatal(ds)
	}
}
