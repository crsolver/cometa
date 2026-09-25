package compiler

import (
	"strings"
	"testing"
)

func TestGlobalVariablesAndConstantsRuntime(t *testing.T) {
	source := `const base = 2
const limite entero = base + 3
var siguiente = limite
var contador entero = siguiente
fn subir()
	contador = contador + 1
fn inicio()
	subir()
	imprimir(contador)
`
	generated, err := Compile("globales.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	goSource := string(generated)
	for _, expected := range []string{
		"const Base int64 = 2",
		"const Limite int64 = (Base + 3)",
		"var Siguiente int64 = func() int64",
		"Contador = (Contador + 1)",
	} {
		if !strings.Contains(goSource, expected) {
			t.Errorf("generated Go does not contain %q:\n%s", expected, goSource)
		}
	}
	runGeneratedGo(t, generated, "6\n")
}

func TestGlobalCompositeValuesAndFunctionCalls(t *testing.T) {
	source := `tipo Usuario
	nombre cadena
fn crear() Usuario Usuario {nombre: "Ana"}
var usuario_global = crear()
var nombres = ["uno", "dos"]
var opcional entero? = .Alguno(7)
fn inicio()
	imprimir(usuario_global.nombre)
	imprimir(nombres[1])
	imprimir(opcional o 0)
`
	generated, err := Compile("globales_compuestas.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "Ana\ndos\n7\n")
}

func TestGlobalDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"constant assignment", "const limite = 3\nfn inicio()\n\tlimite = 4\n", "no puede reasignarse"},
		{"constant list", "const valores = [1]\n", "solo puede ser entero, cadena o bool"},
		{"constant call", "fn valor() entero 1\nconst limite = valor()\n", "expresión constante simple"},
		{"constant reads variable", "var base = 1\nconst limite = base + 1\n", "solo puede referirse a otras constantes"},
		{"global control flow", "fn valor() entero! .Ok(1)\nvar global = intentar valor()\n", "no admite control de flujo"},
		{"global cycle", "var primero = segundo\nvar segundo = primero\n", "ciclo de inicialización global"},
		{"global cycle through function", "var primero = leer()\nfn leer() entero primero\n", "ciclo de inicialización global"},
		{"name conflict", "var dato = 1\nfn dato() entero 2\n", "declarado como global"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Compile(test.name+".hacha", []byte(test.source))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestImportedGlobalsArePublic(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.hacha":   "usar config como c\nconst doble = c.limite * 2\nvar copia = doble\nfn inicio()\n\tc.contador = c.contador + copia\n\timprimir(c.contador)\n",
		"config.hacha": "const limite = 4\nvar contador = 1\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "9\n")
}

func TestLocalVariableShadowsGlobal(t *testing.T) {
	source := "var valor = 1\nfn inicio()\n\tvar valor = 2\n\timprimir(valor)\n"
	generated, err := Compile("sombra_global.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "2\n")
}
