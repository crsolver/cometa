package stdlib

var audioModule = Module{
	Name: "audio",
	Path: "std/pincel/audio",
	Doc:  "Reproduce y controla sonidos: reproducir, pausar, reanudar, detener y ajustar el volumen.",
	Types: []Type{
		{
			Name:  "Onda",
			Doc:   "La forma de onda de audio.tono, que decide su timbre: .Cuadrada (consola antigua), .Triangular (suave), .Sierra (áspera) o .Seno (pura).",
			Value: true,
			Constants: map[string]string{
				"Cuadrada":   "_hgOnda(0)",
				"Seno":       "_hgOnda(3)",
				"Sierra":     "_hgOnda(2)",
				"Triangular": "_hgOnda(1)",
			},
		},
		{
			Name: "Reproduccion",
			Doc:  "Una reproducción de sonido en curso, devuelta por audio.reproducir; permite pausarla, reanudarla o detenerla.",
		},
		{
			Name: "Sonido",
			Doc:  "Un sonido cargado con recursos.sonido o creado con audio.tono o audio.ruido, listo para reproducirse con audio.reproducir.",
		},
	},
	Functions: []Function{
		{Name: "reproducir", Signature: "(sonido Sonido, volumen decimal = 1, bucle bool = falso, tono decimal = 1) Reproduccion", GoName: "reproducir",
			Doc: "Empieza a reproducir `sonido` con el volumen y modo de bucle dados, y devuelve la Reproduccion para controlarla luego. `tono` cambia la altura y la velocidad a la vez: 2 suena una octava más aguda y dura la mitad, 0.5 una octava más grave (se limita a entre 0.25 y 4)."},
		{Name: "tono", Signature: "(frecuencia decimal, duracion decimal, onda Onda = .Cuadrada, hasta decimal = 0, volumen decimal = 1) Sonido", GoName: "audioTono",
			Doc: "Crea un sonido en memoria, sin archivos: un tono de `frecuencia` hercios (440 es un la) que dura `duracion` segundos (hasta 30). `onda` cambia el timbre (.Cuadrada suena a consola antigua); con `hasta` mayor que 0 la frecuencia se desliza hasta ese valor (un salto sube, un láser baja). Guárdalo en una variable global y reprodúcelo con audio.reproducir."},
		{Name: "ruido", Signature: "(duracion decimal, volumen decimal = 1) Sonido", GoName: "audioRuido",
			Doc: "Crea un sonido de ruido blanco de `duracion` segundos (hasta 30), sin archivos: sirve para explosiones, pasos o golpes. Siempre genera el mismo ruido."},
		{Name: "pausar", Signature: "(reproduccion Reproduccion)", GoName: "pausar",
			Doc: "Pausa una reproducción en curso; puede reanudarse después con audio.reanudar."},
		{Name: "reanudar", Signature: "(reproduccion Reproduccion)", GoName: "reanudar",
			Doc: "Continúa una reproducción previamente pausada."},
		{Name: "detener", Signature: "(reproduccion Reproduccion)", GoName: "detener",
			Doc: "Detiene la reproducción por completo (a diferencia de pausar, no puede reanudarse desde donde iba)."},
		{Name: "volumen", Signature: "(reproduccion Reproduccion, valor decimal)", GoName: "volumen",
			Doc: "Cambia el volumen de una reproducción ya iniciada (0 = silencio, 1 = volumen original)."},
	},
}
