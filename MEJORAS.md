# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### El editor no sugiere `formato` en números ni `copiar` en tipos propios

- **Observado:** al escribir `precio.` o `perro.`, el autocompletado no ofrece `formato()` (métodos de `entero`/`decimal`) ni el `copiar()` incorporado de los tipos declarados; ambos funcionan al compilar y muestran su documentación al pasar el cursor solo en el caso de `formato`.
- **Problema:** son funciones nuevas que la gente solo descubre leyendo la documentación.
- **Posible solución:** que `Model.Methods` (o la lista de completado) incluya `NumberMethods()` para números y una entrada sintética `copiar` para los tipos declarados, sin que cuenten para satisfacer interfaces.

### Un literal entero no admite llamadas de método (`7.formato(2)`)

- **Observado:** `7.formato(2)` falla con «se esperaba un dígito después del punto decimal» porque el lexer lee `7.` como el inicio de un decimal; `3.5.formato(2)` sí funciona.
- **Problema:** obliga a usar una variable o paréntesis para un caso poco común, con un mensaje que no lo explica.
- **Posible solución:** que el lexer no consuma el punto si le sigue una letra, o mejorar el mensaje para sugerir `(7).formato(2)`.

### `+=` y compañía no funcionan con entradas de mapa

- **Observado:** `puntos["ana"] += 1` se rechaza con un mensaje que pide escribir `puntos["ana"] = (puntos["ana"] o 0) + 1`.
- **Problema:** contar elementos con un mapa es un patrón muy común y la forma larga es incómoda.
- **Posible solución:** admitir `m[k] += v` tratando la clave ausente como el valor cero del tipo (0, "" ), o añadir un método `sumar(clave, valor)` a los mapas.

### `casos` sobre valores no admite rangos

- **Observado:** las ramas de `casos` sobre `entero` aceptan literales y listas (`2, 3 =>`), pero no rangos como `1..5 =>` ni comparaciones.
- **Problema:** clasificar una puntuación o una edad obliga a usar una cadena de `si`/`osi`.
- **Posible solución:** admitir rangos enteros de fin exclusivo, como en `repetir`.

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
