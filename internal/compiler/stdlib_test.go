package compiler

import (
	"strings"
	"testing"
)

func TestStandardMathWithoutGame(t *testing.T) {
	source := "usar std/mate\nfn inicio()\n\tvar v = mate.Vec2 {3, 4}\n\timprimir(v.longitud())\n\timprimir(mate.piso(3.8))\n\timprimir((v + v).x)\n"
	generated, err := Compile("math.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("math pulled in Ebitengine")
	}
	runGeneratedGo(t, generated, "5\n3\n6\n")
}

func TestPincelExplicitStartup(t *testing.T) {
	source := "usar std/pincel/juego\nusar std/pincel/graficos\ntipo Demo\n\tpub fn actualizar(dt decimal) imprimir(dt)\n\tpub fn pintar() graficos.limpiar(.Negro)\nfn inicio()\n\tjuego.ejecutar(Demo {}) capturar |error|\n\t\timprimir(error)\n"
	generated, err := Compile("game.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ebiten.RunGame", "g.instancia.Actualizar", "g.instancia.Pintar", "func main()"} {
		if !strings.Contains(string(generated), want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestStandardLibraryVisibility(t *testing.T) {
	for _, source := range []string{
		"fn inicio() imprimir(mate.pi)\n",
		"fn inicio()\n\tvar v = Vec2 {}\n",
		"usar std/mate\nfn inicio()\n\tvar v = Vec2 {}\n",
		"usar std/pincel/graficos\nfn inicio() imprimir(mate.pi)\n",
		"usar std/pincel\nfn inicio() imprimir(0)\n",
		"usar std/no_existe\nfn inicio() imprimir(0)\n",
		"usar std/mate\nusar std/mate como m\nfn inicio() imprimir(0)\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatalf("accepted missing import: %s", source)
		}
	}
	source := "usar std/mate como m\ntipo Vec2\n\tx entero\ntipo Color\n\tx entero\nfn inicio()\n\tvar mate = Vec2 {7}\n\timprimir(mate.x)\n\timprimir(m.Vec2 {3,4}.longitud())\n\timprimir(m.piso(m.pi))\n"
	generated, err := Compile("aliases.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "7\n5\n3\n")
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":   "usar helper\nfn inicio() imprimir(mate.pi)\n",
		"helper.cometa": "usar std/mate\npub fn valor() decimal mate.pi\n",
	})
	if _, err := CompileProject(entry, loader); err == nil {
		t.Fatal("dependency import leaked to root")
	}
}

func TestStandardMathModuleAndTypeIdentity(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":   "usar std/mate como m\nusar helper\nfn inicio()\n\tvar p m.Vec2 = helper.punto()\n\timprimir(p.longitud())\n",
		"helper.cometa": "usar std/mate\npub fn punto() mate.Vec2\n\tmate.Vec2 {3,4}\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("math dependency pulled in game")
	}
	runGeneratedGo(t, generated, "5\n")
}
