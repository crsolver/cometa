package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestEmbeddingExample(t *testing.T) {
	source, err := os.ReadFile("../../examples/embebidos.hacha")
	if err != nil {
		t.Fatal(err)
	}
	runHacha(t, string(source), "Ana\nIngeniera\nLuis\nLuis\nInvitado\n")
	code, err := Compile("embedding.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(code), "\t*Persona\n") {
		t.Fatalf("expected anonymous Go field:\n%s", code)
	}
}

func TestEmbeddingRuntime(t *testing.T) {
	runHacha(t, `interfaz Proveedor<T>
	fn obtener() T
tipo Caja<T>
	valor T
	fn obtener() T @valor
	fn elegir(v T = @valor) T v
tipo Medio<T>
	Caja<T>
tipo Final<T>
	Medio<T>
	fn leer() T @elegir()
tipo Base
	n entero
	fn subir(v entero = 1)
		@n = @n + v
tipo Otra
	n entero
	fn subir(v entero = 100)
		@n = @n + v
tipo Profunda
	Otra
tipo Contenedor
	Base
	Profunda
	fn cambiar()
		@n = 4
		@subir(v = 2)
tipo Sombra
	Base
	n cadena
	fn subir() cadena "local"
fn tomar<T Proveedor<entero>>(v T) entero v.obtener()
fn inicio()
	var f = Final<entero> {Medio: {Caja: {valor: 7}}}
	imprimir(f.valor)
	imprimir(f.leer())
	imprimir(f.elegir(v = 8))
	imprimir(tomar(f))
	var a = Contenedor {}
	var b = Contenedor {}
	a.cambiar()
	a.subir()
	imprimir(a.n)
	imprimir(b.n)
	imprimir(a.Profunda.n)
	var s = Sombra {n: "propio"}
	imprimir(s.n)
	imprimir(s.subir())
`, "7\n7\n8\n7\n7\n0\n0\npropio\nlocal\n")
}

func TestEmbeddingDiagnostics(t *testing.T) {
	const base = "tipo A\n\tx entero\n\tfn f() entero @x\ntipo B\n\tx entero\n\tfn f() entero @x\ntipo C\n\tA\n\tB\n"
	for _, tc := range []struct{ name, source, want string }{
		{"read ambiguity", base + "fn g(c C) entero c.x\n", "ambiguo"},
		{"write ambiguity", base + "fn g(c C)\n\tc.x = 1\n", "ambiguo"},
		{"call ambiguity", base + "fn g(c C) entero c.f()\n", "ambiguo"},
		{"receiver ambiguity", base + "\tfn g() entero @x\n", "ambiguo"},
		{"receiver call ambiguity", base + "\tfn g() entero @f()\n", "ambiguo"},
		{"receiver write ambiguity", base + "\tfn g()\n\t\t@x = 1\n", "ambiguo"},
		{"diamond", "tipo A\n\tx entero\ntipo B\n\tA\ntipo C\n\tA\ntipo D\n\tB\n\tC\nfn f(d D) entero d.x\n", "ambiguo"},
		{"field method collision", "tipo A\n\tx entero\ntipo B\n\tfn x() entero 1\ntipo C\n\tA\n\tB\nfn f(c C) entero c.x()\n", "ambiguo"},
		{"promoted literal", base + "fn g() C C {x: 1}\n", "no existe"},
		{"duplicate", "tipo A\n\tx entero\ntipo B\n\tA\n\tA\n", "ya fue declarado"},
		{"named conflict", "tipo A\n\tx entero\ntipo B\n\tA\n\tA entero\n", "ya fue declarado"},
		{"method conflict", "tipo A\n\tx entero\ntipo B\n\tA\n\tfn A() entero 1\n", "ya fue declarado"},
		{"generic duplicate", "tipo A<T>\n\tx T?\ntipo B\n\tA<entero>\n\tA<cadena>\n", "ya fue declarado"},
		{"primitive", "tipo B\n\tentero\n", "solo se pueden embeber"},
		{"list", "tipo B\n\t[entero]\n", "solo se pueden embeber"},
		{"optional", "tipo A\n\tx entero\ntipo B\n\tA?\n", "solo se pueden embeber"},
		{"result", "tipo A\n\tx entero\ntipo B\n\tA!\n", "solo se pueden embeber"},
		{"interface", "interfaz I\ntipo B\n\tI\n", "solo se pueden embeber"},
		{"enum", "enum E\n\tV\ntipo B\n\tE\n", "solo se pueden embeber"},
		{"parameter", "tipo B<T>\n\tT\n", "solo se pueden embeber"},
		{"cycle", "tipo A\n\tB\ntipo B\n\tA\n", "ciclo de campos"},
		{"self cycle", "tipo A\n\tA\n", "ciclo de campos"},
		{"required default", "tipo A\n\tx entero!\ntipo B\n\tA\nfn f() B B {}\n", "A.x requiere inicialización"},
		{"generic default", "tipo A<T>\n\tx T\ntipo B\n\tA<entero>\nfn f() B B {}\n", "A.x requiere inicialización"},
		{"ambiguous interface", base + "interfaz I\n\tfn f() entero\nfn g(c C) I c\n", "I"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Analyze("bad.hacha", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %q", err, tc.want)
			}
		})
	}
}

func TestUnusedAmbiguousEmbeddingAndExplicitPaths(t *testing.T) {
	runHacha(t, "tipo A\n\tx entero\ntipo B\n\tA\ntipo C\n\tA\ntipo D\n\tB\n\tC\nfn inicio()\n\tvar d = D {}\n\td.B.x = 9\n\timprimir(d.B.A.x)\n\timprimir(d.C.x)\n", "9\n0\n")
}

func TestEmbeddingDefaultCallsWithoutFlowLowering(t *testing.T) {
	runHacha(t, `tipo Base
	n entero
	fn leer(v entero = @n) entero v
tipo Izquierda
	Base
tipo Derecha
	Base
tipo Exterior
	Base
	Izquierda
	Derecha
	fn leerBase() entero @leer()
fn inicio()
	var e = Exterior {Base: {n: 6}}
	imprimir(e.leer())
	imprimir(e.leerBase())
	imprimir(e.leer(v = 9))
`, "6\n6\n9\n")
}
