package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "pruebas.cometa")
}

func TestProbarReportsResults(t *testing.T) {
	entry := writeProject(t, map[string]string{
		"solucion.cometa": "pub fn doble(n entero) entero\n\tn * 2\n",
		"pruebas.cometa": `usar solucion
usar std/pruebas

fn prueba_ok()
	pruebas.igual(4, solucion.doble(2))

fn prueba_mal()
	pruebas.igual(7, solucion.doble(3), "doble de 3")

fn prueba_crash()
	var v = [1, 2]
	var i = 5
	imprimir(v[i])
`,
	})
	report, code := probar(entry, "", time.Minute, true)
	if code != exitTestsFailed || report.Status != "fallo" || report.Total != 3 || report.Passed != 1 || report.Failed != 2 {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	status := map[string]testResult{}
	for _, r := range report.Tests {
		status[r.Name] = r
	}
	if status["prueba_ok"].Status != "ok" {
		t.Fatalf("prueba_ok: %+v", status["prueba_ok"])
	}
	if r := status["prueba_mal"]; r.Status != "fallo" || r.Message != "doble de 3: se esperaba 7 pero se obtuvo 6" || r.Line != 8 {
		t.Fatalf("prueba_mal: %+v", r)
	}
	if r := status["prueba_crash"]; r.Status != "error" || r.Line != 13 {
		t.Fatalf("prueba_crash: %+v", r)
	}
}

func TestProbarTimeoutAndCompileError(t *testing.T) {
	entry := writeProject(t, map[string]string{"pruebas.cometa": "fn prueba_a()\n\tmientras verdadero\n\t\tcontinuar\n\nfn prueba_b()\n\timprimir(1)\n"})
	report, code := probar(entry, "", 2*time.Second, true)
	if code != exitTimeout || report.Tests[0].Status != "tiempo_agotado" || report.Tests[1].Status != "omitida" {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	entry = writeProject(t, map[string]string{"pruebas.cometa": "usar std/pruebas\nfn prueba_x()\n\tpruebas.igual(1, \"a\")\n"})
	report, code = probar(entry, "", time.Minute, true)
	if code != exitCompile || report.Status != "error_compilacion" || len(report.Diagnostics) != 1 || report.Diagnostics[0].File != "pruebas.cometa" || report.Diagnostics[0].Line != 3 {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
}

// Visual tests step a game in a hidden window and compare screen pixels.
func TestProbarVisual(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	entry := writeProject(t, map[string]string{
		"juego.cometa": `usar std/pincel/graficos

pub tipo Caja
	pub x decimal

	pub fn actualizar(dt decimal)
		@x += 60 * dt

	pub fn pintar()
		graficos.limpiar(.Azul)
		graficos.rectangulo(@x, 10, 10, 10, .Rojo)
`,
		"pruebas.cometa": `usar juego
usar std/pruebas
usar std/pincel/lienzo

fn prueba_se_mueve()
	var caja = juego.Caja {}
	pruebas.avanzar(caja, 0)
	pruebas.pixel(5, 15, .Rojo)
	pruebas.avanzar(caja, 30)
	pruebas.casi_igual(30.0, caja.x)
	pruebas.pixel(5, 15, .Azul)
	pruebas.pixel(35, 15, .Rojo)
	var pantalla = pruebas.pantalla()
	pruebas.igual(320, lienzo.ancho(pantalla))
	pruebas.igual(180, lienzo.alto(pantalla))

fn prueba_color_equivocado()
	pruebas.avanzar(juego.Caja {}, ancho = 64, alto = 32)
	pruebas.pixel(60, 30, .Verde, mensaje = "fondo")

fn prueba_sin_avanzar()
	pruebas.pixel(0, 0, .Negro)
`,
	})
	report, code := probar(entry, "", 5*time.Minute, true)
	status := map[string]testResult{}
	for _, r := range report.Tests {
		status[r.Name] = r
	}
	if code != exitTestsFailed || report.Passed != 1 || status["prueba_se_mueve"].Status != "ok" {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	if r := status["prueba_color_equivocado"]; r.Status != "fallo" || r.Message != "fondo: en el píxel (60, 30) se esperaba rgba(0, 255, 0, 255) pero se obtuvo rgba(0, 0, 255, 255)" || r.Line != 19 {
		t.Fatalf("prueba_color_equivocado: %+v", r)
	}
	// The screen of an earlier test is not visible to the next one.
	if r := status["prueba_sin_avanzar"]; r.Status != "error" || r.Line != 22 {
		t.Fatalf("prueba_sin_avanzar: %+v", r)
	}
}

// Tests can press keys, move the pointer, wheel, type and use a gamepad through
// the same entrada and ui helpers the game reads.
func TestProbarEntradaSimulada(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	entry := writeProject(t, map[string]string{
		"juego.cometa": `usar std/pincel/graficos
usar std/pincel/entrada
usar std/pincel/ui

pub tipo Juego
	pub contexto ui.Contexto
	pub nombre cadena
	pub x entero
	pub pulsaciones entero
	pub soltadas entero
	pub clic decimal = -1
	pub rueda decimal
	pub mandos entero
	pub boton_a bool
	pub palanca decimal

	pub fn actualizar(dt decimal)
		si entrada.tecla_mantenida(.D)
			@x += 1
		si entrada.tecla_presionada(.Espacio)
			@pulsaciones += 1
		si entrada.tecla_soltada(.Espacio)
			@soltadas += 1
		si entrada.raton_presionado(.Izquierdo)
			@clic = entrada.posicion_raton().x
		@rueda += entrada.rueda().y
		@mandos = entrada.mandos()
		@boton_a = entrada.mando_boton(0, .A)
		@palanca = entrada.mando_eje(0, .IzquierdoX)
		con ui.cuadro(@contexto)
			@nombre = ui.campo_texto("nombre", @nombre)

	pub fn pintar()
		graficos.limpiar(.Negro)
		ui.pintar(@contexto)
`,
		"pruebas.cometa": `usar juego
usar std/pruebas
usar std/pincel/ui

fn prueba_teclas()
	var j = juego.Juego {contexto: ui.crear()}
	pruebas.avanzar(j, 2)
	pruebas.igual(0, j.x)
	pruebas.mantener(.D)
	pruebas.avanzar(j, 3)
	pruebas.igual(3, j.x)
	pruebas.soltar(.D)
	pruebas.avanzar(j, 2)
	pruebas.igual(3, j.x)
	pruebas.pulsar(.Espacio)
	pruebas.avanzar(j, 3)
	pruebas.igual(1, j.pulsaciones)
	pruebas.igual(1, j.soltadas)

fn prueba_la_entrada_no_pasa_a_la_siguiente_prueba()
	var j = juego.Juego {contexto: ui.crear()}
	pruebas.avanzar(j, 2)
	pruebas.igual(0, j.x)

fn prueba_raton_y_rueda()
	var j = juego.Juego {contexto: ui.crear()}
	pruebas.raton(40, 5)
	pruebas.pulsar_raton(.Izquierdo)
	pruebas.rueda(0, -2)
	pruebas.avanzar(j, 1)
	pruebas.igual(40, j.clic)
	pruebas.casi_igual(-2.0, j.rueda)
	pruebas.avanzar(j, 2)
	pruebas.casi_igual(-2.0, j.rueda, mensaje = "la rueda dura un cuadro")

fn prueba_mando()
	var j = juego.Juego {contexto: ui.crear()}
	pruebas.avanzar(j, 1)
	pruebas.igual(0, j.mandos)
	pruebas.mantener_mando(.A)
	pruebas.eje_mando(.IzquierdoX, 0.5)
	pruebas.avanzar(j, 1)
	pruebas.igual(1, j.mandos)
	pruebas.afirmar(j.boton_a, "botón A")
	pruebas.casi_igual(0.5, j.palanca)

fn prueba_texto()
	var j = juego.Juego {contexto: ui.crear()}
	pruebas.avanzar(j, 1)
	pruebas.pulsar(.Tab)
	pruebas.avanzar(j, 2)
	pruebas.texto("hola")
	pruebas.avanzar(j, 1)
	pruebas.igual("hola", j.nombre)
`,
	})
	report, code := probar(entry, "", 5*time.Minute, true)
	if code != 0 {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	if report.Passed != 5 {
		t.Fatalf("unexpected report %+v", report)
	}
}
