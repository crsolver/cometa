# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### Un literal entero no admite llamadas de método (`7.formato(2)`)

- **Observado:** `7.formato(2)` falla con «se esperaba un dígito después del punto decimal» porque el lexer lee `7.` como el inicio de un decimal; `3.5.formato(2)` sí funciona.
- **Problema:** obliga a usar una variable o paréntesis para un caso poco común, con un mensaje que no lo explica.
- **Posible solución:** que el lexer no consuma el punto si le sigue una letra, o mejorar el mensaje para sugerir `(7).formato(2)`.

### `casos` sobre valores no admite rangos

- **Observado:** las ramas de `casos` sobre `entero` aceptan literales y listas (`2, 3 =>`), pero no rangos como `1..5 =>` ni comparaciones.
- **Problema:** clasificar una puntuación o una edad obliga a usar una cadena de `si`/`osi`.
- **Posible solución:** admitir rangos enteros de fin exclusivo, como en `repetir`.

### Las listas no se pueden concatenar con `+`

- **Observado:** `normal + espejadas` falla con «el operador "+" no acepta [graficos.Imagen] y [graficos.Imagen]» (al armar los cuadros de un personaje en `experiments/pesadilla`).
- **Problema:** el mensaje no dice cómo hacerlo; hay que saber que existen `extender` y `agregar` y escribir un bucle o una copia.
- **Posible solución:** sugerir `extender` en el mensaje de error, o admitir `+` entre listas del mismo tipo devolviendo una lista nueva.

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

### `cometa captura` no puede simular entrada ni avanzar por escenas

- **Observado:** para revisar un juego en salas, jefes o pantallas de fin hubo que meter en el propio juego un «bot» temporal (constante de prueba + entrada automática) y capturar tras N cuadros.
- **Problema:** probar visualmente un juego con varias escenas obliga a modificar el código del juego; ya está anotado que `cometa probar` no tiene pruebas visuales.
- **Posible solución:** `cometa captura --entrada guion.txt` con teclas por cuadro (`60 +D`, `90 -D`, `100 Enter`) o `--cuadros 100,300,900` para varias capturas en una sola ejecución.

## Biblioteca estándar

### `graficos.texto` exige aportar un archivo de fuente

- **Observado:** el texto con tipografías vectoriales necesita `recursos.fuente("ruta.ttf")`; el repositorio no incluye ninguna, y solo `retro.texto` funciona sin archivos (8×8 píxeles). `graficos.medir_texto` ya permite centrar, pero hay que traer la fuente.
- **Problema:** quien quiere un marcador legible sin estilo retro debe buscar una fuente y respetar su licencia.
- **Posible solución:** empaquetar una fuente de licencia libre (por ejemplo Go Regular, ya presente en el runtime de Go) y exponerla como `graficos.fuente_predeterminada()`.

### Ayudantes de cámara, temporizadores y ventana

- **Observado:** `Camara2D` solo transforma; no hay seguimiento de un objetivo, límites, sacudida ni conversión pantalla → mundo (`entrada.posicion_raton` ignora la cámara). Tampoco hay un temporizador o enfriamiento, ni ocultar el cursor, poner el icono de la ventana o saber si está en pantalla completa.
- **Problema:** todo juego de desplazamiento reescribe esas piezas y el clic del ratón falla al usar cámara.
- **Posible solución:** `graficos.pantalla_a_mundo(pos)`, un tipo `Temporizador` en `std/pincel/tiempo` y funciones `ventana.cursor`/`ventana.icono`.

### Las rejillas no tienen capas ni animan tiles

- **Observado:** `Rejilla.mover` resuelve colisiones contra celdas sólidas, pero no da la normal del choque ni el tipo de celda tocada, y un mapa con varias capas (suelo, objetos) son varias rejillas dibujadas a mano. Los tiles no se animan.
- **Problema:** las pendientes, plataformas que se atraviesan desde abajo o suelos de hielo exigen más lógica propia.
- **Posible solución:** un tipo `Capas` que agrupe rejillas y un ayudante de cuadros animados.

### `datos` solo guarda texto y depende del nombre de la carpeta del proyecto

- **Observado:** los valores son cadenas (hay que convertir con `cadena()`/`a_entero()`), y el archivo se identifica por `carpeta-del-proyecto`+`archivo`; si se renombra o se mueve el proyecto, la mejor marca «desaparece», y dos proyectos con el mismo par comparten datos.
- **Problema:** puede sorprender y no hay forma de listar las claves guardadas.
- **Posible solución:** aceptar un nombre de juego explícito (`datos.juego("mi_juego")`) y añadir `datos.claves()` y variantes numéricas.

### Los mandos solo admiten la distribución estándar

- **Observado:** `entrada.mandos()` ignora los mandos sin asignación estándar y no hay vibración ni aviso de conexión o desconexión.
- **Problema:** algunos mandos genéricos no aparecen.
- **Posible solución:** exponer los botones y ejes sin procesar como respaldo y un evento de conexión.

### Sin prueba automática del dibujo de hojas, rejillas y mandos

- **Observado:** las pruebas cubren la lógica sin ventana (`rejilla`, `datos`, `azar`, `mate`) y que el resto compila, y se comprobó a mano con capturas, pero ninguna prueba compara píxeles de `graficos.cuadro`/`rejilla.dibujar` ni lee un mando.
- **Problema:** un cambio en la orientación del espejo o del origen podría pasar inadvertido.
- **Posible solución:** una prueba opcional con `cometa captura` que compare una imagen de referencia pequeña generada con `lienzo`.

### `ui.campo_texto` no admite selección de texto ni portapapeles

- **Observado:** el campo de texto de `std/pincel/ui` inserta y borra caracteres y mueve el cursor con las flechas/Inicio/Fin, pero no se puede seleccionar texto (arrastrar o Mayús+flecha), hacer clic para posicionar el cursor en medio del texto, ni pegar desde el portapapeles.
- **Problema:** cubre el caso de un campo corto (nombre, valor numérico), pero cualquier edición más larga es incómoda: no hay forma de reemplazar todo el contenido de una vez ni de corregir un error a mitad de la palabra sin usar solo las flechas.
- **Posible solución:** añadir un segundo índice de selección al nodo, resaltar el rango seleccionado al dibujar, calcular la posición de clic contra las líneas medidas, y usar las funciones de portapapeles de Ebitengine para copiar/pegar.

### No se puede destellar un sprite en blanco ni recortar el dibujo

- **Observado:** `tinte` multiplica el color, así que no sirve para el destello blanco al recibir un golpe; en `pesadilla` cada sprite se genera dos veces (con la paleta normal y con una paleta toda blanca). Tampoco hay una región de recorte: para deslizar una sala dentro de un marco hubo que tapar lo que sobresalía con rectángulos del color de fondo.
- **Problema:** dos efectos muy comunes en juegos pixel art (destello y transiciones dentro de un recuadro) obligan a rodeos.
- **Posible solución:** un parámetro `relleno color.Color?` en `graficos.imagen`/`cuadro` que pinte la silueta, y `graficos.recortar(rect)` / `graficos.quitar_recorte()`.

### Las posiciones decimales desenfocan los sprites de pixel art

- **Observado:** con `pixelado = verdadero`, `graficos.imagen(img, 10.5, 20.3)` dibuja el sprite en una posición fraccionaria y los píxeles salen de distinto grosor; `pesadilla` redondea con un ayudante (`mate.redondear`) en cada llamada.
- **Problema:** es fácil olvidarlo y el resultado se ve mal sin explicación.
- **Posible solución:** que `pixelado = verdadero` ajuste a píxel entero las posiciones de imagen (o un parámetro `ajustar = verdadero`), y documentarlo en `docs/juegos.md`.

## Plataforma de ejercicios

### Falta lectura de entrada estándar en programas de consola

- **Observado:** `imprimir` es la única E/S de consola; no hay forma de leer `stdin`.
- **Problema:** los ejercicios de consola no pueden recibir datos de entrada; hoy solo valen funciones/estructuras y juegos.
- **Posible solución:** un módulo `std/consola` con `leer_linea() cadena?`.

### La API del compilador está bajo `internal/`

- **Observado:** `compiler.AnalyzeProject`/`CompileProject` no se pueden importar desde otro módulo Go.
- **Problema:** un servidor de plataforma tiene que invocar el binario `cometa` como proceso.
- **Posible solución:** publicar un paquete estable (p. ej. `pkg/cometa`) o mantener la CLI con `--json` como contrato.

### `cometa probar` aún no admite pruebas visuales

- **Observado:** solo hay afirmaciones sobre valores; no hay forma de avanzar un `Juego` N cuadros y comprobar píxeles de la pantalla (ver paso 5 del plan).
- **Posible solución:** `pruebas.avanzar(juego, cuadros)` y `pruebas.pantalla()` apoyados en el modo `captura`.
