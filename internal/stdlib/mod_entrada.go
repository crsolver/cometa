package stdlib

import "strconv"

var entradaModule = Module{
	Name: "entrada",
	Path: "std/pincel/entrada",
	Doc:  "Lee el teclado, el ratón y los mandos (gamepads): teclas mantenidas, presionadas o soltadas, botones del ratón y su posición, y botones y palancas de los mandos.",
	Types: []Type{
		{
			Name:  "BotonMando",
			Doc:   "Un botón de un mando estándar (.A, .B, .X, .Y, .Arriba, .HombroIzquierdo, .Inicio...), para usar con las funciones mando_*.",
			Value: true,
			Constants: map[string]string{
				"A":                "_hgBotonMando(ebiten.StandardGamepadButtonRightBottom)",
				"Abajo":            "_hgBotonMando(ebiten.StandardGamepadButtonLeftBottom)",
				"Arriba":           "_hgBotonMando(ebiten.StandardGamepadButtonLeftTop)",
				"Atras":            "_hgBotonMando(ebiten.StandardGamepadButtonCenterLeft)",
				"B":                "_hgBotonMando(ebiten.StandardGamepadButtonRightRight)",
				"Derecha":          "_hgBotonMando(ebiten.StandardGamepadButtonLeftRight)",
				"GatilloDerecho":   "_hgBotonMando(ebiten.StandardGamepadButtonFrontBottomRight)",
				"GatilloIzquierdo": "_hgBotonMando(ebiten.StandardGamepadButtonFrontBottomLeft)",
				"HombroDerecho":    "_hgBotonMando(ebiten.StandardGamepadButtonFrontTopRight)",
				"HombroIzquierdo":  "_hgBotonMando(ebiten.StandardGamepadButtonFrontTopLeft)",
				"Inicio":           "_hgBotonMando(ebiten.StandardGamepadButtonCenterRight)",
				"Izquierda":        "_hgBotonMando(ebiten.StandardGamepadButtonLeftLeft)",
				"PalancaDerecha":   "_hgBotonMando(ebiten.StandardGamepadButtonRightStick)",
				"PalancaIzquierda": "_hgBotonMando(ebiten.StandardGamepadButtonLeftStick)",
				"X":                "_hgBotonMando(ebiten.StandardGamepadButtonRightLeft)",
				"Y":                "_hgBotonMando(ebiten.StandardGamepadButtonRightTop)",
			},
		},
		{
			Name:  "BotonRaton",
			Doc:   "Un botón del ratón (izquierdo, derecho o medio), para usar con las funciones de entrada.",
			Value: true,
			Constants: map[string]string{
				"Derecho":   "_hgBotonRaton(ebiten.MouseButtonRight)",
				"Izquierdo": "_hgBotonRaton(ebiten.MouseButtonLeft)",
				"Medio":     "_hgBotonRaton(ebiten.MouseButtonMiddle)",
			},
		},
		{
			Name:  "EjeMando",
			Doc:   "Una palanca de un mando estándar (.IzquierdoX, .IzquierdoY, .DerechoX, .DerechoY) para entrada.mando_eje.",
			Value: true,
			Constants: map[string]string{
				"DerechoX":   "_hgEjeMando(ebiten.StandardGamepadAxisRightStickHorizontal)",
				"DerechoY":   "_hgEjeMando(ebiten.StandardGamepadAxisRightStickVertical)",
				"IzquierdoX": "_hgEjeMando(ebiten.StandardGamepadAxisLeftStickHorizontal)",
				"IzquierdoY": "_hgEjeMando(ebiten.StandardGamepadAxisLeftStickVertical)",
			},
		},
		{
			Name:      "Tecla",
			Doc:       "Una tecla del teclado, para usar con las funciones de entrada.",
			Value:     true,
			Constants: teclas(),
		},
	},
	Functions: []Function{
		{Name: "tecla_mantenida", Signature: "(tecla Tecla) bool", GoName: "teclaMantenida",
			Doc: "verdadero mientras la tecla está presionada (se dispara en cada cuadro que sigue sostenida)."},
		{Name: "tecla_presionada", Signature: "(tecla Tecla) bool", GoName: "teclaPresionada",
			Doc: "verdadero solo en el cuadro en que la tecla empezó a presionarse."},
		{Name: "tecla_soltada", Signature: "(tecla Tecla) bool", GoName: "teclaSoltada",
			Doc: "verdadero solo en el cuadro en que la tecla se soltó."},
		{Name: "raton_mantenido", Signature: "(boton BotonRaton) bool", GoName: "ratonMantenido",
			Doc: "verdadero mientras el botón del ratón está presionado."},
		{Name: "raton_presionado", Signature: "(boton BotonRaton) bool", GoName: "ratonPresionado",
			Doc: "verdadero solo en el cuadro en que el botón del ratón empezó a presionarse."},
		{Name: "raton_soltado", Signature: "(boton BotonRaton) bool", GoName: "ratonSoltado",
			Doc: "verdadero solo en el cuadro en que el botón del ratón se soltó."},
		{Name: "posicion_raton", Signature: "() Vec2", GoName: "posicionRaton",
			Doc: "Posición actual del cursor en coordenadas de pantalla."},
		{Name: "rueda", Signature: "() Vec2", GoName: "rueda",
			Doc: "Desplazamiento de la rueda del ratón en este cuadro (normalmente pequeño; suma cero cuando no se mueve)."},
		{Name: "mandos", Signature: "() entero", GoName: "mandos",
			Doc: "Cuántos mandos (gamepads) con distribución estándar hay conectados; se numeran desde 0."},
		{Name: "mando_boton", Signature: "(mando entero, boton BotonMando) bool", GoName: "mandoBoton",
			Doc: "verdadero mientras el botón del mando número `mando` está pulsado (falso si ese mando no existe)."},
		{Name: "mando_boton_presionado", Signature: "(mando entero, boton BotonMando) bool", GoName: "mandoBotonPresionado",
			Doc: "verdadero solo en el cuadro en que se pulsó el botón del mando."},
		{Name: "mando_eje", Signature: "(mando entero, eje EjeMando, zona_muerta decimal = 0.15) decimal", GoName: "mandoEje",
			Doc: "Posición de una palanca en [-1, 1]; los valores por debajo de `zona_muerta` (0.15 por defecto) se tratan como 0 para ignorar el temblor. Devuelve 0 si el mando no existe."},
	},
}

// teclas are the entrada.Tecla constants: the named keys plus the letters, the
// digits and the function keys, which follow Ebitengine's own names.
func teclas() map[string]string {
	keys := map[string]string{
		"Izquierda": "ArrowLeft", "Derecha": "ArrowRight", "Arriba": "ArrowUp", "Abajo": "ArrowDown",
		"Espacio": "Space", "Escape": "Escape", "Enter": "Enter", "Tab": "Tab", "Retroceso": "Backspace",
		"Shift": "Shift", "Control": "Control", "Alt": "Alt", "Suprimir": "Delete",
		"Inicio": "Home", "Fin": "End", "RePag": "PageUp", "AvPag": "PageDown",
		"ShiftIzquierdo": "ShiftLeft", "ShiftDerecho": "ShiftRight",
		"ControlIzquierdo": "ControlLeft", "ControlDerecho": "ControlRight",
	}
	for i, name := range []string{"Cero", "Uno", "Dos", "Tres", "Cuatro", "Cinco", "Seis", "Siete", "Ocho", "Nueve"} {
		keys[name] = "Digit" + strconv.Itoa(i)
	}
	for i := 1; i <= 12; i++ {
		keys["F"+strconv.Itoa(i)] = "F" + strconv.Itoa(i)
	}
	for c := 'A'; c <= 'Z'; c++ {
		keys[string(c)] = string(c)
	}
	constants := map[string]string{}
	for name, key := range keys {
		constants[name] = "_hgTecla(ebiten.Key" + key + ")"
	}
	return constants
}
