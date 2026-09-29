package stdlib

// NamespaceDocs holds a short, friendly introduction for each standard
// library module. It's shown once at the top of the module's navigable
// declarations (cometa-std:///std/...) and feeds its hover tooltip.
var NamespaceDocs = map[string]string{
	"mate":     "Funciones matemáticas de uso general: valores absolutos, redondeo, trigonometría y el tipo Vec2/Rect para trabajar con posiciones y áreas.",
	"azar":     "Números aleatorios simples para juegos: un real dentro de un rango o un entero dentro de un rango.",
	"color":    "El tipo Color y una forma de construir colores propios a partir de sus componentes r, g, b, a.",
	"graficos": "Dibuja formas, imágenes y texto en pantalla, y controla la cámara 2D. Todo lo que ves en pantalla pasa por aquí.",
	"entrada":  "Lee el teclado y el ratón: teclas mantenidas, presionadas o soltadas, botones del ratón y su posición.",
	"audio":    "Reproduce y controla sonidos: reproducir, pausar, reanudar, detener y ajustar el volumen.",
	"ventana":  "Controla el título y el modo de pantalla completa de la ventana del juego.",
	"tiempo":   "Cuadros e imágenes por segundo reales del juego en ejecución, útiles para medir rendimiento.",
	"recursos": "Carga imágenes, fuentes y sonidos desde archivos, con la ruta relativa al archivo que los usa.",
	"lienzo":   "Crea y edita imágenes (graficos.Imagen) directamente en código: píxel a píxel, con texto o combinando otras imágenes. Ideal para arte generado sin necesitar archivos externos. Ver docs/lienzo.md.",
	"pincel":   "El núcleo del motor de juegos: pincel.ejecutar arranca el bucle de juego a partir de un valor que implemente pincel.Juego.",
	"retro":    "Sprites y texto de estilo retro (DUNGEON.mode), listos para usar sin cargar ningún archivo. Ver examples/pincel/06_retro.cometa.",
	"curvas":   "31 funciones de easing puras (sin estado) para animar valores de forma más natural que una interpolación lineal. Ver docs/curvas.md.",
	"ruido":    "Ruido 2D pseudoaleatorio pero reproducible, útil para generar terrenos, texturas o variación orgánica a partir de una semilla.",
	"ui":       "Interfaz inmediata (immediate-mode): declara la UI de cada cuadro (botones, casillas, texto) dentro de un contexto en árbol de filas, columnas y paneles.",
}

// FunctionDocs holds a one-line, friendly Spanish explanation for every
// top-level std/* function, keyed by "namespace.nombre". Kept short and
// example-driven for functions whose behavior isn't obvious from the name
// and signature alone.
var FunctionDocs = map[string]string{
	// ruido
	"ruido.suave":   "Ruido 2D suave en (x, y), siempre en [0, 1] y reproducible: la misma semilla y coordenadas dan siempre el mismo resultado.",
	"ruido.fractal": "Como ruido.suave pero sumando varias octavas para un detalle más rico (piensa en montañas con relieve, no solo colinas).",

	// curvas (todas puras: mismo progreso, mismo resultado)
	"curvas.lineal":                     "Progreso sin transformar: lineal(t) = t. Útil como referencia para comparar con las demás curvas.",
	"curvas.cuadratica_entrada":         "Arranca lento y acelera hacia el final (ease-in cuadrático).",
	"curvas.cuadratica_salida":          "Arranca rápido y frena hacia el final (ease-out cuadrático).",
	"curvas.cuadratica_entrada_salida":  "Arranca y termina lento, acelera en el medio (ease-in-out cuadrático).",
	"curvas.cubica_entrada":             "Como cuadratica_entrada pero con una aceleración más marcada (ease-in cúbico).",
	"curvas.cubica_salida":              "Como cuadratica_salida pero con un frenado más marcado (ease-out cúbico).",
	"curvas.cubica_entrada_salida":      "Ease-in-out cúbico: transición más pronunciada que la cuadrática en ambos extremos.",
	"curvas.cuartica_entrada":           "Ease-in de cuarto grado: arranque aún más lento que el cúbico.",
	"curvas.cuartica_salida":            "Ease-out de cuarto grado: frenado aún más marcado que el cúbico.",
	"curvas.cuartica_entrada_salida":    "Ease-in-out de cuarto grado.",
	"curvas.quintica_entrada":           "Ease-in de quinto grado: el arranque más suave de la familia polinómica.",
	"curvas.quintica_salida":            "Ease-out de quinto grado: el frenado más suave de la familia polinómica.",
	"curvas.quintica_entrada_salida":    "Ease-in-out de quinto grado.",
	"curvas.senoidal_entrada":           "Arranque suave basado en una onda seno (más suave que el cuadrático, menos marcado que el cúbico).",
	"curvas.senoidal_salida":            "Frenado suave basado en una onda seno.",
	"curvas.senoidal_entrada_salida":    "Ease-in-out basado en una onda seno.",
	"curvas.circular_entrada":           "Arranque basado en un cuarto de círculo: se siente distinto a las curvas polinómicas o senoidales.",
	"curvas.circular_salida":            "Frenado basado en un cuarto de círculo.",
	"curvas.circular_entrada_salida":    "Ease-in-out circular.",
	"curvas.exponencial_entrada":        "Arranque casi plano que se dispara al final (crecimiento exponencial).",
	"curvas.exponencial_salida":         "Salida disparada al principio que se aplana al final.",
	"curvas.exponencial_entrada_salida": "Ease-in-out exponencial: muy plano en los extremos, muy rápido en el medio.",
	"curvas.elastica_entrada":           "Arranque con rebote tipo resorte antes de asentarse (efecto \"elástico\"). El resultado puede sobrepasar [0, 1].",
	"curvas.elastica_salida":            "Como elastica_entrada pero el rebote ocurre al llegar al final.",
	"curvas.elastica_entrada_salida":    "Rebote elástico tanto al empezar como al terminar.",
	"curvas.retroceso_entrada":          "Retrocede un poco antes de avanzar (como tomar impulso). El resultado puede sobrepasar [0, 1].",
	"curvas.retroceso_salida":           "Se pasa un poco del destino antes de asentarse.",
	"curvas.retroceso_entrada_salida":   "Retroceso al empezar y sobrepaso al terminar.",
	"curvas.rebote_entrada":             "Simula una pelota que rebota antes de llegar al valor inicial, en reversa.",
	"curvas.rebote_salida":              "Simula una pelota rebotando varias veces hasta quedarse quieta en el destino.",
	"curvas.rebote_entrada_salida":      "Rebote tanto al empezar como al terminar.",

	// pincel / retro
	"pincel.ejecutar": "Arranca el bucle de juego con `instancia` (que debe implementar pincel.Juego) y abre la ventana con la configuración dada. No retorna hasta que el juego se cierra.",
	"retro.texto":     "Dibuja `texto` en la fuente DUNGEON-437 (soporta acentos y ñ), en la posición (x, y) con el tamaño de escala entero dado.",
	"retro.icono":     "Dibuja un ícono con nombre del atlas DUNGEON-mode (por ejemplo .Corazon o .Llave) en (x, y).",
	"retro.glifo":     "Dibuja el glifo `indice` del atlas retro indicado (por defecto .Dungeon) en (x, y); útil para tilesets propios sobre el mismo atlas.",
	"retro.ejecutar":  "Como pincel.ejecutar pero con valores por defecto pensados para juegos retro: ventana más pequeña, escala x4 y píxeles nítidos.",

	// mate
	"mate.absoluto":   "Valor absoluto: siempre positivo o cero. Ejemplo: absoluto(-5) → 5",
	"mate.minimo":     "El menor de dos valores. Ejemplo: minimo(3, 7) → 3",
	"mate.maximo":     "El mayor de dos valores. Ejemplo: maximo(3, 7) → 7",
	"mate.limitar":    "Recorta `valor` para que quede dentro de [minimo, maximo]. Ejemplo: limitar(150, 0, 100) → 100",
	"mate.interpolar": "Interpola linealmente entre a y b según t (0 = a, 1 = b, 0.5 = el punto medio). Ejemplo: interpolar(0, 10, 0.5) → 5",
	"mate.piso":       "Redondea hacia abajo al entero más cercano. Ejemplo: piso(2.9) → 2",
	"mate.techo":      "Redondea hacia arriba al entero más cercano. Ejemplo: techo(2.1) → 3",
	"mate.redondear":  "Redondea al entero más cercano (0.5 redondea hacia arriba). Ejemplo: redondear(2.5) → 3",
	"mate.raiz":       "Raíz cuadrada. Ejemplo: raiz(9) → 3",
	"mate.seno":       "Seno del ángulo dado en radianes.",
	"mate.coseno":     "Coseno del ángulo dado en radianes.",
	"mate.atan2":      "Ángulo en radianes del vector (x, y) respecto al eje x, considerando el signo de ambos componentes (útil para apuntar hacia un punto).",

	// azar
	"azar.real":   "Un decimal aleatorio uniforme dentro de [minimo, maximo].",
	"azar.entero": "Un entero aleatorio uniforme en [minimo, maximo): incluye minimo y excluye maximo, como los rangos a..b. Para un dado usa azar.entero(1, 7). Si minimo == maximo devuelve minimo.",

	// color
	"color.rgba": "Construye un Color a partir de sus componentes rojo, verde, azul y alfa (0-255 cada uno; alfa por defecto totalmente opaco).",

	// graficos
	"graficos.limpiar":            "Pinta toda la pantalla del color dado. Suele ser lo primero que se llama en pintar().",
	"graficos.rectangulo":         "Dibuja un rectángulo en (x, y) con el ancho y alto dados. `origen` desplaza el punto de referencia dentro del rectángulo (por defecto la esquina superior izquierda).",
	"graficos.rectangulo_v":       "Como graficos.rectangulo pero recibiendo posición y tamaño como Vec2.",
	"graficos.rectangulo_rect":    "Dibuja el Rect dado directamente.",
	"graficos.circulo":            "Dibuja un círculo relleno centrado en (x, y) con el radio dado.",
	"graficos.circulo_v":          "Como graficos.circulo pero recibiendo el centro como Vec2.",
	"graficos.linea":              "Dibuja una línea recta entre (x1, y1) y (x2, y2) con el grosor dado.",
	"graficos.linea_v":            "Como graficos.linea pero recibiendo ambos puntos como Vec2.",
	"graficos.imagen":             "Dibuja `imagen` en (x, y). `escala` y `rotacion` transforman la imagen alrededor de `origen`; `tinte` la colorea.",
	"graficos.imagen_v":           "Como graficos.imagen pero recibiendo la posición como Vec2.",
	"graficos.imagen_rect":        "Dibuja `imagen` estirada o encogida para llenar exactamente el Rect `destino`.",
	"graficos.region":             "Dibuja solo la región `fuente` (un recorte) de `imagen`, en (x, y). Útil para hojas de sprites (spritesheets).",
	"graficos.region_v":           "Como graficos.region pero recibiendo la posición como Vec2.",
	"graficos.region_rect":        "Como graficos.region pero estirando el recorte para llenar el Rect `destino`.",
	"graficos.texto":              "Dibuja `texto` con la fuente dada en (x, y), con el tamaño y color indicados.",
	"graficos.texto_v":            "Como graficos.texto pero recibiendo la posición como Vec2.",
	"graficos.texto_depuracion":   "Dibuja texto con la fuente de depuración incorporada (sin necesitar cargar ninguna fuente). Pensado para mensajes rápidos de FPS/estado, no para el juego final.",
	"graficos.texto_depuracion_v": "Como graficos.texto_depuracion pero recibiendo la posición como Vec2.",
	"graficos.tamano":             "El tamaño actual de la ventana/pantalla lógica, como Vec2.",
	"graficos.usar_camara":        "A partir de aquí, todo lo dibujado usa la cámara 2D dada (desplazamiento, zoom y rotación) hasta la próxima llamada a usar_camara o restablecer_camara.",
	"graficos.restablecer_camara": "Vuelve a dibujar en coordenadas de pantalla normales, deshaciendo el efecto de usar_camara.",

	// entrada
	"entrada.tecla_mantenida":  "verdadero mientras la tecla está presionada (se dispara en cada cuadro que sigue sostenida).",
	"entrada.tecla_presionada": "verdadero solo en el cuadro en que la tecla empezó a presionarse.",
	"entrada.tecla_soltada":    "verdadero solo en el cuadro en que la tecla se soltó.",
	"entrada.raton_mantenido":  "verdadero mientras el botón del ratón está presionado.",
	"entrada.raton_presionado": "verdadero solo en el cuadro en que el botón del ratón empezó a presionarse.",
	"entrada.raton_soltado":    "verdadero solo en el cuadro en que el botón del ratón se soltó.",
	"entrada.posicion_raton":   "Posición actual del cursor en coordenadas de pantalla.",
	"entrada.rueda":            "Desplazamiento de la rueda del ratón en este cuadro (normalmente pequeño; suma cero cuando no se mueve).",

	// audio
	"audio.reproducir": "Empieza a reproducir `sonido` con el volumen y modo de bucle dados, y devuelve la Reproduccion para controlarla luego.",
	"audio.pausar":     "Pausa una reproducción en curso; puede reanudarse después con audio.reanudar.",
	"audio.reanudar":   "Continúa una reproducción previamente pausada.",
	"audio.detener":    "Detiene la reproducción por completo (a diferencia de pausar, no puede reanudarse desde donde iba).",
	"audio.volumen":    "Cambia el volumen de una reproducción ya iniciada (0 = silencio, 1 = volumen original).",

	// ventana
	"ventana.titulo":            "Cambia el texto de la barra de título de la ventana.",
	"ventana.pantalla_completa": "Activa o desactiva el modo de pantalla completa.",
	"ventana.tamano":            "El tamaño actual de la ventana del sistema operativo, como Vec2.",

	// tiempo
	"tiempo.fps": "Cuadros por segundo reales que se están dibujando (varía con el rendimiento).",
	"tiempo.tps": "Actualizaciones por segundo reales que se están ejecutando (compáralo con el `tps` configurado en pincel.ejecutar).",

	// lienzo
	"lienzo.nuevo":            "Crea una Imagen en blanco de `ancho` x `alto`, rellena con `color` (transparente por defecto).",
	"lienzo.desde_texto":      "Construye un sprite a partir de un dibujo hecho con texto: cada carácter de `filas` se traduce a un color de `colores` (. o espacio = transparente).",
	"lienzo.hoja_desde_texto": "Como lienzo.desde_texto pero para varios cuadros a la vez (una hoja de sprites), devolviendo una lista de imágenes.",
	"lienzo.ancho":            "Ancho en píxeles de la imagen.",
	"lienzo.alto":             "Alto en píxeles de la imagen.",
	"lienzo.copiar":           "Copia independiente de la imagen: modificar la copia no afecta a la original.",
	"lienzo.limpiar":          "Rellena toda la imagen con `color` (transparente por defecto), borrando lo que hubiera antes.",
	"lienzo.pixel":            "Pinta un único píxel en (x, y) del color dado.",
	"lienzo.leer_pixel":       "El color del píxel en (x, y), o Ninguno si (x, y) queda fuera de la imagen.",
	"lienzo.rect":             "Dibuja un rectángulo dentro de la imagen; `relleno` decide si se pinta macizo o solo el borde.",
	"lienzo.linea":            "Dibuja una línea recta entre dos puntos dentro de la imagen.",
	"lienzo.circulo":          "Dibuja un círculo dentro de la imagen; `relleno` decide si se pinta macizo o solo el borde.",
	"lienzo.rellenar":         "Relleno por inundación (flood fill): pinta del color dado toda el área conectada al píxel (x, y) que comparte su color original.",
	"lienzo.pegar":            "Copia `fuente` sobre `destino` en (x, y), respetando la transparencia. `espejo_h`/`espejo_v` voltean la fuente antes de pegarla.",
	"lienzo.guardar":          "Exporta la imagen como archivo PNG en `ruta`, opcionalmente ampliada por `escala` (útil para inspeccionar píxel arte de cerca).",

	// recursos
	"recursos.imagen": "Carga una imagen desde `ruta` (relativa al archivo que llama a esta función).",
	"recursos.fuente": "Carga una fuente desde `ruta` (relativa al archivo que llama a esta función).",
	"recursos.sonido": "Carga un sonido desde `ruta` (relativa al archivo que llama a esta función).",

	// ui
	"ui.crear":           "Crea un nuevo Contexto de interfaz. Se llama una sola vez; el mismo contexto se reutiliza en cada cuadro.",
	"ui.cuadro":          "Empieza a declarar la interfaz de este cuadro dentro de `contexto`. Úsalo con `con` para delimitar el árbol de la UI.",
	"ui.pintar":          "Dibuja la interfaz declarada este cuadro. Llámalo al final, después de cerrar el bloque de ui.cuadro.",
	"ui.captura_raton":   "verdadero si la interfaz está usando el ratón en este cuadro (útil para no mover también la cámara del juego, por ejemplo).",
	"ui.captura_teclado": "verdadero si la interfaz está usando el teclado en este cuadro (por ejemplo, un campo de texto con foco).",
	"ui.fijo":            "Una Medida de tamaño exacto en píxeles.",
	"ui.contenido":       "Una Medida que se ajusta al contenido, opcionalmente acotada entre `minimo` y `maximo`.",
	"ui.expandir":        "Una Medida que crece para llenar el espacio disponible del contenedor padre.",
	"ui.porcentaje":      "Una Medida como fracción del espacio disponible del contenedor padre (0 a 1).",
	"ui.fila":            "Un contenedor que acomoda a sus hijos en fila (uno junto al otro, horizontalmente). Ábrelo con `con`.",
	"ui.columna":         "Un contenedor que acomoda a sus hijos en columna (uno debajo del otro, verticalmente). Ábrelo con `con`.",
	"ui.panel":           "Un contenedor visual (con fondo) que agrupa a sus hijos como una columna.",
	"ui.texto":           "Muestra una etiqueta de texto no interactiva.",
	"ui.imagen":          "Muestra una imagen dentro de la interfaz.",
	"ui.boton":           "Un botón con `etiqueta`; devuelve verdadero en el cuadro en que se hace clic sobre él. `id` debe ser único dentro de su contenedor.",
	"ui.casilla":         "Una casilla de verificación; devuelve el nuevo valor de `valor` tras el clic (o el mismo si no cambió).",
	"ui.deslizador":      "Un control deslizante entre `minimo` y `maximo`; devuelve el valor actual tras la interacción del usuario.",
	"ui.barra":           "Una barra de progreso no interactiva que muestra `valor` entre `minimo` y `maximo`.",
	"ui.separador":       "Una línea delgada para separar visualmente secciones de la interfaz.",
	"ui.espacio":         "Un hueco vacío, útil para empujar otros elementos o dejar aire entre ellos.",
	"ui.campo_texto":     "Un campo de texto editable; devuelve el contenido actual tras la edición del usuario.",
	"ui.retro":           "Un Tema con la paleta y fuente de std/pincel/retro, listo para pasar a ui.tema.",
	"ui.fuente":          "Un Tema que usa `fuente` (y el `tamano` dado) para todo el texto de la interfaz.",
	"ui.tema":            "Aplica `tema` a los elementos declarados dentro de este bloque `con`.",
}

// MethodDocs holds a one-line, friendly Spanish explanation for every
// native struct method, keyed by "Tipo.metodo".
var MethodDocs = map[string]string{
	"Vec2.longitud":         "La magnitud (largo) del vector, es decir su distancia al origen (0, 0).",
	"Vec2.normalizado":      "El mismo vector pero con longitud 1 (misma dirección, sin importar la magnitud original).",
	"Vec2.distancia_a":      "La distancia entre este vector y `otro`, tratados como puntos.",
	"Vec2.producto_punto":   "El producto punto (dot product) con `otro`: útil para saber qué tan alineados están dos vectores.",
	"Vec2.rotado":           "El mismo vector rotado `angulo` radianes alrededor del origen.",
	"Vec2.colision_circulo": "verdadero si un círculo de este radio centrado aquí se superpone con un círculo de `radio_otro` centrado en `otro`.",
	"Rect.interseca":        "verdadero si este rectángulo se superpone con `otro`, aunque sea parcialmente.",
	"Rect.contiene":         "verdadero si `punto` cae dentro de este rectángulo.",
	"AmbitoUI.entrar":       "Entra al ámbito de este contenedor; se usa automáticamente al abrir un bloque `con` sobre él.",
	"AmbitoUI.salir":        "Sale del ámbito de este contenedor; se usa automáticamente al cerrar el bloque `con`.",
}

// TypeDocs holds a short, friendly Spanish explanation for every native
// struct/interface type.
var TypeDocs = map[string]string{
	"Vec2":         "Un vector/punto 2D: dos números decimales, x e y.",
	"Rect":         "Un rectángulo alineado a los ejes, definido por su posición (pos) y tamaño (tamano).",
	"Camara2D":     "Una cámara 2D: posición, origen (punto de la pantalla que corresponde a `pos`), zoom y rotación.",
	"Color":        "Un color RGBA. Usa las constantes con nombre (por ejemplo .Rojo) o color.rgba para crear uno propio.",
	"Imagen":       "Una imagen cargada o creada con lienzo, lista para dibujarse con graficos.imagen y funciones relacionadas.",
	"Fuente":       "Una fuente tipográfica cargada con recursos.fuente, lista para usarse con graficos.texto.",
	"Sonido":       "Un sonido cargado con recursos.sonido, listo para reproducirse con audio.reproducir.",
	"Reproduccion": "Una reproducción de sonido en curso, devuelta por audio.reproducir; permite pausarla, reanudarla o detenerla.",
	"Tecla":        "Una tecla del teclado, para usar con las funciones de entrada.",
	"BotonRaton":   "Un botón del ratón (izquierdo, derecho o medio), para usar con las funciones de entrada.",
	"Icono":        "Un ícono con nombre del atlas retro DUNGEON-mode (por ejemplo .Corazon), para usar con retro.icono.",
	"Atlas":        "Qué atlas retro usar al dibujar un glifo por índice: .Dungeon (gráfico) o .ASCII (texto Unicode).",
	"Juego":        "La interfaz que debe implementar el valor pasado a pincel.ejecutar: actualizar(dt) para la lógica, pintar() para dibujar.",
	"Contexto":     "El estado de la interfaz inmediata entre cuadros; se crea una vez con ui.crear y se reutiliza en cada uno.",
	"AmbitoUI":     "Un contenedor abierto de la interfaz (fila, columna, panel...), devuelto por funciones como ui.fila para usarse con `con`.",
	"Medida":       "Cómo debe calcularse el tamaño de un elemento de interfaz: fijo, según su contenido, expandiendo o como porcentaje del espacio disponible.",
	"Alineacion":   "Cómo alinear los hijos de un contenedor de interfaz dentro del espacio sobrante: .Inicio, .Centro o .Final.",
	"Bordes":       "Un relleno (padding) con un valor distinto por lado: izquierda, derecha, arriba y abajo.",
	"Tema":         "Los colores, fuente y espaciado que usa la interfaz al dibujarse; ver ui.retro y ui.fuente para temas listos para usar.",
}
