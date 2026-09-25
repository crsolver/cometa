package compiler

import (
	"strings"
	"testing"
)

const visibilityLibrary = `pub interfaz Lector
	fn leer() entero
pub tipo Caja<T>
	secreto entero
	pub valor T
	fn oculto() entero @secreto
	pub fn leer() entero @secreto
pub tipo Base
	pub numero entero
	pub fn leer() entero @numero
pub tipo Cerrado
	Base
pub tipo Abierto
	pub Base
pub tipo Sombra
	pub Base
	numero entero
pub tipo Obligatorio
	valor entero!
pub enum Evento
	Dato entero
	Vacio
tipo Oculto
	pub numero entero
	pub fn leer() entero @numero
fn secreto() entero 7
enum Interno
	Uno
interfaz Interna
var privada = 1
const constante = 2
pub var contador = 0
pub const limite = 3
pub fn crear() Oculto Oculto {numero: secreto()}
pub fn predeterminado(valor entero = secreto()) entero valor
pub fn local() entero
	var c = Caja<entero> {secreto: 9, valor: 1}
	c.secreto = c.oculto()
	var texto = "${c.secreto}"
	c.secreto
`

func TestPrivateModuleAccess(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"function", "imprimir(m.secreto())", "privada"},
		{"global read", "imprimir(m.privada)", "privada"},
		{"global write", "m.privada = 2", "privada"},
		{"constant", "imprimir(m.constante)", "privada"},
		{"type", "var x = m.Oculto {}", "privada"},
		{"enum", "var x = m.Interno.Uno", "privada"},
		{"interface", "var x m.Interna = 1", "privada"},
		{"field read", "var c = m.Caja<entero> {valor: 1}\n\timprimir(c.secreto)", "privado"},
		{"field write", "var c = m.Caja<entero> {valor: 1}\n\tc.secreto = 1", "privado"},
		{"method", "var c = m.Caja<entero> {valor: 1}\n\timprimir(c.oculto())", "privado"},
		{"named literal", "var c = m.Caja<entero> {secreto: 1, valor: 2}", "privado"},
		{"positional literal", "var c = m.Caja<entero> {1, 2}", "privado"},
		{"private required field", "var c = m.Obligatorio {}", "requiere"},
		{"promoted field", "var c = m.Cerrado {}\n\timprimir(c.numero)", "privado"},
		{"promoted method", "var c = m.Cerrado {}\n\timprimir(c.leer())", "privado"},
		{"shadow", "var c = m.Sombra {}\n\timprimir(c.numero)", "privado"},
		{"interface path", "var c m.Lector = m.Cerrado {}", "se esperaba"},
		{"interpolation", "var c = m.Caja<entero> {valor: 1}\n\timprimir(\"${c.secreto}\")", "privado"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, loader := memoryProject(t, map[string]string{"main.cometa": "usar biblioteca como m\nfn inicio()\n\t" + tc.body + "\n", "biblioteca.cometa": visibilityLibrary})
			generated, err := CompileProject(entry, loader)
			if err == nil || !strings.Contains(err.Error(), tc.want) || len(generated) != 0 {
				t.Fatalf("wanted %q, got %v", tc.want, err)
			}
		})
	}
}

func TestPublicModuleRuntime(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{"biblioteca.cometa": visibilityLibrary, "main.cometa": `usar biblioteca como m
fn inicio()
	var opaco = m.crear()
	imprimir(opaco.numero)
	var lector m.Lector = opaco
	imprimir(lector.leer())
	imprimir(m.local())
	imprimir(m.predeterminado())
	var caja = m.Caja<entero> {valor: 4}
	imprimir(caja.leer())
	var abierto = m.Abierto {}
	abierto.numero = 5
	var otro m.Lector = abierto
	imprimir(otro.leer())
	m.contador = m.limite
	imprimir(m.contador)
	var e = m.Evento.Dato(6)
	casos e |dato|
		.Dato => imprimir(dato)
		.Vacio => imprimir(0)
`})
	code, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, code, "7\n7\n9\n7\n0\n5\n3\n6\n")
}

func TestPrivateMethodsCannotEscapeThroughInterfaces(t *testing.T) {
	runCometa(t, `interfaz Todo
interfaz Lector
	fn leer() entero
tipo Privado
	fn leer() entero 1
tipo Publico
	pub fn leer() entero 2
tipo Cerrado
	Publico
tipo Abierto
	pub Publico
tipo Sombra
	pub Publico
	fn leer() entero 3
fn comprobar(valor Todo)
	var lector = valor como Lector
	si lector |l| imprimir(l.leer()) sino imprimir(0)
	casos valor
		Lector => imprimir(1)
		_ => imprimir(0)
fn inicio()
	comprobar(Privado {})
	comprobar(Cerrado {})
	comprobar(Sombra {})
	comprobar(Abierto {})
`, "0\n0\n0\n0\n0\n0\n2\n1\n")
}

func TestPrivateMethodsDoNotSatisfyConstraints(t *testing.T) {
	_, _, err := Analyze("private.cometa", []byte("interfaz I\n\tfn leer() entero\ntipo T\n\tfn leer() entero 1\nfn usarlo<A I>(v A) entero v.leer()\nfn inicio() imprimir(usarlo(T {}))\n"))
	if err == nil || !strings.Contains(err.Error(), "restricción") {
		t.Fatal(err)
	}
}
