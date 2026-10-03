# Cambios

Las notas de cada versión. La sección de la etiqueta publicada se usa como texto de la página de versiones.

## v0.1.1

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
