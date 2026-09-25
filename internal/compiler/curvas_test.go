package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestCurvesNumericsWithoutGame(t *testing.T) {
	generated, err := Compile("curvas.cometa", []byte("usar std/mate/curvas\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), `"math"`) || strings.Contains(string(generated), "ebiten") {
		t.Fatal("curves must use math without Ebitengine")
	}
	// Exercise the emitted implementations, including every public entry point.
	harness := `
func main() {
 families := []struct {
  name string
  in, out, both func(float64) float64
  midpoint float64
  monotonic bool
 }{
  {"cuadratica", _hgcurva_cuadratica_entrada, _hgcurva_cuadratica_salida, _hgcurva_cuadratica_entrada_salida, 0.25, true},
  {"cubica", _hgcurva_cubica_entrada, _hgcurva_cubica_salida, _hgcurva_cubica_entrada_salida, 0.125, true},
  {"cuartica", _hgcurva_cuartica_entrada, _hgcurva_cuartica_salida, _hgcurva_cuartica_entrada_salida, 0.0625, true},
  {"quintica", _hgcurva_quintica_entrada, _hgcurva_quintica_salida, _hgcurva_quintica_entrada_salida, 0.03125, true},
  {"senoidal", _hgcurva_senoidal_entrada, _hgcurva_senoidal_salida, _hgcurva_senoidal_entrada_salida, 0.2928932188134524, true},
  {"circular", _hgcurva_circular_entrada, _hgcurva_circular_salida, _hgcurva_circular_entrada_salida, 0.1339745962155614, true},
  {"exponencial", _hgcurva_exponencial_entrada, _hgcurva_exponencial_salida, _hgcurva_exponencial_entrada_salida, 0.03125, true},
  {"elastica", _hgcurva_elastica_entrada, _hgcurva_elastica_salida, _hgcurva_elastica_entrada_salida, -0.02209708691207961, false},
  {"retroceso", _hgcurva_retroceso_entrada, _hgcurva_retroceso_salida, _hgcurva_retroceso_entrada_salida, -0.375, false},
  {"rebote", _hgcurva_rebote_entrada, _hgcurva_rebote_salida, _hgcurva_rebote_entrada_salida, 0.28125, false},
 }
 near := func(got, want float64) {
  if math.IsNaN(got) || math.Abs(got-want) > 1e-12 { panic("unexpected curve sample") }
 }
 check := func(f func(float64) float64, monotonic bool) {
  for _, p := range []float64{math.Inf(-1), -2, 0} { if f(p) != 0 { panic("start endpoint") } }
  for _, p := range []float64{1, 2, math.Inf(1)} { if f(p) != 1 { panic("end endpoint") } }
  if !math.IsNaN(f(math.NaN())) { panic("NaN must propagate") }
  previous := f(0)
  for i := 1; i <= 1000; i++ {
   v := f(float64(i)/1000)
   if math.IsNaN(v) || math.IsInf(v, 0) { panic("nonfinite curve") }
   if monotonic && (v < previous-1e-12 || v < 0 || v > 1) { panic("nonmonotonic curve") }
   previous = v
  }
 }
 check(_hgcurva_lineal, true)
 near(_hgcurva_lineal(0.37), 0.37)
 for _, f := range families {
  check(f.in, f.monotonic); check(f.out, f.monotonic); check(f.both, f.monotonic)
  near(f.in(0.5), f.midpoint); near(f.out(0.5), 1-f.midpoint)
  near(f.both(0.25), f.midpoint/2); near(f.both(0.5), 0.5); near(f.both(0.75), 1-f.midpoint/2)
  if f.name == "elastica" || f.name == "retroceso" {
   if f.in(0.5) >= 0 || f.out(0.5) <= 1 || f.both(0.25) >= 0 || f.both(0.75) <= 1 { panic("lost overshoot") }
  }
 }
 // Bounce piece boundaries are continuous and touch the upper envelope.
 for _, p := range []float64{4.0/11, 8.0/11, 0.9} {
  near(_hgcurva_rebote_salida(p), 1)
  if math.Abs(_hgcurva_rebote_salida(p-1e-9)-_hgcurva_rebote_salida(p+1e-9)) > 1e-7 { panic("bounce discontinuity") }
 }
}
`
	runGeneratedGo(t, append(generated, []byte(harness)...), "")
}

func TestCurvesImportsAndCalls(t *testing.T) {
	runCometa(t, "usar std/mate/curvas\nfn inicio()\n\timprimir(curvas.cubica_entrada(progreso = 0.5))\n\timprimir(curvas.lineal(2))\n", "0.125\n1\n")
	runCometa(t, "usar std/mate/curvas como c\nfn inicio() imprimir(c.cuadratica_salida(0.5))\n", "0.75\n")
	for _, source := range []string{
		"fn inicio() imprimir(curvas.lineal(0.5))\n",
		"usar std/mate\nfn inicio() imprimir(curvas.lineal(0.5))\n",
		"usar std/mate\nfn inicio() imprimir(mate.lineal(0.5))\n",
		"usar std/mate/curvas\nfn inicio() imprimir(mate.pi)\n",
		"usar std/mate/curvas\nfn inicio() imprimir(curvas.pi)\n",
		"usar std/mate/curvas como c\nfn inicio() imprimir(curvas.lineal(0.5))\n",
		"usar std/mate/curvas\nfn inicio() imprimir(curvas.lineal(verdadero))\n",
		"usar std/pincel/curvas\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":   "usar helper\nfn inicio() imprimir(helper.valor())\n",
		"helper.cometa": "usar std/mate/curvas como c\npub fn valor() decimal c.cubica_entrada(0.5)\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("transitive curves imported game runtime")
	}
	runGeneratedGo(t, generated, "0.125\n")
}

func TestCurvesExample(t *testing.T) {
	source, err := os.ReadFile("../../examples/curvas.cometa")
	if err != nil {
		t.Fatal(err)
	}
	runCometa(t, string(source), "20\n27.5\n80\n132.5\n140\n140\n")
}
