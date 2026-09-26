package lspserver

import (
	"context"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"cometa/internal/stdlib"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPincelImportedDefinitions(t *testing.T) {
	for _, tc := range []struct{ source, path, needle string }{
		{"usar std/pincel\nfn f(g pincel.Juego)\n\tpincel.§ejecutar(g, retro = verdadero) capturar |e| imprimir(e)\n", "std/pincel", "retro bool = falso"},
		{"usar std/pincel/retro como r\nfn pintar() r.§texto(\"hola\", 0, 0)\n", "std/pincel/retro", "fn texto("},
		{"usar std/pincel/retro como r\nfn pintar() r.§icono(.Llave, 0, 0)\n", "std/pincel/retro", "fn icono("},
		{"usar std/pincel/retro como r\nfn f(valor r.§Icono) imprimir(valor)\n", "std/pincel/retro", "tipo Icono"},
		{"usar std/pincel/graficos como g\nfn pintar() g.§limpiar(.Negro)\n", "std/pincel/graficos", "fn limpiar("},
		{"usar std/pincel/lienzo como l\nvar s = l.§desde_texto([\"a\"], [\"a\": .Rojo])\n", "std/pincel/lienzo", "fn desde_texto("},
		{"usar std/mate como m\nfn inicio() imprimir(m.Vec2 {3,4}.§longitud())\n", "std/mate", "fn longitud("},
		{"usar std/pincel/color como c\nfn f(valor c.§Color) imprimir(valor)\n", "std/pincel/color", "tipo Color"},
	} {
		h := NewHandler()
		uri := lsp.DocumentURI("file:///native-def.cometa")
		source, pos := markerPosition(tc.source)
		if _, err := h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument: lsp.TextDocumentItem{URI: uri, Text: source}}); err != nil {
			t.Fatal(err)
		}
		locations, err := h.Definition(context.Background(), &lsp.DefinitionParams{TextDocumentPositionParams: lsp.TextDocumentPositionParams{TextDocument: lsp.TextDocumentIdentifier{URI: uri}, Position: pos}})
		if err != nil || len(locations) != 1 {
			t.Fatalf("definition for %s: %v %v", tc.source, locations, err)
		}
		if string(locations[0].URI) != "cometa-std:///"+tc.path+".cometa" {
			t.Fatalf("wrong native URI: %v", locations)
		}
		reference, _ := stdlib.Source(tc.path)
		lines := strings.Split(reference, "\n")
		if !strings.Contains(lines[locations[0].Range.Start.Line], tc.needle) {
			t.Fatalf("wrong declaration line: %v", locations)
		}
	}
}

func TestNoImplicitGameCompletion(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///no-native-import.cometa")
	source, pos := markerPosition("fn inicio()\n\tgraficos.§\n")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	list, err := h.Completion(uri, pos.Line, pos.Character)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("unimported namespace leaked: %v", list)
	}
}

const pincelImports = "usar std/mate\nusar std/azar\nusar std/pincel\nusar std/pincel/graficos\nusar std/pincel/color\nusar std/pincel/entrada\nusar std/pincel/audio\nusar std/pincel/ventana\nusar std/pincel/tiempo\nusar std/pincel/recursos\n"

func TestGameAssetsWithEncodedFileURI(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game with spaces")
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(filepath.Join("..", "..", "examples", "pincel", "assets", "jugador.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "jugador.png"), image, 0600); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(string(fileURI(filepath.Join(dir, "juego.cometa"))))
	if err != nil {
		t.Fatal(err)
	}
	u.RawPath = strings.ReplaceAll(u.EscapedPath(), ":", "%3A")
	uri := lsp.DocumentURI(u.String())
	h := servertest.New(t, NewHandler())
	// The source only exists in the document overlay, as with an unsaved edit.
	source := pincelImports + "var sprite = recursos.imagen(\"assets/jugador.png\")\nfn actualizar(dt decimal) imprimir(dt)\nfn pintar() graficos.imagen_v(sprite, mate.Vec2 {})\n"
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, h, uri); len(diagnostics) != 0 {
		t.Fatalf("asset URI diagnostics: %+v", diagnostics)
	}
	hover, err := h.Hover(uri, strings.Count(pincelImports, "\n"), 5)
	if err != nil || hover == nil || !strings.Contains(hover.Contents.Value(), "Imagen") {
		t.Fatalf("resource hover: %+v, %v", hover, err)
	}
	if _, err := h.DocumentSymbol(uri); err != nil {
		t.Fatal(err)
	}
	missing := strings.ReplaceAll(source, "jugador.png", "missing.png")
	h.ClearDiagnostics()
	if err := h.DidChange(uri, 2, missing); err != nil {
		t.Fatal(err)
	}
	diagnostics := waitForDiagnostics(t, h, uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "missing.png") || strings.Contains(diagnostics[0].Message, "file:") {
		t.Fatalf("missing asset diagnostic: %+v", diagnostics)
	}
	h.ClearDiagnostics()
	if err := h.DidChange(uri, 3, source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, h, uri); len(diagnostics) != 0 {
		t.Fatalf("stale asset diagnostic: %+v", diagnostics)
	}
}

func TestGameCompletion(t *testing.T) {
	for _, tt := range []struct{ body, label, detail string }{
		{"graficos.§", "rectangulo_v", "pos mate.Vec2"},
		{"retro.§", "texto", "x entero"},
		{"retro.icono(.§, 0, 0)", "Corazon", "Icono.Corazon"},
		{"retro.glifo(0, 0, 0, atlas = .§)", "ASCII", "Atlas.ASCII"},
		{"entrada.§", "tecla_presionada", "Tecla"},
		{"graficos.limpiar(.§)", "Rojo", "Color.Rojo"},
		{"entrada.tecla_mantenida(.§)", "Espacio", "Tecla.Espacio"},
		{"var v = mate.Vec2 {}\n\tv.§", "x", "decimal"},
		{"mate.Vec§", "Vec2", "tipo"},
		{"mate.§", "pi", "decimal"},
		{"mate.§", "limitar", "decimal"},
		{"graficos.§", "imagen_rect", "destino mate.Rect"},
		{"var v = mate.Vec2 {}\n\tv.§", "normalizado", "mate.Vec2"},
		{"var r = mate.Rect {}\n\tr.pos.§", "normalizado", "mate.Vec2"},
		{"var vs = [mate.Vec2 {}]\n\tvs[0].§", "longitud", "decimal"},
		{"var v = mate.Vec2 {}\n\tv.normalizado().§", "distancia_a", "otro mate.Vec2"},
		{"mate.Vec2 {}.§", "rotado", "angulo decimal"},
		{"var v = mate.Vec2 {}\n\t(v + v).§", "normalizado", "mate.Vec2"},
		{"var r = mate.Rect {}\n\tr.§", "interseca", "otro mate.Rect"},
	} {
		t.Run(tt.label, func(t *testing.T) {
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///game.cometa")
			source, pos := markerPosition("usar std/pincel/retro\n" + pincelImports + "fn actualizar(dt decimal) imprimir(dt)\nfn pintar()\n\t" + tt.body + "\n")
			if err := h.DidOpen(uri, "cometa", source); err != nil {
				t.Fatal(err)
			}
			list, err := h.Completion(uri, pos.Line, pos.Character)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Items {
				if item.Label == tt.label && strings.Contains(item.Detail, tt.detail) {
					return
				}
			}
			t.Fatalf("missing %s: %+v", tt.label, list)
		})
	}
}

func TestVectorReceiverFieldCompletion(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///vector-receiver.cometa")
	source, pos := markerPosition(pincelImports + "tipo Jugador\n\tpos mate.Vec2\n\tfn mover()\n\t\t@pos.§\n")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	list, err := h.Completion(uri, pos.Line, pos.Character)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		if item.Label == "normalizado" && item.Detail == "pub fn normalizado() mate.Vec2" {
			return
		}
	}
	t.Fatalf("missing vector methods: %+v", list)
}

func TestGameHover(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///game.cometa")
	source, pos := markerPosition(pincelImports + "fn actualizar(dt decimal) imprimir(dt)\nfn pintar()\n\tgraficos.§rectangulo_v(mate.Vec2 {}, mate.Vec2 {x: 2}, .Rojo)\n")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	hover, err := h.Hover(uri, pos.Line, pos.Character)
	if err != nil || hover == nil {
		t.Fatalf("hover: %v %v", hover, err)
	}
}

func TestGameMethodHover(t *testing.T) {
	for _, expr := range []string{"mate.Vec2 {}.§longitud()", "mate.Rect {}.§contiene({})", "mate.§pi"} {
		h := servertest.New(t, NewHandler())
		uri := lsp.DocumentURI("file:///game.cometa")
		source, pos := markerPosition(pincelImports + "fn actualizar(dt decimal) imprimir(dt)\nfn pintar()\n\timprimir(" + expr + ")\n")
		if err := h.DidOpen(uri, "cometa", source); err != nil {
			t.Fatal(err)
		}
		hover, err := h.Hover(uri, pos.Line, pos.Character)
		if err != nil || hover == nil {
			t.Fatalf("hover for %s: %v %v", expr, hover, err)
		}
	}
}
func TestRemovedGameCompletions(t *testing.T) {
	for _, name := range []string{"mat", "vectores", "colisiones"} {
		if list := gameNamespaceCompletion(name + "."); list != nil {
			t.Fatalf("obsolete namespace %s", name)
		}
	}
	for _, item := range gameTopCompletion("").Items {
		switch item.Label {
		case "mat", "vectores", "colisiones", "ConfigJuego", "configurar":
			t.Fatalf("obsolete completion: %s", item.Label)
		}
	}
}
