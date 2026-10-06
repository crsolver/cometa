# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### Un literal entero no admite llamadas de método (`7.formato(2)`)

- **Observado:** `7.formato(2)` falla con «se esperaba un dígito después del punto decimal» porque el lexer lee `7.` como el inicio de un decimal; `3.5.formato(2)` sí funciona.
- **Problema:** obliga a usar una variable o paréntesis para un caso poco común, con un mensaje que no lo explica.
- **Posible solución:** que el lexer no consuma el punto si le sigue una letra, o mejorar el mensaje para sugerir `(7).formato(2)`.

### `repetir` no recorre cadenas

- **Observado:** `repetir (texto) |letra|` falla con «repetir requiere una lista o mapa, no cadena» (`internal/sema/sema.go`). Hay que escribir `repetir (texto.dividir("")) |letra|`.
- **Problema:** recorrer las letras de un texto es un ejercicio típico de principiantes, y el truco de `dividir("")` no es evidente.
- **Posible solución:** aceptar `cadena` en `repetir`, ligando cada punto de código como `cadena` (y el índice opcional como `entero`), igual que `dividir("")`; o al menos sugerir `dividir("")` en el mensaje de error.

### Las constantes de tipos de la biblioteca no se pueden escribir con su tipo (`retro.Icono.Arriba`)

- **Observado:** en una rama de `casos` que debe producir un ícono, `retro.Icono.Arriba` falla con «el nombre "retro.Icono" no existe». Solo funciona la forma contextual `.Arriba`, y para eso hay que anotar el tipo de la variable (`var flecha retro.Icono = casos ...`). Con los enums propios, `Semaforo.Rojo` sí funciona.
- **Problema:** cuando no hay un tipo esperado (un `var` sin anotar, un valor de `casos`), no hay forma directa de nombrar un ícono, color o tecla, y el mensaje no sugiere la anotación.
- **Posible solución:** aceptar `modulo.Tipo.Constante` para los tipos de la biblioteca con constantes (`Icono`, `Color`, `Tecla`…), igual que con los enums propios; o que el error sugiera anotar el tipo.

### Una función con resultado no puede terminar en un `repetir` infinito

- **Observado:** esta función falla con «el bloque debe producir entero en todos los caminos», aunque el ciclo solo se abandona con `retornar`:

  ```cometa
  fn primer_multiplo(numero entero, desde entero) entero
  	var candidato = desde
  	repetir
  		si candidato % numero == 0
  			retornar candidato
  		candidato += 1
  ```

- **Problema:** «buscar hasta encontrar y devolver» es un patrón natural para principiantes; obliga a reescribirlo con `mientras` y un valor final, o a añadir una última línea que nunca se ejecuta. El mensaje tampoco explica que el problema es el final de la función. Visto al escribir el curso «desde cero» de `learn` (lección `18-dragon`).
- **Posible solución:** tratar un `repetir` sin rango y sin `romper` como un bloque que nunca termina (igual que Go con `for {}`), de modo que no se exija un valor después.

### La forma corta de `si` no admite una variante contextual (`si grande .Claro sino .Oscuro`)

- **Observado:** con un tipo enum esperado, `si grande .Claro sino .Oscuro` falla con «se esperaba una expresión, pero se encontró 'sino'» (también con la condición entre paréntesis); con un literal antes, `si valor > 1 .Amarillo sino .Celeste` da «se esperaba el valor de la rama 'si'». El punto se lee como acceso a un miembro de la condición. La forma en varias líneas sí funciona.
- **Problema:** la forma corta funciona con números y cadenas (`si a > b a sino b`), así que sorprende que falle justo con variantes y constantes como los colores, y el mensaje no da ninguna pista.
- **Posible solución:** al analizar la condición de un `si`, no continuar con `.` cuando va precedido de un espacio; o que el error sugiera escribir las ramas en líneas separadas.

### La forma corta de `si` tampoco admite una rama negativa (`si izquierda -1 sino 0`)

- **Observado:** `direccion = si izquierda -1 sino 0` falla con «se esperaba el valor de la rama 'si', pero se encontró 'sino'»: se lee como la resta `izquierda - 1`. Con paréntesis, `si izquierda (-1) sino 0`, da el mismo error porque se lee como la llamada `izquierda(-1)`. Visto en `experiments/tetris` (dirección de la repetición de teclas).
- **Problema:** elegir entre `-1`, `0` y `1` es de lo más común en un juego, y ninguna de las dos formas naturales funciona; hay que usar tres líneas. Es el mismo origen que la entrada anterior: la condición no tiene un final marcado.
- **Posible solución:** un separador opcional para la forma corta (`si izquierda entonces -1 sino 0`), o en la condición de un `si` corto no continuar con `-`/`(` cuando van precedidos de un espacio y seguidos sin espacio; como mínimo, que el error sugiera la forma en varias líneas.

### Una función de una línea no puede empezar su cuerpo con `!` (`fn en_suelo() bool !@cabe(...)`)

- **Observado:** `fn en_suelo() bool !@cabe(@giro, @px, @py + 1)` compila la declaración sin queja, pero el tipo de resultado se lee como `bool!` (un resultado) y el cuerpo como `@cabe(...)`. El error aparece lejos, en cada uso: `si @en_suelo()` da «la condición debe ser bool, no bool!». Visto en `experiments/tetris`.
- **Problema:** negar una condición es un cuerpo de una línea muy natural, y el mensaje señala la llamada y no la declaración, así que cuesta relacionarlo con el `!`.
- **Posible solución:** exigir que el `!` de un tipo de resultado vaya pegado al tipo y el de la negación pegado a su operando (aquí hay un espacio antes del `!`, que ya los distingue); o avisar en la declaración cuando el tipo `T!` va seguido de un espacio y una expresión en la misma línea.

### El error de `.Alguno |valor| =>` no dice dónde va la ligadura

- **Observado:** en un `casos` sobre un opcional, escribir la ligadura en la rama (`.Alguno |pista| => ...`) falla con «se esperaba '=>' después del patrón, pero se encontró '|'». La forma correcta la pone en la cabecera: `casos opcional |pista|`.
- **Problema:** `si opcional |valor|` y `atrapar |error|` ligan justo donde se usa el valor, así que es fácil esperar lo mismo en la rama; el mensaje no orienta.
- **Posible solución:** que el error diga «la ligadura se escribe en la cabecera: `casos valor |nombre|`».

### No hay operadores de bits (`&`, `|`, `^`, `<<`, `>>`)

- **Observado:** en `experiments/ajedrez/motor.cometa` una jugada se guarda en un entero (casilla de salida, de llegada, coronación y clase). Sin desplazamientos ni máscaras hay que empaquetar con `d + h * 64 + corona * 4096` y leer con `m / 64 % 64`; los cuatro derechos de enroque acabaron en cuatro campos `entero` porque no se puede apagar un bit, y la clave de la posición suma números al azar en vez de combinarlos con XOR (Zobrist).
- **Problema:** banderas, máscaras, colores `0xRRGGBB` y claves de dispersión son habituales en juegos; la aritmética los imita, pero es más frágil (con negativos `%` da restos negativos) y oculta la intención.
- **Posible solución:** operadores `&`, `|`, `^`, `<<` y `>>` sobre `entero`, o funciones `mate.bit_y`, `mate.bit_o`, `mate.bit_xor`, `mate.desplazar` si no se quieren más símbolos; junto con los literales `0x…` que ya pide otra entrada.

## Herramientas

### `cometa captura` escribe en los datos guardados reales del juego

- **Observado:** al capturar `experiments/tetris` con un guion de entrada que llega al final de la partida, `datos.guardar_entero("mejor", ...)` escribió en `%AppData%\cometa\cometa_tetris`, la misma carpeta que usa el juego al jugarlo de verdad. Hubo que borrarla a mano.
- **Problema:** una captura o una prueba visual deja una mejor puntuación falsa (o pisa la del jugador), y dos capturas seguidas no parten del mismo estado.
- **Posible solución:** que `captura` y `probar` redirijan `std/pincel/datos` a una carpeta temporal vacía (o a memoria), con una opción `--datos carpeta` para partir de un guardado concreto.

### Los errores en tiempo de ejecución no indican la columna

- **Observado:** un pánico (índice fuera de rango, división entre cero) ahora muestra archivo, línea y el texto de la línea, pero no la columna ni qué expresión falló.
- **Problema:** en una línea con varias listas o divisiones no se sabe cuál causó el error.
- **Posible solución:** emitir directivas `//line archivo:línea:columna` (Go las admite) en las expresiones que pueden fallar (índices y divisiones), no solo en cada sentencia.

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

### Las listas no se pueden ordenar

- **Observado:** `[T]` no tiene `ordenar`. En `experiments/ajedrez/motor.cometa` las jugadas se ordenan por puntuación con una selección escrita a mano sobre dos listas paralelas (`movs` y `puntos`), y las piezas capturadas se insertan ya ordenadas con un `mientras`.
- **Problema:** ordenar una tabla de récords, un inventario o los sprites por su `y` antes de dibujarlos es de lo primero que pide un juego, y obliga a enseñar un algoritmo de ordenación antes de tiempo.
- **Posible solución:** `lista.ordenar()` para `entero`, `decimal` y `cadena` (con `descendente = falso`), y una forma con clave para estructuras, por ejemplo `lista.ordenar_por(claves [decimal])` mientras no haya funciones como valores.

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

- **Observado:** la rejilla de colores de un mapa (`Mapa.colores`, o el parámetro `colores` de `rejilla.dibujar`) guarda cada color como `rojo * 65536 + verde * 256 + azul`. No hay forma de obtener ese número de un `color.Color`, ni de escribirlo en hexadecimal (Cometa no tiene literales `0x…`).
- **Problema:** recolorear una celda al jugar obliga a calcular el número a mano (`colores.poner(x, y, 255 * 65536)`) en vez de escribir `.Rojo`.
- **Posible solución:** `color.a_entero(c Color) entero` y `color.desde_entero(n entero) Color`, y literales hexadecimales en el lexer.

### `retro.texto` no dibuja las mayúsculas acentuadas

- **Observado:** `retro.texto("LA PIRÁMIDE", ...)` dibuja «LA PIR?MIDE»; las minúsculas (`á`, `ó`) y `¡` sí salen. Visto con el `cometa` que acompaña a `learn/bin` (cometa dev); falta confirmarlo con la versión actual.
- **Problema:** los títulos de los juegos suelen ir en mayúsculas, y en español obliga a escribirlos en minúsculas o sin tilde. El «?» aparece sin ningún aviso.
- **Posible solución:** añadir `Á É Í Ó Ú Ü` a la fuente de `retro` (o dibujarlas con la letra base y la tilde encima).
