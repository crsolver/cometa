# Cambios

Las notas de cada versión. La sección de la etiqueta publicada se usa como texto de la página de versiones.

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
