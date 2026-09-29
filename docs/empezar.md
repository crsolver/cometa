# Empezar con Cometa

Cometa es un lenguaje con sintaxis en español, pensado para aprender y para hacer juegos 2D por diversión. Esta guía te lleva desde la instalación hasta tu primer juego.

## 1. Qué necesitas

- El programa `cometa` (descárgalo de la página de versiones y ponlo en tu `PATH`).
- **Go 1.25 o superior** ([go.dev/dl](https://go.dev/dl/)). Cometa se traduce a Go, y `cometa ejecutar` usa Go para compilar tu programa. Comprueba que funciona con `go version`.
- Un editor. Para VS Code hay una extensión (`.vsix`) con colores, errores mientras escribes y autocompletado.
- En Linux, los juegos necesitan además las bibliotecas de desarrollo de Ebitengine (consulta [ebitengine.org](https://ebitengine.org/en/documents/install.html)).

Comprueba la instalación:

```console
cometa --version
```

## 2. Tu primer programa

```console
cometa nuevo hola
cd hola
cometa ejecutar principal.cometa
```

`cometa nuevo` crea la carpeta `hola` con un archivo `principal.cometa`:

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
