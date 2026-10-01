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

## Herramientas

### El lexer rechaza archivos con BOM UTF-8

- **Observado:** un `.cometa` guardado con BOM (`Set-Content -Encoding utf8` de Windows PowerShell 5.1, algunos editores de Windows) falla en 1:1 con «carácter inesperado '﻿'», y los errores siguientes son confusos (`módulo desconocido "mate"`, porque se pierde el `usar` de la primera línea).
- **Problema:** el carácter es invisible, así que el mensaje no ayuda a un principiante a entender qué pasa.
- **Posible solución:** ignorar un BOM inicial al leer el archivo (en el lexer o en el cargador de fuentes).

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

### El guion de `cometa captura --entrada` no escribe texto ni usa mandos

- **Observado:** el guion simula teclas, botones y posición del ratón, pero no la escritura de caracteres (`ui.campo_texto` recibe texto vacío) ni los mandos, y la rueda siempre vale cero.
- **Problema:** no se puede capturar un formulario rellenado ni un juego que solo se controle con mando.
- **Posible solución:** acciones `texto "hola"`, `rueda 0 -1` y `+MandoA`/`eje IzquierdoX 1` en el guion, leídas por los ayudantes de `entrada` igual que las teclas.

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

## Plataforma de ejercicios

### Las pruebas visuales no pueden simular teclado ni ratón

- **Observado:** `pruebas.avanzar` ignora los dispositivos reales, pero no hay forma de pulsar una tecla o mover el ratón desde una prueba; el guion de `cometa captura --entrada` no está expuesto en `std/pruebas`.
- **Problema:** un ejercicio del tipo «al pulsar D el jugador se mueve» solo se puede comprobar llamando a los métodos del juego, no a través de `entrada`.
- **Posible solución:** `pruebas.mantener(tecla)`, `pruebas.soltar(tecla)`, `pruebas.pulsar(tecla)` y `pruebas.raton(x, y)` que escriban en el estado simulado (`_hgsimAhora`, `_hgsimCursor`) que ya leen los ayudantes de `entrada`.
