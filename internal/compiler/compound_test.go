package compiler

import (
	"strings"
	"testing"
)

func TestCompoundAssignmentRuns(t *testing.T) {
	source := `tipo Contador
	total entero
	pos decimal
	fn sumar(n entero)
		@total += n
		@pos *= 2

fn inicio()
	var x = 10
	x += 5
	x -= 3
	x *= 2
	x /= 5
	x %= 3
	imprimir(x)
	var lista = [1, 2, 3]
	var i = 1
	lista[i + 1] += 10
	lista[i] -= 1
	imprimir(lista)
	var texto = "ho"
	texto += "la"
	imprimir(texto)
	var d = 1.5
	d += 1
	d *= 2
	imprimir(d)
	var c = Contador {total: 1, pos: 1.5}
	c.sumar(4)
	c.total += 1
	imprimir(c.total)
	imprimir(c.pos)
	var y = 2
	y *= 3 + 4
	imprimir(y)
`
	got, err := Compile("compuesta.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "1\n[1, 1, 13]\nhola\n5\n6\n3\n14\n")
}

func TestCompoundAssignmentMapEntries(t *testing.T) {
	runCometa(t, `tipo Marcador
	puntos [cadena: entero]
fn inicio()
	var conteo [cadena: entero] = [:]
	repetir (["el", "sol", "el"]) |palabra|
		conteo[palabra] += 1
	imprimir(conteo["el"] o -1)
	imprimir(conteo["sol"] o -1)
	conteo["el"] *= 10
	conteo["nada"] *= 10
	imprimir(conteo["el"] o -1)
	imprimir(conteo["nada"] o -1)
	var pesos = [1: 1.5]
	pesos[1] += 1
	pesos[2] -= 0.5
	imprimir(pesos[1] o -1)
	imprimir(pesos[2] o -1)
	var textos [bool: cadena] = [:]
	textos[verdadero] += "ho"
	textos[verdadero] += "la"
	imprimir(textos[verdadero] o "?")
	var m = Marcador {}
	m.puntos["ana"] += 5
	m.puntos["ana"] += 2
	imprimir(m.puntos["ana"] o -1)
`, "2\n1\n20\n0\n2.5\n-0.5\nhola\n7\n")
}

func TestCompoundAssignmentErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"call target", "fn f() entero 1\nfn inicio()\n\tvar l = [1, 2]\n\tl[f()] += 1\n", "no puede contener llamadas"},
		{"map entry of bool", "fn inicio()\n\tvar m = [\"a\": verdadero]\n\tm[\"a\"] += verdadero\n", "solo funciona con valores entero, decimal o cadena"},
		{"map entry type mismatch", "fn inicio()\n\tvar m = [\"a\": 1]\n\tm[\"a\"] += 1.5\n", "no se puede asignar"},
		{"type mismatch", "fn inicio()\n\tvar x = 1\n\tx += 1.5\n", "no se puede asignar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("se esperaba %q, se obtuvo %v", tc.want, err)
			}
		})
	}
}
