package compiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Compiles and runs a console program: rejilla, azar.semilla and the new
// mate helpers need no window, so their behavior can be checked directly.
func TestRejillaAndMathHelpersRun(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/mate
usar std/azar
usar std/pincel/rejilla

fn inicio()
	var mapa = rejilla.desde_texto(["#..", ".#.", "..#"], ["#": 1, ".": 0])
	imprimir(mapa.columnas())
	imprimir(mapa.filas())
	imprimir(mapa.obtener(1, 1))
	imprimir(mapa.obtener(5, 1))
	imprimir(mapa.poner(2, 0, 7))
	imprimir(mapa.poner(9, 9, 1))
	imprimir(mapa.choca(mate.Rect {pos: {15, 0}, tamano: {4, 4}}, {16, 16}, [1]))
	imprimir(mapa.choca(mate.Rect {pos: {32, 0}, tamano: {4, 4}}, {16, 16}, [1]))
	imprimir(mapa.choca(mate.Rect {pos: {32, 0}, tamano: {4, 4}}, {16, 16}, [7]))
	imprimir(mapa.choca(mate.Rect {pos: {-50, -50}, tamano: {4, 4}}, {16, 16}, [1]))
	imprimir(mapa.choca(mate.Rect {pos: {16, 16}, tamano: {0, 4}}, {16, 16}, [1]))
	var vacia = rejilla.nueva(2, 2, -1)
	imprimir(vacia.obtener(0, 0))
	vacia.rellenar(4)
	imprimir(vacia.obtener(1, 1))
	azar.semilla(42)
	var a = azar.entero(0, 1000)
	var b = azar.real(0, 1)
	azar.semilla(42)
	imprimir(a == azar.entero(0, 1000) && b == azar.real(0, 1))
	imprimir(mate.signo(-3))
	imprimir(mate.signo(2.5))
	imprimir(mate.potencia(2, 10))
	imprimir(mate.distancia(0, 0, 3, 4))
	imprimir(mate.grados(mate.radianes(90)).formato(3))
	imprimir(mate.angulo(0, 0, 0, 1).formato(4))
	var v = mate.Vec2 {x: 1, y: 0}
	imprimir(v.perpendicular().y)
	imprimir(v.interpolar({3, 0}, 0.5).x)
	imprimir(v.reflejar({1, 0}).x)
	imprimir(v.angulo())
	var r = mate.Rect {pos: {0, 0}, tamano: {10, 10}}
	imprimir(r.centro().x)
	imprimir(r.desplazado({5, 5}).interseccion(r).tamano.x)
	imprimir(r.desplazado({20, 20}).interseccion(r).tamano.x)
	imprimir(r.colision_circulo({12, 5}, 3))
	imprimir(r.colision_circulo({14, 5}, 3))
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"3", "3", "Alguno(1)", "Ninguno", "verdadero", "falso",
		"verdadero", "falso", "verdadero", "falso", "falso",
		"Alguno(-1)", "Alguno(4)", "verdadero",
		"-1", "1", "1024", "5", "90.000", "1.5708",
		"1", "2", "-1", "0", "5", "5", "0", "verdadero", "falso",
	}, "\n") + "\n"
	runGeneratedGo(t, generated, want)
}

func TestRejillaErrorsAreClear(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"unknown symbol", `rejilla.desde_texto(["#?"], ["#": 1])`, `el símbolo "?"`},
		{"ragged rows", `rejilla.desde_texto(["##", "#"], ["#": 1])`, "la fila 2 mide 1"},
		{"bad size", `rejilla.nueva(0, 2)`, "deben ser positivas"},
		{"bad cell size", `rejilla.nueva(2, 2).choca(mate.Rect {tamano: {1, 1}}, {0, 16}, [1])`, "tamaño de celda"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, loader := memoryProject(t, map[string]string{"main.cometa": "usar std/mate\nusar std/pincel/rejilla\nfn inicio()\n\tvar x = " + tc.body + "\n\timprimir(x)\n"})
			generated, err := CompileProject(entry, loader)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.go"), generated, 0o600); err != nil {
				t.Fatal(err)
			}
			out := runGoExpectingFailure(t, filepath.Join(dir, "main.go"))
			if !strings.Contains(out, tc.want) {
				t.Errorf("falta %q en la salida:\n%s", tc.want, out)
			}
		})
	}
}

// datos keeps values between runs in the user's config folder, which the test redirects.
func TestDatosStoresValues(t *testing.T) {
	config := t.TempDir()
	for _, name := range []string{"APPDATA", "XDG_CONFIG_HOME", "HOME"} {
		t.Setenv(name, config)
	}
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/pincel/datos

fn inicio()
	imprimir(datos.leer("x"))
	datos.guardar("x", "42") capturar |e1|
		imprimir(e1)
	imprimir(datos.leer("x"))
	imprimir((datos.leer("x") o "0").a_entero() o -1)
	datos.borrar("x") capturar |e2|
		imprimir(e2)
	imprimir(datos.leer("x"))
	datos.guardar("", "1") capturar |e3|
		imprimir(e3)
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "Ninguno\nAlguno(\"42\")\n42\nNinguno\nla clave no puede estar vacía\n")
	found := false
	filepath.WalkDir(config, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Name() == "datos.json" {
			found = true
		}
		return nil
	})
	if !found {
		t.Error("datos.json no se creó en la carpeta de configuración")
	}
}

// The extras fixture uses every new drawing, gamepad and quit function.
func TestPincelExtrasFixtureCompiles(t *testing.T) {
	if _, err := CompileProject(filepath.Join("testdata", "programas", "pincel_extras.cometa"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestNewKeyConstants(t *testing.T) {
	source := `usar std/pincel
usar std/pincel/entrada
tipo P
	pub fn actualizar(dt decimal)
		var ok = entrada.tecla_presionada(.Uno) || entrada.tecla_presionada(.Nueve) || entrada.tecla_presionada(.F12) || entrada.tecla_mantenida(.Alt) || entrada.tecla_mantenida(.ShiftDerecho) || entrada.tecla_presionada(.RePag) || entrada.tecla_presionada(.Suprimir)
		si ok
			pincel.salir()
	pub fn pintar()
		imprimir(0)
fn inicio()
	pincel.ejecutar(P {}) capturar |e|
		imprimir(e)
`
	entry, loader := memoryProject(t, map[string]string{"main.cometa": source})
	if _, err := CompileProject(entry, loader); err != nil {
		t.Fatal(err)
	}
}

// mover stops flush against solid cells, slides along the other axis and never tunnels through a thin wall.
func TestRejillaMover(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/mate
usar std/pincel/rejilla

fn inicio()
	var mapa = rejilla.desde_texto(["....#", ".....", "#####"], ["#": 1, ".": 0])
	var cuerpo = mate.Rect {pos: {10, 10}, tamano: {10, 10}}
	var libre = mapa.mover(cuerpo, {16, 16}, {5, 0}, [1])
	imprimir(libre.x)
	var pared = mapa.mover(cuerpo, {16, 16}, {100, 0}, [1])
	imprimir(pared.x.formato(2))
	imprimir(pared.y)
	var suelo = mapa.mover(cuerpo, {16, 16}, {0, 100}, [1])
	imprimir(suelo.y.formato(2))
	var diagonal = mapa.mover(cuerpo, {16, 16}, {100, 100}, [1])
	imprimir(diagonal.x.formato(2))
	imprimir(diagonal.y.formato(2))
	imprimir(mapa.mover(cuerpo, {16, 16}, {0, 0}, [1]).x)
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	// The wall (column 4) starts at x=64 and the floor (row 2) at y=32; the body spans 10..20 on both axes.
	runGeneratedGo(t, generated, "5\n44.00\n0\n12.00\n44.00\n12.00\n0\n")
}
