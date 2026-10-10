package stdlib

var ruidoModule = Module{
	Name: "ruido",
	Path: "std/mate/ruido",
	Doc:  "Ruido 2D pseudoaleatorio pero reproducible, útil para generar terrenos, texturas o variación orgánica a partir de una semilla.",
	Notes: []string{
		"Coordenadas finitas en (-2^52, 2^52), incluso en cada octava.",
		"Octavas: 1..16; persistencia: [0, 1]; lacunaridad finita >= 1.",
	},
	Functions: []Function{
		{Name: "suave", Signature: "(x decimal, y decimal, semilla entero = 0) decimal", GoName: "ruidoSuave",
			Doc: "Ruido 2D suave en (x, y), siempre en [0, 1] y reproducible: la misma semilla y coordenadas dan siempre el mismo resultado."},
		{Name: "fractal", Signature: "(x decimal, y decimal, semilla entero = 0, octavas entero = 4, persistencia decimal = 0.5, lacunaridad decimal = 2.0) decimal", GoName: "ruidoFractal",
			Doc: "Como ruido.suave pero sumando varias octavas para un detalle más rico (piensa en montañas con relieve, no solo colinas)."},
	},
}
