package compiler

import (
	"strings"
	"testing"
)

func TestDefaultsRuntime(t *testing.T) {
	runHacha(t, `fn inicio()
	imprimir(sumar())
	imprimir(sumar(0))
	imprimir(sumar(b = 9))
	imprimir(sumar(2, b = 3))
	imprimir(sumar(valores = [3, 4]))
	imprimir(sumar(1, 2, [3, 4]...))
	imprimir(sumar(1, 2, 3, 4))
	imprimir(recursiva(3))
	imprimir(anidada())
fn sumar(a num = 5, b num = a + 1, valores ...num) num
	var total = a + b
	repetir (valores) |v| total = total + v
	total
fn recursiva(n num, paso num = 1) num
	si (n == 0) 0 sino paso + recursiva(n - 1)
fn anidada(a num = sumar(), b num = si (a > 0) a sino 0,) num b
`, "11\n1\n14\n5\n18\n10\n10\n3\n11\n")
}

func TestDefaultEvaluationOrderAndScope(t *testing.T) {
	runHacha(t, `tipo Caja
	valor num
	fn siguiente() num
		@valor = @valor + 1
		@valor
	fn poner(a num = @siguiente(), b num = a + @siguiente(), c num = 0)
		imprimir(a)
		imprimir(b)
		imprimir(c)
	fn reenviar() @poner(c = 8)
fn receptor(c Caja) Caja
	imprimir("receptor")
	c
fn argumento(n num) num
	imprimir(n)
	n
fn inicio()
	var c = Caja {}
	receptor(c).poner(c = argumento(9))
	imprimir(c.valor)
	c.poner(c = argumento(7), b = argumento(6), a = argumento(0))
	imprimir(c.valor)
	c.reenviar()
`, "receptor\n9\n1\n3\n9\n2\n7\n6\n0\n0\n6\n7\n2\n3\n7\n8\n")
}

func TestDefaultContextAndFreshReferences(t *testing.T) {
	runHacha(t, `tipo Caja
	valor num
enum E
	A
	B num
fn crear(c Caja = {valor: 1}, alias Caja = c) Caja
	alias.valor = alias.valor + 1
	c
fn lista(xs [num] = [1]) num
	xs[0] = xs[0] + 1
	xs[0]
fn elegir(e E = .B(4), xs [Caja] = [], texto cadena = "hola", activo bool = verdadero)
	casos e
		.A => imprimir("A")
		.B => imprimir(texto)
	imprimir(activo)
fn inicio()
	imprimir(crear().valor)
	imprimir(crear().valor)
	imprimir(lista())
	imprimir(lista())
	elegir()
	elegir(.A, activo = falso, texto = "")
`, "2\n2\n2\n2\nhola\ntrue\nA\nfalse\n")
}

func TestInvalidDefaults(t *testing.T) {
	cases := []struct{ source, message string }{
		{"fn f(a num = verdadero) num a\n", "debe ser num"},
		{"fn f(a ...num = []) num 0\n", "variádico"},
		{"fn f(a num = 1, b num) num b\n", "obligatorio"},
		{"fn f(a num = a) num a\n", "no existe"},
		{"fn f(a num = b, b num = 1) num a\n", "no existe"},
		{"fn f(a num = local) num\n\tvar local = 1\n\ta\n", "no existe"},
		{"fn f(a num = @valor) num a\n", "método"},
		{"fn f(a num, b num = 1) num a\nfn inicio() f(b = 2)\n", "falta"},
		{"fn f(a num = 1) num a\nfn inicio() f(verdadero)\n", "debe ser num"},
		{"fn f(a num = 1) num a\nfn inicio() f(1, a = 2)\n", "duplicado"},
		{"fn inicio(a num = 1) imprimir(a)\n", "inicio"},
		{"fn f(a num = ) num a\n", "expresión"},
	}
	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			_, err := Compile("defaults.hacha", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want %q", err, tt.message)
			}
		})
	}
}
