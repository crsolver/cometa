package compiler

import (
	"strings"
	"testing"
)

func TestMapsRuntime(t *testing.T) {
	runCometa(t, `var global = ["g": 7]
tipo Nodo
	hijos [cadena: Nodo]
fn poner<T>(m [cadena: T], valor T) T?
	m["x"] = valor
	m.obtener(clave = "x")
fn inicio()
	var m = [
		"a": 0,
		// La última entrada gana.
		"b": 2,
		"b": 3,
	]
	imprimir(m["a"] o -1)
	imprimir(m["ausente"] o -1)
	imprimir(m.longitud())
	imprimir(m.contiene(clave = "b"))
	var alias = m
	alias["c"] = 5
	imprimir(m.obtener("c") o -1)
	var copia = m.copiar()
	copia["c"] = 99
	imprimir(m["c"] o -1)
	imprimir(copia["c"] o -1)
	imprimir(m.claves().longitud())
	imprimir(m.valores().longitud())
	var suma = 0
	repetir (m) |valor, clave|
		suma = suma + valor
		imprimir(m.contiene(clave))
	imprimir(suma)
	imprimir(m.eliminar("a"))
	imprimir(m.eliminar("a"))
	alias.vaciar()
	imprimir(m.esta_vacia())
	imprimir(copia.longitud())
	imprimir(global["g"] o -1)
	imprimir(poner(m, 42) o -1)
	var a = Nodo {}
	var b = Nodo {}
	a.hijos["b"] = b
	imprimir(b.hijos.longitud())
	imprimir(a.hijos.longitud())
	var booleanos = [verdadero: "sí", falso: "no"]
	imprimir(booleanos[verdadero] o "error")
	var enteros = [1: "uno"]
	imprimir(enteros[1] o "error")
`, "0\n-1\n2\ntrue\n5\n5\n99\n3\n3\ntrue\ntrue\ntrue\n8\ntrue\nfalse\ntrue\n3\n7\n42\n0\n1\nsí\nuno\n")
}

func TestMapsNestedAndContextualRuntime(t *testing.T) {
	runCometa(t, `tipo Objeto
	n entero
enum E
	A
fn inicio()
	var opcionales [cadena: entero?] = ["nada": .Ninguno, "algo": 4]
	si (opcionales["nada"]) |interior|
		imprimir(interior o -1)
	sino
		imprimir(-2)
	si (opcionales["ausente"]) |interior|
		imprimir(interior o -3)
	sino
		imprimir(-4)
	var objetos [cadena: Objeto] = ["a": {n: 5}]
	var copia = objetos.copiar()
	var objeto = copia["a"] o Objeto {}
	objeto.n = 8
	var original = objetos["a"] o Objeto {}
	imprimir(original.n)
	var anidados [cadena: [entero: E]] = ["a": [1: .A], "b": [:]]
	var extraido = anidados["b"] o [:]
	extraido[2] = .A
	imprimir(extraido.longitud())
	var decimales [cadena: decimal] = ["a": 1]
	imprimir(decimales["a"] o 0.0)
`, "-1\n-4\n8\n1\n1\n")
}

func TestMapsEvaluationOrderAndPropagation(t *testing.T) {
	runCometa(t, `var m [cadena: entero] = [:]
fn receptor() [cadena: entero]
	imprimir("receptor")
	m
fn clave() cadena
	imprimir("clave")
	"a"
fn valor() entero
	imprimir("valor")
	9
fn falla() entero! .Error("fallo")
fn cambiar() !
	receptor()[clave()] = intentar falla()
	.Ok
fn construir() [cadena: entero]!
	[clave(): intentar falla(), "b": valor()]
fn inicio()
	receptor()[clave()] = valor()
	imprimir(receptor()[clave()] o -1)
	cambiar() capturar |e| imprimir(e)
	imprimir(m["a"] o -1)
	var construido = construir() capturar |e|
		imprimir(e)
		[:]
	var duplicados = [clave(): valor(), clave(): 10]
	imprimir(duplicados["a"] o -1)
	receptor().vaciar()
	imprimir(m.longitud())
`, "receptor\nclave\nvalor\nreceptor\nclave\n9\nreceptor\nclave\nfallo\n9\nclave\nfallo\nclave\nvalor\nclave\n10\nreceptor\n0\n")
}

func TestMapDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"var m = [:]", "mapa vacío"},
		{"var m = [1.0: 2]", "clave de mapa"},
		{"var m [decimal: entero] = [:]", "clave de mapa"},
		{"var m = [\"a\": 1, 2: 2]", "clave debe ser cadena"},
		{"var m = [\"a\": 1, \"b\": verdadero]", "valor debe ser entero"},
		{"var m = [\"a\": 1]\n\tm[1] = 2", "clave debe ser cadena"},
		{"var m = [\"a\": 1]\n\tvar n entero = m[\"a\"]", "entero?"},
		{"var m [cadena: entero] = []", "no una lista"},
		{"var m = [\"a\": 1]\n\tvar n [cadena: decimal] = m", "[cadena: entero]"},
		{"var m = [\"a\": 1]\n\timprimir(m == m)", "operador"},
	} {
		_, err := Compile("mapas.cometa", []byte("fn inicio()\n\t"+tc.source+"\n"))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %s", tc.source, err, tc.want)
		}
	}
	_, err := Compile("mapas.cometa", []byte("fn f<K>(m [K: entero]) entero 1\nfn inicio() imprimir(1)\n"))
	if err == nil || !strings.Contains(err.Error(), "clave de mapa") {
		t.Fatalf("generic key: %v", err)
	}
}

func TestMapsExplicitExitsAndLazyLookup(t *testing.T) {
	runCometa(t, `fn literal() entero
	var m = ["a": retornar 7]
	9
fn clave() entero
	var m = [(retornar 8): 1]
	9
fn asignar(m [cadena: entero]) entero
	m[(retornar 6)] = 100
fn leer(m [cadena: entero]) entero
	m[(retornar 5)]
fn alternativa() entero
	imprimir("alternativa")
	3
fn inicio()
	var m = ["a": 0]
	imprimir(literal())
	imprimir(clave())
	imprimir(asignar(m))
	imprimir(leer(m))
	imprimir(m["a"] o alternativa())
	imprimir(m["b"] o alternativa())
`, "7\n8\n6\n5\n0\nalternativa\n3\n")
}

func TestMapsProjectTypesAndVisibility(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa": `usar modelos como m
fn primero<T>(mapa [cadena: T]) T? mapa["a"]
fn inicio()
	var valores [cadena: m.Dato] = ["a": {numero: 12}]
	var dato = primero(valores) o m.Dato {}
	imprimir(dato.numero)
	var caja = m.Caja<m.Dato> {}
	caja.valores["a"] = dato
	imprimir(caja.valores.longitud())
	var ocultos = m.crear()
	var copia = ocultos.copiar()
	imprimir(copia.longitud())
`,
		"modelos.cometa": `pub tipo Dato
	pub numero entero
pub tipo Caja<T>
	pub valores [cadena: T]
tipo Oculto
	valor entero
pub fn crear() [cadena: Oculto] ["x": {}]
`,
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "12\n1\n1\n")
}

func TestMapInterfaceValues(t *testing.T) {
	runCometa(t, `interfaz Cualquiera
fn inicio()
	var valores [cadena: Cualquiera] = ["a": 1, "b": "dos"]
	var valor = valores["a"] o 0
	imprimir((valor como entero) o -1)
	var objeto Cualquiera = ["x": 4]
	var m = (objeto como [cadena: entero]) o [:]
	imprimir(m["x"] o -1)
`, "1\n4\n")
	_, err := Compile("mapas.cometa", []byte("interfaz Medible\n\tfn longitud() entero\nfn inicio()\n\tvar m Medible = [\"a\": 1]\n"))
	if err == nil {
		t.Fatal("map intrinsics must not satisfy interface methods")
	}
}

func TestMapsIterationAndFreshDefaults(t *testing.T) {
	runCometa(t, `tipo Contenedor
	mapa [entero: entero]
fn nuevo(m [cadena: entero] = [:]) entero
	m["a"] = (m["a"] o 0) + 1
	m["a"] o 0
fn inicio()
	imprimir(nuevo())
	imprimir(nuevo())
	var m = [1: 4, 2: 8, 3: 16]
	var suma = 0
	repetir (m) |valor|
		suma = suma + valor
	imprimir(suma)
	var vueltas = 0
	repetir (m) |valor, clave|
		vueltas = vueltas + 1
		m.vaciar()
		continuar
	imprimir(vueltas)
	imprimir(m.longitud())
	var booleanos = [verdadero: 7]
	repetir (booleanos) |valor, clave|
		imprimir(clave)
		imprimir(valor)
		romper
	var cajas = [Contenedor {}, Contenedor {}]
	cajas[0].mapa[1] = 10
	imprimir(cajas[1].mapa.longitud())
	var claves = cajas[0].mapa.claves()
	claves[0] = 99
	imprimir(cajas[0].mapa.contiene(1))
	var valores = cajas[0].mapa.valores()
	valores[0] = 99
	imprimir(cajas[0].mapa[1] o 0)
`, "1\n1\n28\n1\n0\ntrue\n7\n0\ntrue\n10\n")
}
