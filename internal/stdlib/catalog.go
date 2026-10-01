// Package stdlib is the compiler-owned, backend-independent game API catalog.
package stdlib

import (
	"cometa/internal/ast"
	"cometa/internal/lexer"
	"cometa/internal/parser"
	"strconv"
	"strings"
)

const EbitenVersion = "v2.10.1"

type Function struct {
	Namespace, Name, Signature, GoName string
	Draw, Resource                     bool
}

var Functions = []Function{
	{"pruebas", "afirmar", `condicion bool, mensaje cadena = "")`, "pruebasAfirmar", false, false},
	{"pruebas", "igual", `esperado cadena, obtenido cadena, mensaje cadena = "")`, "pruebasIgual", false, false},
	{"pruebas", "casi_igual", `esperado decimal, obtenido decimal, tolerancia decimal = 0.000001, mensaje cadena = "")`, "pruebasCasiIgual", false, false},
	{"pruebas", "fallar", "mensaje cadena)", "pruebasFallar", false, false},
	{"ruido", "suave", "x decimal, y decimal, semilla entero = 0) decimal", "ruidoSuave", false, false},
	{"ruido", "fractal", "x decimal, y decimal, semilla entero = 0, octavas entero = 4, persistencia decimal = 0.5, lacunaridad decimal = 2.0) decimal", "ruidoFractal", false, false},
	{"curvas", "lineal", "progreso decimal) decimal", "curva_lineal", false, false},
	{"curvas", "cuadratica_entrada", "progreso decimal) decimal", "curva_cuadratica_entrada", false, false},
	{"curvas", "cuadratica_salida", "progreso decimal) decimal", "curva_cuadratica_salida", false, false},
	{"curvas", "cuadratica_entrada_salida", "progreso decimal) decimal", "curva_cuadratica_entrada_salida", false, false},
	{"curvas", "cubica_entrada", "progreso decimal) decimal", "curva_cubica_entrada", false, false},
	{"curvas", "cubica_salida", "progreso decimal) decimal", "curva_cubica_salida", false, false},
	{"curvas", "cubica_entrada_salida", "progreso decimal) decimal", "curva_cubica_entrada_salida", false, false},
	{"curvas", "cuartica_entrada", "progreso decimal) decimal", "curva_cuartica_entrada", false, false},
	{"curvas", "cuartica_salida", "progreso decimal) decimal", "curva_cuartica_salida", false, false},
	{"curvas", "cuartica_entrada_salida", "progreso decimal) decimal", "curva_cuartica_entrada_salida", false, false},
	{"curvas", "quintica_entrada", "progreso decimal) decimal", "curva_quintica_entrada", false, false},
	{"curvas", "quintica_salida", "progreso decimal) decimal", "curva_quintica_salida", false, false},
	{"curvas", "quintica_entrada_salida", "progreso decimal) decimal", "curva_quintica_entrada_salida", false, false},
	{"curvas", "senoidal_entrada", "progreso decimal) decimal", "curva_senoidal_entrada", false, false},
	{"curvas", "senoidal_salida", "progreso decimal) decimal", "curva_senoidal_salida", false, false},
	{"curvas", "senoidal_entrada_salida", "progreso decimal) decimal", "curva_senoidal_entrada_salida", false, false},
	{"curvas", "circular_entrada", "progreso decimal) decimal", "curva_circular_entrada", false, false},
	{"curvas", "circular_salida", "progreso decimal) decimal", "curva_circular_salida", false, false},
	{"curvas", "circular_entrada_salida", "progreso decimal) decimal", "curva_circular_entrada_salida", false, false},
	{"curvas", "exponencial_entrada", "progreso decimal) decimal", "curva_exponencial_entrada", false, false},
	{"curvas", "exponencial_salida", "progreso decimal) decimal", "curva_exponencial_salida", false, false},
	{"curvas", "exponencial_entrada_salida", "progreso decimal) decimal", "curva_exponencial_entrada_salida", false, false},
	{"curvas", "elastica_entrada", "progreso decimal) decimal", "curva_elastica_entrada", false, false},
	{"curvas", "elastica_salida", "progreso decimal) decimal", "curva_elastica_salida", false, false},
	{"curvas", "elastica_entrada_salida", "progreso decimal) decimal", "curva_elastica_entrada_salida", false, false},
	{"curvas", "retroceso_entrada", "progreso decimal) decimal", "curva_retroceso_entrada", false, false},
	{"curvas", "retroceso_salida", "progreso decimal) decimal", "curva_retroceso_salida", false, false},
	{"curvas", "retroceso_entrada_salida", "progreso decimal) decimal", "curva_retroceso_entrada_salida", false, false},
	{"curvas", "rebote_entrada", "progreso decimal) decimal", "curva_rebote_entrada", false, false},
	{"curvas", "rebote_salida", "progreso decimal) decimal", "curva_rebote_salida", false, false},
	{"curvas", "rebote_entrada_salida", "progreso decimal) decimal", "curva_rebote_entrada_salida", false, false},
	{"pincel", "ejecutar", `instancia Juego, ancho entero = 320, alto entero = 180, titulo cadena = "Cometa", escala decimal = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = falso, retro bool = falso) !`, "ejecutar", false, false},
	{"pincel", "salir", ")", "salir", false, false},
	{"retro", "texto", "texto cadena, x entero, y entero, escala entero = 1, color Color = .Blanco)", "retroTexto", true, false},
	{"retro", "ancho_texto", "texto cadena, escala entero = 1) entero", "retroAnchoTexto", false, false},
	{"retro", "icono", "icono Icono, x entero, y entero, escala entero = 1, color Color = .Blanco)", "retroIcono", true, false},
	{"retro", "glifo", "indice entero, x entero, y entero, escala entero = 1, color Color = .Blanco, atlas Atlas = .Dungeon)", "retroGlifo", true, false},
	{"retro", "ejecutar", `instancia Juego, ancho entero = 320, alto entero = 200, titulo cadena = "Cometa", escala decimal = 4, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = verdadero, retro bool = falso) !`, "retroEjecutar", false, false},
	{"mate", "absoluto", "valor decimal) decimal", "abs", false, false},
	{"mate", "minimo", "a decimal, b decimal) decimal", "min", false, false},
	{"mate", "maximo", "a decimal, b decimal) decimal", "max", false, false},
	{"mate", "limitar", "valor decimal, minimo decimal, maximo decimal) decimal", "limitar", false, false},
	{"mate", "interpolar", "a decimal, b decimal, t decimal) decimal", "interpolar", false, false},
	{"mate", "piso", "valor decimal) entero", "piso", false, false},
	{"mate", "techo", "valor decimal) entero", "techo", false, false},
	{"mate", "redondear", "valor decimal) entero", "redondear", false, false},
	{"mate", "raiz", "valor decimal) decimal", "raiz", false, false},
	{"mate", "seno", "angulo decimal) decimal", "seno", false, false},
	{"mate", "coseno", "angulo decimal) decimal", "coseno", false, false},
	{"mate", "atan2", "y decimal, x decimal) decimal", "atan2", false, false},
	{"mate", "tangente", "angulo decimal) decimal", "tangente", false, false},
	{"mate", "potencia", "base decimal, exponente decimal) decimal", "potencia", false, false},
	{"mate", "signo", "valor decimal) decimal", "signo", false, false},
	{"mate", "distancia", "x1 decimal, y1 decimal, x2 decimal, y2 decimal) decimal", "distanciaXY", false, false},
	{"mate", "angulo", "x1 decimal, y1 decimal, x2 decimal, y2 decimal) decimal", "anguloXY", false, false},
	{"mate", "radianes", "grados decimal) decimal", "radianes", false, false},
	{"mate", "grados", "radianes decimal) decimal", "grados", false, false},
	{"azar", "real", "minimo decimal, maximo decimal) decimal", "azarReal", false, false},
	{"azar", "entero", "minimo entero, maximo entero) entero", "azarEntero", false, false},
	{"azar", "semilla", "valor entero)", "azarSemilla", false, false},
	{"datos", "guardar", "clave cadena, valor cadena) !", "datosGuardar", false, false},
	{"datos", "leer", "clave cadena) cadena?", "datosLeer", false, false},
	{"datos", "borrar", "clave cadena) !", "datosBorrar", false, false},
	{"datos", "juego", "nombre cadena)", "datosJuego", false, false},
	{"datos", "claves", ") [cadena]", "datosClaves", false, false},
	{"datos", "guardar_entero", "clave cadena, valor entero) !", "datosGuardarEntero", false, false},
	{"datos", "leer_entero", "clave cadena) entero?", "datosLeerEntero", false, false},
	{"datos", "guardar_decimal", "clave cadena, valor decimal) !", "datosGuardarDecimal", false, false},
	{"datos", "leer_decimal", "clave cadena) decimal?", "datosLeerDecimal", false, false},
	{"rejilla", "nueva", "columnas entero, filas entero, valor entero = 0) Rejilla", "rejillaNueva", false, false},
	{"rejilla", "desde_texto", "filas [cadena], simbolos [cadena: entero]) Rejilla", "rejillaDesdeTexto", false, false},
	{"rejilla", "dibujar", "mapa Rejilla, hoja Hoja, pos Vec2 = {}, escala Vec2 = {x: 1, y: 1}, tinte Color = .Blanco, animaciones [entero: [entero]] = [:], cuadros_por_segundo decimal = 8)", "rejillaDibujar", true, false},
	{"color", "rgba", "r entero, g entero, b entero, a entero = 255) Color", "rgba", false, false},
	{"graficos", "limpiar", "color Color)", "limpiar", true, false},
	{"graficos", "rectangulo", "x decimal, y decimal, ancho decimal, alto decimal, color Color, origen Vec2 = {}, rotacion decimal = 0)", "rectanguloXY", true, false},
	{"graficos", "rectangulo_v", "pos Vec2, tamano Vec2, color Color, origen Vec2 = {}, rotacion decimal = 0)", "rectangulo", true, false},
	{"graficos", "rectangulo_rect", "rect Rect, color Color, origen Vec2 = {}, rotacion decimal = 0)", "rectanguloRect", true, false},
	{"graficos", "circulo", "x decimal, y decimal, radio decimal, color Color)", "circuloXY", true, false},
	{"graficos", "circulo_v", "centro Vec2, radio decimal, color Color)", "circulo", true, false},
	{"graficos", "linea", "x1 decimal, y1 decimal, x2 decimal, y2 decimal, color Color, grosor decimal = 1)", "lineaXY", true, false},
	{"graficos", "linea_v", "a Vec2, b Vec2, color Color, grosor decimal = 1)", "linea", true, false},
	{"graficos", "imagen", "imagen Imagen, x decimal, y decimal, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "imagenXY", true, false},
	{"graficos", "imagen_v", "imagen Imagen, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "imagen", true, false},
	{"graficos", "imagen_rect", "imagen Imagen, destino Rect, origen Vec2 = {}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "imagenRect", true, false},
	{"graficos", "region", "imagen Imagen, fuente Rect, x decimal, y decimal, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "regionXY", true, false},
	{"graficos", "region_v", "imagen Imagen, fuente Rect, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "region", true, false},
	{"graficos", "region_rect", "imagen Imagen, fuente Rect, destino Rect, origen Vec2 = {}, rotacion decimal = 0, tinte Color = .Blanco, relleno Color = .Transparente)", "regionRect", true, false},
	{"graficos", "texto", "texto cadena, fuente Fuente, x decimal, y decimal, tamano decimal = 20, color Color = .Blanco, origen Vec2 = {}, rotacion decimal = 0)", "textoXY", true, false},
	{"graficos", "texto_v", "texto cadena, fuente Fuente, pos Vec2, tamano decimal = 20, color Color = .Blanco, origen Vec2 = {}, rotacion decimal = 0)", "texto", true, false},
	{"graficos", "hoja", "imagen Imagen, ancho_cuadro entero, alto_cuadro entero) Hoja", "hoja", false, false},
	{"graficos", "cuadros", "hoja Hoja) entero", "cuadros", false, false},
	{"graficos", "tamano_cuadro", "hoja Hoja) Vec2", "tamanoCuadro", false, false},
	{"graficos", "cuadro", "hoja Hoja, indice entero, x decimal, y decimal, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, espejo_h bool = falso, espejo_v bool = falso, relleno Color = .Transparente)", "cuadroXY", true, false},
	{"graficos", "cuadro_v", "hoja Hoja, indice entero, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte Color = .Blanco, espejo_h bool = falso, espejo_v bool = falso, relleno Color = .Transparente)", "cuadro", true, false},
	{"graficos", "medir_texto", "texto cadena, fuente Fuente, tamano decimal = 20) Vec2", "medirTexto", false, false},
	{"graficos", "fuente_predeterminada", ") Fuente", "fuentePredeterminada", false, false},
	{"graficos", "texto_depuracion", "texto cadena, x decimal = 0, y decimal = 0)", "textoDepuracionXY", true, false},
	{"graficos", "texto_depuracion_v", "texto cadena, pos Vec2 = {})", "textoDepuracion", true, false},
	{"graficos", "tamano", ") Vec2", "tamano", false, false},
	{"graficos", "recortar", "rect Rect)", "recortar", true, false},
	{"graficos", "quitar_recorte", ")", "quitarRecorte", true, false},
	{"graficos", "usar_camara", "camara Camara2D)", "usarCamara", true, false},
	{"graficos", "restablecer_camara", ")", "restablecerCamara", true, false},
	{"entrada", "tecla_mantenida", "tecla Tecla) bool", "teclaMantenida", false, false},
	{"entrada", "tecla_presionada", "tecla Tecla) bool", "teclaPresionada", false, false},
	{"entrada", "tecla_soltada", "tecla Tecla) bool", "teclaSoltada", false, false},
	{"entrada", "raton_mantenido", "boton BotonRaton) bool", "ratonMantenido", false, false},
	{"entrada", "raton_presionado", "boton BotonRaton) bool", "ratonPresionado", false, false},
	{"entrada", "raton_soltado", "boton BotonRaton) bool", "ratonSoltado", false, false},
	{"entrada", "posicion_raton", ") Vec2", "posicionRaton", false, false},
	{"entrada", "rueda", ") Vec2", "rueda", false, false},
	{"entrada", "mandos", ") entero", "mandos", false, false},
	{"entrada", "mando_boton", "mando entero, boton BotonMando) bool", "mandoBoton", false, false},
	{"entrada", "mando_boton_presionado", "mando entero, boton BotonMando) bool", "mandoBotonPresionado", false, false},
	{"entrada", "mando_eje", "mando entero, eje EjeMando, zona_muerta decimal = 0.15) decimal", "mandoEje", false, false},
	{"audio", "reproducir", "sonido Sonido, volumen decimal = 1, bucle bool = falso) Reproduccion", "reproducir", false, false},
	{"audio", "pausar", "reproduccion Reproduccion)", "pausar", false, false},
	{"audio", "reanudar", "reproduccion Reproduccion)", "reanudar", false, false},
	{"audio", "detener", "reproduccion Reproduccion)", "detener", false, false},
	{"audio", "volumen", "reproduccion Reproduccion, valor decimal)", "volumen", false, false},
	{"ventana", "titulo", "titulo cadena)", "titulo", false, false},
	{"ventana", "pantalla_completa", "activa bool)", "pantallaCompleta", false, false},
	{"ventana", "tamano", ") Vec2", "ventanaTamano", false, false},
	{"ventana", "cursor", "visible bool)", "ventanaCursor", false, false},
	{"ventana", "icono", "imagen Imagen)", "ventanaIcono", false, false},
	{"ventana", "es_pantalla_completa", ") bool", "esPantallaCompleta", false, false},
	{"tiempo", "fps", ") decimal", "fps", false, false},
	{"tiempo", "tps", ") decimal", "tps", false, false},
	{"tiempo", "temporizador", "duracion decimal, bucle bool = falso) Temporizador", "temporizador", false, false},
	{"lienzo", "nuevo", "ancho entero, alto entero, color Color = .Transparente) Imagen", "lienzoNuevo", false, false},
	{"lienzo", "desde_texto", "filas [cadena], colores [cadena: Color]) Imagen", "lienzoDesdeTexto", false, false},
	{"lienzo", "hoja_desde_texto", "cuadros [[cadena]], colores [cadena: Color]) [Imagen]", "lienzoHojaDesdeTexto", false, false},
	{"lienzo", "ancho", "imagen Imagen) entero", "lienzoAncho", false, false},
	{"lienzo", "alto", "imagen Imagen) entero", "lienzoAlto", false, false},
	{"lienzo", "copiar", "imagen Imagen) Imagen", "lienzoCopiar", false, false},
	{"lienzo", "limpiar", "imagen Imagen, color Color = .Transparente)", "lienzoLimpiar", false, false},
	{"lienzo", "pixel", "imagen Imagen, x entero, y entero, color Color)", "lienzoPixel", false, false},
	{"lienzo", "leer_pixel", "imagen Imagen, x entero, y entero) Color?", "lienzoLeerPixel", false, false},
	{"lienzo", "rect", "imagen Imagen, x entero, y entero, ancho entero, alto entero, color Color, relleno bool = verdadero)", "lienzoRect", false, false},
	{"lienzo", "linea", "imagen Imagen, x1 entero, y1 entero, x2 entero, y2 entero, color Color)", "lienzoLinea", false, false},
	{"lienzo", "circulo", "imagen Imagen, x entero, y entero, radio entero, color Color, relleno bool = verdadero)", "lienzoCirculo", false, false},
	{"lienzo", "rellenar", "imagen Imagen, x entero, y entero, color Color)", "lienzoRellenar", false, false},
	{"lienzo", "pegar", "destino Imagen, fuente Imagen, x entero, y entero, espejo_h bool = falso, espejo_v bool = falso)", "lienzoPegar", false, false},
	{"lienzo", "guardar", "imagen Imagen, ruta cadena, escala entero = 1) !", "lienzoGuardar", false, false},
	{"recursos", "imagen", "ruta cadena) Imagen", "cargarImagen", false, true},
	{"recursos", "fuente", "ruta cadena) Fuente", "cargarFuente", false, true},
	{"recursos", "sonido", "ruta cadena) Sonido", "cargarSonido", false, true},
}

// Methods use Namespace as the receiver type, never as a callable namespace.
var Methods = []Function{
	{"Vec2", "longitud", ") decimal", "longitud", false, false},
	{"Vec2", "normalizado", ") Vec2", "normalizar", false, false},
	{"Vec2", "distancia_a", "otro Vec2) decimal", "distancia", false, false},
	{"Vec2", "producto_punto", "otro Vec2) decimal", "productoPunto", false, false},
	{"Vec2", "rotado", "angulo decimal) Vec2", "rotar", false, false},
	{"Vec2", "colision_circulo", "radio decimal, otro Vec2, radio_otro decimal) bool", "circulos", false, false},
	{"Rect", "interseca", "otro Rect) bool", "rectangulos", false, false},
	{"Rect", "contiene", "punto Vec2) bool", "contiene", false, false},
	{"Vec2", "angulo", ") decimal", "anguloVec", false, false},
	{"Vec2", "interpolar", "otro Vec2, t decimal) Vec2", "interpolarVec", false, false},
	{"Vec2", "reflejar", "normal Vec2) Vec2", "reflejar", false, false},
	{"Vec2", "perpendicular", ") Vec2", "perpendicular", false, false},
	{"Rect", "centro", ") Vec2", "centroRect", false, false},
	{"Rect", "interseccion", "otro Rect) Rect", "interseccionRect", false, false},
	{"Rect", "desplazado", "delta Vec2) Rect", "desplazadoRect", false, false},
	{"Rect", "colision_circulo", "centro Vec2, radio decimal) bool", "circuloRect", false, false},
	{"Rejilla", "columnas", ") entero", "columnas", false, false},
	{"Rejilla", "filas", ") entero", "filas", false, false},
	{"Rejilla", "obtener", "x entero, y entero) entero?", "obtener", false, false},
	{"Rejilla", "poner", "x entero, y entero, valor entero) bool", "poner", false, false},
	{"Rejilla", "rellenar", "valor entero)", "rellenar", false, false},
	{"Rejilla", "choca", "area Rect, tamano_celda Vec2, solidos [entero]) bool", "choca", false, false},
	{"Rejilla", "mover", "area Rect, tamano_celda Vec2, delta Vec2, solidos [entero], plataformas [entero] = []) Vec2", "mover", false, false},
	{"Rejilla", "celda_en", "punto Vec2, tamano_celda Vec2) entero?", "celdaEn", false, false},
	{"Rejilla", "valores_en", "area Rect, tamano_celda Vec2) [entero]", "valoresEn", false, false},
	{"Camara2D", "a_mundo", "punto Vec2) Vec2", "aMundo", false, false},
	{"Camara2D", "a_pantalla", "punto Vec2) Vec2", "aPantalla", false, false},
	{"Camara2D", "siguiendo", "objetivo Vec2, suavizado decimal) Camara2D", "siguiendo", false, false},
	{"Camara2D", "limitada", "mundo Rect) Camara2D", "limitada", false, false},
	{"Camara2D", "sacudida", "intensidad decimal) Camara2D", "sacudida", false, false},
	{"Camara2D", "visible", ") Rect", "visible", false, false},
	{"Temporizador", "avanzar", "dt decimal) bool", "avanzar", false, false},
	{"Temporizador", "terminado", ") bool", "terminado", false, false},
	{"Temporizador", "terminar", ")", "terminar", false, false},
	{"Temporizador", "reiniciar", ")", "reiniciar", false, false},
	{"Temporizador", "restante", ") decimal", "restante", false, false},
	{"Temporizador", "progreso", ") decimal", "progreso", false, false},
}

var Fields = map[string]string{
	"Icono": "", "Atlas": "",
	"Vec2": "x decimal\ny decimal", "Rect": "pos Vec2\ntamano Vec2",
	"Camara2D": "pos Vec2\norigen Vec2\nzoom decimal\nrotacion decimal",
	"Color":    "", "Imagen": "", "Fuente": "", "Sonido": "", "Reproduccion": "", "Tecla": "", "BotonRaton": "",
	"BotonMando": "", "EjeMando": "", "Hoja": "", "Rejilla": "", "Temporizador": "",
}

// Native Go names are deliberately outside the exported user-name space.
func GoType(name string) string {
	if IsType(name) {
		return "_hg" + PublicName(name)
	}
	return ""
}
func IsType(name string) bool { _, ok := Fields[name]; return ok }
func IsValue(name string) bool {
	if name == Symbol("Bordes") || name == Symbol("Alineacion") {
		return true
	}
	if name == Symbol("Icono") || name == Symbol("Atlas") {
		return true
	}
	return name == Symbol("Vec2") || name == Symbol("Rect") || name == Symbol("Camara2D") || name == Symbol("Color") || name == Symbol("Tecla") || name == Symbol("BotonRaton") || name == Symbol("BotonMando") || name == Symbol("EjeMando")
}
func IsNamespace(name string) bool {
	for _, f := range Functions {
		if f.Namespace == name {
			return true
		}
	}
	return false
}
func Reserved(name string) bool {
	return name == "entero" || name == "decimal" || strings.HasPrefix(name, "_hg") || strings.HasPrefix(name, "__std_")
}
func Lookup(namespace, name string) (Function, bool) {
	for _, f := range Functions {
		if f.Namespace == namespace && f.Name == name {
			return f, true
		}
	}
	return Function{}, false
}
func (f Function) Declaration() *ast.FuncDecl {
	tokens, err := lexer.Lex("<stdlib>", "fn "+f.Name+"("+f.Signature+"\n\timprimir(0)\n")
	if err != nil {
		panic(err)
	}
	for i := range tokens {
		tokens[i].Pos.Filename = "<stdlib>"
	}
	p, err := parser.Parse("<stdlib>", tokens)
	if err != nil {
		panic(err)
	}
	d := p.Decls[0].(*ast.FuncDecl)
	d.Public = true
	if f.Namespace == "ui" {
		for _, param := range d.Params {
			if call, ok := param.Default.(*ast.CallExpr); ok {
				if id, ok := call.Callee.(*ast.IdentExpr); ok {
					id.Name = FunctionSymbol("ui", id.Name)
				}
			}
		}
	}
	relocate(d, f.Namespace, "fn "+f.Name+"(")
	BindTypes(d)
	d.Name = FunctionSymbol(f.Namespace, f.Name)
	return d
}
func TypeDeclaration(name string) *ast.TypeDecl {
	if Fields[name] == "" {
		return &ast.TypeDecl{Name: name, Public: true}
	}
	source := "pub tipo " + PublicName(name) + "\n"
	if Fields[name] != "" {
		source += "\tpub " + strings.ReplaceAll(Fields[name], "\n", "\n\tpub ") + "\n"
	}
	tokens, err := lexer.Lex("<stdlib>", source)
	if err != nil {
		panic(err)
	}
	for i := range tokens {
		tokens[i].Pos.Filename = "<stdlib>"
	}
	p, err := parser.Parse("<stdlib>", tokens)
	if err != nil {
		panic(err)
	}
	d := p.Decls[0].(*ast.TypeDecl)
	d.Public = true
	for _, field := range d.Fields {
		field.Public = true
	}
	relocate(d, TypeModules[PublicName(name)], "pub tipo "+PublicName(name))
	d.Name = name
	BindTypes(d)
	return d
}

var Constants = map[string]map[string]string{
	"Atlas": {"Dungeon": "_hgAtlas(0)", "ASCII": "_hgAtlas(1)"},
	"Icono": {"Corazon": "_hgIcono(3)", "CorazonVacio": "_hgIcono(19)", "Espada": "_hgIcono(156)", "Escudo": "_hgIcono(157)", "Llave": "_hgIcono(175)", "Calavera": "_hgIcono(237)", "Derecha": "_hgIcono(12)", "Izquierda": "_hgIcono(13)", "Arriba": "_hgIcono(14)", "Abajo": "_hgIcono(15)"},
	"Color": {
		"Negro":         "_hgColor{0,0,0,255}",
		"Rojo":          "_hgColor{255,0,0,255}",
		"Verde":         "_hgColor{0,255,0,255}",
		"Azul":          "_hgColor{0,0,255,255}",
		"Amarillo":      "_hgColor{255,255,0,255}",
		"Cian":          "_hgColor{0,255,255,255}",
		"Magenta":       "_hgColor{255,0,255,255}",
		"VerdePino":     "_hgColor{0,43,36,255}",
		"AzulNoche":     "_hgColor{24,30,42,255}",
		"Oliva":         "_hgColor{84,106,0,255}",
		"Indigo":        "_hgColor{25,17,74,255}",
		"VioletaOscuro": "_hgColor{47,42,76,255}",
		"Carbon":        "_hgColor{68,63,65,255}",
		"Petroleo":      "_hgColor{8,66,72,255}",
		"VerdeBosque":   "_hgColor{40,84,72,255}",
		"Gris":          "_hgColor{82,82,76,255}",
		"Marron":        "_hgColor{115,97,80,255}",
		"Caqui":         "_hgColor{119,120,91,255}",
		"Malva":         "_hgColor{94,82,107,255}",
		"Ocre":          "_hgColor{130,91,49,255}",
		"Carmesi":       "_hgColor{181,59,89,255}",
		"Rosa":          "_hgColor{255,87,119,255}",
		"Ambar":         "_hgColor{255,185,21,255}",
		"Crema":         "_hgColor{255,224,119,255}",
		"AzulIndigo":    "_hgColor{67,62,166,255}",
		"Cobalto":       "_hgColor{71,114,191,255}",
		"Violeta":       "_hgColor{150,102,238,255}",
		"Hierba":        "_hgColor{87,176,103,255}",
		"Celeste":       "_hgColor{153,215,229,255}",
		"Marfil":        "_hgColor{255,249,228,255}",
		"Blanco":        "_hgColor{255,255,255,255}",
		"Transparente":  "_hgColor{}",
	},
	"Tecla":      {"Izquierda": "_hgTecla(ebiten.KeyArrowLeft)", "Derecha": "_hgTecla(ebiten.KeyArrowRight)", "Arriba": "_hgTecla(ebiten.KeyArrowUp)", "Abajo": "_hgTecla(ebiten.KeyArrowDown)", "Espacio": "_hgTecla(ebiten.KeySpace)", "Escape": "_hgTecla(ebiten.KeyEscape)", "Enter": "_hgTecla(ebiten.KeyEnter)", "Tab": "_hgTecla(ebiten.KeyTab)", "Retroceso": "_hgTecla(ebiten.KeyBackspace)", "Shift": "_hgTecla(ebiten.KeyShift)", "Control": "_hgTecla(ebiten.KeyControl)"},
	"BotonMando": {
		"A": "_hgBotonMando(ebiten.StandardGamepadButtonRightBottom)", "B": "_hgBotonMando(ebiten.StandardGamepadButtonRightRight)",
		"X": "_hgBotonMando(ebiten.StandardGamepadButtonRightLeft)", "Y": "_hgBotonMando(ebiten.StandardGamepadButtonRightTop)",
		"Arriba": "_hgBotonMando(ebiten.StandardGamepadButtonLeftTop)", "Abajo": "_hgBotonMando(ebiten.StandardGamepadButtonLeftBottom)",
		"Izquierda": "_hgBotonMando(ebiten.StandardGamepadButtonLeftLeft)", "Derecha": "_hgBotonMando(ebiten.StandardGamepadButtonLeftRight)",
		"HombroIzquierdo": "_hgBotonMando(ebiten.StandardGamepadButtonFrontTopLeft)", "HombroDerecho": "_hgBotonMando(ebiten.StandardGamepadButtonFrontTopRight)",
		"GatilloIzquierdo": "_hgBotonMando(ebiten.StandardGamepadButtonFrontBottomLeft)", "GatilloDerecho": "_hgBotonMando(ebiten.StandardGamepadButtonFrontBottomRight)",
		"Atras": "_hgBotonMando(ebiten.StandardGamepadButtonCenterLeft)", "Inicio": "_hgBotonMando(ebiten.StandardGamepadButtonCenterRight)",
		"PalancaIzquierda": "_hgBotonMando(ebiten.StandardGamepadButtonLeftStick)", "PalancaDerecha": "_hgBotonMando(ebiten.StandardGamepadButtonRightStick)",
	},
	"EjeMando": {
		"IzquierdoX": "_hgEjeMando(ebiten.StandardGamepadAxisLeftStickHorizontal)", "IzquierdoY": "_hgEjeMando(ebiten.StandardGamepadAxisLeftStickVertical)",
		"DerechoX": "_hgEjeMando(ebiten.StandardGamepadAxisRightStickHorizontal)", "DerechoY": "_hgEjeMando(ebiten.StandardGamepadAxisRightStickVertical)",
	},
	"BotonRaton": {"Izquierdo": "_hgBotonRaton(ebiten.MouseButtonLeft)", "Derecho": "_hgBotonRaton(ebiten.MouseButtonRight)", "Medio": "_hgBotonRaton(ebiten.MouseButtonMiddle)"},
}

func init() {
	installUI()
	fields := map[string]string{}
	for name, value := range Fields {
		fields[Symbol(name)] = value
	}
	Fields = fields
	constants := map[string]map[string]string{}
	for name, value := range Constants {
		constants[Symbol(name)] = value
	}
	Constants = constants
	for i := range Methods {
		Methods[i].Namespace = Symbol(Methods[i].Namespace)
	}
	digits := []string{"Cero", "Uno", "Dos", "Tres", "Cuatro", "Cinco", "Seis", "Siete", "Ocho", "Nueve"}
	for i, name := range digits {
		Constants[Symbol("Tecla")][name] = "_hgTecla(ebiten.KeyDigit" + string(rune('0'+i)) + ")"
	}
	for i := 1; i <= 12; i++ {
		Constants[Symbol("Tecla")]["F"+itoa(i)] = "_hgTecla(ebiten.KeyF" + itoa(i) + ")"
	}
	for name, key := range map[string]string{
		"Alt": "Alt", "Suprimir": "Delete", "Inicio": "Home", "Fin": "End", "RePag": "PageUp", "AvPag": "PageDown",
		"ShiftIzquierdo": "ShiftLeft", "ShiftDerecho": "ShiftRight",
		"ControlIzquierdo": "ControlLeft", "ControlDerecho": "ControlRight",
	} {
		Constants[Symbol("Tecla")][name] = "_hgTecla(ebiten.Key" + key + ")"
	}
	for c := 'A'; c <= 'Z'; c++ {
		Constants[Symbol("Tecla")][string(c)] = "_hgTecla(ebiten.Key" + string(c) + ")"
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
