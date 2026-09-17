// Package gameapi is the compiler-owned, backend-independent game API catalog.
package gameapi

import (
	"hacha/internal/ast"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"strings"
)

const EbitenVersion = "v2.10.1"

type Function struct {
	Namespace, Name, Signature, GoName string
	Draw, Resource                     bool
}

var Functions = []Function{
	{"juego", "configuracion", `ancho num = 320, alto num = 180, titulo cadena = "Hacha", escala num = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps num = 60)`, "configuracion", false, false},
	{"mate", "absoluto", "valor num) num", "abs", false, false},
	{"mate", "minimo", "a num, b num) num", "min", false, false},
	{"mate", "maximo", "a num, b num) num", "max", false, false},
	{"mate", "limitar", "valor num, minimo num, maximo num) num", "limitar", false, false},
	{"mate", "interpolar", "a num, b num, t num) num", "interpolar", false, false},
	{"mate", "piso", "valor num) num", "piso", false, false},
	{"mate", "techo", "valor num) num", "techo", false, false},
	{"mate", "redondear", "valor num) num", "redondear", false, false},
	{"mate", "raiz", "valor num) num", "raiz", false, false},
	{"mate", "seno", "angulo num) num", "seno", false, false},
	{"mate", "coseno", "angulo num) num", "coseno", false, false},
	{"mate", "atan2", "y num, x num) num", "atan2", false, false},
	{"azar", "real", "minimo num, maximo num) num", "azarReal", false, false},
	{"azar", "entero", "minimo num, maximo num) num", "azarEntero", false, false},
	{"color", "rgba", "r num, g num, b num, a num = 255) Color", "rgba", false, false},
	{"graficos", "limpiar", "color Color)", "limpiar", true, false},
	{"graficos", "rectangulo", "x num, y num, ancho num, alto num, color Color, origen Vec2 = {}, rotacion num = 0)", "rectanguloXY", true, false},
	{"graficos", "rectangulo_v", "pos Vec2, tamano Vec2, color Color, origen Vec2 = {}, rotacion num = 0)", "rectangulo", true, false},
	{"graficos", "rectangulo_rect", "rect Rect, color Color, origen Vec2 = {}, rotacion num = 0)", "rectanguloRect", true, false},
	{"graficos", "circulo", "x num, y num, radio num, color Color)", "circuloXY", true, false},
	{"graficos", "circulo_v", "centro Vec2, radio num, color Color)", "circulo", true, false},
	{"graficos", "linea", "x1 num, y1 num, x2 num, y2 num, color Color, grosor num = 1)", "lineaXY", true, false},
	{"graficos", "linea_v", "a Vec2, b Vec2, color Color, grosor num = 1)", "linea", true, false},
	{"graficos", "imagen", "imagen Imagen, x num, y num, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)", "imagenXY", true, false},
	{"graficos", "imagen_v", "imagen Imagen, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)", "imagen", true, false},
	{"graficos", "imagen_rect", "imagen Imagen, destino Rect, origen Vec2 = {}, rotacion num = 0, tinte Color = .Blanco)", "imagenRect", true, false},
	{"graficos", "region", "imagen Imagen, fuente Rect, x num, y num, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)", "regionXY", true, false},
	{"graficos", "region_v", "imagen Imagen, fuente Rect, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)", "region", true, false},
	{"graficos", "region_rect", "imagen Imagen, fuente Rect, destino Rect, origen Vec2 = {}, rotacion num = 0, tinte Color = .Blanco)", "regionRect", true, false},
	{"graficos", "texto", "texto cadena, fuente Fuente, x num, y num, tamano num = 20, color Color = .Blanco, origen Vec2 = {}, rotacion num = 0)", "textoXY", true, false},
	{"graficos", "texto_v", "texto cadena, fuente Fuente, pos Vec2, tamano num = 20, color Color = .Blanco, origen Vec2 = {}, rotacion num = 0)", "texto", true, false},
	{"graficos", "texto_depuracion", "texto cadena, x num = 0, y num = 0)", "textoDepuracionXY", true, false},
	{"graficos", "texto_depuracion_v", "texto cadena, pos Vec2 = {})", "textoDepuracion", true, false},
	{"graficos", "tamano", ") Vec2", "tamano", false, false},
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
	{"audio", "reproducir", "sonido Sonido, volumen num = 1, bucle bool = falso) Reproduccion", "reproducir", false, false},
	{"audio", "pausar", "reproduccion Reproduccion)", "pausar", false, false},
	{"audio", "reanudar", "reproduccion Reproduccion)", "reanudar", false, false},
	{"audio", "detener", "reproduccion Reproduccion)", "detener", false, false},
	{"audio", "volumen", "reproduccion Reproduccion, valor num)", "volumen", false, false},
	{"ventana", "titulo", "titulo cadena)", "titulo", false, false},
	{"ventana", "pantalla_completa", "activa bool)", "pantallaCompleta", false, false},
	{"ventana", "tamano", ") Vec2", "ventanaTamano", false, false},
	{"tiempo", "fps", ") num", "fps", false, false},
	{"tiempo", "tps", ") num", "tps", false, false},
	{"recursos", "imagen", "ruta cadena) Imagen", "cargarImagen", false, true},
	{"recursos", "fuente", "ruta cadena) Fuente", "cargarFuente", false, true},
	{"recursos", "sonido", "ruta cadena) Sonido", "cargarSonido", false, true},
}

// Methods use Namespace as the receiver type, never as a callable namespace.
var Methods = []Function{
	{"Vec2", "longitud", ") num", "longitud", false, false},
	{"Vec2", "normalizado", ") Vec2", "normalizar", false, false},
	{"Vec2", "distancia_a", "otro Vec2) num", "distancia", false, false},
	{"Vec2", "producto_punto", "otro Vec2) num", "productoPunto", false, false},
	{"Vec2", "rotado", "angulo num) Vec2", "rotar", false, false},
	{"Vec2", "colision_circulo", "radio num, otro Vec2, radio_otro num) bool", "circulos", false, false},
	{"Rect", "interseca", "otro Rect) bool", "rectangulos", false, false},
	{"Rect", "contiene", "punto Vec2) bool", "contiene", false, false},
}

var Fields = map[string]string{
	"Vec2": "x num\ny num", "Rect": "pos Vec2\ntamano Vec2",
	"Camara2D": "pos Vec2\norigen Vec2\nzoom num\nrotacion num",
	"Color":    "", "Imagen": "", "Fuente": "", "Sonido": "", "Reproduccion": "", "Tecla": "", "BotonRaton": "",
}

// Native Go names are deliberately outside the exported user-name space.
func GoType(name string) string {
	if IsType(name) {
		return "_hg" + name
	}
	return ""
}
func IsType(name string) bool { _, ok := Fields[name]; return ok }
func IsValue(name string) bool {
	return name == "Vec2" || name == "Rect" || name == "Camara2D" || name == "Color" || name == "Tecla" || name == "BotonRaton"
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
	return IsType(name) || IsNamespace(name) || strings.HasPrefix(name, "_hg")
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
	tokens, err := lexer.Lex("<gameapi>", "fn "+f.Name+"("+f.Signature+"\n\timprimir(0)\n")
	if err != nil {
		panic(err)
	}
	for i := range tokens {
		tokens[i].Pos.Filename = "<gameapi>"
	}
	p, err := parser.Parse("<gameapi>", tokens)
	if err != nil {
		panic(err)
	}
	return p.Decls[0].(*ast.FuncDecl)
}
func TypeDeclaration(name string) *ast.TypeDecl {
	if Fields[name] == "" {
		return &ast.TypeDecl{Name: name}
	}
	source := "tipo " + name + "\n"
	if Fields[name] != "" {
		source += "\t" + strings.ReplaceAll(Fields[name], "\n", "\n\t") + "\n"
	}
	tokens, err := lexer.Lex("<gameapi>", source)
	if err != nil {
		panic(err)
	}
	for i := range tokens {
		tokens[i].Pos.Filename = "<gameapi>"
	}
	p, err := parser.Parse("<gameapi>", tokens)
	if err != nil {
		panic(err)
	}
	return p.Decls[0].(*ast.TypeDecl)
}

var Constants = map[string]map[string]string{
	"Color":      {"Blanco": "_hgColor{255,255,255,255}", "Negro": "_hgColor{0,0,0,255}", "Rojo": "_hgColor{255,0,0,255}", "Verde": "_hgColor{0,255,0,255}", "Azul": "_hgColor{0,0,255,255}", "Amarillo": "_hgColor{255,255,0,255}", "Magenta": "_hgColor{255,0,255,255}", "Cian": "_hgColor{0,255,255,255}", "Transparente": "_hgColor{}"},
	"Tecla":      {"Izquierda": "_hgTecla(ebiten.KeyArrowLeft)", "Derecha": "_hgTecla(ebiten.KeyArrowRight)", "Arriba": "_hgTecla(ebiten.KeyArrowUp)", "Abajo": "_hgTecla(ebiten.KeyArrowDown)", "Espacio": "_hgTecla(ebiten.KeySpace)", "Escape": "_hgTecla(ebiten.KeyEscape)", "Enter": "_hgTecla(ebiten.KeyEnter)", "Tab": "_hgTecla(ebiten.KeyTab)", "Retroceso": "_hgTecla(ebiten.KeyBackspace)", "Shift": "_hgTecla(ebiten.KeyShift)", "Control": "_hgTecla(ebiten.KeyControl)"},
	"BotonRaton": {"Izquierdo": "_hgBotonRaton(ebiten.MouseButtonLeft)", "Derecho": "_hgBotonRaton(ebiten.MouseButtonRight)", "Medio": "_hgBotonRaton(ebiten.MouseButtonMiddle)"},
}

func init() {
	for c := 'A'; c <= 'Z'; c++ {
		Constants["Tecla"][string(c)] = "_hgTecla(ebiten.Key" + string(c) + ")"
	}
}
