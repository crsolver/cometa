# Empezar con Cometa

Cometa es un lenguaje con sintaxis en español, pensado para aprender y para hacer juegos 2D por diversión. Esta guía te lleva desde la instalación hasta tu primer juego.

## 1. Qué necesitas

- El programa `cometa`: descarga el archivo de tu sistema de la [página de versiones](https://github.com/crsolver/cometa/releases/latest), descomprímelo y pon `cometa` en tu `PATH`.
- **Go 1.25 o superior** ([go.dev/dl](https://go.dev/dl/)). Cometa se traduce a Go, y `cometa ejecutar` usa Go para compilar tu programa. Comprueba que funciona con `go version`.
- Un editor. Para VS Code hay una extensión con colores, errores mientras escribes y autocompletado (ver [Extensión de VS Code](#extensión-de-vs-code)).
- En Linux, los juegos necesitan además las bibliotecas de desarrollo de Ebitengine (consulta [ebitengine.org](https://ebitengine.org/en/documents/install.html)).

Comprueba la instalación:

```console
cometa --version
```

El ejecutable no está firmado, así que el sistema puede desconfiar de él la primera vez:

- **Windows:** si aparece «Windows protegió su PC», pulsa **Más información** y luego **Ejecutar de todas formas**.
- **macOS:** si dice que no se puede abrir porque no se puede verificar al desarrollador, ejecuta `xattr -d com.apple.quarantine cometa` en la carpeta donde lo pusiste.

## Extensión de VS Code

La extensión da colores, errores mientras escribes, autocompletado, ir a la definición y sangría con tabuladores. Todavía no está en el Marketplace de VS Code, así que se instala a mano. Necesitas VS Code 1.106 o superior.

**Opción A: desde un `.vsix`.** Descarga el archivo de tu sistema desde la [página de versiones](https://github.com/crsolver/cometa/releases/latest) (por ejemplo `cometa-win32-x64.vsix`, `cometa-linux-x64.vsix` o `cometa-darwin-arm64.vsix`). Ya incluye el servidor de lenguaje. Instálalo con:

```console
code --install-extension cometa-win32-x64.vsix
```

También puedes abrir la vista de extensiones, pulsar `⋯` y elegir **Instalar desde VSIX…**.

**Opción B: desde el código fuente.** Necesitas Go 1.25+ y Node.js 22+:

```console
git clone https://github.com/crsolver/cometa.git
cd cometa/vscode-extension
npm install
npm run build
npx @vscode/vsce package
code --install-extension cometa-language-0.1.0.vsix
```

Si solo quieres probarla sin instalarla, abre la carpeta `cometa` en VS Code y pulsa `F5` (**Run Cometa Extension**).

Reinicia VS Code y abre un archivo `.cometa`: deberías ver el logo de Cometa y los colores. Si algo falla:

- Mira **Salida → Cometa Language Server**.
- Si usas un `cometa` propio, indícalo en el ajuste `cometa.server.path` (ruta absoluta al ejecutable). Si lo dejas vacío, se usa el que viene con la extensión.
- Algunos temas de íconos (como Material Icon Theme) muestran su propio ícono en lugar del logo; los colores y el resto de funciones no cambian.

## 2. Tu primer programa

```console
cometa nuevo hola
cd hola
cometa ejecutar principal.cometa
```

`cometa nuevo` crea la carpeta `hola` con un archivo `principal.cometa` y, para quien use un asistente de IA, `AGENTS.md` y `CLAUDE.md`: unas instrucciones que le explican el lenguaje y le piden actuar como profesor en lugar de escribirte el código (Cometa es para aprender y divertirse). Añade `--sin-agentes` si no las quieres.

El archivo `principal.cometa` contiene:

```cometa
fn saludar(nombre cadena) cadena
	"¡Hola, ${nombre}!"

fn inicio()
	imprimir(saludar("mundo"))
```

- `fn inicio()` es donde empieza el programa.
- **Las sangrías se escriben con tabuladores**, no con espacios. La extensión de VS Code ya lo hace por ti.
- La última expresión de una función es lo que devuelve.
- `${...}` inserta valores dentro de un texto.

Prueba a cambiar `"mundo"` por tu nombre y vuelve a ejecutar.

## 3. Tu primer juego

```console
cometa nuevo mi_juego --juego
cd mi_juego
cometa ejecutar principal.cometa
```

**La primera vez tarda unos minutos**: se descargan y compilan las dependencias del juego (hace falta internet). Después es rápido.

Se abre una ventana con un cuadrado amarillo que mueves con las flechas o WASD. El juego tiene dos partes:

- `actualizar(dt)` se llama unas 60 veces por segundo. Aquí lees el teclado y mueves cosas; `dt` son los segundos desde el cuadro anterior.
- `pintar()` dibuja el cuadro actual.

## 4. Cuando algo falla

- Los errores del compilador dicen el archivo, la línea y la columna, y a veces sugieren cómo corregirlo («¿quisiste decir…?»).
- Si tu programa falla mientras corre, verás el motivo en español, el archivo y la línea.
- Si ves `error interno del compilador`, es un fallo de Cometa, no tuyo: por favor repórtalo.
- `no se encontró Go`: instala Go y vuelve a abrir la terminal, o define `COMETA_GO` con la ruta de `go`.

## 5. Y ahora qué

Los [ejemplos](../examples/README.md) están ordenados de menor a mayor dificultad: primero programas de consola (`basico/`) y después juegos (`pincel/`). La [guía de juegos](juegos.md) explica la biblioteca Pincel, y [specs.md](../specs.md) describe el lenguaje completo.
