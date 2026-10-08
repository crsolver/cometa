package stdlib

var ventanaModule = Module{
	Name: "ventana",
	Path: "std/pincel/ventana",
	Doc:  "Controla el título, el icono, el cursor y el modo de pantalla completa de la ventana del juego.",
	Functions: []Function{
		{Name: "titulo", Signature: "(titulo cadena)", GoName: "titulo",
			Doc: "Cambia el texto de la barra de título de la ventana."},
		{Name: "pantalla_completa", Signature: "(activa bool)", GoName: "pantallaCompleta",
			Doc: "Activa o desactiva el modo de pantalla completa."},
		{Name: "tamano", Signature: "() Vec2", GoName: "ventanaTamano",
			Doc: "El tamaño actual de la ventana del sistema operativo, como Vec2."},
		{Name: "cursor", Signature: "(visible bool)", GoName: "ventanaCursor",
			Doc: "Muestra (`verdadero`) u oculta (`falso`) el cursor del ratón cuando está sobre la ventana."},
		{Name: "icono", Signature: "(imagen Imagen)", GoName: "ventanaIcono",
			Doc: "Usa `imagen` como icono de la ventana (por ejemplo una de recursos.imagen o lienzo). Conviene un tamaño pequeño y cuadrado, como 32×32."},
		{Name: "es_pantalla_completa", Signature: "() bool", GoName: "esPantallaCompleta",
			Doc: "Indica si la ventana está ahora en pantalla completa."},
	},
}
