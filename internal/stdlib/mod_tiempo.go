package stdlib

var tiempoModule = Module{
	Name:       "tiempo",
	Path:       "std/pincel/tiempo",
	OnlyCalled: true,
	Doc:        "Temporizadores para enfriamientos y eventos periódicos, y los cuadros por segundo reales del juego en ejecución.",
	Types: []Type{
		{
			Name: "Temporizador",
			Doc:  "Cuenta segundos hacia una duración, creado con tiempo.temporizador; sirve para enfriamientos, invulnerabilidad o eventos periódicos. Es un objeto compartido.",
			Methods: []Function{
				{Name: "avanzar", Signature: "(dt decimal) bool", GoName: "avanzar",
					Doc: "Suma `dt` segundos y devuelve `verdadero` en el paso en que el temporizador llega a su duración. Llámalo una vez por cuadro en actualizar."},
				{Name: "terminado", Signature: "() bool", GoName: "terminado",
					Doc: "Indica si un temporizador sin repetición ya llegó a su duración (un temporizador que se repite nunca queda terminado)."},
				{Name: "terminar", Signature: "()", GoName: "terminar",
					Doc: "Lo da por terminado ya, por ejemplo para que un enfriamiento empiece listo."},
				{Name: "reiniciar", Signature: "()", GoName: "reiniciar",
					Doc: "Vuelve a contar desde cero."},
				{Name: "restante", Signature: "() decimal", GoName: "restante",
					Doc: "Segundos que faltan para que termine."},
				{Name: "progreso", Signature: "() decimal", GoName: "progreso",
					Doc: "Fracción transcurrida, de 0 a 1; útil para barras de carga o con std/mate/curvas."},
			},
		},
	},
	Functions: []Function{
		{Name: "fps", Signature: "() decimal", GoName: "fps",
			Doc: "Cuadros por segundo reales que se están dibujando (varía con el rendimiento)."},
		{Name: "tps", Signature: "() decimal", GoName: "tps",
			Doc: "Actualizaciones por segundo reales que se están ejecutando (compáralo con el `tps` configurado en pincel.ejecutar)."},
		{Name: "ahora", Signature: "() decimal", GoName: "ahora",
			Doc: "Segundos transcurridos desde que empezó el programa, con decimales. Sirve para cronometrar o para repartir un cálculo largo entre cuadros: `var limite = tiempo.ahora() + 0.008`. No depende de `dt` ni del reloj del sistema."},
		{Name: "temporizador", Signature: "(duracion decimal, bucle bool = falso) Temporizador", GoName: "temporizador",
			Doc: "Crea un Temporizador que dura `duracion` segundos. Con `bucle = verdadero` vuelve a empezar solo cada vez que termina (eventos periódicos); si no, se queda terminado hasta que lo reinicies (enfriamientos)."},
	},
}
