package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNumericRuntime(t *testing.T) {
	runCometa(t, `interfaz Valor
enum E
	A
	B
tipo Caja
	x decimal
fn aceptar(x decimal) decimal x
fn opcional(x entero) decimal? x
fn resultado(x entero) decimal! x
fn identidad<T>(x T) T x
fn inicio()
	var n = 5
	var d decimal = n
	imprimir(n / 2)
	imprimir(-n / 2)
	imprimir(-n % 2)
	imprimir(n / 2.0)
	imprimir(d + n)
	imprimir(aceptar(n))
	imprimir(Caja {x: n}.x)
	var xs [decimal] = [n, 2.5]
	xs.agregar(n)
	imprimir(xs[2])
	imprimir(opcional(n) o 0)
	imprimir(resultado(n) capturar 0)
	imprimir(entero(2.9))
	imprimir(entero(-2.9))
	imprimir(decimal(n) / 2)
	var a = si verdadero n sino 1.5
	var b = si falso 1.5 sino n
	imprimir(a)
	imprimir(b)
	var c = casos E.A
		.A => n
		.B => 1.5
	imprimir(c)
	var mezclados = [n, 1.5]
	imprimir(mezclados[0])
	var v Valor = n
	imprimir((v como entero) o -1)
	imprimir((v como decimal) o -1.0)
	imprimir(identidad<decimal>(n))
	var minimo = -9223372036854775808
	var maximo = 9223372036854775807
	imprimir("${minimo} ${maximo}")
	var grande = 9007199254740993
	imprimir(decimal(grande) == 9007199254740992.0)
`, "2\n-2\n-1\n2.5\n10\n5\n5\n5\n5\n5\n2\n-2\n2.5\n5\n5\n5\n5\n5\n-1\n5\n-9223372036854775808 9223372036854775807\ntrue\n")
}

func TestNumericSimpleAndFlowGeneration(t *testing.T) {
	for _, forceFlow := range []string{"", "\tvar activar entero? = 0\n"} {
		source := `enum E
	A
fn decimal_desde_rama(x entero) decimal
	si verdadero
		x / 2
	sino
		1.5
fn inicio()
` + forceFlow + `	var n = 5
	var x decimal = n / 2
	imprimir(x)
	imprimir((si verdadero n sino 1) + 0.5)
	imprimir(decimal_desde_rama(n))
	decimal(n)
	entero(1.5)
	var r decimal = casos E.A
		.A => n
	imprimir(r)
`
		runCometa(t, source, "2\n5.5\n2\n5\n")
	}
}

func TestNumericGameRuntime(t *testing.T) {
	source := `fn iniciar()
	var n = 3
	var a entero = mate.minimo(n, 2)
	var b entero = mate.maximo(n, 2)
	var c entero = mate.limitar(n, 0, 2)
	var d entero = mate.absoluto(-n)
	var e decimal = mate.minimo(n, 2.5)
	imprimir("${a} ${b} ${c} ${d} ${e}")
	imprimir("${mate.piso(-1.5)} ${mate.techo(-1.5)} ${mate.redondear(-1.5)}")
	var fijo entero = azar.entero(7, 7)
	imprimir(fijo)
	var canales color.Color = color.rgba(300, -1, 20)
fn actualizar(dt decimal) imprimir(dt)
fn pintar() imprimir(0)
`
	generated, err := Compile("numeric-game.cometa", []byte(pincelImports+source))
	if err != nil {
		t.Fatal(err)
	}
	runnable := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	runnable += `
func main() {
 Iniciar()
 for i:=0;i<100;i++ { v:=_hgazarEntero(-9223372036854775808,9223372036854775807); if v==9223372036854775807 {panic("límite incluido")} }
 fmt.Println(_hgrgba(300,-1,20,255))
}
`
	runGeneratedGo(t, []byte(runnable), "2 3 2 3 2.5\n-2 -1 -2\n7\n{255 0 20 255}\n")
}

func TestNumericDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"fn f(x num) num x\n", "num fue eliminado; use entero o decimal"},
		{"fn inicio()\n\tvar n = [1][1.7]\n", "índice de una lista debe ser entero"},
		{"fn inicio()\n\tvar xs = [1]\n\tvar n = xs.obtener(1.5)\n", "debe ser entero"},
		{"fn inicio()\n\tvar n = \"abc\".obtener(1.5)\n", "debe ser entero"},
		{"fn inicio()\n\trepetir (0.5..2) |i| imprimir(i)\n", "inicio del rango debe ser entero"},
		{"fn inicio()\n\trepetir (0..2.0) |i| imprimir(i)\n", "final del rango debe ser entero"},
		{"fn inicio()\n\tvar n entero = 1.0\n", "no se puede asignar decimal a entero"},
		{"fn inicio()\n\tvar xs = [1]\n\tvar ds [decimal] = xs\n", "no se puede asignar [entero] a [decimal]"},
		{"fn inicio()\n\tvar x entero? = 1\n\tvar y decimal? = x\n", "no se puede asignar entero? a decimal?"},
		{"fn inicio() imprimir(1.0 % 2)\n", "requiere enteros"},
		{"fn inicio() imprimir(9223372036854775808)\n", "fuera del rango de int64"},
		{"fn inicio() imprimir(-9223372036854775809)\n", "fuera del rango de int64"},
		{"fn inicio() imprimir(entero(verdadero))\n", "argumento numérico"},
	} {
		_, err := Compile("numeric.cometa", []byte(tc.source))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", tc.source, err, tc.want)
		}
	}
}

func TestNumericConversionFailureRuntime(t *testing.T) {
	generated, err := Compile("numeric.cometa", []byte("fn inicio() imprimir(entero(1.5))\n"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	source = strings.Replace(source, `import "fmt"`, "import (\"fmt\"; \"math\")", 1)
	source += `
func main() {
 for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 9223372036854775808.0, -9223372036854777856.0} {
  func() { defer func(){ if r:=recover(); r == nil { panic("faltó validación") } else {fmt.Println(r)} }(); _hentero(v) }()
 }
 fmt.Println(_hentero(-9223372036854775808.0))
}
`
	runGeneratedGo(t, []byte(source), strings.Repeat("conversión a entero inválida: valor no finito o fuera del rango de int64\n", 5)+"-9223372036854775808\n")
}

func TestNumericExamples(t *testing.T) {
	// Entry programs live one level down (basico/, pincel/); deeper files are imported modules.
	paths, err := filepath.Glob("../../examples/*/*.cometa")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no examples found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path))+"/"+filepath.Base(path), func(t *testing.T) {
			abs, err := filepath.Abs(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = CompileProject(abs, os.ReadFile)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
