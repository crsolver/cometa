# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### Una lista no se puede interpolar en una cadena

- **Observado:** `imprimir("Notas: ${notas}")` falla con «la interpolación requiere cadena, entero, decimal o bool, no [entero]», y `cadena(notas)` tampoco existe. `imprimir(notas)` sí la escribe (`[7, 9, 6]`).
- **Problema:** para mostrar una lista junto a un texto hay que usar dos `imprimir` o construir la cadena con un bucle; lo mismo ocurre con mapas, estructuras y enums.
- **Posible solución:** aceptar en la interpolación (y en `cadena(x)`) cualquier valor que `imprimir` sepa escribir, con el mismo formato.

## Herramientas

### El manejo de fallos de juegos Ebitengine no está probado de extremo a extremo

- **Observado:** la traducción de pánicos (`cmd/cometa/errors.go`) se comprobó a mano con un juego de Pincel que falla dentro de `actualizar` (mensaje y línea correctos), pero no hay una prueba automática: exige descargar y compilar Ebitengine.
- **Problema:** una actualización de Ebitengine podría cambiar el formato del pánico sin que ninguna prueba lo detecte.
- **Posible solución:** una prueba opcional (activada por variable de entorno) que ejecute con `cometa captura` un juego que falle y compruebe el mensaje traducido; correrla en CI.

### Los flujos de CI y de publicación no se han ejecutado

- **Observado:** `.github/workflows/ci.yml` y `release.yml`, junto con `.goreleaser.yaml`, se escribieron sin poder ejecutarlos.
- **Problema:** el trabajo de Linux supone que una pantalla virtual (`xvfb`) basta para la prueba de captura con ventana oculta, y el empaquetado de los `.vsix` por plataforma no se ha probado.
- **Posible solución:** publicar una etiqueta de prueba (`v0.0.1-rc1`) en un repositorio de ensayo y corregir lo que falle.

### `cometa nuevo` no ofrece plantillas más allá de consola y un juego mínimo

- **Observado:** solo crea `principal.cometa` con «hola» o un cuadrado que se mueve.
- **Problema:** no hay punto de partida para un juego con varios archivos (el ejemplo `12_escenas` muestra las escenas, pero no hay plantilla).
- **Posible solución:** más plantillas (`--plantilla plataformas`, `menu`) cuando existan esos ejemplos.

### La entrada simulada solo controla el mando 0

- **Observado:** `pruebas.mantener_mando`, `pruebas.eje_mando` y el guion de `cometa captura --entrada` (`+MandoA`, `eje IzquierdoX 1`) afectan solo al mando 0; `entrada.mandos()` vale 0 o 1 mientras se simula.
- **Problema:** un juego de dos jugadores con un mando cada uno no se puede probar ni capturar.
- **Posible solución:** un parámetro `mando` opcional en las funciones de `pruebas` y un prefijo en el guion (`+Mando1A`), con los códigos de botón desplazados por mando.

## Biblioteca estándar

### `audio` no genera sonidos: un juego sin archivos es mudo

- **Observado:** `std/pincel/audio` solo reproduce lo que carga `recursos.sonido` desde un WAV/Ogg/MP3, y `audio.reproducir` no cambia el tono ni la velocidad. Para `experiments/tetris` hubo que escribir un script de Node (`sonidos.js`) que sintetiza doce WAV (pitidos, ruido y la música).
- **Problema:** `retro` y `lienzo` permiten un juego completo sin recursos, salvo el sonido: el principiante tiene que buscar o fabricar archivos fuera de Cometa. Sin control de tono tampoco se puede variar un efecto (un pitido más agudo por cada fila de un combo) ni acelerar la música con el nivel.
- **Posible solución:** un sintetizador mínimo, por ejemplo `audio.tono(frecuencia, duracion, onda = .Cuadrada, hasta = frecuencia, volumen = 1) Sonido` y `audio.ruido(duracion)`, que devuelvan un `Sonido` creado en memoria (como `lienzo.nuevo` con las imágenes); y un parámetro `tono decimal = 1` en `audio.reproducir`.

### No se puede medir el tiempo real

- **Observado:** `std/pincel/tiempo` da `fps()`, `tps()` y temporizadores que avanzan con `dt`, pero no un reloj. El motor de `experiments/ajedrez` reparte su búsqueda entre cuadros contando nodos (`presupuesto = 14000` por cuadro), un número ajustado a mano midiendo con `cometa probar`.
- **Problema:** un cálculo largo (búsqueda, generación de un nivel, camino más corto) no puede decir «trabaja 8 ms y sigue en el próximo cuadro», así que en una máquina lenta la pantalla se traba y en una rápida se desaprovecha. Tampoco se puede mostrar la hora ni cronometrar un trozo de código.
- **Posible solución:** `tiempo.ahora() decimal` (segundos desde que empezó el programa, con reloj monótono), sin depender de Ebitengine para poder usarlo también en pruebas.

### `color.Color` no se puede mezclar ni descomponer

- **Observado:** no hay forma de interpolar dos colores, leer sus canales ni cambiarles el alfa. En `experiments/tetris` la rampa ámbar (`rampa(nivel, alfa)`) repite a mano los números de la paleta e interpola cada canal con `mate.interpolar` y `mate.redondear`.
- **Problema:** los desvanecimientos, destellos y degradados (lo más habitual al pulir un juego) obligan a guardar los colores dos veces: como `Color` y como tres enteros sueltos.
- **Posible solución:** `color.mezclar(a Color, b Color, t decimal) Color`, `Color.con_alfa(alfa entero) Color` y campos de lectura `r`, `g`, `b`, `a`.

### No hay forma directa de rellenar un número con ceros o repetir una cadena

- **Observado:** para pintar `0000208` o `03:07` hay que construir los ceros con un bucle (`repetir (texto.longitud()..cifras) ceros += "0"`) o una función propia (`dos_cifras`). `formato(n)` solo fija los decimales.
- **Problema:** marcadores y relojes aparecen en casi todos los juegos.
- **Posible solución:** `cadena.rellenar_izquierda(ancho entero, relleno cadena = " ")` (y `rellenar_derecha`), `cadena.repetir(veces entero)`, o un segundo parámetro en `formato(decimales, ancho = 0)`.

### Las rejillas no admiten pendientes

- **Observado:** ya hay plataformas de un solo sentido (`mover(..., plataformas = [...])`), consulta de celdas (`celda_en`, `valores_en`) y tiles animados, pero las celdas son siempre cuadradas y sólidas por completo.
- **Problema:** un plataformas con rampas exige calcular la altura del suelo a mano.
- **Posible solución:** un valor de celda con forma (rampa a 45°/22,5°) que `mover` resuelva ajustando y.

### Los mandos solo admiten la distribución estándar

- **Observado:** `entrada.mandos()` ignora los mandos sin asignación estándar y no hay vibración ni aviso de conexión o desconexión.
- **Problema:** algunos mandos genéricos no aparecen.
- **Posible solución:** exponer los botones y ejes sin procesar como respaldo y un evento de conexión.

### Sin prueba automática de los mandos

- **Observado:** `TestSpriteSheetAndGridPixels` ya compara píxeles de hojas, espejos, orígenes, rejillas y tiles animados, pero ninguna prueba lee un mando: Ebitengine no permite simular uno.
- **Problema:** un cambio en la asignación de botones o ejes estándar pasaría inadvertido.
- **Posible solución:** aislar la conversión de `BotonMando`/`EjeMando` en funciones puras y probarlas, o una prueba manual guiada con un mando conectado.

### `ui.campo_texto` no admite selección de texto ni portapapeles

- **Observado:** el campo de texto de `std/pincel/ui` inserta y borra caracteres y mueve el cursor con las flechas/Inicio/Fin, pero no se puede seleccionar texto (arrastrar o Mayús+flecha), hacer clic para posicionar el cursor en medio del texto, ni pegar desde el portapapeles.
- **Problema:** cubre el caso de un campo corto (nombre, valor numérico), pero cualquier edición más larga es incómoda: no hay forma de reemplazar todo el contenido de una vez ni de corregir un error a mitad de la palabra sin usar solo las flechas.
- **Posible solución:** añadir un segundo índice de selección al nodo, resaltar el rango seleccionado al dibujar, calcular la posición de clic contra las líneas medidas, y usar las funciones de portapapeles de Ebitengine para copiar/pegar.


### `retro.icono` no se puede voltear

- **Observado:** `retro.icono` dibuja la celda del atlas tal cual (`_hgretroIcono` llama a `_hgretroGlifo` sin más opciones); no hay forma de espejarla.
- **Problema:** para tener un icono mirando a la izquierda y a la derecha, o boca arriba y boca abajo, hay que añadir al atlas una copia espejada de cada uno.
- **Posible solución:** parámetros opcionales `voltear_h` y `voltear_v` (bool) en `retro.icono` (y quizá en `glifo`) que inviertan los ejes al dibujar, como ya hace `lienzo.pegar` con los espejos.


### Los mapas del editor no tienen entidades

- **Observado:** el editor (`tools/mapa-editor`) guarda solo el icono y el color de cada celda. Para saber dónde empieza el jugador o cuántas llaves hay, el juego recorre la rejilla con dos `repetir` y `obtener(x, y) o -1` (ver `examples/pincel/16_editor_mapas.cometa`).
- **Problema:** colocar enemigos, puntos de inicio o puertas exige reservar índices «mágicos» y escribir el mismo bucle de búsqueda en cada juego.
- **Posible solución:** `Rejilla.buscar(valor) [mate.Vec2]` (las celdas que tienen ese valor) para leer entidades de una capa de objetos; en el editor, una capa de marcas con nombre exportada como lista.

### Un color de rejilla es un entero sin conversión desde `Color`

- **Observado:** la rejilla de colores de un mapa (`Mapa.colores`, o el parámetro `colores` de `rejilla.dibujar`) guarda cada color como `rojo * 65536 + verde * 256 + azul`. No hay forma de obtener ese número de un `color.Color`, aunque ya se puede escribir en hexadecimal (`0xFF0000`).
- **Problema:** recolorear una celda al jugar obliga a calcular el número a mano (`colores.poner(x, y, 255 * 65536)`) en vez de escribir `.Rojo`.
- **Posible solución:** `color.a_entero(c Color) entero` y `color.desde_entero(n entero) Color`.

### `retro.texto` no dibuja las mayúsculas acentuadas

- **Observado:** `retro.texto("LA PIRÁMIDE", ...)` dibuja «LA PIR?MIDE»; las minúsculas (`á`, `ó`) y `¡` sí salen. Visto con el `cometa` que acompaña a `learn/bin` (cometa dev); falta confirmarlo con la versión actual.
- **Problema:** los títulos de los juegos suelen ir en mayúsculas, y en español obliga a escribirlos en minúsculas o sin tilde. El «?» aparece sin ningún aviso.
- **Posible solución:** añadir `Á É Í Ó Ú Ü` a la fuente de `retro` (o dibujarlas con la letra base y la tilde encima).
