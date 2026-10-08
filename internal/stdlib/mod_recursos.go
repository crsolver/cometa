package stdlib

var recursosModule = Module{
	Name: "recursos",
	Path: "std/pincel/recursos",
	Doc:  "Carga imágenes, fuentes, sonidos y textos desde archivos, con la ruta relativa al archivo que los usa.",
	Functions: []Function{
		{Name: "imagen", Signature: "(ruta cadena) Imagen", GoName: "cargarImagen", Resource: true,
			Doc: "Carga una imagen desde `ruta` (relativa al archivo que llama a esta función)."},
		{Name: "fuente", Signature: "(ruta cadena) Fuente", GoName: "cargarFuente", Resource: true,
			Doc: "Carga una fuente desde `ruta` (relativa al archivo que llama a esta función)."},
		{Name: "sonido", Signature: "(ruta cadena) Sonido", GoName: "cargarSonido", Resource: true,
			Doc: "Carga un sonido desde `ruta` (relativa al archivo que llama a esta función)."},
		{Name: "texto", Signature: "(ruta cadena) cadena", GoName: "cargarTexto", Resource: true,
			Doc: "Lee un archivo de texto UTF-8 desde `ruta` (relativa al archivo que llama a esta función) y lo devuelve como cadena; por ejemplo un mapa para rejilla.desde_csv."},
	},
}
