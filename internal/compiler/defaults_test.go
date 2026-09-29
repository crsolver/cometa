package compiler

import (
	"strings"
	"testing"
)

func TestDefaultsRuntime(t *testing.T) {
	runCometa(t, `fn inicio()
	imprimir(sumar())
	imprimir(sumar(0))
	imprimir(sumar(b = 9))
	imprimir(sumar(2, b = 3))
	imprimir(sumar(valores = [3, 4]))
	imprimir(sumar(1, 2, [3, 4]...))
	imprimir(sumar(1, 2, 3, 4))
	imprimir(recursiva(3))
	imprimir(anidada())
fn sumar(a entero = 5, b entero = a + 1, valores ...entero) entero
	var total = a + b
	repetir (valores) |v| total = total + v
	total
fn recursiva(n entero, paso entero = 1) entero
	si (n == 0) 0 sino paso + recursiva(n - 1)
fn anidada(a entero = sumar(), b entero = si (a > 0) a sino 0,) entero b
`, "11\n1\n14\n5\n18\n10\n10\n3\n11\n")
}

func TestDefaultEvaluationOrderAndScope(t *testing.T) {
	runCometa(t, `tipo Caja
	valor entero
	fn siguiente() entero
		@valor = @valor + 1
		@valor
	fn poner(a entero = @siguiente(), b entero = a + @siguiente(), c entero = 0)
		imprimir(a)
		imprimir(b)
		imprimir(c)
	fn reenviar() @poner(c = 8)
fn receptor(c Caja) Caja
	imprimir("receptor")
	c
fn argumento(n entero) entero
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
	runCometa(t, `tipo Caja
	valor entero
enum E
	A
	B entero
fn crear(c Caja = {valor: 1}, alias Caja = c) Caja
	alias.valor = alias.valor + 1
	c
fn lista(xs [entero] = [1]) entero
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
`, "2\n2\n2\n2\nhola\nverdadero\nA\nfalso\n")
}

func TestInvalidDefaults(t *testing.T) {
	cases := []struct{ source, message string }{
		{"fn f(a entero = verdadero) entero a\n", "debe ser entero"},
		{"fn f(a ...entero = []) entero 0\n", "variádico"},
		{"fn f(a entero = 1, b entero) entero b\n", "obligatorio"},
		{"fn f(a entero = a) entero a\n", "no existe"},
		{"fn f(a entero = b, b entero = 1) entero a\n", "no existe"},
		{"fn f(a entero = local) entero\n\tvar local = 1\n\ta\n", "no existe"},
		{"fn f(a entero = @valor) entero a\n", "método"},
		{"fn f(a entero, b entero = 1) entero a\nfn inicio() f(b = 2)\n", "falta"},
		{"fn f(a entero = 1) entero a\nfn inicio() f(verdadero)\n", "debe ser entero"},
		{"fn f(a entero = 1) entero a\nfn inicio() f(1, a = 2)\n", "duplicado"},
		{"fn inicio(a entero = 1) imprimir(a)\n", "inicio"},
		{"fn f(a entero = ) entero a\n", "expresión"},
	}
	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			_, err := Compile("defaults.cometa", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want %q", err, tt.message)
			}
		})
	}
}
