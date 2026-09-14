# Hacha

Hacha is a statically typed, indentation-based programming language with Spanish syntax. Its compiler is written in Go and emits formatted, type-checked Go source.

```hacha
fn sumar(a num, b num) num a + b

fn inicio()
	imprimir("hola desde Hacha")
```

## Variadic parameters and named arguments

```hacha
fn sumar(base num, valores ...num) num
	var total = base
	repetir (valores) |valor|
		total = total + valor
	total

fn inicio()
	imprimir(sumar(10, 1, 2))
	var lista = [1, 2]
	imprimir(sumar(10, lista...))
	imprimir(sumar(valores = lista, base = 10))
	imprimir(sumar(base = 10))
```

The final parameter may use `...T`; its body sees a `[T]` list. Calls accept zero or more elements, or one final `lista...` sharing the list's storage, as in Go. Individual variadic elements cannot be mixed with expansion.

Named arguments use `name = value` in any order. Positional arguments must come first; fixed parameters without defaults are required exactly once. A named variadic argument supplies the entire list (`valores = lista`, also `valores = lista...` when last). Expressions evaluate once in source order, with the receiver evaluated first. Functions and methods support both features; `imprimir` accepts `valor = ...`, while enum payload constructors remain positional. See `examples/argumentos.hacha` for a runnable example.

## Default parameter values

```hacha
fn saludar(nombre cadena = "mundo", saludo cadena = "hola " + nombre) cadena
	saludo

fn inicio()
	imprimir(saludar())                         // hola mundo
	imprimir(saludar("Ana"))                    // hola Ana
	imprimir(saludar(saludo = "buenos días"))    // buenos días
```

Functions and methods accept `name Type = expression`. Required parameters must precede defaulted parameters; a final variadic parameter is allowed but cannot declare a default. Positional arguments fill parameters from left to right; named arguments can skip defaults. Explicit zero, false, and empty values override defaults.

Omitted defaults evaluate once per call, in declaration order, after the receiver and all explicit arguments. Defaults are checked against the parameter type, even for unused functions, and can use earlier parameters, ordinary function calls, and `@` members in methods. They cannot reference themselves, later parameters, body locals, or caller locals. Lists and structs created by defaults are fresh on each evaluation; references to earlier parameters retain their usual sharing. See `examples/defaults.hacha`.

## Optionals and errors

Use `T?` for a value that may be absent, `T!` for a value or string error, and `T!E` for a typed error. `!` and `!E` represent operations with no success payload. A plain value implicitly becomes present/successful when the expected type allows one wrapping step. Absence and failure use `.Ninguno` and `.Error(error)` explicitly.

```hacha
fn buscar(existe bool) num?
	si existe 42
	sino .Ninguno

fn cargar() num! .Error("no disponible")

fn siguiente() num!
	var valor = intentar cargar()
	valor + 1

fn inicio()
	imprimir(buscar(falso) o 0)
	imprimir(siguiente() capturar 0)
```

Use exhaustive `casos`, optional `si valor |payload|` bindings, lazy `o` fallbacks, or `capturar |error|` recovery. `intentar` propagates one layer of absence or a compatible error; `retornar` exits explicitly. Access to the contained value always requires extraction. Discarded wrappers and unread local wrapper variables are errors.

Scalars, lists and optionals retain valid defaults. Omitted struct fields allocate fresh, recursively defaulted objects; result and enum fields require explicit initialization, including when nested. Required struct cycles are rejected: migrate a recursive `referido Usuario` to `referido Usuario?`. See [the executable example](examples/errores.hacha) and [the complete rules](specs.md).

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

Numeric ranges are available only in loops: `repetir (0..5) |i| imprimir(i)` prints `0` through `4`, and `repetir (5..0) |i| imprimir(i)` prints `5` through `1`. Bounds are `num` expressions evaluated once, with an exclusive end and an automatic step of `1` or `-1`. Equal bounds produce no iterations. An optional second binding receives the index starting at zero.

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
