package compiler

import (
	"strings"
	"testing"
)

func TestInfiniteRepeatEndsFunction(t *testing.T) {
	t.Parallel()
	source := `fn primer_multiplo(numero entero, desde entero) entero
	var candidato = desde
	repetir
		si candidato % numero == 0
			retornar candidato
		candidato += 1

fn buscar(lista [entero], valor entero) entero?
	var i = 0
	repetir
		si i >= lista.longitud()
			retornar .Ninguno
		si lista[i] == valor
			retornar i
		i += 1

fn signo(n entero) cadena
	si n < 0
		"negativo"
	sino
		var paso = 0
		repetir
			paso += 1
			repetir (0..3) |i|
				si i == 1 romper
			si paso == 2
				retornar "positivo tras ${paso}"

tipo Contador
	total entero

	fn hasta(tope entero) entero
		repetir
			@total += 1
			si @total >= tope retornar @total

fn inicio()
	imprimir(primer_multiplo(7, 30))
	imprimir(buscar([4, 5, 6], 6) o -1)
	imprimir(buscar([4, 5, 6], 9) o -1)
	imprimir(signo(-1))
	imprimir(signo(1))
	var c = Contador {}
	imprimir(c.hasta(3))
`
	got, err := Compile("infinito.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "35\n2\n-1\nnegativo\npositivo tras 2\n3\n")

	for _, tc := range []struct{ name, body string }{
		{"romper", "\trepetir\n\t\tsi n > 0 romper\n"},
		{"romper en casos", "\trepetir\n\t\tcasos n\n\t\t\t0 => romper\n\t\t\t_ => retornar n\n"},
		{"mientras", "\tmientras n > 0\n\t\tretornar n\n"},
		{"rango", "\trepetir (0..n) |i|\n\t\tretornar i\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte("fn f(n entero) entero\n"+tc.body+"\nfn inicio()\n\timprimir(f(1))\n"))
			if err == nil || !strings.Contains(err.Error(), "en todos los caminos") {
				t.Fatalf("se esperaba el error de caminos; got %v", err)
			}
		})
	}
}
