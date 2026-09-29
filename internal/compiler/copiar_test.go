package compiler

import (
	"strings"
	"testing"
)

func TestStructCopiar(t *testing.T) {
	source := `tipo Arma
	dano entero

tipo Jugador
	nombre cadena
	vida entero
	arma Arma
	items [cadena]

tipo Caja<T>
	valor T

tipo Propio
	x entero
	pub fn copiar() entero 99

fn inicio()
	var a = Jugador {nombre: "Ana", vida: 10, arma: {dano: 5}, items: ["espada"]}
	var b = a.copiar()
	b.vida = 3
	b.nombre = "Bea"
	imprimir("${a.nombre} ${a.vida} | ${b.nombre} ${b.vida}")
	// La copia es superficial: los objetos anidados se comparten.
	b.arma.dano = 50
	imprimir(a.arma.dano)
	// Las listas comparten sus elementos pero agregar a una copia de la lista no afecta a la otra.
	b.items.agregar("escudo")
	imprimir(a.items.longitud())
	var caja = Caja<entero> {valor: 1}
	var otra = caja.copiar()
	otra.valor = 2
	imprimir("${caja.valor} ${otra.valor}")
	var p = Propio {x: 1}
	imprimir(p.copiar())
	var c = a.copiar().copiar()
	c.vida = 77
	imprimir(a.vida)
`
	got, err := Compile("copiar.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "Ana 10 | Bea 3\n50\n1\n1 2\n99\n10\n")
}

func TestStructCopiarErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"args", "tipo A\n\tx entero\nfn inicio()\n\tvar a = A {}\n\tvar b = a.copiar(1)\n", "no acepta argumentos"},
		{"library type", "usar std/mate\nfn inicio()\n\tvar v = mate.Vec2 {}\n\tvar w = v.copiar()\n", "copiar"},
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
