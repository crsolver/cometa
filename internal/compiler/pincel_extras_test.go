package compiler

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// configDir points os.UserConfigDir at a temporary folder. The go tool derives
// its caches from HOME on Linux and macOS, so they are pinned to their real
// locations first; otherwise `go run` starts from an empty cache inside the
// temporary folder.
func configDir(t *testing.T) string {
	t.Helper()
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	for _, name := range []string{"GOCACHE", "GOMODCACHE", "GOPATH"} {
		out, err := exec.Command(filepath.Join(runtime.GOROOT(), "bin", goName), "env", name).Output()
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv(name, strings.TrimSpace(string(out)))
	}
	config := t.TempDir()
	for _, name := range []string{"APPDATA", "XDG_CONFIG_HOME", "HOME"} {
		t.Setenv(name, config)
	}
	return config
}

func TestDatosStoresValues(t *testing.T) {
	config := configDir(t)
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/pincel/datos

fn inicio()
	imprimir(datos.leer("x"))
	datos.guardar("x", "42") atrapar |e1|
		imprimir(e1)
	imprimir(datos.leer("x"))
	imprimir((datos.leer("x") o "0").a_entero() o -1)
	datos.borrar("x") atrapar |e2|
		imprimir(e2)
	imprimir(datos.leer("x"))
	datos.guardar("", "1") atrapar |e3|
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

// datos.juego picks a stable folder; numeric helpers round-trip and reject text that is not a number.
func TestDatosJuegoYNumeros(t *testing.T) {
	config := configDir(t)
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/pincel/datos

fn inicio()
	datos.juego("Mi juego!")
	datos.guardar_entero("mejor", 120) atrapar |e1|
		imprimir(e1)
	datos.guardar_decimal("volumen", 0.75) atrapar |e2|
		imprimir(e2)
	datos.guardar("nombre", "Ana") atrapar |e3|
		imprimir(e3)
	imprimir(datos.leer_entero("mejor") o 0)
	imprimir(datos.leer_decimal("volumen") o 1)
	imprimir(datos.leer_entero("nombre"))
	imprimir(datos.leer_decimal("falta"))
	imprimir(datos.claves())
	datos.guardar_decimal("x", 1.0 / 0.0) atrapar |e4|
		imprimir(e4)
	datos.juego("otro")
	imprimir(datos.claves().longitud())
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "120\n0.75\nNinguno\nNinguno\n[\"mejor\", \"nombre\", \"volumen\"]\nel valor debe ser un número finito\n0\n")
	var folders []string
	filepath.WalkDir(config, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Name() == "datos.json" {
			folders = append(folders, filepath.Base(filepath.Dir(path)))
		}
		return nil
	})
	if len(folders) != 1 || folders[0] != "Mi_juego_" {
		t.Errorf("carpetas de datos = %v, se esperaba [Mi_juego_]", folders)
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
	pincel.ejecutar(P {}) atrapar |e|
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
	// Row 1 holds one-way platforms (2): they stop a fall from above but not a jump from below.
	var nubes = rejilla.desde_texto(["....", "=...", "...."], ["=": 2, ".": 0])
	var encima = mate.Rect {pos: {2, 0}, tamano: {10, 10}}
	imprimir(nubes.mover(encima, {16, 16}, {0, 20}, [1], plataformas = [2]).y.formato(2))
	imprimir(nubes.mover(encima, {16, 16}, {0, 20}, [1]).y.formato(2))
	var debajo = mate.Rect {pos: {2, 36}, tamano: {10, 10}}
	imprimir(nubes.mover(debajo, {16, 16}, {0, -20}, [1], [2]).y.formato(2))
	var dentro = mate.Rect {pos: {2, 12}, tamano: {10, 10}}
	imprimir(nubes.mover(dentro, {16, 16}, {0, 5}, [1], [2]).y.formato(2))
	// Omitted and named native defaults also work where wrappers force flow lowering.
	imprimir(((nubes.celda_en({3, 17}, {16, 16}) o 0) + nubes.mover(encima, {16, 16}, {0, 20}, [1]).y).formato(2))
	imprimir(((nubes.celda_en({3, 17}, {16, 16}) o 0) + nubes.mover(encima, solidos = [1], delta = {0, 20}, tamano_celda = {16, 16}, plataformas = [2]).y).formato(2))
	imprimir(nubes.celda_en({3, 17}, {16, 16}))
	imprimir(nubes.celda_en({-1, 17}, {16, 16}))
	imprimir(nubes.valores_en(mate.Rect {pos: {10, 10}, tamano: {10, 10}}, {16, 16}))
	imprimir(caida.y.formato(2))
	imprimir(apoyo.y.formato(2))

// Native defaults also fill omitted arguments in global initializers.
var cielo = rejilla.desde_texto(["....", "=...", "...."], ["=": 2, ".": 0])
var caida = cielo.mover(mate.Rect {pos: {2, 0}, tamano: {10, 10}}, {16, 16}, {0, 20}, [1])
var apoyo = cielo.mover(mate.Rect {pos: {2, 0}, tamano: {10, 10}}, plataformas = [2], tamano_celda = {16, 16}, delta = {0, 20}, solidos = [1])
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	// The wall (column 4) starts at x=64 and the floor (row 2) at y=32; the body spans 10..20 on both axes.
	runGeneratedGo(t, generated, "5\n44.00\n0\n12.00\n44.00\n12.00\n0\n6.00\n20.00\n-20.00\n5.00\n22.00\n8.00\nAlguno(2)\nNinguno\n[0, 2]\n20.00\n6.00\n")
}

// Camera helpers invert the drawing transform and use the default 320x180 logical screen; timers need no window.
func TestCamaraYTemporizador(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/mate
usar std/pincel/graficos
usar std/pincel/tiempo

fn ver(v mate.Vec2) cadena
	"${v.x.formato(1)},${v.y.formato(1)}"

fn inicio()
	var cam = graficos.Camara2D {pos: {100, 50}, origen: {160, 90}, zoom: 2}
	imprimir(ver(cam.a_mundo({160, 90})))
	imprimir(ver(cam.a_mundo({0, 0})))
	imprimir(ver(cam.a_pantalla({20, 5})))
	var giro = graficos.Camara2D {pos: {7, -3}, origen: {10, 20}, zoom: 1.5, rotacion: 0.7}
	imprimir(ver(giro.a_pantalla(giro.a_mundo({33, 44}))))
	var vista = cam.visible()
	imprimir("${ver(vista.pos)} ${ver(vista.tamano)}")
	var origen = graficos.Camara2D {pos: {0, 0}, origen: {160, 90}, zoom: 2}
	imprimir(ver(origen.limitada({pos: {0, 0}, tamano: {200, 100}}).pos))
	imprimir(ver(origen.limitada({pos: {0, 0}, tamano: {100, 50}}).pos))
	imprimir(ver(origen.siguiendo({10, 0}, 0.5).pos))
	imprimir(ver(origen.sacudida(0).pos))
	var t = tiempo.temporizador(1)
	imprimir(t.avanzar(0.6))
	imprimir(t.avanzar(0.6))
	imprimir(t.terminado())
	imprimir(t.avanzar(1))
	imprimir("${t.restante().formato(1)} ${t.progreso().formato(1)}")
	t.reiniciar()
	imprimir("${t.terminado()} ${t.progreso().formato(1)}")
	t.terminar()
	imprimir(t.terminado())
	var r = tiempo.temporizador(0.5, bucle = verdadero)
	imprimir(r.avanzar(0.6))
	imprimir("${r.terminado()} ${r.restante().formato(1)}")
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"100.0,50.0", "20.0,5.0", "0.0,0.0", "33.0,44.0", "20.0,5.0 160.0,90.0",
		"80.0,45.0", "50.0,25.0", "5.0,0.0", "0.0,0.0",
		"falso", "verdadero", "verdadero", "falso", "0.0 1.0", "falso 0.0", "verdadero",
		"verdadero", "falso 0.4",
	}, "\n") + "\n"
	runGeneratedGo(t, generated, want)
}
