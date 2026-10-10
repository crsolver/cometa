package stdlib

var pincelModule = Module{
	Name: "pincel",
	Path: "std/pincel",
	Doc:  "El núcleo del motor de juegos: pincel.ejecutar arranca el bucle de juego a partir de un valor que implemente pincel.Juego.",
	Types: []Type{
		{
			Name:      "Juego",
			Doc:       "La interfaz que debe implementar el valor pasado a pincel.ejecutar: actualizar(dt) para la lógica, pintar() para dibujar.",
			Interface: true,
			Methods: []Function{
				{Name: "actualizar", Signature: "(dt decimal)"},
				{Name: "pintar", Signature: "()"},
			},
		},
	},
	Functions: []Function{
		{Name: "ejecutar", Signature: `(instancia Juego, ancho entero = 320, alto entero = 180, titulo cadena = "Cometa", escala decimal = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = falso, retro bool = falso) !`, GoName: "ejecutar",
			Doc: "Arranca el bucle de juego con `instancia` (que debe implementar pincel.Juego) y abre la ventana con la configuración dada. No retorna hasta que el juego se cierra."},
		{Name: "salir", Signature: "()", GoName: "salir",
			Doc: "Pide cerrar el juego: la ventana se cierra al terminar el cuadro actual y pincel.ejecutar retorna."},
	},
}
