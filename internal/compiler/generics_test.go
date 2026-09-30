package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInterfacesGenericsExample(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "programas", "interfaces_genericos.cometa"))
	if err != nil {
		t.Fatal(err)
	}
	runCometa(t, string(source), "Ana\nAna\n42\nAna\nAna\n7\notro valor\nLuis\n")
}

func TestInterfacesAndGenericsRuntime(t *testing.T) {
	source := `interfaz Cualquiera
interfaz Describible
	fn describir(prefijo cadena) cadena
interfaz Nombrado
	Describible
tipo Usuario
	nombre cadena
	pub fn describir(texto cadena = "predeterminado") cadena texto
tipo Caja<T>
	valor T
	pub fn obtener() T @valor
enum Evento<T>
	Vacio
	Dato T
interfaz Proveedor<T>
	fn obtener() T
fn identidad<T>(valor T) T valor
fn leer<T, F Proveedor<T>>(fuente F) T fuente.obtener()
fn nombre<T Describible>(valor T) cadena valor.describir(prefijo = "interfaz")
fn primero<T>(valores [T]) T? valores[0]
fn inicio()
	var u = Usuario {nombre: "Ana"}
	var i Nombrado = u
	imprimir(i.describir(prefijo = "hola"))
	imprimir(u.describir())
	imprimir(nombre(u))
	var caja = Caja<Usuario> {valor: u}
	imprimir(caja.obtener().nombre)
	imprimir(leer<Usuario, Caja<Usuario>>(caja).nombre)
	imprimir(identidad(3))
	imprimir(identidad<entero>(4))
	var presente = primero([5])
	imprimir(presente o 0)
	var evento = Evento<Usuario>.Dato(u)
	casos evento |p|
		Evento<Usuario>.Dato => p.nombre = "Luis"
		_ => imprimir("vacio")
	var valor Cualquiera = u
	var convertido = valor como Usuario
	si convertido |p| imprimir(p.nombre)
	var no = valor como entero
	imprimir(no o 9)
	var descripcion = valor como Describible
	si descripcion |p| imprimir(p.describir("ok"))
	casos valor |p|
		Describible => imprimir(p.describir("primero"))
		Usuario => imprimir("segundo")
		_ => imprimir("otro")
	var mixtos [Cualquiera] = [1, "dos", u, [3]]
	repetir (mixtos) |v|
		casos v |p|
			entero => imprimir(p)
			cadena => imprimir(p)
			Usuario => imprimir(p.nombre)
			[entero] => imprimir(p[0])
			_ => imprimir("otro")
`
	generated := runCometa(t, source, "hola\npredeterminado\ninterfaz\nAna\nAna\n3\n4\n5\nLuis\n9\nok\nprimero\n1\ndos\nLuis\n3\n")
	for _, part := range []string{"interface {", "type Caja[_T_T any]", "*Caja[*Usuario]", "_cometa_default_Describir"} {
		if !strings.Contains(string(generated), part) {
			t.Errorf("missing %q", part)
		}
	}
}

func TestGenericAndInterfaceErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"expanding type", "tipo A<T>\n\thijos [A<[T]>]\n", "expande"},
		{"expanding function", "fn f<T>(v T)\n\tf<[T]>([v])\n", "expande"},
		{"generic required cycle", "tipo Caja<T>\n\tvalor T\ntipo A\n\tcaja Caja<A>\n", "ciclo de campos"},
		{"generic invariance", "interfaz I\ntipo Caja<T>\n\tvalor T\nfn f(v Caja<I>) imprimir(v)\nfn inicio() f(Caja<entero> {valor: 1})\n", "se esperaba Caja<I>"},
		{"no parameter equality", "fn f<T>(a T, b T) bool a == b\n", "no acepta"},
		{"method type parameters", "tipo A\n\tpub fn f<T>(v T) T v\n", "métodos no pueden"},
		{"unused generic constraint", "interfaz I\n\tfn f()\ntipo Caja<T I>\n\tvalor T\nfn inutil<T>(v Caja<T>) imprimir(v)\n", "no satisface"},
		{"too few explicit types", "fn f<T, U>(t T, u U) T t\nfn inicio() imprimir(f<entero>(1, 2))\n", "requiere 2"},
		{"no type inference", "tipo Caja<T>\n\tvalor T\nfn inicio()\n\tvar c = Caja {valor: 1}\n", "requiere 1"},
		{"nested required generic", "tipo Caja<T>\n\tvalor T\ntipo Externa<T>\n\tcaja Caja<T>\nfn inicio()\n\tvar c = Externa<entero> {}\n", "caja.valor"},
		{"wrapper invariant", "interfaz I\nfn f(v entero?) I? v\n", "produce entero?"},
		{"argument names", "interfaz I\n\tfn f(n entero)\ninterfaz J\n\tfn f(x entero)\ninterfaz K\n\tI\n\tJ\n", "redeclaración"},
		{"signature conflict", "interfaz I\n\tfn f(n entero)\ninterfaz J\n\tfn f(n cadena)\ninterfaz K\n\tI\n\tJ\n", "incompatibles"},
		{"missing interface argument", "interfaz I\n\tfn f(n entero)\ntipo A\n\tpub fn f(n entero = 1) imprimir(n)\nfn inicio()\n\tvar a I = A {}\n\ta.f()\n", "falta el parámetro"},
		{"missing method", "interfaz I\n\tfn f() entero\ntipo A\n\tn entero\nfn inicio()\n\tvar i I = A {}\n", "se esperaba I"},
		{"wrong return", "interfaz I\n\tfn f() entero\ntipo A\n\tpub fn f() cadena \"a\"\nfn inicio()\n\tvar i I = A {}\n", "se esperaba I"},
		{"interface defaults", "interfaz I\n\tfn f(n entero = 1)\n", "no admite valores predeterminados"},
		{"composition cycle", "interfaz A\n\tB\ninterfaz B\n\tA\n", "ciclo"},
		{"required interface", "interfaz I\ntipo A\n\ti I\nfn inicio()\n\tvar a = A {}\n", "inicialización explícita"},
		{"required parameter", "tipo Caja<T>\n\tvalor T\nfn inicio()\n\tvar c = Caja<entero> {}\n", "inicialización explícita"},
		{"unused invalid body", "fn f<T>(v T) entero v + 1\n", "no acepta"},
		{"constraint", "interfaz I\n\tfn f()\nfn consumir<T I>(v T)\n\tv.f()\nfn inicio() consumir(1)\n", "no satisface"},
		{"inference conflict", "fn f<T>(a T, b T) T a\nfn inicio() imprimir(f(1, verdadero))\n", "incompatibles"},
		{"no return inference", "fn f<T>() T? .Ninguno\nfn inicio()\n\tvar v entero? = f()\n", "no se puede inferir"},
		{"invariant list", "interfaz I\nfn f(v [I]) imprimir(v)\nfn inicio()\n\tvar ns = [1]\n\tf(ns)\n", "debe ser"},
		{"interface equality", "interfaz I\nfn f(a I, b I) bool a == b\n", "no acepta"},
		{"type match fallback", "interfaz I\nfn f(a I)\n\tcasos a\n\t\tentero => imprimir(1)\n", "rama '_'"},
		{"duplicate pattern", "interfaz I\nfn f(a I)\n\tcasos a\n\t\tentero => imprimir(1)\n\t\tentero => imprimir(2)\n\t\t_ => imprimir(3)\n", "duplicado"},
		{"impossible assertion", "interfaz I\n\tfn f()\nfn f(a I) entero? a como entero\n", "no puede implementar"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("error.cometa", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestGenericInferenceDefaultsAndWrappers(t *testing.T) {
	source := `interfaz Todo
interfaz Primero
	fn elegir(a entero, b entero) entero
interfaz Segundo
	fn elegir(x entero, y entero) entero
interfaz Ambos
	Primero
	Segundo
	fn elegir(izquierda entero, derecha entero) entero
tipo Contador
	valor entero
	pub fn siguiente() entero
		@valor = @valor + 1
		@valor
	pub fn elegir(x entero, y entero = 8) entero x * 10 + y
tipo Caja<T>
	valor T
	lista [T]
	opcional T?
	pub fn elegir(alternativa T = @valor) T alternativa
tipo Externa<T>
	interna Caja<T>
enum E<T>
	Dato T
	Nada
fn ultimos<T>(primero T, resto ...T) [T] resto
fn defecto<T>(a T, b T = a) T b
fn copia<T>(c Caja<T>) T c.valor
fn error<T>(v T) T! .Ok(v)
fn envuelto(v entero) Todo? v
fn resultado(v entero) Todo! v
fn obtener<T>(v Todo) T? v como T
fn contar<T>(v Todo) entero
	casos v
		T => 1
		entero => 2
		_ => 3
fn inicio()
	var c = Contador {}
	var i Ambos = c
	imprimir(i.elegir(derecha = c.siguiente(), izquierda = c.siguiente()))
	imprimir(c.elegir(3))
	var caja = Caja<entero> {valor: 7}
	imprimir(caja.elegir())
	imprimir(caja.elegir(9))
	imprimir(copia(caja))
	var anidada = Caja<Caja<entero>> {valor: caja}
	imprimir(anidada.valor.valor)
	var contextual Caja<entero> = {valor: 4}
	imprimir(contextual.valor)
	var objeto Todo? = Caja<entero> {valor: 1}
	si objeto |v|
		var caja_extraida = v como Caja<entero>
		si caja_extraida |p| imprimir(p.valor)
	var lista Todo? = [2]
	si lista |v|
		var lista_extraida = v como [entero]
		si lista_extraida |p| imprimir(p[0])
	imprimir(defecto(b = 6, a = 2))
	imprimir(defecto(5))
	imprimir(ultimos(1, 2, 3)[1])
	imprimir(ultimos(1, [4, 5]...)[0])
	imprimir(ultimos(resto = [6, 7], primero = 0)[1])
	var presente = envuelto(8)
	si presente |v|
		var numero = v como entero
		imprimir(numero o 0)
	var r = resultado(9)
	casos r |v|
		.Ok =>
			var numero = v como entero
			imprimir(numero o 0)
		.Error => imprimir(v)
	var rr = error(10)
	imprimir(rr o 0)
	var e E<cadena> = .Dato("hola")
	casos e |v|
		.Dato => imprimir(v)
		.Nada => imprimir("nada")
	var numero = obtener<entero>(11)
	imprimir(numero o 0)
	imprimir(contar<entero>(12))
	imprimir(contar<cadena>(12))
	var envoltorio Todo = presente
	var extraido = envoltorio como Todo?
	si extraido |interno|
		si interno |v|
			var n = v como entero
			imprimir(n o 0)
`
	runCometa(t, source, "21\n38\n7\n9\n7\n7\n4\n1\n2\n6\n5\n3\n4\n7\n8\n9\n10\nhola\n11\n1\n2\n8\n")
}

func TestInterfaceInspectionEvaluationAndControlFlow(t *testing.T) {
	source := `interfaz Todo
tipo Contador
	n entero
	pub fn siguiente() Todo
		@n = @n + 1
		@n
fn describir(c Contador) cadena
	casos c.siguiente() |v|
		entero =>
			imprimir(v)
			"numero"
		_ => "otro"
fn recuperar(c Contador) entero
	var n = c.siguiente() como cadena o retornar 99
	0
fn inicio()
	var c = Contador {}
	imprimir(describir(c))
	imprimir(c.n)
	imprimir(recuperar(c))
	imprimir(c.n)
	var total = 0
	repetir (0..5) |n|
		var v Todo = n
		casos v |p|
			entero =>
				si p == 0 continuar
				si p == 3 romper
				total = total + p
			_ => imprimir("otro")
	imprimir(total)
`
	runCometa(t, source, "1\nnumero\n1\n99\n2\n3\n")
}

func TestTypeParameterInspectionFromMethodInterface(t *testing.T) {
	source := `interfaz I
	fn numero() entero
tipo A
	pub fn numero() entero 1
fn extraer<T>(v I) T? v como T
fn comprobar<T>(v I) entero
	casos v
		T => 1
		_ => 0
fn inicio()
	var a I = A {}
	var encontrado = extraer<A>(a)
	si encontrado |v| imprimir(v.numero())
	var ausente = extraer<entero>(a)
	imprimir(ausente o 2)
	imprimir(comprobar<A>(a))
	imprimir(comprobar<entero>(a))
`
	runCometa(t, source, "1\n2\n1\n0\n")
}
