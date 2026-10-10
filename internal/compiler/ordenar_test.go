package compiler

import (
	"strings"
	"testing"
)

func TestListSorting(t *testing.T) {
	t.Parallel()
	source := `tipo Sprite
	nombre cadena
	y decimal

tipo Tabla
	puntos [entero]

	fn ordenar_tabla()
		@puntos.ordenar(descendente = verdadero)

fn inicio()
	var numeros = [5, 3, 9, 1, 3]
	numeros.ordenar()
	imprimir(numeros)
	numeros.ordenar(verdadero)
	imprimir(numeros)
	var precios = [2.5, -1.0, 2.25]
	precios.ordenar()
	imprimir(precios)
	var nombres = ["pera", "Uva", "manzana"]
	nombres.ordenar()
	imprimir(nombres)
	var vacia [entero] = []
	vacia.ordenar()
	vacia.ordenar_por([])
	imprimir(vacia)

	var sprites = [Sprite {nombre: "arbol", y: 40}, Sprite {nombre: "heroe", y: 12.5}, Sprite {nombre: "roca", y: 40}, Sprite {nombre: "nube", y: 3}]
	var alturas [decimal] = []
	repetir (sprites) |sprite| alturas.agregar(sprite.y)
	sprites.ordenar_por(alturas)
	repetir (sprites) |sprite| imprimir(sprite.nombre)
	imprimir(alturas)

	var jugadas = ["a", "b", "c", "d"]
	var puntos = [10, 30, 10, 20]
	jugadas.ordenar_por(puntos, descendente = verdadero)
	imprimir(jugadas)
	jugadas.ordenar_por(claves = ["z", "y", "x", "w"])
	imprimir(jugadas)

	var tabla = Tabla {puntos: [1, 3, 2]}
	tabla.ordenar_tabla()
	imprimir(tabla.puntos)
	var filas = [[3, 1], [2, 0]]
	filas[0].ordenar()
	imprimir(filas)
	var alias = numeros
	alias.ordenar()
	imprimir(numeros)
`
	got, err := Compile("ordenar.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"[1, 3, 3, 5, 9]", "[9, 5, 3, 3, 1]", "[-1, 2.25, 2.5]", `["Uva", "manzana", "pera"]`, "[]",
		"nube", "heroe", "arbol", "roca", "[40, 12.5, 40, 3]",
		`["b", "d", "a", "c"]`, `["c", "a", "d", "b"]`,
		"[3, 2, 1]", "[[1, 3], [2, 0]]", "[1, 3, 3, 5, 9]",
	}, "\n") + "\n"
	runGeneratedGo(t, got, want)

	for _, tc := range []struct{ name, source, want string }{
		{"estructuras", "tipo P\n\tx entero\nfn inicio()\n\tvar l = [P {}]\n\tl.ordenar()\n", "solo admite listas de entero, decimal o cadena"},
		{"claves bool", "fn inicio()\n\tvar l = [1]\n\tl.ordenar_por([verdadero])\n", "las claves de ordenar_por deben ser una lista de entero, decimal o cadena, no [bool]"}, {"sin claves", "fn inicio()\n\tvar l = [1]\n\tl.ordenar_por()\n", `falta el parámetro "claves"`},
		{"descendente", "fn inicio()\n\tvar l = [1]\n\tl.ordenar(1)\n", "debe ser bool"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}

func TestSortByKeysOfAnotherLength(t *testing.T) {
	t.Parallel()
	source := "fn inicio()\n\tvar l = [1, 2]\n\tl.ordenar_por([1])\n"
	got, err := Compile("largo.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "ordenar_por: la lista tiene %d elementos y las claves %d") {
		t.Fatalf("falta la comprobación de longitud\n%s", got)
	}
}
