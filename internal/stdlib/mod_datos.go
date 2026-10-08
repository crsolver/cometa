package stdlib

var datosModule = Module{
	Name: "datos",
	Path: "std/pincel/datos",
	Doc:  "Guarda y lee pequeños datos de tu juego (mejor puntuación, opciones) como texto o números asociados a una clave; se conservan entre partidas en la carpeta de configuración del usuario.",
	Functions: []Function{
		{Name: "guardar", Signature: "(clave cadena, valor cadena) !", GoName: "datosGuardar",
			Doc: "Guarda `valor` bajo `clave` en el disco y lo conserva entre partidas; falla si no se puede escribir. Para números usa guardar_entero o guardar_decimal."},
		{Name: "leer", Signature: "(clave cadena) cadena?", GoName: "datosLeer",
			Doc: "El valor guardado bajo `clave`, o .Ninguno si no existe todavía; por ejemplo datos.leer(\"nombre\") o \"Anónimo\"."},
		{Name: "borrar", Signature: "(clave cadena) !", GoName: "datosBorrar",
			Doc: "Elimina la clave; no falla si no existía."},
		{Name: "juego", Signature: "(nombre cadena)", GoName: "datosJuego",
			Doc: `Fija el nombre con el que se guardan los datos (por ejemplo datos.juego("mi_juego")), para que no cambien al renombrar o mover la carpeta del proyecto. Llámalo al principio de inicio, antes de leer nada.`},
		{Name: "claves", Signature: "() [cadena]", GoName: "datosClaves",
			Doc: "Todas las claves guardadas, en orden alfabético."},
		{Name: "guardar_entero", Signature: "(clave cadena, valor entero) !", GoName: "datosGuardarEntero",
			Doc: "Guarda un número entero bajo `clave`; se lee con leer_entero."},
		{Name: "leer_entero", Signature: "(clave cadena) entero?", GoName: "datosLeerEntero",
			Doc: "El entero guardado bajo `clave`, o .Ninguno si no existe o no es un entero; por ejemplo datos.leer_entero(\"mejor\") o 0."},
		{Name: "guardar_decimal", Signature: "(clave cadena, valor decimal) !", GoName: "datosGuardarDecimal",
			Doc: "Guarda un número decimal bajo `clave`; falla si no es finito. Se lee con leer_decimal."},
		{Name: "leer_decimal", Signature: "(clave cadena) decimal?", GoName: "datosLeerDecimal",
			Doc: "El decimal guardado bajo `clave`, o .Ninguno si no existe o no es un número."},
	},
}
