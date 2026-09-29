package compiler

import (
	"strings"
	"testing"
)

func TestFieldDefaultsRuntime(t *testing.T) {
	runCometa(t, `tipo Mascota
	nombre cadena
	energia entero = 10
fn contador() entero
	5
tipo Caja
	valor entero = contador() + 1
fn inicio()
	var gato = Mascota {nombre: "Michi"}
	imprimir(gato.energia)
	var perro = Mascota {nombre: "Rex", energia: 3}
	imprimir(perro.energia)
	var c1 = Caja {}
	var c2 = Caja {}
	imprimir(c1.valor)
	imprimir(c2.valor)
`, "10\n3\n6\n6\n")
}

func TestFieldDefaultFreshPerLiteral(t *testing.T) {
	runCometa(t, `tipo Caja
	valores [entero] = [1]
fn inicio()
	var a = Caja {}
	var b = Caja {}
	a.valores[0] = a.valores[0] + 1
	imprimir(a.valores[0])
	imprimir(b.valores[0])
`, "2\n1\n")
}

func TestFieldDefaultUnderFlowCodegen(t *testing.T) {
	// Generic type params on any declaration force flow-mode codegen for the
	// whole file (see model.HasScopes / TypeParams checks in codegen.go), so
	// this also exercises defaultFields' flow-aware branch for Caja's field.
	runCometa(t, `tipo Generico<T>
	valor T
tipo Caja
	activo bool = si (contador() > 0) verdadero sino falso
fn contador() entero
	1
fn inicio()
	var g = Generico<entero> {valor: 1}
	imprimir(g.valor)
	var c = Caja {}
	imprimir(c.activo)
`, "1\nverdadero\n")
}

func TestFieldDefaultBreaksRequiredCycle(t *testing.T) {
	// Without a default, a required self-referential field is rejected as a cycle
	// (see "cycle and independent" in diagnostics_test.go). A supplied default
	// breaks that static cycle check; actually constructing an instance without
	// an explicit, non-recursive override is the caller's responsibility.
	_, err := Compile("cycle.cometa", []byte(`tipo Nodo
	valor entero
	siguiente Nodo = Nodo {valor: 0, siguiente: raiz()}
fn raiz() Nodo
	Nodo {valor: 0, siguiente: Nodo {valor: 0, siguiente: raiz()}}
fn inicio()
	imprimir("ok")
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidFieldDefaults(t *testing.T) {
	cases := []struct{ source, message string }{
		{"tipo T\n\tcampo entero = campo\nfn inicio()\n\tT {}\n", "no existe"},
		{"tipo T\n\ta entero\n\tb entero = a\nfn inicio()\n\tT {a: 1}\n", "no existe"},
		{"tipo T\n\tcampo entero = @campo\nfn inicio()\n\tT {}\n", "método"},
		{"tipo T\n\tcampo bool = 5\nfn inicio()\n\tT {}\n", "debe ser"},
		{"tipo Base<T>\n\tvalor T\ntipo T\n\tBase<entero> = algo\nfn inicio()\n\tT {}\n", "embebidos no admiten"},
		{"tipo Caja<T>\n\tcontenido T = 0\nfn inicio()\n\tCaja<entero> {}\n", "debe ser"},
	}
	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			_, err := Compile("field_defaults.cometa", []byte(tt.source))
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want %q", err, tt.message)
			}
		})
	}
}
