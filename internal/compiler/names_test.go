package compiler

import (
	"strings"
	"testing"
)

// Cometa names are case-sensitive, but every top-level name is capitalized in
// Go: a function and a type that differ only in their first letter must not
// end up as the same Go identifier.
func TestFunctionAndTypeDifferingOnlyInCase(t *testing.T) {
	t.Parallel()
	runCometa(t, `tipo Punto
	x entero
	y entero

fn punto(x entero, y entero) Punto
	Punto {x: x, y: y}

fn Suma(a entero, b entero) entero
	a + b

fn suma(a entero) entero
	a + 1

fn inicio()
	var p = punto(1, 2)
	imprimir(p.x + p.y)
	imprimir(p)
	imprimir(Suma(1, 2))
	imprimir(suma(1))
`, "3\nPunto {x: 1, y: 2}\n3\n2\n")
}

// Locals keep their spelling in Go, so the names the generated code itself
// relies on (builtins and imported packages) have to be kept apart.
func TestLocalsNamedLikeGoIdentifiers(t *testing.T) {
	t.Parallel()
	runCometa(t, `fn contar(len entero, string cadena, append [entero]) entero
	var copy = append.copiar()
	copy.agregar(len)
	var make = "${string}:${copy.longitud()}:${copy}"
	imprimir(make)
	var int64 = 2.5
	var float64 = entero(int64)
	var fmt = [1: "uno"]
	var strings = string.mayusculas()
	var strconv = "7".a_entero() o 0
	var sort = [3, 1, 2]
	sort.ordenar()
	var nil = fmt[1] o "nada"
	var true = sort.contiene(3)
	var reflect = 0
	repetir (string) |rune, int|
		reflect += int
	var panic = float64 + strconv + reflect
	var any = 1.5 * decimal(panic)
	var negado = false_o_no(true)
	fmt.eliminar(1)
	fmt.vaciar()
	imprimir(strings)
	imprimir(nil)
	imprimir(true)
	imprimir(sort)
	imprimir(any)
	imprimir(negado)
	// Shadowing these is harmless: the generated code never mentions them.
	var error = 1
	var min = 2
	var max = 3
	var new = 4
	var color = 5
	panic + error + min + max + new + color

fn false_o_no(false bool) bool
	!false

fn inicio()
	imprimir(contar(4, "ab", [1, 2]))
`, "ab:3:[1, 2, 4]\nAB\nuno\nverdadero\n[1, 2, 3]\n15\nfalso\n25\n")
}

func TestMembersAndTypesDifferingOnlyInCase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, want string }{
		{"campo y método", "tipo Caja\n\tvalor entero\n\tfn Valor() entero\n\t\t@valor\nfn inicio()\n\timprimir(1)\n", `"valor" y "Valor" solo se distinguen por la mayúscula inicial`},
		{"dos campos", "tipo Caja\n\tvalor entero\n\tValor entero\nfn inicio()\n\timprimir(1)\n", `"valor" y "Valor" solo se distinguen por la mayúscula inicial`},
		{"dos tipos", "tipo caja\n\tvalor entero\nenum Caja\n\tA\nfn inicio()\n\timprimir(1)\n", `"caja" y "Caja" solo se distinguen por la mayúscula inicial`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("nombres.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("se esperaba un error con %q, se obtuvo %v", tc.want, err)
			}
		})
	}
}
