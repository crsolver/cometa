# Pincel: juegos 2D

Pincel es la familia de bibliotecas de juego incluida con Cometa. Se importa cada módulo necesario:

| Importación | Contenido |
| --- | --- |
| `std/mate` | Matemáticas, `mate.Vec2`, `mate.Rect` y colisiones; no requiere Ebitengine |
| `std/mate/curvas` | 31 [curvas de animación](curvas.md) escalares; no requiere Ebitengine |
| `std/azar` | Números aleatorios; no requiere Ebitengine |
| `std/pincel/juego` | Interfaz `Juego` y `ejecutar` |
| `std/pincel/graficos` | Dibujo, cámara, `graficos.Imagen` y `graficos.Fuente` |
| `std/pincel/color` | `color.Color` y `rgba` |
| `std/pincel/entrada` | Teclado y ratón |
| `std/pincel/audio` | `audio.Sonido`, `audio.Reproduccion` y reproducción |
| `std/pincel/ventana` | Propiedades de ventana |
| `std/pincel/tiempo` | FPS y TPS |
| `std/pincel/recursos` | Carga y empaquetado de recursos |
| `std/pincel/retro` | Texto bitmap e iconos incorporados de 8×8 |

El último segmento es el namespace; `usar std/pincel/graficos como g` permite `g.limpiar(.Negro)`. Cada archivo declara sus imports. No se importan automáticamente tipos ni otros módulos.

## Inicio explícito

Para empezar sin archivos de recursos, consulta [Juegos retro](#juegos-retro-sin-recursos).

```cometa
usar std/mate
usar std/pincel/juego
usar std/pincel/graficos

tipo Partida
	pos mate.Vec2
	pub fn actualizar(dt decimal)
		@pos.x = @pos.x + 40 * dt
	pub fn pintar()
		graficos.rectangulo_v(@pos, {10, 10}, .Rojo)

fn inicio()
	juego.ejecutar(Partida {}, titulo = "Mi juego") capturar |error|
		imprimir(error)
```

`juego.Juego` es una interfaz estructural con `actualizar(dt decimal)` y `pintar()`, sin resultados. El objeto puede declararse en otro módulo. Inicializa el estado antes de llamar a `ejecutar`. Las funciones globales `actualizar`, `pintar` e `iniciar` son ordinarias; no existe `juego.configuracion`.

`ejecutar` bloquea hasta cerrar la ventana y devuelve `!`. La configuración inválida, los errores del backend y una segunda ejecución en el mismo proceso producen `.Error(cadena)`; el cierre normal produce `.Ok`. Debe manejarse el resultado con las construcciones habituales de Cometa.

Defaults: 320×180, escala 1, título "Cometa", 60 TPS, sin redimensionamiento ni pantalla completa. Dimensiones/TPS son enteros positivos; escala positiva y finita. El máximo es 32768 y el tamaño físico debe ser al menos un píxel. `dt` es el tiempo fijo por tick en segundos. El tamaño lógico permanece fijo al redimensionar la ventana.

Dibujar, cambiar o restablecer la cámara fuera de la fase activa de `pintar` produce un error en ejecución. Los helpers llamados durante esa fase pueden dibujar.

## Juegos retro sin recursos

`std/pincel/retro` incluye los atlas originales DUNGEON.mode de [datagoblin](https://datagoblin.itch.io/dungeonmode), distribuidos bajo CC0. No requiere PNG, TTF ni llamadas a `recursos` en el proyecto del usuario.

```cometa
usar std/pincel/juego
usar std/pincel/graficos
usar std/pincel/retro

tipo Partida
	pub fn actualizar(dt decimal) retornar
	pub fn pintar()
		graficos.limpiar(.Negro)
		retro.texto("¡Hola, niño!", 8, 8)
		retro.icono(.Corazon, 8, 24, color = .Rojo)
		retro.icono(.Llave, 24, 24, color = .Amarillo)

fn inicio()
	juego.ejecutar(Partida {}, 180, 180, escala = 4, pixelado = verdadero) capturar |error|
		imprimir(error)
```

Firmas (todos los dibujos requieren la fase `pintar`):

```cometa
retro.texto(texto cadena, x entero, y entero, escala entero = 1, color color.Color = .Blanco)
retro.icono(icono retro.Icono, x entero, y entero, escala entero = 1, color color.Color = .Blanco)
retro.glifo(indice entero, x entero, y entero, escala entero = 1, color color.Color = .Blanco, atlas retro.Atlas = .Dungeon)
```

Las posiciones son píxeles de la esquina superior izquierda; cada celda ocupa `8 * escala` píxeles. La escala debe ser un entero positivo. Los píxeles negros del atlas se vuelven transparentes; los blancos toman el color indicado. Para fondos, dibuja antes un rectángulo.

`texto` usa Dungeon-437: incluye `áéíóúüñÑ¿¡`, símbolos y líneas de caja. Los caracteres ausentes aparecen como `?` (por ejemplo, no todas las vocales mayúsculas acentuadas están presentes). Un espacio avanza una celda sin dibujar; `\n` empieza otra línea de ocho píxeles escalados, `\t` avanza hasta el siguiente múltiplo de cuatro columnas y `\r` se ignora.

Iconos disponibles: `.Corazon`, `.CorazonVacio`, `.Espada`, `.Escudo`, `.Llave`, `.Calavera`, `.Arriba`, `.Abajo`, `.Izquierda`, `.Derecha`. `glifo` acepta índices de 0 a 255, calculados como `fila * 16 + columna`, y `.Dungeon` o `.ASCII` como atlas. Índices o escalas inválidos producen un error en ejecución.

`juego.ejecutar` acepta los parámetros finales `pixelado bool = falso` y `retro bool = falso`. Con `pixelado = verdadero` desactiva el filtro de presentación de Ebitengine; combínalo con una ventana fija a escala entera, como 4. El muestreo de los glifos siempre usa vecino más cercano. Traslaciones y zoom enteros de cámara sin rotación mantienen píxeles uniformes; transformaciones fraccionarias, rotaciones o escalas de ventana fraccionarias no lo garantizan. No se implementa letterboxing entero al redimensionar.

Para una pantalla CRT sutil, usa `juego.ejecutar(Partida {}, escala = 4, retro = verdadero) !`. Este modo activa vecino más cercano incluso con `pixelado = falso` y aplica líneas de barrido suaves, una máscara RGB tenue y una viñeta ligera a toda la imagen, incluido el texto. Los patrones finos se atenúan a escalas pequeñas. No curva la imagen ni añade parpadeo; mantiene la resolución lógica, la cámara, las coordenadas del ratón, la relación de aspecto y las bandas al redimensionar o usar pantalla completa. Funciona con cualquier juego Pincel sin importar `std/pincel/retro`. Con `retro = falso`, `pixelado` conserva su comportamiento habitual. Los errores al inicializar el shader producen `.Error(cadena)`.

Ejemplos: [calabozo jugable](../examples/dungeon2.cometa) y [galería de ambos atlas](../examples/retro.cometa). La galería muestra las celdas en orden de índice y los iconos nombrados.

### La última luz

[Esta aventura completa](../examples/escape_retro.cometa) tiene tres pisos, diálogos de historia y una paleta fija de 24 colores. No necesita recursos externos. Ejecuta `cometa ejecutar examples/escape_retro.cometa`.

- **Flechas:** mover un paso o atacar al esqueleto de la casilla vecina. Atacar lo derrota sin mover al jugador.
- **Espacio:** esperar un turno. Los esqueletos supervivientes actúan después de cada movimiento, ataque o espera; chocar con un muro o una salida cerrada no gasta turno.
- **Enter:** avanzar el diálogo. Mientras lees, la partida está pausada.
- **R:** reiniciar los tres pisos, incluso durante un diálogo o al terminar.

Recoge la llave dorada y entra en la salida `>` de cada piso. Los corazones verdes recuperan una vida hasta un máximo de cinco; la salud se conserva entre pisos. Los enemigos adyacentes golpean y los demás se acercan una casilla, sin atravesar muros, salidas u otros enemigos. El tercer piso termina la historia. La paleta está al principio del ejemplo y el estado del juego vive en `Partida`.

### Bajo la tierra

[Este plataformas](../examples/plataformas.cometa) es un sandbox pequeño con superficie, cuevas conectadas, minería y combate. Ejecuta desde la raíz del repositorio:

```powershell
.\vscode-extension\bin\cometa.exe ejecutar .\examples\plataformas.cometa
```

- **A/D o flechas:** moverse con aceleración, frenado y control aéreo.
- **Espacio:** saltar. Mantenerlo produce un salto más alto; soltarlo lo acorta. Incluye 100 ms de tolerancia al abandonar una plataforma y 120 ms de memoria de salto antes de aterrizar.
- **Ratón:** apuntar libremente. Mantener el botón izquierdo dispara; el derecho pica el primer bloque visible del rayo dentro de 48 píxeles. La piedra y la roca profunda tardan más que la tierra. El mineral turquesa aumenta el contador.
- **R:** volver al refugio con cinco corazones, conservando el terreno excavado y el mineral. Morir hace lo mismo automáticamente.
- **N:** crear un mundo con otra semilla; descarta la partida actual.
- **H:** mostrar de nuevo la ayuda.

La resolución lógica es **320×180**, a escala entera 4 y sin filtro. Cada sección muestra **40×22½ bloques de 8×8**: la cuadrícula sigue continua y los bordes verticales recortan medias celdas. La cámara permanece fija y cambia instantáneamente cuando el centro del jugador cruza un borde. El mundo tiene seis secciones horizontales y tres verticales, con límites sólidos. Los 240×68 bloques cubren 1920×544 píxeles; los cuatro píxeles inferiores quedan fuera de la última pantalla.

El generador combina ruido con galerías y rampas de conexión; la plataforma del refugio no puede destruirse. Los slimes saltan y los murciélagos se acercan volando, sin atravesar terreno. Las balas se detienen en el primer bloque o enemigo. Los cambios de terreno y el estado de los enemigos se conservan al cambiar de sección; los enemigos lejanos quedan pausados. No hay guardado, construcción, inventario, crafting ni condición de victoria.

La paleta y los efectos se dibujan con Pincel y `retro`, que incorpora los mismos atlas CC0 de `examples/assets/dungeonmode/bitmap` con fondo transparente. No se necesitan recursos externos. La simulación vive en [plataformas/mundo.cometa](../examples/plataformas/mundo.cometa), sin dependencia gráfica. Para reproducir un terreno, reemplaza el argumento aleatorio de `partida.generar(...)` en `inicio` por la semilla que aparece al pie de la pantalla.

## Valores, coordenadas y recursos

`mate.Vec2` tiene `x decimal` e `y decimal`. `mate.Rect` tiene `pos mate.Vec2` y `tamano mate.Vec2`.
`graficos.Camara2D` tiene `pos mate.Vec2`, `origen mate.Vec2`, `zoom decimal` (1 por defecto) y
`rotacion decimal`. Estos tipos se copian por valor.
Las estructuras del usuario conservan referencias.

X crece a la derecha, Y hacia abajo. Distancias en píxeles lógicos y ángulos en
radianes. `mate.Vec2` admite suma/resta de vectores, negación, multiplicación por escalar
en ambos órdenes y división por escalar. Normalizar cero devuelve cero. Se admiten literales nombrados y posicionales, además de `{2, 3}` cuando se conoce el tipo esperado.

`color.Color`, `entrada.Tecla` y `entrada.BotonRaton` son valores opacos con constantes contextuales:

- color.Color: los 24 tonos de la paleta siguiente y `.Transparente` (RGBA 0, 0, 0, 0).
  `color.rgba` permite colores personalizados y limita los canales a 0–255.
- entrada.Tecla: `.A` a `.Z`, `.Izquierda`, `.Derecha`, `.Arriba`, `.Abajo`, `.Espacio`,
  `.Escape`, `.Enter`, `.Tab`, `.Retroceso`, `.Shift`, `.Control`.
- entrada.BotonRaton: `.Izquierdo`, `.Derecho`, `.Medio`.

La paleta de `std/pincel/color` usa alfa 255 en todos sus tonos:

| Constante | R | G | B |
| --- | ---: | ---: | ---: |
| `.Negro` | 0 | 0 | 0 |
| `.VerdePino` | 0 | 43 | 36 |
| `.AzulNoche` | 24 | 30 | 42 |
| `.Oliva` | 84 | 106 | 0 |
| `.Indigo` | 25 | 17 | 74 |
| `.VioletaOscuro` | 47 | 42 | 76 |
| `.Carbon` | 68 | 63 | 65 |
| `.Petroleo` | 8 | 66 | 72 |
| `.VerdeBosque` | 40 | 84 | 72 |
| `.Gris` | 82 | 82 | 76 |
| `.Marron` | 115 | 97 | 80 |
| `.Caqui` | 119 | 120 | 91 |
| `.Malva` | 94 | 82 | 107 |
| `.Ocre` | 130 | 91 | 49 |
| `.Rojo` | 181 | 59 | 89 |
| `.Rosa` | 255 | 87 | 119 |
| `.Amarillo` | 255 | 185 | 21 |
| `.Crema` | 255 | 224 | 119 |
| `.AzulIndigo` | 67 | 62 | 166 |
| `.Azul` | 71 | 114 | 191 |
| `.Violeta` | 150 | 102 | 238 |
| `.Verde` | 87 | 176 | 103 |
| `.Celeste` | 153 | 215 | 229 |
| `.Blanco` | 255 | 249 | 228 |

`.Blanco` es un blanco cálido; como tinte predeterminado también aporta ese tono
a imágenes y glifos. Para conservar los colores originales de una imagen, usa
`tinte = color.rgba(255, 255, 255)`.

No son enums del usuario y no admiten `casos`. Se requiere un tipo esperado para
las constantes, por ejemplo `entrada.tecla_mantenida(.Espacio)`.

`graficos.Imagen`, `graficos.Fuente`, `audio.Sonido` y `audio.Reproduccion` son handles opacos compartidos.
No admiten literales ni nulos; un campo de recurso requiere inicialización.
Se puede usar `graficos.Imagen?` para ausencia.

```cometa
usar std/pincel/recursos

var sprite = recursos.imagen("assets/jugador.png")
var fuente = recursos.fuente("assets/texto.ttf")
var sonido = recursos.sonido("assets/recoger.wav")
```

Las llamadas de recursos solo se aceptan como inicializadores globales directos,
con rutas literales relativas al módulo que las declara. Se validan los datos y
se incorporan al Go generado. Formatos: PNG/JPEG, TTF/OTF, WAV/Ogg Vorbis/MP3.
WAV admite PCM de 8/16 bits, mono/estéreo. Audio se carga completo en memoria y se
decodifica a PCM estéreo de 48 kHz. No se necesitan archivos externos al ejecutar.

## API de los módulos

Las firmas siguientes requieren importar sus respectivos módulos. Los parámetros después de `=`
son opcionales. Las expresiones de argumentos se evalúan una vez, en orden fuente,
también al usar argumentos nombrados.

```cometa
juego.ejecutar(instancia juego.Juego, ancho entero = 320, alto entero = 180, titulo cadena = "Cometa", escala decimal = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = falso, retro bool = falso) !
mate.Vec2.longitud() decimal
mate.Vec2.normalizado() mate.Vec2
mate.Vec2.distancia_a(otro mate.Vec2) decimal
mate.Vec2.producto_punto(otro mate.Vec2) decimal
mate.Vec2.rotado(angulo decimal) mate.Vec2
mate.Vec2.colision_circulo(radio decimal, otro mate.Vec2, radio_otro decimal) bool
mate.Rect.interseca(otro mate.Rect) bool
mate.Rect.contiene(punto mate.Vec2) bool
mate.pi // constante decimal
mate.absoluto(valor decimal) decimal
mate.minimo(a decimal, b decimal) decimal
mate.maximo(a decimal, b decimal) decimal
mate.limitar(valor decimal, minimo decimal, maximo decimal) decimal
mate.interpolar(a decimal, b decimal, t decimal) decimal
mate.piso(valor decimal) entero
mate.techo(valor decimal) entero
mate.redondear(valor decimal) entero
mate.raiz(valor decimal) decimal
mate.seno(angulo decimal) decimal
mate.coseno(angulo decimal) decimal
mate.atan2(y decimal, x decimal) decimal
azar.real(minimo decimal, maximo decimal) decimal
azar.entero(minimo entero, maximo entero) entero
color.rgba(r entero, g entero, b entero, a entero = 255) color.Color

graficos.limpiar(color color.Color)
graficos.rectangulo(x decimal, y decimal, ancho decimal, alto decimal, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.rectangulo_v(pos mate.Vec2, tamano mate.Vec2, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.rectangulo_rect(rect mate.Rect, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.circulo(x decimal, y decimal, radio decimal, color color.Color)
graficos.circulo_v(centro mate.Vec2, radio decimal, color color.Color)
graficos.linea(x1 decimal, y1 decimal, x2 decimal, y2 decimal, color color.Color, grosor decimal = 1)
graficos.linea_v(a mate.Vec2, b mate.Vec2, color color.Color, grosor decimal = 1)
graficos.imagen(imagen graficos.Imagen, x decimal, y decimal, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.imagen_v(imagen graficos.Imagen, pos mate.Vec2, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.imagen_rect(imagen graficos.Imagen, destino mate.Rect, origen mate.Vec2 = {}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.region(imagen graficos.Imagen, fuente mate.Rect, x decimal, y decimal, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.region_v(imagen graficos.Imagen, fuente mate.Rect, pos mate.Vec2, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.region_rect(imagen graficos.Imagen, fuente mate.Rect, destino mate.Rect, origen mate.Vec2 = {}, rotacion decimal = 0, tinte color.Color = .Blanco)
graficos.texto(texto cadena, fuente graficos.Fuente, x decimal, y decimal, tamano decimal = 20, color color.Color = .Blanco, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.texto_v(texto cadena, fuente graficos.Fuente, pos mate.Vec2, tamano decimal = 20, color color.Color = .Blanco, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.texto_depuracion(texto cadena, x decimal = 0, y decimal = 0)
graficos.texto_depuracion_v(texto cadena, pos mate.Vec2 = {})
graficos.tamano() mate.Vec2
graficos.usar_camara(camara graficos.Camara2D)
graficos.restablecer_camara()

entrada.tecla_mantenida(tecla entrada.Tecla) bool
entrada.tecla_presionada(tecla entrada.Tecla) bool
entrada.tecla_soltada(tecla entrada.Tecla) bool
entrada.raton_mantenido(boton entrada.BotonRaton) bool
entrada.raton_presionado(boton entrada.BotonRaton) bool
entrada.raton_soltado(boton entrada.BotonRaton) bool
entrada.posicion_raton() mate.Vec2
entrada.rueda() mate.Vec2

audio.reproducir(sonido audio.Sonido, volumen decimal = 1, bucle bool = falso) audio.Reproduccion
audio.pausar(reproduccion audio.Reproduccion)
audio.reanudar(reproduccion audio.Reproduccion)
audio.detener(reproduccion audio.Reproduccion)
audio.volumen(reproduccion audio.Reproduccion, valor decimal)
ventana.titulo(titulo cadena)
ventana.pantalla_completa(activa bool)
ventana.tamano() mate.Vec2
tiempo.fps() decimal
tiempo.tps() decimal
recursos.imagen(ruta cadena) graficos.Imagen
recursos.fuente(ruta cadena) graficos.Fuente
recursos.sonido(ruta cadena) audio.Sonido
```

`azar` incluye el mínimo y excluye el máximo cuando son distintos; límites iguales
devuelven ese valor. Los límites deben ser finitos y ordenados; para `entero`
deben ser enteros. mate.Rectángulo–rectángulo requiere área solapada, círculo–círculo
incluye contacto y punto–rectángulo incluye arriba/izquierda, excluye abajo/derecha.

Las formas son rellenas. La región de sprite usa píxeles de la imagen original.
La cámara traslada por `-pos`, rota por `-rotacion`, escala por `zoom` y traslada
por `origen`; su zoom debe ser positivo. Cada frame restablece la cámara.
`limpiar` y `texto_depuracion` usan pantalla; el ratón usa coordenadas lógicas de
pantalla. Las imágenes con posición/escala aplican su origen antes de escala/rotación/posición.
Las variantes `_rect` de imagen y región ajustan la imagen o recorte completo al
rectángulo destino; su origen se mide en píxeles del destino, después de escalar.
mate.Rectángulos y texto aplican origen, rotación y posición antes de la cámara.
Todas las rotaciones son radianes y valen cero por defecto. Círculos, líneas y
texto de depuración no tienen rotación. Los métodos de mate.Vec2 devuelven valores
nuevos y no mutan el receptor; normalizar cero devuelve cero.

El compilador rechaza dibujo alcanzable desde globales,
inicialización o actualización, incluyendo helpers, métodos y defaults.
Para interfaces comprueba conservadoramente los métodos con ese nombre.
Los helpers llamados desde `pintar` pueden dibujar; el runtime también verifica
el contexto. Lea los estados presionado/soltado en `actualizar`: corresponden al tick.

Audio retiene las reproducciones activas aunque se descarte el handle. Al terminar
libera el reproductor; `reanudar` reinicia un sonido terminado/detenido. Volumen 0–1.

## Comandos y alcance

`cometa compilar juego.cometa -o juego.go` produce Go formateado y sintácticamente
válido. `cometa ejecutar juego.cometa` construye y ejecuta. `cometa construir
juego.cometa -o juego.exe` construye sin abrir ventana; sin `-o` usa el nombre del
archivo sin extensión (`.exe` en Windows). Ambos usan un módulo temporal, eliminado
al finalizar. La primera compilación requiere red para descargar dependencias.

Se fija Ebitengine v2.10.1. Go 1.25+ y las dependencias nativas de Ebitengine son
necesarios. `COMETA_GO` permite elegir Go. La entrega inicial soporta escritorio;
web, móvil, gamepads, touch, física completa, tilemaps, partículas, UI y shaders
quedan fuera. Los tipos y namespaces están reservados; la interfaz genérica de
los ejemplos antes llamada `graficos.Fuente` ahora se llama `Proveedor`.
