package compiler

import (
	"strings"
	"testing"
)

func TestListMethodsRuntime(t *testing.T) {
	runCometa(t, `fn inicio()
	var valores = [10, 20, 20]
	imprimir(valores.longitud())
	imprimir(valores.esta_vacia())
	imprimir(valores.contiene(valor = 20))
	imprimir(valores.buscar_indice(20) o -1)
	imprimir(valores.buscar_indice(99) o -1)
	imprimir(valores.obtener(1) o -1)
	imprimir(valores.obtener(99) o -1)
	imprimir(valores.primero() o -1)
	imprimir(valores.ultimo() o -1)
	valores.agregar(30)
	valores.extender([40, 50])
	imprimir(valores.insertar(valor = 5, indice = 0))
	imprimir(valores.insertar(7, 60))
	imprimir(valores.insertar(-1, 0))
	imprimir(valores.eliminar(2))
	imprimir(valores.eliminar(20))
	valores.invertir()
	var copia = valores.copiar()
	copia[0] = 999
	imprimir(valores[0])
	imprimir(copia[0])
	imprimir(valores.longitud())
	var vacia [entero] = []
	imprimir(vacia.primero() o -1)
`, "3\nfalso\nverdadero\n1\n-1\n20\n-1\n10\n20\nverdadero\nverdadero\nfalso\nverdadero\nfalso\n60\n999\n7\n-1\n")
}

func TestListMutationReceiversAliasingAndOrder(t *testing.T) {
	runCometa(t, `tipo Objeto
	valor entero
tipo Caja
	valores [entero]
fn sumar_local(valores [entero]) entero
	valores.agregar(9)
	valores.longitud()
fn inicio()
	var caja = Caja {valores: [1, 2]}
	var alias = caja.valores
	caja.valores.agregar(3)
	imprimir(alias.longitud())
	imprimir(caja.valores.longitud())
	imprimir(sumar_local(caja.valores))
	imprimir(caja.valores.longitud())
	var matrices = [[1], [2]]
	matrices[0].agregar(4)
	imprimir(matrices[0].ultimo() o -1)
	matrices[0].extender(matrices[0])
	imprimir(matrices[0].longitud())
	var base = [1, 2, 3]
	var compartida = base
	base.eliminar(0)
	imprimir(compartida[0])
	imprimir(compartida[2])
	var objeto = Objeto {valor: 1}
	var objetos = [objeto]
	var objetos_copia = objetos.copiar()
	objetos_copia[0].valor = 8
	imprimir(objetos[0].valor)
`, "2\n3\n4\n3\n4\n4\n2\n3\n8\n")
}

func TestGenericListMethodsRuntime(t *testing.T) {
	runCometa(t, `fn agregar_y_primero<T>(valores [T], valor T) T?
	valores.agregar(valor)
	valores.primero()
fn inicio()
	imprimir(agregar_y_primero([7], 8) o -1)
`, "7\n")
}

func TestListConcatenationRuntime(t *testing.T) {
	runCometa(t, `var globales = [1] + [2, 3]
fn unir<T>(a [T], b [T]) [T] a + b
fn inicio()
	var a = [1, 2]
	var b = [3]
	var c = a + b
	c[0] = 99
	imprimir(a[0])
	imprimir(c)
	var vacia [entero] = []
	var d = a + vacia
	d[1] = 7
	imprimir(a[1])
	a += [4]
	imprimir(a.longitud())
	imprimir(unir(["x"], ["y", "z"]))
	imprimir(globales.longitud())
	imprimir(([[1]] + [[2]]).longitud())
`, "1\n[99, 2, 3]\n2\n3\n[\"x\", \"y\", \"z\"]\n3\n2\n")
}

func TestListMutationWaitsForPropagatingArguments(t *testing.T) {
	runCometa(t, `tipo Caja
	valores [entero]
fn falla() entero! .Error("fallo")
fn cambiar(c Caja) !
	c.valores.agregar(intentar falla())
	.Ok
fn inicio()
	var c = Caja {valores: [1]}
	cambiar(c) atrapar |e| imprimir(e)
	imprimir(c.valores.longitud())
`, "fallo\n1\n")
}

func TestListMethodDiagnostics(t *testing.T) {
	tests := []struct {
		name, source, want string
	}{
		{"wrong element", "fn inicio()\n\tvar xs = [1]\n\txs.agregar(verdadero)\n", "debe ser entero"},
		{"unknown named argument", "fn inicio()\n\tvar xs = [1]\n\txs.agregar(otro = 2)\n", `el parámetro "otro" no existe`},
		{"temporary mutation", "fn lista() [entero] [1]\nfn inicio()\n\tlista().agregar(2)\n", "requiere una lista asignable"},
		{"nested list equality", "fn inicio()\n\tvar xs = [[1]]\n\timprimir(xs.contiene([1]))\n", "no admite elementos de tipo [entero]"},
		{"enum equality", "enum E\n\tA\nfn inicio()\n\tvar xs = [E.A]\n\timprimir(xs.contiene(E.A))\n", "no admite elementos de tipo E"},
		{"interface equality", "interfaz I\nfn inicio()\n\tvar xs [I] = [1]\n\timprimir(xs.contiene(1))\n", "no admite elementos de tipo I"},
		{"optional equality", "fn inicio()\n\tvar xs [entero?] = [1]\n\timprimir(xs.contiene(.Ninguno))\n", "no admite elementos de tipo entero?"},
		{"concat mismatch", "fn inicio()\n\tvar xs = [1] + [\"a\"]\n", `no acepta [entero] y [cadena]`},
		{"concat element", "fn inicio()\n\tvar xs = [1] + 2\n", "usa agregar"},
		{"list minus", "fn inicio()\n\tvar xs = [1] - [1]\n", `no acepta [entero] y [entero]`},
		{"type parameter equality", "fn buscar<T>(xs [T], valor T) bool xs.contiene(valor)\nfn inicio() imprimir(verdadero)\n", "no admite elementos de tipo T"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Compile("listas.cometa", []byte(test.source))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}
