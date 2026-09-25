package compiler

import (
	"strings"
	"testing"
)

func TestContextualEnumRuntime(t *testing.T) {
	runHacha(t, `enum IP
	V4
	V6
	Otro IP
tipo Caja
	ip IP
	fn poner(ip IP)
		@ip = ip
fn a_cadena(ip IP) cadena
	casos ip
		IP.V4 => "v4"
		IP.V6 => "v6"
		IP.Otro => "otro"
fn a_cadena2(ip IP) cadena
	casos ip |p|
		.V4 => "v4"
		.V6 => "v6"
		.Otro => a_cadena(p)
fn crear() IP
	.V4
fn elegir() IP
	si verdadero
		.V6
	sino
		.V4
fn inicio()
	imprimir(a_cadena(.V4))
	imprimir(a_cadena(.V6))
	var ip IP = .V4
	ip = .V6
	var caja Caja = {ip: .V4}
	caja.poner(.V6)
	imprimir(a_cadena(caja.ip))
	var lista [IP] = [.V4, .Otro(.V6)]
	imprimir(a_cadena2(lista[1]))
	imprimir(a_cadena(crear()))
	imprimir(a_cadena(elegir()))
	var resultado IP = casos ip
		.V4 => .V6
		_ => .V4
	imprimir(a_cadena(resultado))
`, "v4\nv6\nv6\nv6\nv4\nv6\nv4\n")
}

func TestContextualPayloadReferences(t *testing.T) {
	runHacha(t, `tipo Caja
	valor cadena
enum Evento
	Caja Caja
	Otro Evento
fn leer(e Evento) cadena
	casos e |p|
		.Caja => p.valor
		.Otro => leer(p)
fn inicio()
	var caja Caja = {valor: "antes"}
	var e Evento = .Otro(.Caja(caja))
	caja.valor = "despues"
	imprimir(leer(e))
`, "despues\n")
}

func TestContextualEnumErrors(t *testing.T) {
	const declarations = "enum IP\n\tV4\n\tV6\n\tOtro IP\nenum Otro\n\tV4\nfn f(ip IP)\n\timprimir(ip)\nfn inicio()\n\t"
	for _, tt := range []struct{ source, want string }{
		{"var ip = .V4", "no se puede inferir"},
		{"imprimir(.V4)", "no se puede inferir"},
		{"var n entero = .V4", "no se puede inferir"},
		{"f(.Nada)", "no existe en IP"},
		{"f(.Otro)", "requiere un payload"},
		{"f(.Otro())", "exactamente un payload"},
		{"f(.Otro(1))", "payload debe ser IP"},
		{"f(.V4())", "no acepta paréntesis"},
		{"casos IP.V4\n\t\tV4 => 1", "requieren Enum.Variante"},
		{"casos IP.V4\n\t\tOtro.V4 => 1", "debe pertenecer a IP"},
		{"casos IP.V4\n\t\tIP.V4 => 1\n\t\t.V4 => 2", "duplicada"},
		{"casos IP.V4\n\t\t.V4 => 1", "faltan: V6, Otro"},
		{"casos IP.V4\n\t\t._ => 1", "comodín"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			_, err := Compile("contextual.hacha", []byte(declarations+tt.source+"\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v; want %q", err, tt.want)
			}
		})
	}
}
