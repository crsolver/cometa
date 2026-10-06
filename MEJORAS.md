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

## Herramientas

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
