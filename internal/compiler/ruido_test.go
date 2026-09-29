package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestNoiseNumerics(t *testing.T) {
	generated, err := Compile("ruido.cometa", []byte("usar std/mate/ruido\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ebiten", "math/rand", "_hgazar", "_hgcurva_"} {
		if strings.Contains(string(generated), forbidden) {
			t.Fatalf("unexpected dependency %s", forbidden)
		}
	}
	harness := `
func main() {
 near := func(got, want float64) { if math.IsNaN(got) || math.Abs(got-want)>1e-14 { panic("noise reference mismatch") } }
 for _, s := range []struct{x,y float64; seed int64; want float64}{
  {0,0,0,0.4917453265072494}, {1,2,42,0.9342569175613852},
  {-2,-3,-7,0.24081248639792388}, {-0.25,0.75,42,0.6429085207146035},
  {0.5,0.5,0,0.4768410255192784},
 } { near(_hgruidoSuave(s.x,s.y,s.seed),s.want) }
 for _, seed := range []int64{0,42,-7,math.MinInt64,math.MaxInt64} {
  for i:=-100;i<=100;i++ {
   x,y:=float64(i)*0.071,float64(i)*-0.039
   a:=_hgruidoSuave(x,y,seed)
   b:=_hgruidoFractal(x,y,seed,16,1,2)
   if a<0||a>1||b<0||b>1||math.IsNaN(b) {panic("range")}
   _hgruidoSuave(y,x,seed+1)
   if a!=_hgruidoSuave(x,y,seed)||b!=_hgruidoFractal(x,y,seed,16,1,2) {panic("stateful noise")}
   near(_hgruidoFractal(x,y,seed,1,0.5,2),a)
   near(_hgruidoFractal(x,y,seed,16,0,2),a)
  }
 }
 if _hgruidoSuave(0.2,0.3,1)==_hgruidoSuave(0.2,0.3,2) {panic("seed ignored")}
 // Check normalization against explicit octave samples and independently fixed mixed seed.
 seed2:=int64(-4767286540954276203) // mix(42 + golden-ratio increment)
 want:=(_hgruidoSuave(0.2,-0.3,42)+0.5*_hgruidoSuave(0.4,-0.6,seed2))/1.5
 near(_hgruidoFractal(0.2,-0.3,42,2,0.5,2),want)
 for i:=-3;i<=3;i++ {
  p:=float64(i)
  if math.Abs(_hgruidoSuave(p-1e-8,0.37,42)-_hgruidoSuave(p+1e-8,0.37,42))>1e-7 {panic("x discontinuity")}
  if math.Abs(_hgruidoSuave(0.37,p-1e-8,42)-_hgruidoSuave(0.37,p+1e-8,42))>1e-7 {panic("y discontinuity")}
 }
 mustPanic:=func(f func()) { defer func(){if recover()==nil {panic("expected invalid noise input to panic")}}(); f() }
 for _,bad:=range []float64{math.NaN(),math.Inf(1),math.Inf(-1),4503599627370496,-4503599627370496} {
  mustPanic(func(){_hgruidoSuave(bad,0,0)})
  mustPanic(func(){_hgruidoSuave(0,bad,0)})
  mustPanic(func(){_hgruidoFractal(bad,0,0,4,0.5,2)})
 }
 for _,n:=range []int64{0,-1,17} {mustPanic(func(){_hgruidoFractal(0,0,0,n,0.5,2)})}
 for _,p:=range []float64{-0.1,1.1,math.NaN(),math.Inf(1),math.Inf(-1)} {mustPanic(func(){_hgruidoFractal(0,0,0,4,p,2)})}
 for _,l:=range []float64{0,0.9,math.NaN(),math.Inf(1),math.Inf(-1)} {mustPanic(func(){_hgruidoFractal(0,0,0,4,0.5,l)})}
 for _,p:=range []float64{0,0.5} {mustPanic(func(){_hgruidoFractal(1,0,0,2,p,4503599627370496)})}
 mustPanic(func(){_hgruidoFractal(0,2,0,2,0.5,math.MaxFloat64)})
 _hgruidoSuave(math.Nextafter(4503599627370496,0),math.Nextafter(-4503599627370496,0),0)
 _hgruidoFractal(0.2,0.3,0,16,1,1)
}
`
	runGeneratedGo(t, append(generated, []byte(harness)...), "")
}

func TestNoiseCalls(t *testing.T) {
	runCometa(t, `usar std/mate/ruido como r
fn inicio()
	imprimir(r.suave(0, 0) == r.suave(y = 0, x = 0, semilla = 0))
	imprimir(r.fractal(1, 2) == r.fractal(lacunaridad = 2, persistencia = 0.5, octavas = 4, semilla = 0, y = 2, x = 1))
	imprimir(r.fractal(1, 2, octavas = 1, semilla = 42) == r.suave(1, 2, 42))
`, "verdadero\nverdadero\nverdadero\n")
	for _, source := range []string{
		"fn inicio() imprimir(ruido.suave(0, 0))\n",
		"usar std/mate\nfn inicio() imprimir(ruido.suave(0, 0))\n",
		"usar std/mate/ruido como r\nfn inicio() imprimir(ruido.suave(0, 0))\n",
		"usar std/mate/ruido\nfn inicio() imprimir(ruido.suave(verdadero, 0))\n",
		"usar std/mate/ruido\nfn inicio() imprimir(ruido.suave(0, 0, 1.5))\n",
		"usar std/mate/ruido\nfn inicio() imprimir(ruido.fractal(0, 0, octavas = 1.5))\n",
		"usar std/pincel/ruido\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":   "usar helper\nfn inicio() imprimir(helper.valor())\n",
		"helper.cometa": "usar std/mate/ruido\npub fn valor() bool ruido.suave(1, 2, 42) > 0.9\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "verdadero\n")
}

func TestNoiseExample(t *testing.T) {
	source, err := os.ReadFile("testdata/programas/ruido.cometa")
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the example's map builders twice in reverse order, without replacing their logic.
	source = []byte(strings.Replace(string(source), "fn inicio()", "fn mostrar()", 1))
	source = append(source, []byte(`
fn inicio()
	var t = terreno(42)
	var c = cuevas(42)
	imprimir(c == cuevas(42))
	imprimir(t == terreno(42))
	imprimir(t != terreno(43))
	imprimir(c != cuevas(43))
`)...)
	runCometa(t, string(source), "verdadero\nverdadero\nverdadero\nverdadero\n")
}
