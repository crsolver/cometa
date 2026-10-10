package stdlib

var azarModule = Module{
	Name: "azar",
	Path: "std/azar",
	Doc:  "Números aleatorios simples para juegos: un real o un entero dentro de un rango, y una semilla para que las partidas sean reproducibles.",
	Functions: []Function{
		{Name: "real", Signature: "(minimo decimal, maximo decimal) decimal", GoName: "azarReal",
			Doc: "Un decimal aleatorio uniforme dentro de [minimo, maximo]."},
		{Name: "entero", Signature: "(minimo entero, maximo entero) entero", GoName: "azarEntero",
			Doc: "Un entero aleatorio uniforme en [minimo, maximo): incluye minimo y excluye maximo, como los rangos a..b. Para un dado usa azar.entero(1, 7). Si minimo == maximo devuelve minimo."},
		{Name: "semilla", Signature: "(valor entero)", GoName: "azarSemilla",
			Doc: "Fija la semilla del generador: con la misma semilla, azar.entero y azar.real producen siempre la misma secuencia (niveles reproducibles, partidas del día). Sin llamarla, cada ejecución es distinta."},
	},
}
