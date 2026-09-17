package compiler

import (
	"bytes"
	"hacha/internal/gameapi"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalGame = "fn actualizar(dt num)\n\timprimir(dt)\nfn pintar()\n\tgraficos.rectangulo_v(Vec2 {}, Vec2 {x: 10, y: 10}, .Rojo)\n"

func TestGameGeneration(t *testing.T) {
	source := `tipo Jugador
	pos Vec2
	vel Vec2
var jugador = Jugador {}
fn iniciar()
	juego.configuracion(320, 240, titulo = "Prueba", escala = 2)
fn actualizar(dt num)
	jugador.pos = jugador.pos + jugador.vel * dt
	jugador.pos.x = 7
fn pintar()
	graficos.rectangulo_v(
		jugador.pos,
		Vec2 {x: 10, y: 10},
		.Rojo
	)
`
	got, err := Compile("game.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func main()", "ebiten.RunGame", "func (*_hgGame) Update() error", "_hgscale(", "Pos _hgVec2", "_hgrectangulo("} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(string(got), "Pos *_hgVec2") {
		t.Fatal("vectors must be values")
	}
}

func TestGameGolden(t *testing.T) {
	file := filepath.Join("testdata", "juego.hacha")
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
		{"missing paint", "fn actualizar(dt num) imprimir(dt)\n", "requiere fn pintar"},
		{"bad update", "fn actualizar(dt cadena) imprimir(dt)\nfn pintar() imprimir(0)\n", "firma inválida"},
		{"old entry", minimalGame + "fn inicio() imprimir(0)\n", "no puede declarar inicio"},
		{"draw helper", "fn dibujar() graficos.circulo_v(Vec2 {}, 2, .Azul)\nfn actualizar(dt num) dibujar()\nfn pintar() dibujar()\n", "solo se permite desde pintar"},
		{"draw default", "fn dibujar() num\n\tgraficos.limpiar(.Negro)\n\t1\nfn f(x num = dibujar()) imprimir(x)\nfn actualizar(dt num) f()\nfn pintar() imprimir(0)\n", "solo se permite desde pintar"},
		{"resource local", "fn actualizar(dt num)\n\tvar img = recursos.imagen(\"foo.png\")\nfn pintar() imprimir(0)\n", "inicializador global directo"},
		{"resource expression", "var p = \"foo.png\"\nvar img = recursos.imagen(p)\n" + minimalGame, "cadena literal"},
		{"opaque literal", "var img = Imagen {}\n" + minimalGame, "opaco"},
		{"missing resource field", "tipo T\n\timg Imagen\nvar t = T {}\n" + minimalGame, "requiere inicialización"},
		{"bad vector operator", "fn actualizar(dt num)\n\tvar x = Vec2 {} * Vec2 {}\nfn pintar() imprimir(0)\n", "operador"},
		{"bad color", "fn actualizar(dt num) imprimir(dt)\nfn pintar() graficos.limpiar(.Violeta)\n", "constante inválida"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("game.hacha", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want %q; got %v", tt.want, err)
			}
		})
	}
}

func TestAllGameSignatures(t *testing.T) {
	var all strings.Builder
	// Signature defaults are parsed and checked by the same checker as user calls.
	for _, f := range gameapi.Functions {
		d := f.Declaration()
		var args []string
		for _, p := range d.Params {
			if p.Default != nil {
				continue
			}
			switch p.Type.Name {
			case "num":
				args = append(args, "1")
			case "cadena":
				args = append(args, `"x"`)
			case "bool":
				args = append(args, "falso")
			default:
				args = append(args, p.Name)
			}
		}
		if f.Resource || f.Namespace == "juego" {
			continue
		}
		var params []string
		for _, p := range d.Params {
			if p.Default == nil && p.Type.Name != "num" && p.Type.Name != "cadena" && p.Type.Name != "bool" {
				params = append(params, "arg_"+p.Name+" "+p.Type.Name)
				for i, a := range args {
					if a == p.Name {
						args[i] = "arg_" + a
					}
				}
			}
		}
		source := "fn helper(" + strings.Join(params, ",") + ")\n\t" + f.Namespace + "." + f.Name + "(" + strings.Join(args, ",") + ")\n" + minimalGame
		all.WriteString(strings.Replace(strings.TrimSuffix(source, minimalGame), "fn helper(", "fn "+f.Namespace+"_"+f.Name+"(", 1))
		t.Run(f.Namespace+"."+f.Name, func(t *testing.T) {
			tokens, e := lexer.Lex("game.hacha", source)
			if e != nil {
				t.Fatal(e)
			}
			p, e := parser.Parse("game.hacha", tokens)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = sema.Check("game.hacha", p); e != nil {
				t.Fatal(e)
			}
		})
	}
	generated, err := Compile("game.hacha", []byte(all.String()+minimalGame))
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
	if err := os.WriteFile(filepath.Join(sub, "model.hacha"), []byte("var sprite = recursos.imagen(\"sprite.png\")\n"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "game.hacha")
	if err := os.WriteFile(entry, []byte("usar assets/model\n"+minimalGame), 0600); err != nil {
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
	pos Vec2
var jugador = Jugador {}
var orden num = 0
const pi = mate.pi
fn siguiente() num
	orden = orden + 1
	orden
fn iniciar()
	var a = Vec2 {x: 2, y: 3}
	var b = a
	b.x = 9
	imprimir(a.x)
	imprimir(b.x)
	var c = 2 * a + -a / 2
	imprimir(c.x)
	jugador.pos.x = 12
	imprimir(jugador.pos.x)
	var normal = Vec2 {}.normalizado()
	imprimir(normal.x)
	juego.configuracion(alto = siguiente(), ancho = siguiente())
	imprimir(graficos.tamano().x)
	imprimir(graficos.tamano().y)
	var cam = Camara2D {}
	imprimir(cam.zoom)
fn actualizar(dt num) imprimir(dt)
fn pintar() imprimir(0)
`
	generated, err := Compile("game.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	// Exercise generated game code without starting a window or event loop.
	runnable := strings.Replace(string(generated), "func main() {", "func _hgUnusedMain() {", 1) + "\nfunc main() { _hgstarting = true; Iniciar() }\n"
	runGeneratedGo(t, []byte(runnable), "2\n9\n3\n12\n0\n2\n1\n1\n")
}
