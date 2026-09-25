package compiler

import (
	"strings"
	"testing"
)

func TestVariadicAndNamedRuntime(t *testing.T) {
	runCometa(t, `tipo Contador
	valor entero
	fn siguiente() entero
		@valor = @valor + 1
		@valor
	fn sumar(base entero, valores ...entero) entero
		base + sumar(valores...)
	fn reenviar(valores ...entero) entero @sumar(valores = valores, base = 10)
enum E
	A
	B entero
fn sumar(valores ...entero) entero
	var total = 0
	repetir (valores) |valor|
		total = total + valor
	total
fn cambiar(valores ...entero)
	valores[0] = 99
fn diferencia(a entero, b entero) entero a - b
fn enumValor(e E, otros ...E) E e
fn inicio()
	imprimir(sumar())
	imprimir(sumar(1, 2, 3))
	var lista = [4, 5]
	imprimir(sumar(lista...))
	cambiar(lista...)
	imprimir(lista[0])
	var c = Contador {}
	imprimir(diferencia(b = c.siguiente(), a = c.siguiente()))
	imprimir(c.sumar(10, 1, 2))
	imprimir(c.reenviar(1, 2))
	imprimir(c.sumar(valores = [2, 3], base = 10))
	imprimir(c.sumar(base = 10))
	imprimir(diferencia(8, b = 3))
	var e = enumValor(otros = [.B(2)], e = .A)
	casos e
		.A => imprimir("A")
		_ => imprimir("error")
	imprimir(valor = sumar(valores = []))
`, "0\n6\n9\n99\n1\n13\n13\n15\n10\n5\nA\n0\n")
}

func TestInvalidCallArguments(t *testing.T) {
	cases := []struct{ name, source, message string }{
		{"variadic first", "fn f(a ...entero, b entero) entero b\n", "último"},
		{"two variadics", "fn f(a ...entero, b ...entero) entero 0\n", "último"},
		{"missing", "fn f(a entero, b entero) entero a\nfn inicio() f(b = 1)\n", "falta"},
		{"unknown", "fn f(a entero) entero a\nfn inicio() f(b = 1)\n", "no existe"},
		{"duplicate", "fn f(a entero) entero a\nfn inicio() f(a = 1, a = 2)\n", "duplicado"},
		{"positional duplicate", "fn f(a entero) entero a\nfn inicio() f(1, a = 2)\n", "duplicado"},
		{"positional after named", "fn f(a entero, b entero) entero a\nfn inicio() f(a = 1, 2)\n", "posicional"},
		{"wrong named type", "fn f(a entero) entero a\nfn inicio() f(a = verdadero)\n", "debe ser entero"},
		{"wrong element", "fn f(a ...entero) entero 0\nfn inicio() f(verdadero)\n", "debe ser entero"},
		{"wrong spread", "fn f(a ...entero) entero 0\nfn inicio() f([verdadero]...)\n", "debe ser"},
		{"spread fixed", "fn f(a [entero]) entero 0\nfn inicio() f([1]...)\n", "expansión"},
		{"spread not last", "fn f(a ...entero) entero 0\nfn inicio() f([1]..., 2)\n", "último"},
		{"mixed spread", "fn f(a ...entero) entero 0\nfn inicio() f(1, [2]...)\n", "mezcla"},
		{"named variadic scalar", "fn f(a ...entero) entero 0\nfn inicio() f(a = 1)\n", "debe ser [entero]"},
		{"named variadic duplicate", "fn f(a ...entero) entero 0\nfn inicio() f(1, a = [2])\n", "duplicado"},
		{"missing fixed variadic", "fn f(a entero, b ...entero) entero a\nfn inicio() f()\n", "falta"},
		{"enum named", "enum E\n\tA entero\nfn inicio()\n\tE.A(valor = 1)\n", "payload"},
		{"context enum spread", "enum E\n\tA entero\nfn inicio()\n\tvar e E = .A([1]...)\n", "payload"},
		{"print spread", "fn inicio() imprimir([1]...)\n", "sin expansión"},
		{"entry", "fn inicio(a ...entero) imprimir(1)\n", "inicio"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("calls.cometa", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want %q", err, tt.message)
			}
		})
	}
}

func TestNamedReceiverOrderAndVariadicReferences(t *testing.T) {
	runCometa(t, `tipo Caja
	valor entero
	fn poner(a entero, b entero)
		@valor = a * 10 + b
fn receptor(c Caja) Caja
	imprimir("receptor")
	c
fn argumento(n entero) entero
	imprimir(n)
	n
fn referencias(cs ...Caja,) Caja
	cs[0].valor = 7
	cs[0]
fn listas(xs ...[entero]) entero xs[0][0]
fn cambiar(xs ...entero)
	xs[0] = 9
fn inicio()
	var c = Caja {}
	receptor(c).poner(b = argumento(2), a = argumento(1))
	imprimir(c.valor)
	var alias = referencias(c)
	imprimir(alias.valor)
	var otro = referencias(cs = [{valor: 3}])
	imprimir(otro.valor)
	imprimir(listas([4], [5]))
	var xs = [1]
	cambiar(xs[0])
	imprimir(xs[0])
	cambiar(xs = xs...)
	imprimir(xs[0])
`, "receptor\n2\n1\n12\n7\n7\n4\n1\n9\n")
}
