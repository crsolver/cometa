package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestEmbeddingExample(t *testing.T) {
	source, err := os.ReadFile("testdata/programas/embebidos.cometa")
	if err != nil {
		t.Fatal(err)
	}
	runCometa(t, string(source), "Ana\nIngeniera\nLuis\nLuis\nInvitado\n")
	code, err := Compile("embedding.cometa", source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(code), "\t*Persona\n") {
		t.Fatalf("expected anonymous Go field:\n%s", code)
	}
}

func TestEmbeddingRuntime(t *testing.T) {
	runCometa(t, `interfaz Proveedor<T>
	fn obtener() T
tipo Caja<T>
	valor T
	pub fn obtener() T @valor
	pub fn elegir(v T = @valor) T v
tipo Medio<T>
	pub Caja<T>
tipo Final<T>
	pub Medio<T>
	pub fn leer() T @elegir()
tipo Base
	n entero
	pub fn subir(v entero = 1)
		@n = @n + v
tipo Otra
	n entero
	pub fn subir(v entero = 100)
		@n = @n + v
tipo Profunda
	pub Otra
tipo Contenedor
	pub Base
	pub Profunda
	pub fn cambiar()
		@n = 4
		@subir(v = 2)
tipo Sombra
	pub Base
	n cadena
	pub fn subir() cadena "local"
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
	const base = "tipo A\n\tx entero\n\tpub fn f() entero @x\ntipo B\n\tx entero\n\tpub fn f() entero @x\ntipo C\n\tpub A\n\tpub B\n"
	for _, tc := range []struct{ name, source, want string }{
		{"read ambiguity", base + "fn g(c C) entero c.x\n", "ambiguo"},
		{"write ambiguity", base + "fn g(c C)\n\tc.x = 1\n", "ambiguo"},
		{"call ambiguity", base + "fn g(c C) entero c.f()\n", "ambiguo"},
		{"receiver ambiguity", base + "\tfn g() entero @x\n", "ambiguo"},
		{"receiver call ambiguity", base + "\tfn g() entero @f()\n", "ambiguo"},
		{"receiver write ambiguity", base + "\tfn g()\n\t\t@x = 1\n", "ambiguo"},
		{"diamond", "tipo A\n\tx entero\ntipo B\n\tpub A\ntipo C\n\tpub A\ntipo D\n\tpub B\n\tpub C\nfn f(d D) entero d.x\n", "ambiguo"},
		{"field method collision", "tipo A\n\tx entero\ntipo B\n\tpub fn x() entero 1\ntipo C\n\tpub A\n\tpub B\nfn f(c C) entero c.x()\n", "ambiguo"},
		{"promoted literal", base + "fn g() C C {x: 1}\n", "no existe"},
		{"duplicate", "tipo A\n\tx entero\ntipo B\n\tpub A\n\tpub A\n", "ya fue declarado"},
		{"named conflict", "tipo A\n\tx entero\ntipo B\n\tpub A\n\tA entero\n", "ya fue declarado"},
		{"method conflict", "tipo A\n\tx entero\ntipo B\n\tpub A\n\tpub fn A() entero 1\n", "ya fue declarado"},
		{"generic duplicate", "tipo A<T>\n\tx T?\ntipo B\n\tpub A<entero>\n\tpub A<cadena>\n", "ya fue declarado"},
		{"primitive", "tipo B\n\tentero\n", "solo se pueden embeber"},
		{"list", "tipo B\n\t[entero]\n", "solo se pueden embeber"},
		{"optional", "tipo A\n\tx entero\ntipo B\n\tA?\n", "solo se pueden embeber"},
		{"result", "tipo A\n\tx entero\ntipo B\n\tA!\n", "solo se pueden embeber"},
		{"interface", "interfaz I\ntipo B\n\tpub I\n", "solo se pueden embeber"},
		{"enum", "enum E\n\tV\ntipo B\n\tpub E\n", "solo se pueden embeber"},
		{"parameter", "tipo B<T>\n\tpub T\n", "solo se pueden embeber"},
		{"cycle", "tipo A\n\tpub B\ntipo B\n\tpub A\n", "ciclo de campos"},
		{"self cycle", "tipo A\n\tpub A\n", "ciclo de campos"},
		{"required default", "tipo A\n\tx entero!\ntipo B\n\tpub A\nfn f() B B {}\n", "A.x requiere inicialización"},
		{"generic default", "tipo A<T>\n\tx T\ntipo B\n\tpub A<entero>\nfn f() B B {}\n", "A.x requiere inicialización"},
		{"ambiguous interface", base + "interfaz I\n\tfn f() entero\nfn g(c C) I c\n", "I"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Analyze("bad.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %q", err, tc.want)
			}
		})
	}
}

func TestUnusedAmbiguousEmbeddingAndExplicitPaths(t *testing.T) {
	runCometa(t, "tipo A\n\tx entero\ntipo B\n\tpub A\ntipo C\n\tpub A\ntipo D\n\tpub B\n\tpub C\nfn inicio()\n\tvar d = D {}\n\td.B.x = 9\n\timprimir(d.B.A.x)\n\timprimir(d.C.x)\n", "9\n0\n")
}

func TestEmbeddingDefaultCallsWithoutFlowLowering(t *testing.T) {
	runCometa(t, `tipo Base
	n entero
	pub fn leer(v entero = @n) entero v
tipo Izquierda
	pub Base
tipo Derecha
	pub Base
tipo Exterior
	pub Base
	pub Izquierda
	pub Derecha
	pub fn leerBase() entero @leer()
fn inicio()
	var e = Exterior {Base: {n: 6}}
	imprimir(e.leer())
	imprimir(e.leerBase())
	imprimir(e.leer(v = 9))
`, "6\n6\n9\n")
}
