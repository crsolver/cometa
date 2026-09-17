package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameAssetsWithEncodedFileURI(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game with spaces")
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(filepath.Join("..", "..", "examples", "assets", "jugador.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "jugador.png"), image, 0600); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(string(fileURI(filepath.Join(dir, "juego.hacha"))))
	if err != nil {
		t.Fatal(err)
	}
	u.RawPath = strings.ReplaceAll(u.EscapedPath(), ":", "%3A")
	uri := lsp.DocumentURI(u.String())
	h := servertest.New(t, NewHandler())
	// The source only exists in the document overlay, as with an unsaved edit.
	source := "var sprite = recursos.imagen(\"assets/jugador.png\")\nfn actualizar(dt num) imprimir(dt)\nfn pintar() graficos.imagen_v(sprite, Vec2 {})\n"
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, h, uri); len(diagnostics) != 0 {
		t.Fatalf("asset URI diagnostics: %+v", diagnostics)
	}
	hover, err := h.Hover(uri, 0, 5)
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
		{"graficos.§", "rectangulo_v", "pos Vec2"},
		{"entrada.§", "tecla_presionada", "Tecla"},
		{"graficos.limpiar(.§)", "Rojo", "Color.Rojo"},
		{"entrada.tecla_mantenida(.§)", "Espacio", "Tecla.Espacio"},
		{"var v = Vec2 {}\n\tv.§", "x", "num"},
		{"Vec§", "Vec2", "incorporado"},
		{"mate.§", "pi", "num"},
		{"mate.§", "limitar", "num"},
		{"graficos.§", "imagen_rect", "destino Rect"},
		{"var v = Vec2 {}\n\tv.§", "normalizado", "Vec2"},
		{"var r = Rect {}\n\tr.pos.§", "normalizado", "Vec2"},
		{"var vs = [Vec2 {}]\n\tvs[0].§", "longitud", "num"},
		{"var v = Vec2 {}\n\tv.normalizado().§", "distancia_a", "otro Vec2"},
		{"Vec2 {}.§", "rotado", "angulo num"},
		{"var v = Vec2 {}\n\t(v + v).§", "normalizado", "Vec2"},
		{"var r = Rect {}\n\tr.§", "interseca", "otro Rect"},
	} {
		t.Run(tt.label, func(t *testing.T) {
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///game.hacha")
			source, pos := markerPosition("fn actualizar(dt num) imprimir(dt)\nfn pintar()\n\t" + tt.body + "\n")
			if err := h.DidOpen(uri, "hacha", source); err != nil {
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
	uri := lsp.DocumentURI("file:///vector-receiver.hacha")
	source, pos := markerPosition("tipo Jugador\n\tpos Vec2\n\tfn mover()\n\t\t@pos.§\n")
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	list, err := h.Completion(uri, pos.Line, pos.Character)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		if item.Label == "normalizado" && item.Detail == "fn normalizado() Vec2" {
			return
		}
	}
	t.Fatalf("missing vector methods: %+v", list)
}

func TestGameHover(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///game.hacha")
	source, pos := markerPosition("fn actualizar(dt num) imprimir(dt)\nfn pintar()\n\tgraficos.§rectangulo_v(Vec2 {}, Vec2 {x: 2}, .Rojo)\n")
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	hover, err := h.Hover(uri, pos.Line, pos.Character)
	if err != nil || hover == nil {
		t.Fatalf("hover: %v %v", hover, err)
	}
}

func TestGameMethodHover(t *testing.T) {
	for _, expr := range []string{"Vec2 {}.§longitud()", "Rect {}.§contiene({})", "mate.§pi"} {
		h := servertest.New(t, NewHandler())
		uri := lsp.DocumentURI("file:///game.hacha")
		source, pos := markerPosition("fn actualizar(dt num) imprimir(dt)\nfn pintar()\n\timprimir(" + expr + ")\n")
		if err := h.DidOpen(uri, "hacha", source); err != nil {
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
