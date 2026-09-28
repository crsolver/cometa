package stdlib

import "regexp"

// UI declarations use the same native catalog as the rest of Pincel.
func installUI() {
 for name, fields := range map[string]string{
  "Contexto":"", "AmbitoUI":"", "Medida":"", "Alineacion":"",
  "Bordes":"izquierda decimal\nderecha decimal\narriba decimal\nabajo decimal",
  "Tema":"fondo Color\nnormal Color\nsobre Color\npresionado Color\nfoco Color\ndeshabilitado Color\ntexto Color\nespacio decimal",
 } { Fields[name]=fields; TypeModules[name]="ui" }
 typeName = regexp.MustCompile(`\b(Vec2|Rect|Color|Camara2D|Imagen|Fuente|Tecla|BotonRaton|Sonido|Reproduccion|Juego|Icono|Atlas|Contexto|AmbitoUI|Medida|Alineacion|Bordes|Tema)\b`)
 Constants["Alineacion"]=map[string]string{"Inicio":"_hgAlineacion(0)","Centro":"_hgAlineacion(1)","Final":"_hgAlineacion(2)"}
 add := func(name,signature,goName string,draw bool) { Functions=append(Functions,Function{"ui",name,signature,goName,draw,false}) }
 add("crear",") Contexto","uiCrear",false)
 add("cuadro","contexto Contexto) AmbitoUI","uiCuadro",false)
 add("pintar","contexto Contexto)","uiPintar",true)
 add("captura_raton","contexto Contexto) bool","uiCapturaRaton",false)
 add("captura_teclado","contexto Contexto) bool","uiCapturaTeclado",false)
 add("fijo","valor decimal) Medida","uiFijo",false)
 add("contenido","minimo decimal = 0, maximo decimal = 1000000000) Medida","uiContenido",false)
 add("expandir","minimo decimal = 0, maximo decimal = 1000000000) Medida","uiExpandir",false)
 add("porcentaje","fraccion decimal) Medida","uiPorcentaje",false)
 container := `id cadena, ancho Medida = contenido(), alto Medida = contenido(), espacio decimal = -1, relleno Bordes = {}, horizontal Alineacion = .Inicio, vertical Alineacion = .Inicio, desplazar bool = falso) AmbitoUI`
 add("fila",container,"uiFila",false); add("columna",container,"uiColumna",false); add("panel",container,"uiPanel",false)
 add("texto",`valor cadena, ancho Medida = contenido(), alto Medida = contenido(), color Color = .Transparente)`,"uiTexto",false)
 add("imagen",`imagen Imagen, ancho Medida = contenido(), alto Medida = contenido(), tinte Color = .Blanco)`,"uiImagen",false)
 add("boton",`id cadena, etiqueta cadena, ancho Medida = contenido(), alto Medida = contenido(), habilitado bool = verdadero) bool`,"uiBoton",false)
 add("casilla",`id cadena, valor bool, etiqueta cadena = "", ancho Medida = contenido(), alto Medida = contenido(), habilitado bool = verdadero) bool`,"uiCasilla",false)
 add("deslizador",`id cadena, valor decimal, minimo decimal = 0, maximo decimal = 1, ancho Medida = fijo(120), alto Medida = fijo(16), habilitado bool = verdadero) decimal`,"uiDeslizador",false)
 add("barra",`valor decimal, minimo decimal = 0, maximo decimal = 1, ancho Medida = fijo(120), alto Medida = fijo(10))`,"uiBarra",false)
 add("separador",`ancho Medida = expandir(), alto Medida = fijo(1))`,"uiSeparador",false)
 add("espacio",`ancho Medida = expandir(), alto Medida = expandir())`,"uiEspacio",false)
 add("campo_texto",`id cadena, valor cadena, ancho Medida = contenido(0, 200), alto Medida = contenido(), habilitado bool = verdadero) cadena`,"uiCampoTexto",false)
 add("retro",`escala entero = 1) Tema`,"uiRetro",false)
 add("fuente",`fuente Fuente, tamano decimal = 20) Tema`,"uiFuente",false)
 add("tema",`tema Tema) AmbitoUI`,"uiTema",false)
 Methods=append(Methods,Function{"AmbitoUI","entrar",")","uiEntrar",false,false},Function{"AmbitoUI","salir",")","uiSalir",false,false})
}
