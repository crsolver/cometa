package compiler

import (
	"strings"
	"testing"
)

func TestStringInterpolationConcatenationAndMethodsRuntime(t *testing.T) {
	runCometa(t, `fn saludar(nombre cadena, prefijo cadena = "Hola " + nombre) cadena prefijo
fn inicio()
	var nombre = "Ana"
	imprimir(saludar(nombre))
	imprimir("Hola " + nombre)
	imprimir("${nombre}, ${20 + 1}, ${verdadero}, ${"a" + "b"}")
	imprimir("\${nombre}")
	var texto = "  Ána🙂ana  "
	imprimir(texto.longitud())
	imprimir(texto.esta_vacia())
	imprimir(texto.contiene("🙂"))
	imprimir(texto.buscar_indice("🙂") o -1)
	imprimir(texto.empieza_con("  Á"))
	imprimir(texto.termina_con("  "))
	imprimir(texto.mayusculas())
	imprimir(texto.minusculas())
	imprimir(texto.recortar())
	imprimir("banana".reemplazar(buscar = "a", reemplazo = "X"))
	imprimir("a,,b,".dividir(",").longitud())
	imprimir("ñ🙂".obtener(1) o "no")
	imprimir("añ🙂z".subcadena(fin = 3, inicio = 1) o "no")
	imprimir("abc".subcadena(1, 1) o "no")
	imprimir("abc".obtener(99) o "no")
	imprimir("".dividir("").longitud())
	imprimir("ab".reemplazar("", "-"))
`, "Hola Ana\nHola Ana\nAna, 21, verdadero, ab\n${nombre}\n11\nfalso\nverdadero\n5\nverdadero\nverdadero\n  ÁNA🙂ANA  \n  ána🙂ana  \nÁna🙂ana\nbXnXnX\n4\n🙂\nñ🙂\n\nno\n0\n-a-b-\n")
}

func TestStringDiagnosticsAndConstants(t *testing.T) {
	tests := []struct{ name, source, want string }{
		{"mixed concatenation", "fn inicio() imprimir(\"x\" + 1)\n", `no acepta cadena y entero`},
		{"interpolation type", "fn inicio() imprimir(\"${[1]}\")\n", `requiere cadena, entero, decimal o bool`},
		{"empty interpolation", "fn inicio() imprimir(\"${}\")\n", `no puede estar vacía`},
		{"unclosed interpolation", "fn inicio() imprimir(\"${1\")\n", `interpolación sin cerrar`},
		{"interpolated const", "const x = \"${1}\"\nfn inicio() imprimir(x)\n", `inicializador de una constante`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Compile("cadenas.cometa", []byte(test.source))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	if _, err := Compile("constante.cometa", []byte("const x = \"a\" + \"b\"\nfn inicio() imprimir(x)\n")); err != nil {
		t.Fatal(err)
	}
}

func TestInterpolationEvaluationOrderNestingAndPropagation(t *testing.T) {
	runCometa(t, `var contador = 0
fn siguiente() entero
	contador = contador + 1
	contador
fn falla() entero! .Error("fallo")
fn texto() cadena!
	"valor ${intentar falla()}"
fn inicio()
	imprimir("${siguiente()} ${siguiente()}")
	imprimir("anidada ${"x ${1}"}")
	imprimir(texto() capturar |e| e)
`, "1 2\nanidada x 1\nfallo\n")
}

func TestInterpolationBindsImportedReferences(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":  "usar datos como d\nfn inicio() imprimir(\"valor ${d.valor()}\")\n",
		"datos.cometa": "pub fn valor() cadena \"externo\"\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "valor externo\n")
}
