package compiler

import (
 "strings"
 "testing"
)

func TestScopeCleanup(t *testing.T) {
 runHacha(t, `tipo Ambito
	n entero
	fn entrar() imprimir(@n)
	fn salir() imprimir(0 - @n)
fn crear(n entero) Ambito
	imprimir(100 + n)
	Ambito {n: n}
fn prueba() entero
	con crear(1)
		repetir (0..3) |i|
			con crear(2)
				si i == 0 continuar
				romper
		con crear(3)
			retornar 9
fn fallar() entero! .Error("fallo")
fn propagar() entero!
	con crear(4)
		var n = intentar fallar()
		retornar n
fn inicio()
	imprimir(prueba())
	imprimir(propagar() capturar |e|
		imprimir(e)
		0
	)
`, "101\n1\n102\n2\n-2\n102\n2\n-2\n103\n3\n-3\n-1\n9\n104\n4\n-4\nfallo\n0\n")
}

func TestScopeProtocol(t *testing.T) {
 for _, source := range []string{
 "fn inicio()\n\tcon 1\n\t\timprimir(1)\n",
 "tipo A\n\tfn entrar(n entero = 0) retornar\n\tfn salir() retornar\nfn inicio()\n\tcon A {}\n\t\timprimir(1)\n",
 } {
  _, err := Compile("scope.hacha", []byte(source))
  if err == nil || !strings.Contains(err.Error(), "con requiere") { t.Fatalf("unexpected error: %v",err) }
 }
}

func TestScopeInterfacesAndReturnSnapshot(t *testing.T) {
 runHacha(t, `interfaz Ambito
	fn entrar()
	fn salir()
tipo Estado
	n entero
	fn entrar() @n = @n + 1
	fn salir() @n = @n + 10
tipo Exterior
	Estado
fn leer<T Ambito>(valor T, estado Estado) entero
	con valor
		retornar estado.n
fn inicio()
	var estado = Estado {}
	var ambito Ambito = Exterior {Estado: estado}
	imprimir(leer(ambito, estado))
	imprimir(estado.n)
	con estado
		repetir
			romper
		imprimir(estado.n)
	imprimir(estado.n)
`, "1\n11\n12\n22\n")
}

func TestScopeModuleBinding(t *testing.T) {
 entry,loader:=memoryProject(t,map[string]string{
 "main.hacha":"usar util\nfn inicio()\n\tcon util.crear()\n\t\tvar n = 7\n\t\timprimir(n)\n\tvar n = 8\n\timprimir(n)\n",
 "util.hacha":"tipo A\n\tfn entrar() imprimir(1)\n\tfn salir() imprimir(2)\nfn crear() A A {}\n",
 })
 generated,err:=CompileProject(entry,loader);if err!=nil {t.Fatal(err)};runGeneratedGo(t,generated,"1\n7\n2\n8\n")
}
