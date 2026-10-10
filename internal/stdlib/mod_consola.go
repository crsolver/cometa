package stdlib

var consolaModule = Module{
	Name: "consola",
	Path: "std/consola",
	Doc:  "Entrada y salida de texto para programas de consola: leer línea a línea lo que escribe la persona y escribir sin saltar de línea.",
	Functions: []Function{
		{Name: "leer_linea", Signature: "() cadena?", GoName: "consolaLeerLinea",
			Doc: `Espera una línea de la entrada estándar y la devuelve sin el salto de línea final. Devuelve Ninguno cuando la entrada se acaba (fin de archivo, Ctrl+Z o Ctrl+D). Para leer un número: (consola.leer_linea() o "").recortar().a_entero().`},
		{Name: "escribir", Signature: "(texto cadena)", GoName: "consolaEscribir",
			Doc: "Escribe el texto sin salto de línea al final, a diferencia de imprimir. Sirve para mostrar una pregunta y leer la respuesta en la misma línea."},
	},
}
