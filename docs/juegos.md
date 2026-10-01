# Pincel: juegos 2D

Pincel es la familia de bibliotecas de juego incluida con Cometa. Se importa cada módulo necesario:

| Importación | Contenido |
| --- | --- |
| `std/mate` | Matemáticas, `mate.Vec2`, `mate.Rect` y colisiones; no requiere Ebitengine |
| `std/mate/curvas` | 31 [curvas de animación](curvas.md) escalares; no requiere Ebitengine |
| `std/azar` | Números aleatorios; no requiere Ebitengine |
| `std/pincel` | Interfaz `Juego` y `ejecutar` |
| `std/pincel/graficos` | Dibujo, cámara, `graficos.Imagen` y `graficos.Fuente` |
| `std/pincel/color` | `color.Color` y `rgba` |
| `std/pincel/entrada` | Teclado y ratón |
| `std/pincel/audio` | `audio.Sonido`, `audio.Reproduccion` y reproducción |
| `std/pincel/ventana` | Propiedades de ventana |
| `std/pincel/tiempo` | FPS y TPS |
| `std/pincel/recursos` | Carga y empaquetado de recursos |
| `std/pincel/retro` | Texto bitmap e iconos incorporados de 8×8 |
| `std/pincel/lienzo` | [Pixel art con código](lienzo.md): sprites desde texto, lienzos editables y PNG |
| `std/pincel/rejilla` | Mapas de tiles: cuadrículas de enteros con dibujo y colisión |
| `std/pincel/datos` | Guardar y leer texto y números por clave (mejor puntuación, opciones) |
| `std/pincel/ui` | [Interfaz en modo inmediato](ui.md): botones, campos de texto, barras |

El último segmento es el namespace; `usar std/pincel/graficos como g` permite `g.limpiar(.Negro)`. Cada archivo declara sus imports. No se importan automáticamente tipos ni otros módulos.

## Inicio explícito

Para empezar sin archivos de recursos, consulta [Juegos retro](#juegos-retro-sin-recursos).

```cometa
usar std/mate
usar std/pincel
usar std/pincel/graficos

tipo Partida
	pos mate.Vec2
	pub fn actualizar(dt decimal)
		@pos.x = @pos.x + 40 * dt
	pub fn pintar()
		graficos.rectangulo_v(@pos, {10, 10}, .Rojo)

fn inicio()
	pincel.ejecutar(Partida {}, titulo = "Mi juego") atrapar |error|
		imprimir(error)
```

`pincel.Juego` es una interfaz estructural con `actualizar(dt decimal)` y `pintar()`, sin resultados. El objeto puede declararse en otro módulo. Inicializa el estado antes de llamar a `ejecutar`. Las funciones globales `actualizar`, `pintar` e `iniciar` son ordinarias; no existe `pincel.configuracion`.

`ejecutar` bloquea hasta cerrar la ventana y devuelve `!`. La configuración inválida, los errores del backend y una segunda ejecución en el mismo proceso producen `.Error(cadena)`; el cierre normal produce `.Ok`. Debe manejarse el resultado con las construcciones habituales de Cometa.

Defaults: 320×180, escala 1, título "Cometa", 60 TPS, sin redimensionamiento ni pantalla completa. Dimensiones/TPS son enteros positivos; escala positiva y finita. El máximo es 32768 y el tamaño físico debe ser al menos un píxel. `dt` es el tiempo fijo por tick en segundos. El tamaño lógico permanece fijo al redimensionar la ventana.

Dibujar, cambiar o restablecer la cámara fuera de la fase activa de `pintar` produce un error en ejecución. Los helpers llamados durante esa fase pueden dibujar.

## Juegos retro sin recursos

`std/pincel/retro` incluye los atlas originales DUNGEON.mode de [datagoblin](https://datagoblin.itch.io/dungeonmode), distribuidos bajo CC0. No requiere PNG, TTF ni llamadas a `recursos` en el proyecto del usuario.

```cometa
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
	retro.ejecutar(Partida {}) atrapar |error|
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

`pincel.ejecutar` acepta los parámetros finales `pixelado bool = falso` y `retro bool = falso`. Con `pixelado = verdadero` desactiva el filtro de presentación de Ebitengine; combínalo con una ventana fija a escala entera, como 4. El muestreo de los glifos siempre usa vecino más cercano. Las posiciones decimales (de sprites o de la cámara) no desenfocan: las imágenes se dibujan con vecino más cercano, así que no hace falta redondearlas. Lo que da píxeles de distinto grosor son las escalas o el zoom no enteros, las rotaciones y las escalas de ventana fraccionarias. No se implementa letterboxing entero al redimensionar.

`std/pincel/retro` también expone `retro.ejecutar(instancia Juego, ancho entero = 320, alto entero = 200, titulo cadena = "Cometa", escala decimal = 4, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = verdadero, retro bool = falso) !`: los mismos parámetros que `pincel.ejecutar`, pero con defaults de consola de fantasía — 320×200 (40×25 celdas exactas de 8×8), escala entera 4 (ventana de 1280×800) y `pixelado = verdadero`. Úsalo en vez de `pincel.ejecutar` cuando el juego dibuja con la fuente Dungeon; los parámetros nombrados siguen sobrescribiendo cualquier default.

Para una pantalla CRT sutil, usa `pincel.ejecutar(Partida {}, escala = 4, retro = verdadero) !`. Este modo activa vecino más cercano incluso con `pixelado = falso` y aplica líneas de barrido suaves, una máscara RGB tenue y una viñeta ligera a toda la imagen, incluido el texto. Los patrones finos se atenúan a escalas pequeñas. No curva la imagen ni añade parpadeo; mantiene la resolución lógica, la cámara, las coordenadas del ratón, la relación de aspecto y las bandas al redimensionar o usar pantalla completa. Funciona con cualquier juego Pincel sin importar `std/pincel/retro`. Con `retro = falso`, `pixelado` conserva su comportamiento habitual. Los errores al inicializar el shader producen `.Error(cadena)`.

Ejemplo: [texto e iconos retro](../examples/pincel/06_retro.cometa). Hay más ejemplos para principiantes en [examples](../examples/README.md).

## Valores, coordenadas y recursos

`mate.Vec2` tiene `x decimal` e `y decimal`. `mate.Rect` tiene `pos mate.Vec2` y `tamano mate.Vec2`.
`graficos.Camara2D` tiene `pos mate.Vec2`, `origen mate.Vec2`, `zoom decimal` (1 por defecto) y
`rotacion decimal`. Estos tipos se copian por valor.
Las estructuras del usuario conservan referencias.

Los métodos de la cámara devuelven una copia nueva, así que se asignan de vuelta:

```cometa
// en actualizar
@camara = @camara.siguiendo(@jugador.pos, 0.1).limitada({pos: {0, 0}, tamano: {1280, 720}})
var clic = @camara.a_mundo(entrada.posicion_raton())   // el ratón, en coordenadas del mundo
// en pintar
graficos.usar_camara(@camara.sacudida(@temblor))
```

`siguiendo` acerca la cámara al objetivo (1 = de golpe, 0.1 = con retraso),
`limitada` evita mostrar lo que queda fuera del nivel (ignora la rotación),
`sacudida` la desplaza al azar hasta la intensidad dada, `visible()` da el
rectángulo del mundo que se ve y `a_mundo`/`a_pantalla` convierten puntos entre
pantalla y mundo. `entrada.posicion_raton()` siempre está en coordenadas de pantalla.

`tiempo.temporizador(duracion, bucle = falso)` cuenta segundos para enfriamientos
o eventos periódicos. Se avanza una vez por cuadro:

```cometa
// campo: disparo tiempo.Temporizador = tiempo.temporizador(0.3)
@disparo.avanzar(dt)
si entrada.tecla_mantenida(.Espacio) && @disparo.terminado()
	disparar()
	@disparo.reiniciar()
```

`avanzar` devuelve `verdadero` en el paso en que llega a su duración; con
`bucle = verdadero` vuelve a empezar solo y nunca queda `terminado()`.
`terminar()` lo da por acabado (para que un enfriamiento empiece listo), y
`restante()`/`progreso()` dan los segundos que faltan y la fracción de 0 a 1.

X crece a la derecha, Y hacia abajo. Distancias en píxeles lógicos y ángulos en
radianes. `mate.Vec2` admite suma/resta de vectores, negación, multiplicación por escalar
en ambos órdenes y división por escalar. Normalizar cero devuelve cero. Se admiten literales nombrados y posicionales, además de `{2, 3}` cuando se conoce el tipo esperado.

`color.Color`, `entrada.Tecla`, `entrada.BotonRaton`, `entrada.BotonMando` y `entrada.EjeMando` son valores opacos con constantes contextuales:

- color.Color: los 24 tonos de la paleta siguiente y `.Transparente` (RGBA 0, 0, 0, 0).
  `color.rgba` permite colores personalizados y limita los canales a 0–255.
- entrada.Tecla: `.A` a `.Z`, los dígitos `.Cero` a `.Nueve`, `.F1` a `.F12`, `.Izquierda`,
  `.Derecha`, `.Arriba`, `.Abajo`, `.Espacio`, `.Escape`, `.Enter`, `.Tab`, `.Retroceso`,
  `.Shift`, `.Control`, `.Alt`, `.Suprimir`, `.Inicio`, `.Fin`, `.RePag`, `.AvPag`,
  `.ShiftIzquierdo`, `.ShiftDerecho`, `.ControlIzquierdo`, `.ControlDerecho`.
- entrada.BotonRaton: `.Izquierdo`, `.Derecho`, `.Medio`.
- entrada.BotonMando (distribución estándar): `.A`, `.B`, `.X`, `.Y`, `.Arriba`, `.Abajo`,
  `.Izquierda`, `.Derecha`, `.HombroIzquierdo`, `.HombroDerecho`, `.GatilloIzquierdo`,
  `.GatilloDerecho`, `.Atras`, `.Inicio`, `.PalancaIzquierda`, `.PalancaDerecha`.
- entrada.EjeMando: `.IzquierdoX`, `.IzquierdoY`, `.DerechoX`, `.DerechoY`.

La paleta de `std/pincel/color` usa alfa 255 en todos sus tonos:

| Constante | R | G | B |
| --- | ---: | ---: | ---: |
| `.Negro` | 0 | 0 | 0 |
| `.Rojo` | 255 | 0 | 0 |
| `.Verde` | 0 | 255 | 0 |
| `.Azul` | 0 | 0 | 255 |
| `.Amarillo` | 255 | 255 | 0 |
| `.Cian` | 0 | 255 | 255 |
| `.Magenta` | 255 | 0 | 255 |
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
| `.Carmesi` | 181 | 59 | 89 |
| `.Rosa` | 255 | 87 | 119 |
| `.Ambar` | 255 | 185 | 21 |
| `.Crema` | 255 | 224 | 119 |
| `.AzulIndigo` | 67 | 62 | 166 |
| `.Cobalto` | 71 | 114 | 191 |
| `.Violeta` | 150 | 102 | 238 |
| `.Hierba` | 87 | 176 | 103 |
| `.Celeste` | 153 | 215 | 229 |
| `.Marfil` | 255 | 249 | 228 |
| `.Blanco` | 255 | 255 | 255 |

`.Blanco` es blanco puro y, como tinte predeterminado, conserva los colores originales
de imágenes y glifos. `.Marfil` es el blanco cálido de la paleta original.

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

Para texto legible sin traer un `.ttf`, `graficos.fuente_predeterminada()` devuelve
la fuente Go Regular incorporada (licencia BSD, con acentos y ñ). Se puede llamar
en cualquier sitio, no solo en un global:

```cometa
graficos.texto("Puntos: 10", graficos.fuente_predeterminada(), 8, 8, tamano = 16)
```

## API de los módulos

Las firmas siguientes requieren importar sus respectivos módulos. Los parámetros después de `=`
son opcionales. Las expresiones de argumentos se evalúan una vez, en orden fuente,
también al usar argumentos nombrados.

```cometa
pincel.salir()   // cierra el juego al terminar el cuadro actual
pincel.ejecutar(instancia pincel.Juego, ancho entero = 320, alto entero = 180, titulo cadena = "Cometa", escala decimal = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps entero = 60, pixelado bool = falso, retro bool = falso) !
mate.Vec2.longitud() decimal
mate.Vec2.normalizado() mate.Vec2
mate.Vec2.distancia_a(otro mate.Vec2) decimal
mate.Vec2.producto_punto(otro mate.Vec2) decimal
mate.Vec2.rotado(angulo decimal) mate.Vec2
mate.Vec2.colision_circulo(radio decimal, otro mate.Vec2, radio_otro decimal) bool
mate.Rect.interseca(otro mate.Rect) bool
mate.Vec2.angulo() decimal
mate.Vec2.interpolar(otro mate.Vec2, t decimal) mate.Vec2
mate.Vec2.reflejar(normal mate.Vec2) mate.Vec2
mate.Vec2.perpendicular() mate.Vec2
mate.Rect.contiene(punto mate.Vec2) bool
mate.Rect.centro() mate.Vec2
mate.Rect.interseccion(otro mate.Rect) mate.Rect   // tamaño cero si no se tocan
mate.Rect.desplazado(delta mate.Vec2) mate.Rect
mate.Rect.colision_circulo(centro mate.Vec2, radio decimal) bool
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
mate.tangente(angulo decimal) decimal
mate.potencia(base decimal, exponente decimal) decimal
mate.signo(valor decimal) decimal                  // con enteros devuelve un entero
mate.distancia(x1 decimal, y1 decimal, x2 decimal, y2 decimal) decimal
mate.angulo(x1 decimal, y1 decimal, x2 decimal, y2 decimal) decimal
mate.radianes(grados decimal) decimal
mate.grados(radianes decimal) decimal
azar.real(minimo decimal, maximo decimal) decimal
azar.entero(minimo entero, maximo entero) entero  // incluye minimo, excluye maximo: un dado es azar.entero(1, 7)
azar.semilla(valor entero)                          // partidas reproducibles
color.rgba(r entero, g entero, b entero, a entero = 255) color.Color

graficos.limpiar(color color.Color)
graficos.rectangulo(x decimal, y decimal, ancho decimal, alto decimal, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.rectangulo_v(pos mate.Vec2, tamano mate.Vec2, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.rectangulo_rect(rect mate.Rect, color color.Color, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.circulo(x decimal, y decimal, radio decimal, color color.Color)
graficos.circulo_v(centro mate.Vec2, radio decimal, color color.Color)
graficos.linea(x1 decimal, y1 decimal, x2 decimal, y2 decimal, color color.Color, grosor decimal = 1)
graficos.linea_v(a mate.Vec2, b mate.Vec2, color color.Color, grosor decimal = 1)
graficos.imagen(imagen graficos.Imagen, x decimal, y decimal, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.imagen_v(imagen graficos.Imagen, pos mate.Vec2, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.imagen_rect(imagen graficos.Imagen, destino mate.Rect, origen mate.Vec2 = {}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.region(imagen graficos.Imagen, fuente mate.Rect, x decimal, y decimal, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.region_v(imagen graficos.Imagen, fuente mate.Rect, pos mate.Vec2, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.region_rect(imagen graficos.Imagen, fuente mate.Rect, destino mate.Rect, origen mate.Vec2 = {}, rotacion decimal = 0, tinte color.Color = .Blanco, relleno color.Color = .Transparente)
graficos.texto(texto cadena, fuente graficos.Fuente, x decimal, y decimal, tamano decimal = 20, color color.Color = .Blanco, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.texto_v(texto cadena, fuente graficos.Fuente, pos mate.Vec2, tamano decimal = 20, color color.Color = .Blanco, origen mate.Vec2 = {}, rotacion decimal = 0)
graficos.medir_texto(texto cadena, fuente graficos.Fuente, tamano decimal = 20) mate.Vec2
graficos.fuente_predeterminada() graficos.Fuente   // Go Regular incorporada
graficos.hoja(imagen graficos.Imagen, ancho_cuadro entero, alto_cuadro entero) graficos.Hoja
graficos.cuadros(hoja graficos.Hoja) entero
graficos.tamano_cuadro(hoja graficos.Hoja) mate.Vec2
graficos.cuadro(hoja graficos.Hoja, indice entero, x decimal, y decimal, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, espejo_h bool = falso, espejo_v bool = falso, relleno color.Color = .Transparente)
graficos.cuadro_v(hoja graficos.Hoja, indice entero, pos mate.Vec2, origen mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, rotacion decimal = 0, tinte color.Color = .Blanco, espejo_h bool = falso, espejo_v bool = falso, relleno color.Color = .Transparente)
graficos.texto_depuracion(texto cadena, x decimal = 0, y decimal = 0)
graficos.texto_depuracion_v(texto cadena, pos mate.Vec2 = {})
graficos.tamano() mate.Vec2
graficos.recortar(rect mate.Rect)                 // solo dibuja dentro de rect (pantalla)
graficos.quitar_recorte()
graficos.usar_camara(camara graficos.Camara2D)
graficos.restablecer_camara()
graficos.Camara2D.a_mundo(punto mate.Vec2) mate.Vec2
graficos.Camara2D.a_pantalla(punto mate.Vec2) mate.Vec2
graficos.Camara2D.siguiendo(objetivo mate.Vec2, suavizado decimal) graficos.Camara2D
graficos.Camara2D.limitada(mundo mate.Rect) graficos.Camara2D
graficos.Camara2D.sacudida(intensidad decimal) graficos.Camara2D
graficos.Camara2D.visible() mate.Rect

entrada.tecla_mantenida(tecla entrada.Tecla) bool
entrada.tecla_presionada(tecla entrada.Tecla) bool
entrada.tecla_soltada(tecla entrada.Tecla) bool
entrada.raton_mantenido(boton entrada.BotonRaton) bool
entrada.raton_presionado(boton entrada.BotonRaton) bool
entrada.raton_soltado(boton entrada.BotonRaton) bool
entrada.posicion_raton() mate.Vec2
entrada.rueda() mate.Vec2
entrada.mandos() entero
entrada.mando_boton(mando entero, boton entrada.BotonMando) bool
entrada.mando_boton_presionado(mando entero, boton entrada.BotonMando) bool
entrada.mando_eje(mando entero, eje entrada.EjeMando, zona_muerta decimal = 0.15) decimal

datos.guardar(clave cadena, valor cadena) !
datos.leer(clave cadena) cadena?
datos.borrar(clave cadena) !
datos.juego(nombre cadena)
datos.claves() [cadena]
datos.guardar_entero(clave cadena, valor entero) !
datos.leer_entero(clave cadena) entero?
datos.guardar_decimal(clave cadena, valor decimal) !
datos.leer_decimal(clave cadena) decimal?

rejilla.nueva(columnas entero, filas entero, valor entero = 0) rejilla.Rejilla
rejilla.desde_texto(filas [cadena], simbolos [cadena: entero]) rejilla.Rejilla
rejilla.dibujar(mapa rejilla.Rejilla, hoja graficos.Hoja, pos mate.Vec2 = {}, escala mate.Vec2 = {x: 1, y: 1}, tinte color.Color = .Blanco, animaciones [entero: [entero]] = [:], cuadros_por_segundo decimal = 8)
rejilla.Rejilla.columnas() entero
rejilla.Rejilla.filas() entero
rejilla.Rejilla.obtener(x entero, y entero) entero?
rejilla.Rejilla.poner(x entero, y entero, valor entero) bool
rejilla.Rejilla.rellenar(valor entero)
rejilla.Rejilla.choca(area mate.Rect, tamano_celda mate.Vec2, solidos [entero]) bool
rejilla.Rejilla.mover(area mate.Rect, tamano_celda mate.Vec2, delta mate.Vec2, solidos [entero], plataformas [entero] = []) mate.Vec2
rejilla.Rejilla.celda_en(punto mate.Vec2, tamano_celda mate.Vec2) entero?
rejilla.Rejilla.valores_en(area mate.Rect, tamano_celda mate.Vec2) [entero]

audio.reproducir(sonido audio.Sonido, volumen decimal = 1, bucle bool = falso) audio.Reproduccion
audio.pausar(reproduccion audio.Reproduccion)
audio.reanudar(reproduccion audio.Reproduccion)
audio.detener(reproduccion audio.Reproduccion)
audio.volumen(reproduccion audio.Reproduccion, valor decimal)
ventana.titulo(titulo cadena)
ventana.pantalla_completa(activa bool)
ventana.tamano() mate.Vec2
ventana.cursor(visible bool)
ventana.icono(imagen graficos.Imagen)
ventana.es_pantalla_completa() bool
tiempo.fps() decimal
tiempo.tps() decimal
tiempo.temporizador(duracion decimal, bucle bool = falso) tiempo.Temporizador
tiempo.Temporizador.avanzar(dt decimal) bool
tiempo.Temporizador.terminado() bool
tiempo.Temporizador.terminar()
tiempo.Temporizador.reiniciar()
tiempo.Temporizador.restante() decimal
tiempo.Temporizador.progreso() decimal
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

`tinte` multiplica los colores, así que no sirve para un destello blanco. Para eso
está `relleno`: pinta la silueta (respetando la transparencia) del color dado, con la
fuerza de su alfa. `.Transparente` (por defecto) no cambia nada, `.Blanco` da el
destello de un golpe y `color.rgba(255, 255, 255, 128)` uno a medias, así que se puede
apagar poco a poco:

```cometa
graficos.cuadro(@hoja, @cuadro, @pos.x, @pos.y, relleno = color.rgba(255, 255, 255, @destello))
```

`graficos.recortar(rect)` limita el dibujo a un rectángulo de la pantalla (no le
afecta la cámara) hasta `graficos.quitar_recorte()` o el siguiente cuadro; sirve para
deslizar una sala o un texto dentro de un marco sin tapar lo que sobresale.
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
Si no hay dispositivo de audio, el juego imprime un aviso y sigue sin sonido.

## Escenas: menú, juego y fin

Casi todo juego tiene varias pantallas. La receta más simple es un `enum` con una variante por
escena, un campo con la escena actual y un `casos` en `actualizar` y otro en `pintar`:

```cometa
enum Escena
	Menu
	Jugando
	Fin

tipo Partida
	escena Escena = .Menu

	pub fn actualizar(dt decimal)
		casos @escena
			.Menu => si entrada.tecla_presionada(.Enter) @escena = .Jugando
			.Jugando => @jugar(dt)
			.Fin => si entrada.tecla_presionada(.Enter) @escena = .Menu
```

Cambiar de escena es asignar `@escena`. Como `casos` debe cubrir todas las variantes, al añadir una
escena nueva el compilador te avisa de cada lugar donde falta tratarla. Prepara el estado de la escena
en una función (`empezar`, `terminar`) que también cambie `@escena`, y deja el estado que sobrevive
entre escenas (la mejor puntuación) en campos de `Partida`. Para guardar datos entre ejecuciones usa
`std/pincel/datos`. Ejemplo completo: [12_escenas](../examples/pincel/12_escenas.cometa).

## Hojas de sprites y mapas de tiles

`graficos.hoja(imagen, ancho_cuadro, alto_cuadro)` corta una imagen (cargada con
`recursos.imagen` o dibujada con `lienzo`) en cuadros iguales numerados desde 0, de
izquierda a derecha y de arriba abajo. `graficos.cuadro` y `graficos.cuadro_v` dibujan
un cuadro con los mismos parámetros que `imagen` más `espejo_h` y `espejo_v`, que voltean
el cuadro dentro de su propio rectángulo. Un índice fuera de la hoja no dibuja nada, así que `-1` sirve como
«vacío». Como `graficos.hoja` es una función normal, no exige rutas literales.

`std/pincel/rejilla` guarda una cuadrícula de enteros, normalmente el número de tile de
cada celda. `rejilla.desde_texto` construye una a partir de filas de texto y un mapa de
símbolos (`["#": 1, ".": 0]`); un símbolo desconocido o filas de distinto largo detienen el
programa con un mensaje que indica la fila y la columna. `rejilla.dibujar` dibuja todas las
celdas con una hoja y omite las que quedan fuera de la pantalla.
`Rejilla.choca(area, tamano_celda, solidos)` dice si un rectángulo del mundo toca alguna
celda cuyo valor está en `solidos`; lo que queda fuera de la rejilla nunca es sólido
(rodéala de paredes si quieres que lo sea).
`Rejilla.mover(area, tamano_celda, delta, solidos)` desplaza el rectángulo (primero en x y luego en y) y devuelve lo que realmente se movió: se detiene pegado a la pared, así que no queda hueco, y si el resultado de un eje es menor que el pedido, hubo un choque en ese eje (caer y chocar es tocar el suelo). Mantén `delta` menor que una celda por llamada. Ver [el plataformas](../examples/pincel/13_plataformas.cometa). Es un objeto compartido: dos variables pueden apuntar a la misma rejilla.
La normal del choque se deduce del resultado: si `movido.x` es menor que `delta.x`, la pared está del lado hacia el que te movías (normal `-mate.signo(delta.x)`), y lo mismo en y.
`Rejilla.mover(..., plataformas = [2])` añade celdas de un solo sentido: frenan al caer sobre ellas, pero se atraviesan saltando desde abajo o de lado.
`Rejilla.valores_en(area, tamano_celda)` lista los valores que toca un rectángulo (por ejemplo una franja de 1 píxel bajo los pies para saber si pisas hielo) y `Rejilla.celda_en(punto, tamano_celda)` da el valor bajo un punto, como el ratón.

Para animar tiles, `rejilla.dibujar` acepta `animaciones`: un mapa de valor de celda a lista de cuadros, que se recorre a `cuadros_por_segundo` (todas las celdas van a la vez):

```cometa
rejilla.dibujar(@mapa, @hoja, animaciones = [4: [4, 5, 6]], cuadros_por_segundo = 6)
```

Para varias capas (suelo, decoración, objetos) usa una rejilla por capa con la misma hoja y dibújalas en orden; colisiona solo contra la que tenga los sólidos.
Ver [el ejemplo del laberinto](../examples/pincel/11_mapa.cometa).

## Guardar datos

`std/pincel/datos` guarda pares clave-texto en un archivo dentro de la carpeta de
configuración del usuario (`%AppData%` en Windows, `~/.config` en Linux, `~/Library/Application Support`
en macOS). Por defecto la subcarpeta es `cometa/<carpeta-del-proyecto>-<archivo>`, así que
renombrar o mover el proyecto «pierde» los datos y dos proyectos con el mismo par los
comparten. Para evitarlo, da un nombre fijo al principio de `inicio`, antes de leer nada:
`datos.juego("mi_juego")` usa `cometa/mi_juego` (los caracteres raros se cambian por `_`).
No depende del título de la ventana, así que puede leerse antes de `pincel.ejecutar`.

Para números usa `guardar_entero`/`leer_entero` y `guardar_decimal`/`leer_decimal`:
`var mejor = datos.leer_entero("mejor") o 0` (devuelven `Ninguno` si la clave falta o no
contiene un número). `datos.claves()` lista las claves guardadas en orden alfabético.
`guardar`, `guardar_*` y `borrar` devuelven un resultado (`!`) porque el disco puede fallar.

## Mandos

`entrada.mandos()` cuenta los mandos conectados con distribución estándar (los de Xbox,
PlayStation y similares); se numeran desde 0. `mando_boton` y `mando_boton_presionado` leen
botones y `mando_eje` una palanca en `[-1, 1]`, ignorando por defecto lo que quede por debajo
de `zona_muerta = 0.15`. Un mando que no existe devuelve `falso` o `0`. No se admiten pantallas táctiles.

## Números aleatorios reproducibles

`azar.semilla(n)` fija la semilla: a partir de ahí `azar.entero` y `azar.real` producen siempre
la misma secuencia, útil para niveles o «retos del día». Sin llamarla, cada ejecución es distinta.

## Comandos y alcance

`cometa compilar juego.cometa -o juego.go` produce Go formateado y sintácticamente
válido. `cometa ejecutar juego.cometa` construye y ejecuta. `cometa construir
juego.cometa -o juego.exe` construye sin abrir ventana; sin `-o` usa el nombre del
archivo sin extensión (`.exe` en Windows). Ambos usan un módulo temporal, eliminado
al finalizar. La primera compilación requiere red para descargar dependencias.

Se fija Ebitengine v2.10.1. Go 1.25+ y las dependencias nativas de Ebitengine son
necesarios. `COMETA_GO` permite elegir Go. La entrega inicial soporta escritorio;
web, móvil, pantallas táctiles, física completa, partículas y shaders quedan fuera; la interfaz
básica vive en `std/pincel/ui` (ver [ui.md](ui.md)). Los tipos y namespaces están reservados; la interfaz genérica de
los ejemplos antes llamada `graficos.Fuente` ahora se llama `Proveedor`.
