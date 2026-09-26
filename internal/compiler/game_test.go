package compiler

import (
	"bytes"
	"cometa/internal/ast"
	"cometa/internal/stdlib"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pincelImports = "usar std/mate\nusar std/azar\nusar std/pincel\nusar std/pincel/graficos\nusar std/pincel/color\nusar std/pincel/entrada\nusar std/pincel/audio\nusar std/pincel/ventana\nusar std/pincel/tiempo\nusar std/pincel/recursos\n"

const minimalGame = "fn actualizar(dt decimal)\n\timprimir(dt)\nfn pintar()\n\tgraficos.rectangulo_v(mate.Vec2 {}, mate.Vec2 {x: 10, y: 10}, .Rojo)\n"

func TestGameGeneration(t *testing.T) {
	source := `tipo Jugador
	pos mate.Vec2
	vel mate.Vec2
var jugador = Jugador {}
fn actualizar(dt decimal)
	jugador.pos = jugador.pos + jugador.vel * dt
	jugador.pos.x = 7
fn pintar()
	graficos.rectangulo_v(
		jugador.pos,
		mate.Vec2 {x: 10, y: 10},
		.Rojo
	)
tipo Partida
	pub fn actualizar(dt decimal) actualizar(dt)
	pub fn pintar() pintar()
fn inicio()
	pincel.ejecutar(Partida {}, 320, 240, titulo = "Prueba", escala = 2) capturar |e| imprimir(e)
`
	got, err := Compile("game.cometa", []byte(pincelImports+source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func main()", "ebiten.RunGame", "func (g *_hgGame) Update() error", "_hgscale(", "Pos _hgVec2", "_hgrectangulo("} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(string(got), "Pos *_hgVec2") {
		t.Fatal("vectors must be values")
	}
}

func TestGameGolden(t *testing.T) {
	file := filepath.Join("testdata", "juego.cometa")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Compile(file, data)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "juego.go.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))) {
		t.Fatal("generated game differs from golden")
	}
}

func TestGameErrors(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"resource local", "fn actualizar(dt decimal)\n\tvar img = recursos.imagen(\"foo.png\")\nfn pintar() imprimir(0)\n", "inicializador global directo"},
		{"resource expression", "var p = \"foo.png\"\nvar img = recursos.imagen(p)\n" + minimalGame, "cadena literal"},
		{"opaque literal", "var img = graficos.Imagen {}\n" + minimalGame, "opaco"},
		{"missing resource field", "tipo T\n\timg graficos.Imagen\nvar t = T {}\n" + minimalGame, "requiere inicialización"},
		{"bad vector operator", "fn actualizar(dt decimal)\n\tvar x = mate.Vec2 {} * mate.Vec2 {}\nfn pintar() imprimir(0)\n", "operador"},
		{"bad color", "fn actualizar(dt decimal) imprimir(dt)\nfn pintar() graficos.limpiar(.ColorInexistente)\n", "constante inválida"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("game.cometa", []byte(pincelImports+tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want %q; got %v", tt.want, err)
			}
		})
	}
}

func TestAllGameSignatures(t *testing.T) {
	const imports = "usar std/mate/curvas\nusar std/mate/ruido\nusar std/pincel/retro\nusar std/pincel/ui\nusar std/pincel/lienzo\n" + pincelImports
	var all strings.Builder
	// Signature defaults are parsed and checked by the same checker as user calls.
	for _, f := range stdlib.Functions {
		d := f.Declaration()
		var args []string
		for _, p := range d.Params {
			if p.Default != nil {
				continue
			}
			switch p.Type.Name {
			case "decimal":
				args = append(args, "1")
			case "cadena":
				args = append(args, `"x"`)
			case "bool":
				args = append(args, "falso")
			default:
				args = append(args, p.Name)
			}
		}
		if f.Resource || f.Namespace == "pincel" {
			continue
		}
		var params []string
		for _, p := range d.Params {
			if p.Default == nil && p.Type.Name != "decimal" && p.Type.Name != "cadena" && p.Type.Name != "bool" {
				params = append(params, "arg_"+p.Name+" "+publicTypeSource(p.Type))
				for i, a := range args {
					if a == p.Name {
						args[i] = "arg_" + a
					}
				}
			}
		}
		call := f.Namespace + "." + f.Name + "(" + strings.Join(args, ",") + ")"
		// Wrapper results cannot be discarded.
		if d.ReturnType != nil && d.ReturnType.Wrapper == "?" {
			call = "var opcional = " + call
		} else if d.ReturnType != nil && d.ReturnType.Wrapper == "!" {
			call += " capturar |e| imprimir(e)"
		}
		source := "fn helper(" + strings.Join(params, ",") + ")\n\t" + call + "\n" + minimalGame
		all.WriteString(strings.Replace(strings.TrimSuffix(source, minimalGame), "fn helper(", "fn "+f.Namespace+"_"+f.Name+"(", 1))
		t.Run(f.Namespace+"."+f.Name, func(t *testing.T) {
			if _, _, e := Analyze("game.cometa", []byte(imports+source)); e != nil {
				t.Fatal(e)
			}
		})
	}
	generated, err := Compile("game.cometa", []byte(imports+all.String()+minimalGame))
	if err != nil {
		t.Fatal(err)
	}
	runnable := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1) + "\nfunc main() {}\n"
	runGeneratedGo(t, []byte(runnable), "")
}

func TestEmbeddedImageAndModulePaths(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "assets")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "sprite.png"), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "model.cometa"), []byte("usar std/pincel/recursos\nvar sprite = recursos.imagen(\"sprite.png\")\n"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "game.cometa")
	if err := os.WriteFile(entry, []byte(pincelImports+"usar assets/model\n"+minimalGame), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CompileProject(entry, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "iVBORw0KGgo") {
		t.Fatal("image bytes not embedded")
	}
	again, err := CompileProject(entry, nil)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatal("non-reproducible output", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "sprite.png"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = CompileProject(entry, nil); err == nil || !strings.Contains(err.Error(), "sprite.png") {
		t.Fatalf("missing asset diagnostic: %v", err)
	}
}

func TestGameVectorValuesAndDefaultsRuntime(t *testing.T) {
	source := `tipo Jugador
	pos mate.Vec2
var jugador = Jugador {}
var orden entero = 0
const pi = mate.pi
fn siguiente() entero
	orden = orden + 1
	orden
fn iniciar()
	var a = mate.Vec2 {x: 2, y: 3}
	var b = a
	b.x = 9
	imprimir(a.x)
	imprimir(b.x)
	var c = 2 * a + -a / 2
	imprimir(c.x)
	jugador.pos.x = 12
	imprimir(jugador.pos.x)
	var normal = mate.Vec2 {}.normalizado()
	imprimir(normal.x)
	var cam = graficos.Camara2D {}
	imprimir(cam.zoom)
fn actualizar(dt decimal) imprimir(dt)
fn pintar() imprimir(0)
`
	generated, err := Compile("game.cometa", []byte(pincelImports+source))
	if err != nil {
		t.Fatal(err)
	}
	// Exercise generated game code without starting a window or event loop.
	runnable := strings.Replace(string(generated), "func main() {", "func _hgUnusedMain() {", 1) + "\nfunc main() { Iniciar() }\n"
	runGeneratedGo(t, []byte(runnable), "2\n9\n3\n12\n0\n1\n")
}

func publicTypeSource(t ast.TypeRef) string {
	if t.Key != nil {
		return "[" + publicTypeSource(*t.Key) + ": " + publicTypeSource(*t.Element) + "]"
	}
	if t.Element != nil {
		return "[" + publicTypeSource(*t.Element) + "]"
	}
	return publicLibraryType(t.Name)
}

func publicLibraryType(name string) string {
	if owner := stdlib.TypeModules[stdlib.PublicName(name)]; owner != "" {
		return owner + "." + stdlib.PublicName(name)
	}
	return name
}
