# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### El error de una interfaz no cumplida no dice qué falta

- **Observado:** al escribir la lectura `19-interfaces` y el ejercicio `20-bestiario` del curso `04-intermedio`. Si un tipo no cumple una interfaz, el error es «el literal es Lobo, pero se esperaba Criatura», «el elemento debe ser Criatura, no Lobo» o «no se puede asignar Golem a Criatura». Sale igual cuando falta un método, cuando su resultado es de otro tipo, cuando el método existe pero no lleva `pub`, y cuando viene de un tipo embebido que no es `pub`.
- **Problema:** el caso de `pub` es el más confuso: el método está escrito, con la firma correcta y en el mismo archivo, y nada indica que la visibilidad sea el motivo. El curso tiene que dar una lista de tres cosas para revisar a mano.
- **Posible solución:** añadir el motivo al mensaje: «a Lobo le falta el método atacar() entero», «Lobo.atacar devuelve decimal y Criatura pide entero», «Lobo.atacar existe pero no es pub».

### `imprimir` muestra el campo embebido en minúscula

- **Observado:** con `tipo Lobo` que embebe `Posicion`, `imprimir(lobo)` escribe `Lobo {posicion: Posicion {x: 2, y: 0}, vida: 5}`.
- **Problema:** en el código el campo se llama `Posicion` (`lobo.Posicion`, `Lobo {Posicion: {...}}`); `posicion` en minúscula no existe y copiarlo de la salida da un error.
- **Posible solución:** imprimir el nombre del campo embebido con la capitalización del tipo.

### En las pruebas, dos `pruebas.pulsar` en cuadros seguidos cuentan como una sola pulsación

- **Observado:** en las pruebas de `25-escenas`. `pruebas.pulsar(.Derecha)` seguido de `pruebas.avanzar(juego, 1)`, repetido tres veces, solo produce un `entrada.tecla_presionada`. Con `pruebas.avanzar(juego, 2)` después de cada pulsación funcionan las tres.
- **Problema:** `docs/pruebas.md` dice que la tecla «se suelta al cuadro siguiente», pero no que una nueva pulsación en ese mismo cuadro se pierde. La prueba falla con un número de pasos menor sin ninguna pista de por qué.
- **Posible solución:** que `pulsar` sobre una tecla que se está soltando cuente como pulsación nueva, o documentar que entre dos pulsaciones de la misma tecla hace falta un cuadro libre.

### `repetir (0..n)` cuenta hacia atrás cuando `n` es negativo

- **Observado:** al revisar el generador de código. Un rango cuyo inicio es mayor que el fin se recorre hacia atrás (`repetir (5..0)`), y eso también pasa cuando el fin sale de una cuenta: con una lista vacía, `repetir (0..lista.longitud() - 1) |i|` recorre `0..-1`, ejecuta el cuerpo una vez con `i = 0` y `lista[i]` falla con «índice fuera de rango».
- **Problema:** quien escribe `0..n` espera cero vueltas cuando `n` es menor que cero, como en casi todos los lenguajes; el recorrido hacia atrás aparece sin haberlo pedido y el fallo llega en otra línea.
- **Posible solución:** que la dirección no se decida al ejecutar: un rango ascendente vacío no da vueltas, y el recorrido hacia atrás se pide de forma explícita (por ejemplo `repetir (5..0, paso = -1)` o `repetir (0..5).invertido()`).

## Herramientas

### El Go generado guarda cada operando en una variable temporal

- **Observado:** al unificar los dos generadores de código. `var lista = [1, 2, 3]` se convierte en cuatro líneas de Go (`var _cometa1 int64 = 1`, …, `var lista []int64 = _cometa4`). Antes los programas más simples (sin listas, cadenas ni biblioteca) salían en una forma directa; ahora todos pasan por el mismo camino, que conserva el orden de evaluación.
- **Problema:** `cometa compilar` sirve también para leer el Go que produce un programa, y las temporales lo alargan. No afecta a lo que hace el programa ni, en la práctica, a su velocidad.
- **Posible solución:** una pasada final (junto a `eliminateUnusedLocals`) que sustituya las temporales de un solo uso cuyo valor es un literal; es segura porque un literal no depende del momento en que se evalúa.

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


### El editor de mapas no tiene marcas con nombre

- **Observado:** `Rejilla.buscar(valor)` ya devuelve las celdas de un icono (el inicio del jugador, las llaves), pero el editor (`tools/mapa-editor`) solo guarda el icono y el color de cada celda: una entidad sigue siendo un índice «mágico» del atlas.
- **Problema:** para colocar enemigos, puntos de inicio o puertas hay que reservar un icono por cada clase de entidad y recordar su número.
- **Posible solución:** en el editor, una capa de marcas con nombre exportada como lista, y `Mapa.marcas(nombre) [mate.Vec2]`.
