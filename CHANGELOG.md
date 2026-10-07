# Cambios

Las notas de cada versión. La sección de la etiqueta publicada se usa como texto de la página de versiones.

## v0.1.1

### Pintar en una imagen

- **`con graficos.pintar_en(imagen)`:** lo que se dibuja dentro del bloque va a la imagen en vez de a la pantalla, sin cámara ni recorte, y al salir todo vuelve a como estaba. Sirve para minimapas, escenas a baja resolución que luego se amplían o fondos que se pintan una sola vez; el bloque puede usarse en cualquier momento, no solo desde `pintar`. Las funciones de `lienzo` ven lo pintado.
- **`graficos.recortar` con `con`:** `con graficos.recortar(rect)` quita el recorte al salir del bloque; los bloques anidados se limitan al que los contiene.
- **`graficos.liberar(imagen)`:** libera en el momento la copia de una imagen en la tarjeta gráfica; la imagen sigue siendo válida y vuelve a subirse al dibujarla.

### Entrada simulada

- **Pruebas:** `std/pruebas` puede simular la entrada: `pruebas.mantener`, `soltar` y `pulsar` (teclas), sus variantes `_raton` y `_mando`, `pruebas.raton(x, y)`, `pruebas.rueda(x, y)`, `pruebas.texto("...")` y `pruebas.eje_mando(eje, valor)`. Un ejercicio como «al pulsar D el jugador se mueve» ya se comprueba a través de `entrada`.
- **`cometa captura --entrada`:** el guion admite `texto "hola"`, `rueda x y`, botones del mando 0 (`+MandoA`) y palancas (`eje IzquierdoX 1`), y `std/pincel/ui` recibe el texto escrito.

### Editor de mapas

- **`cometa mapa`:** abre un editor de mapas de tiles en el navegador (`tools/mapa-editor`, un solo archivo HTML), con el atlas dungeon como paleta, un color por celda, capas, pintar, borrar, rellenar y rectángulo. Guarda y abre archivos `.mapa.txt` y exporta un módulo `.cometa` con los iconos y los colores de cada capa.
- **`rejilla.dibujar(..., colores = otra_rejilla)`:** una rejilla de colores (`0xRRGGBB` por celda) pinta cada celda de su color; sin ella se usa `tinte`, como antes.
- **`retro.cargar_mapa(ruta)` y `retro.mapa(mapa)`:** cargan un `.mapa.txt` y lo dibujan como se ve en el editor. `mapa.capa("suelo")` da la rejilla de una capa para consultar celdas, colisionar o cambiar el mapa mientras se juega.
- **`retro.hoja(atlas = .Dungeon)`:** el atlas retro como `graficos.Hoja`, para dibujar una rejilla con los iconos dungeon: `rejilla.dibujar(mapa, retro.hoja(), tinte = .Gris)`.
- **`rejilla.desde_csv(texto, capa = "")`:** crea una rejilla a partir del texto de un `.mapa.txt` (una fila por línea, valores separados por comas en decimal o `0x…`, `# nombre` antes de cada capa).
- **`recursos.texto(ruta)`:** incorpora un archivo de texto al programa, como `recursos.imagen` hace con las imágenes.

### Correcciones

- **Nombres que solo cambian en la mayúscula inicial:** una función `punto` y un tipo `Punto` ya pueden convivir (antes daban «error interno del compilador»). Dos campos, métodos o tipos que solo se distinguen así dan un error claro. Las variables llamadas como algo de Go (`len`, `string`, `copy`, `fmt`…) ya no rompen el programa.
- **`"${ valor }"`:** una interpolación puede llevar espacios alrededor de la expresión; antes daba un error de indentación.
- **Rutas en Windows:** los errores, los fallos en ejecución y «ir a la definición» muestran los archivos con sus mayúsculas, no la ruta entera en minúsculas.
- **Editor más ágil en juegos con recursos:** el editor ya no vuelve a leer y decodificar cada imagen y sonido con cada tecla; solo cuando el archivo cambia.
- **Go instalado dos veces:** si el primer `go` del `PATH` no funciona, Cometa prueba el siguiente en vez de fallar, y usa el del `PATH` antes que el de la carpeta del instalador.
- **`mientras` con una condición que no es `bool`:** el error habla de la condición y no de un operador `!` que no está escrito.
- Un dígito de otro alfabeto (`٣`) en un número da un error que lo explica.
- **La pista de `@` llega a todos los programas:** escribir `vida` en vez de `@vida` dentro de un método explica que falta la `@` también en los programas que usan `usar` (antes solo en los que no importaban nada), y con campos de tipos embebidos.
- **Autocompletado de `.Variante` en cualquier archivo:** las etiquetas de `casos`, los opcionales y los resultados (`.Alguno`, `.Ok`…) se completan también en los archivos con `usar`, que antes se quedaban sin esas sugerencias.
- **Errores de nombres repetidos más claros:** dicen si el nombre ya era una función, un tipo o una global.

### Herramientas

- **`cometa compilar` es más rápido** en programas de consola (de alrededor de 1 s a 0,2 s) y ya no necesita tener Go instalado: el Go generado deja de comprobarse en cada compilación; esa comprobación corre ahora en las pruebas del compilador.
- **El editor analiza menos:** los errores se recalculan cuando dejas de escribir un instante, no con cada tecla, y pasar el ratón o ir a una definición reutiliza el último análisis.
- El Go de los programas más simples sale ahora por el mismo camino que el de los demás, con una variable temporal por operando (más largo, mismo resultado).

## v0.1.0

Primera versión pública de Cometa: un lenguaje de programación con sintaxis en español, pensado para aprender y para hacer juegos 2D por diversión. Los programas se traducen a Go.

### Qué incluye

- **El lenguaje:** tipos estáticos, bloques con tabuladores, listas, mapas, opcionales (`T?`), resultados (`T!`), enums con `casos`, interfaces, genéricos y módulos con `usar`. La referencia completa está en [specs.md](https://github.com/crsolver/cometa/blob/main/specs.md).
- **Pincel**, la biblioteca de juegos 2D: gráficos, sprites y hojas, texto, cámara, teclado, ratón y mandos, audio, rejillas con colisiones, interfaz (`ui`), fuente retro, curvas de suavizado, ruido y guardado de datos.
- **Herramientas:** `cometa nuevo`, `ejecutar`, `construir`, `probar` (con pruebas visuales) y `captura`.
- **Extensión de VS Code:** colores, errores mientras escribes, autocompletado e ir a la definición.
- Errores en español, con sugerencias («¿quisiste decir…?») y la línea de tu programa cuando algo falla al ejecutar.

### Instalar

1. Descarga el archivo `cometa_…` de tu sistema (abajo), descomprímelo y pon `cometa` en tu `PATH`.
2. Instala [Go 1.25 o superior](https://go.dev/dl/).
3. Opcional: descarga el `.vsix` de tu sistema e instálalo con `code --install-extension cometa-win32-x64.vsix` (cambia el nombre por el de tu archivo).

Los ejecutables no están firmados; la [guía para empezar](https://github.com/crsolver/cometa/blob/main/docs/empezar.md) explica cómo abrirlos en Windows y macOS, y sigue con tu primer programa y tu primer juego.

### Limitaciones conocidas

Es una versión temprana: el lenguaje y la biblioteca pueden cambiar. Las asperezas conocidas están en [MEJORAS.md](https://github.com/crsolver/cometa/blob/main/MEJORAS.md).
