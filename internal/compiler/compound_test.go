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

func TestCompoundAssignmentErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"call target", "fn f() entero 1\nfn inicio()\n\tvar l = [1, 2]\n\tl[f()] += 1\n", "no puede contener llamadas"},
		{"map entry", "fn inicio()\n\tvar m = [\"a\": 1]\n\tm[\"a\"] += 1\n", "no funciona con entradas de un mapa"},
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
