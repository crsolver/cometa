# Hacha

Hacha is a statically typed, indentation-based programming language with Spanish syntax. Its compiler is written in Go and emits formatted, type-checked Go source.

```hacha
fn sumar(a num, b num) num a + b

fn inicio()
	imprimir("hola desde Hacha")
```

## Try it

```console
go run ./cmd/hacha compilar examples/usuario.hacha -o usuario.go
go run usuario.go
```

## Language server

The same executable provides a Language Server Protocol server over standard input/output:

```console
hacha lsp
```

Configure an LSP-capable editor to launch that command for `.hacha` files using the `hacha` language ID. The server synchronizes incremental document changes, publishes compiler diagnostics while you type, provides a hierarchical document outline including enums and variants, completes fields and methods after `.` on typed variables or `@` inside methods, and shows inferred variable types and function signatures on hover. It also completes enum variants after `Evento.` and payload members inside `casos` arms. Consecutive `//` lines immediately above a function are included as its hover documentation.

## Test in VS Code

```console
npm install
npm run build
```

Open this repository in VS Code, press `F5`, and choose **Run Hacha Extension**. The build task compiles the TypeScript client and places the Go server in `bin/` before opening an Extension Development Host on the `examples` directory.

Open `usuario.hacha`, replace one leading tab with spaces, and check the **Problems** panel for a Hacha diagnostic. Restore the tab and the diagnostic should disappear immediately. The **Outline** view should show `Direccion`, `Evento`, their variants, `Buton_Presionado`, and the functions.

If the server executable lives elsewhere, set `hacha.server.path` in VS Code settings to its absolute path. Open **Output → Hacha Language Server** to inspect client or server startup failures.

The source language and current MVP boundaries are documented in [specs.md](specs.md).

Hacha also supports list loops with `repetir (lista) |elemento, indice|`, loop control through `continuar` and `romper`, and infinite inline loops such as `repetir imprimir("hola")`.

## Enums and exhaustive matches

Variants optionally declare one explicit payload type. Struct payloads retain their existing references; primitives are stored by value.

```hacha
enum Evento
	Cargar
	Texto cadena

fn describir(evento Evento) cadena
	casos evento |e|
		.Cargar => "cargando pagina"
		.Texto => e

fn inicio()
	var evento = Evento.Texto("hola")
	var texto = casos evento |e|
		.Texto => e
		_ => "otro evento"
	imprimir(texto)
```

Arm labels are indented beneath `casos` and use `Evento.Cargar`, `.Cargar`, or `_`; bare variant labels are invalid. After `=>`, write one statement/expression on the same line or an indented block on following lines. Every match must cover all variants, optionally using a final `_`. In value contexts, every arm must produce the same type. The optional binding is available only in explicitly named payload arms.

Enum constructors also support contextual shorthand: `var evento Evento = .Texto("hola")`, or `mostrar(.Cargar)` when the parameter expects `Evento`. Expected enum types propagate through assignments, returns, fields, lists, and branch results using the existing contextual inference rules. Without an expected enum, `var evento = .Cargar` and `imprimir(.Cargar)` are invalid. Typing `.` offers variants of the expected enum, including in case labels; hover shows the qualified variant and payload type.
