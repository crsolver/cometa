package sema

import (
	"strings"
	"testing"
)

func TestEnumSemanticErrors(t *testing.T) {
	const declarations = "tipo Boton\n\tcaracter cadena\nenum Evento\n\tCargar\n\tBoton Boton\n\tTexto cadena\n"
	tests := []struct{ name, source, want string }{
		{"unknown variant", "var e = Evento.Verano", "no existe en Evento"},
		{"payload required", "var e = Evento.Texto", "requiere un payload"},
		{"payload wrong type", "var e = Evento.Texto(1)", "payload debe ser cadena"},
		{"payload arity", "var e = Evento.Texto()", "exactamente un payload"},
		{"unit call", "var e = Evento.Cargar()", "no acepta paréntesis"},
		{"unit payload", "var e = Evento.Cargar(1)", "no acepta paréntesis"},
		{"missing variant", "casos Evento.Cargar\n\t\t.Cargar => imprimir(1)", "faltan: Boton, Texto"},
		{"unknown arm", "casos Evento.Cargar\n\t\t.Verano => imprimir(1)", "no existe en Evento"},
		{"duplicate arm", "casos Evento.Cargar\n\t\t.Cargar => imprimir(1)\n\t\t.Cargar => imprimir(2)", "duplicada en casos"},
		{"after wildcard", "casos Evento.Cargar\n\t\t_ => imprimir(1)\n\t\t.Cargar => imprimir(2)", "después de '_'"},
		{"duplicate wildcard", "casos Evento.Cargar\n\t\t_ => imprimir(1)\n\t\t_ => imprimir(2)", "después de '_'"},
		{"unit binding", "casos Evento.Cargar |e|\n\t\t.Cargar => imprimir(e)\n\t\t_ => imprimir(1)", "el nombre \"e\" no existe"},
		{"wildcard binding", "casos Evento.Cargar |e|\n\t\t_ => imprimir(e)", "el nombre \"e\" no existe"},
		{"binding escapes", "casos Evento.Cargar |e|\n\t\t.Texto => imprimir(e)\n\t\t_ => imprimir(1)\n\timprimir(e)", "el nombre \"e\" no existe"},
		{"binding collision", "var e = 1\n\tcasos Evento.Cargar |e|\n\t\t_ => imprimir(1)", "ya fue declarada"},
		{"primitive match", "casos 1\n\t\t_ => imprimir(1)", "requiere un enum"},
		{"mismatched result", "var x = casos Evento.Cargar\n\t\t.Cargar => 1\n\t\t_ => falso", "las ramas producen entero y bool"},
		{"void result", "var x = casos Evento.Cargar\n\t\t_ => imprimir(1)", "debe producir un valor"},
		{"incomplete conditional result", "var x = casos Evento.Cargar\n\t\t_ =>\n\t\t\tsi verdadero 1", "debe producir un valor"},
		{"break through value match", "repetir\n\t\tvar x = casos Evento.Cargar\n\t\t\t_ =>\n\t\t\t\tromper\n\t\t\t\t1", "'romper' solo puede"},
		{"continue through value match", "repetir\n\t\tvar x = casos Evento.Cargar\n\t\t\t_ =>\n\t\t\t\tcontinuar\n\t\t\t\t1", "'continuar' solo puede"},
		{"enum equality", "imprimir(Evento.Cargar == Evento.Cargar)", "no acepta Evento y Evento"},
		{"no implicit conversion", "var e Evento = Boton {}", "el literal es Boton, pero se esperaba Evento"},
		{"enum struct literal", "var e = Evento {}", "el tipo \"Evento\" no existe"},
		{"enum fields private", "var e = Evento.Cargar\n\timprimir(e.tag)", "no tiene miembros"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSource(declarations + "fn inicio()\n\t" + tt.source + "\n")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
	for _, source := range []string{
		"enum E\n\tA\n\tA\n", "enum E\n\t_\n", "enum E\n\tA\nenum E\n\tB\n",
		"tipo E\n\tx entero\nenum E\n\tA\n", "enum E\n\tA\ntipo E\n\tx entero\n", "enum E\n\tA\nfn E() imprimir(1)\n",
		"enum E\n\tA Desconocido\n",
	} {
		if err := checkSource(source); err == nil {
			t.Fatalf("accepted invalid declarations: %s", source)
		}
	}
}

func TestEnumPayloadTypeReferences(t *testing.T) {
	source := "enum E\n\tVacio\n\tNumero entero\n\tTexto cadena\n\tActivo bool\n\tLista [E]\n\tOtro F\n\tObjeto T\nenum F\n\tE E\ntipo T\n\te E\nfn f(e E) E\n\tcasos e |p|\n\t\t.Lista =>\n\t\t\tvar x [E] = p\n\t\t\tE.Lista(x)\n\t\t_ => E.Vacio\nfn inicio()\n\tvar e E = E.Numero(1)\n\tvar lista [E] = [e, E.Texto(\"a\"), E.Activo(verdadero), E.Lista([]), E.Objeto({e: e})]\n\timprimir(lista)\n"
	if err := checkSource(source); err != nil {
		t.Fatal(err)
	}
}
