package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestWrapperExample(t *testing.T) {
	source, err := os.ReadFile("../../examples/errores.cometa")
	if err != nil {
		t.Fatal(err)
	}
	runCometa(t, string(source), "Ana\nInvitado\nno se pudo leer el usuario\n")
}

func TestUnusedOptionalLocalsCompile(t *testing.T) {
	source := `tipo Usuario
	nombre cadena
	mascota cadena?
fn nulable()
	var usuario = Usuario {mascota: "hola"}
	var mascota = usuario.mascota
	var talvez_usuario Usuario? = .Alguno({})
	var ausente Usuario? = .Ninguno
	var implicito Usuario? = Usuario {}
	var copia = mascota
`
	if _, err := Compile("opcionales.cometa", []byte(source)); err != nil {
		t.Fatal(err)
	}
}

func TestWrapperContextsAndLoops(t *testing.T) {
	runCometa(t, `tipo Caja
	valor entero?
	pendiente (entero!)?
	lista [entero?]
fn envolver(n entero) entero! n
fn aceptar(n entero? = .Ninguno, valores ...entero?) entero
	var suma = n o 0
	repetir (valores) |v|
		suma = suma + (v o 0)
	suma
fn elegir(n entero?) entero
	si n |v|
		retornar v
	sino
		retornar 9
fn condicional(n entero?) entero
	var valor = si n |v| v sino 8
	valor
fn prueba(n entero?) entero
	var valor = si n |v|
		v
	sino
		7
	valor
fn operar() entero!
	var total = 0
	repetir (0..4) |i|
		casos envolver(i) |n|
			.Ok =>
				si n == 1 continuar
				si n == 3 romper
				total = total + n
			.Error => retornar .Error(n)
	total
fn inicio()
	var caja = Caja {valor: 2, lista: [1, .Ninguno]}
	imprimir(caja.valor o 0)
	caja.valor = 3
	caja.lista[1] = 4
	imprimir(aceptar(valores = caja.lista, n = caja.valor))
	imprimir(aceptar())
	imprimir(elegir(.Ninguno))
	imprimir(condicional(5))
	imprimir(prueba(.Ninguno))
	imprimir(operar() capturar 99)
	casos caja.pendiente
		.Ninguno => imprimir("pendiente")
		.Alguno => imprimir("resuelto")
`, "2\n8\n0\n9\n5\n7\n2\npendiente\n")
}

func TestWrapperShortCircuitAndReferenceCapture(t *testing.T) {
	runCometa(t, `tipo Caja
	n entero
	fn leer(ignorado entero, valor entero) entero @n + valor
fn cambiar(c Caja) entero
	c.n = 10
	1
fn falla() bool! .Error("falló")
fn logica() bool!
	var a = falso && intentar falla()
	var b = verdadero || intentar falla()
	a || b
fn vacio() !bool .Error(falso)
fn delegar() !bool
	intentar vacio()
	.Ok
fn inicio()
	var c = Caja {n: 2}
	imprimir(c.leer(valor = c.n, ignorado = cambiar(c)))
	imprimir(logica() capturar falso)
	delegar() capturar |error| imprimir(error)
`, "12\ntrue\nfalse\n")
}

func TestReturnsInsideValueConstruction(t *testing.T) {
	runCometa(t, `fn aceptar(n entero) entero n
fn llamada() entero!
	var valor = aceptar(retornar .Error("llamada"))
	valor
fn constructor() entero!
	var valor entero! = .Ok(retornar .Error("constructor"))
	valor
fn asignacion() entero!
	var numero = 0
	numero = retornar .Error("asignacion")
	numero
fn ramas(b bool) entero!
	si b
		retornar 3
	sino
		retornar .Error("ramas")
	99
fn inicio()
	imprimir(llamada() capturar |e|
		imprimir(e)
		0
	)
	constructor() capturar |e|
		imprimir(e)
		0
	asignacion() capturar |e|
		imprimir(e)
		0
	imprimir(ramas(verdadero) capturar 0)
`, "llamada\n0\nconstructor\nasignacion\n3\n")
}

func TestWrappersRuntime(t *testing.T) {
	runCometa(t, `tipo Direccion
	ciudad cadena
tipo Usuario
	nombre cadena
	direccion Direccion
	referido Usuario?
enum Problema
	Ausente
	Detalle cadena
fn buscar(existe bool) Usuario?
	si existe
		Usuario {nombre: "Ana"}
	sino
		.Ninguno
fn cargar(existe bool) Usuario!Problema
	si existe retornar Usuario {nombre: "Luis"}
	.Error(.Detalle("falló"))
fn nombre(existe bool) cadena!Problema
	var usuario = intentar cargar(existe)
	usuario.nombre
fn nombre_opcional(existe bool) cadena?
	var usuario = intentar buscar(existe)
	usuario.nombre
fn guardar() !
	.Ok
fn guardar_otro() !
	intentar guardar()
	.Ok
fn inicio()
	var primero = Usuario {}
	var segundo = Usuario {}
	primero.direccion.ciudad = "SJ"
	imprimir(segundo.direccion.ciudad)
	si buscar(verdadero) |u|
		imprimir(u.nombre)
	sino imprimir("ausente")
	var invitado = buscar(falso) o Usuario {nombre: "Invitado"}
	imprimir(invitado.nombre)
	imprimir(nombre_opcional(verdadero) o "nadie")
	imprimir(nombre_opcional(falso) o "nadie")
	casos nombre(verdadero) |v|
		.Ok => imprimir(v)
		.Error => imprimir("error")
	var recuperado = nombre(falso) capturar |e|
		casos e |detalle|
			.Ausente => "ausente"
			.Detalle => detalle
	imprimir(recuperado)
	guardar_otro() capturar |e| imprimir(e)
	var anidado Usuario?! = .Ok(.Ninguno)
	casos anidado |valor|
		.Ok => imprimir((valor o Usuario {nombre: "vacío"}).nombre)
		.Error => imprimir(valor)
`, "\nAna\nInvitado\nAna\nnadie\nLuis\nfalló\nvacío\n")
}

func TestWrapperPropagationOrderAndLaziness(t *testing.T) {
	runCometa(t, `tipo Contador
	n entero
fn paso(c Contador, n entero) entero
	c.n = c.n * 10 + n
	n
fn falla(c Contador) entero!
	paso(c, 2)
	.Error("fallo")
fn suma(a entero, b entero) entero a + b
fn operacion(c Contador) entero!
	var valor = suma(b = paso(c, 1), a = intentar falla(c))
	paso(c, 3)
	valor
fn salida(c Contador) entero!
	var valor = casos verdadero_opcional()
		.Alguno => intentar falla(c)
		.Ninguno => 0
	valor
fn verdadero_opcional() bool? verdadero
fn recuperar(c Contador) entero!
	var valor = falla(c) capturar |e|
		retornar .Error(e)
	paso(c, 9)
	valor
fn inicio()
	var c = Contador {}
	imprimir(operacion(c) capturar 42)
	imprimir(c.n)
	var presente entero? = 7
	imprimir(presente o paso(c, 3))
	var correcto entero! = 8
	imprimir(correcto capturar paso(c, 4))
	imprimir(c.n)
	imprimir(salida(c) capturar 5)
	imprimir(c.n)
	imprimir(recuperar(c) capturar 6)
	imprimir(c.n)
`, "42\n12\n7\n8\n12\n5\n122\n6\n1222\n")
}

func TestWrapperDiagnostics(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"unchecked field", "tipo U\n\tn entero\nfn f(u U?) entero u.n\n", "no tiene miembros"},
		{"unchecked argument", "fn f(n entero) entero n\nfn g(n entero?) entero f(n)\n", "entero"},
		{"nonexhaustive", "fn f(n entero?) entero\n\tcasos n |v|\n\t\t.Alguno => v\n", "Ninguno"},
		{"discarded", "fn f() entero! 1\nfn inicio()\n\tf()\n", "descartar"},
		{"unread", "fn inicio()\n\tvar r entero! = 1\n", "debe usarse"},
		{"default result", "tipo C\n\tr entero!\nfn f() C C {}\n", "r requiere"},
		{"nested default result", "tipo C\n\tr entero!\ntipo P\n\tc C\nfn f() P P {}\n", "c.r requiere"},
		{"default enum", "enum E\n\tA\ntipo C\n\te E\nfn f() C C {}\n", "e requiere"},
		{"cycle", "tipo U\n\tu U\n", "ciclo"},
		{"indirect cycle", "tipo A\n\tb B\ntipo B\n\ta A\n", "ciclo"},
		{"incompatible try", "fn f(n entero?) entero! intentar n\n", "propagar"},
		{"typed error mismatch", "fn f(n entero!bool) entero! intentar n\n", "propagar"},
		{"default try", "fn f(n entero!, otro entero = intentar n) entero otro\n", "predeterminados"},
		{"default return", "fn f(n entero = retornar 1) entero n\n", "predeterminados"},
		{"nested implicit", "fn f() entero?! 1\n", "produce"},
		{"bare result return", "fn f() !\n\tretornar\n", "requiere"},
		{"wrapper comparison", "fn f(n entero?) bool n == n\n", "no acepta"},
		{"wrapper indexing", "fn f(n [entero]?) entero n[0]\n", "solo se pueden indexar"},
		{"wrapper arithmetic", "fn f(n entero!) entero n + 1\n", "no acepta"},
		{"bare absence", "fn inicio()\n\tvar n = .Ninguno\n", "inferir"},
		{"unit parentheses", "fn f() ! .Ok()\n", "no acepta"},
		{"unit payload binding", "fn f(n !)\n\tcasos n |v|\n\t\t.Ok => imprimir(v)\n\t\t.Error => imprimir(v)\n", "no existe"},
		{"wrong recovery type", "fn f(n entero!) entero n capturar \"no\"\n", "recuperación"},
		{"branch unused result", "fn inicio()\n\tsi verdadero\n\t\tvar n entero! = 1\n\tsino\n\t\tvar n entero! = 2\n\t\tcasos n\n\t\t\t_ => imprimir(1)\n", "debe usarse"},
		{"optional binding scope", "fn f(n entero?) entero\n\tsi n |v| imprimir(v)\n\tv\n", "no existe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("wrapper.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v", tc.want, err)
			}
		})
	}
}
