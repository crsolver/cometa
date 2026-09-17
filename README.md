# Hacha

Hacha is a statically typed language with Spanish syntax for small 2D games, powered by Go and Ebitengine. Tabs define blocks. Vectors, drawing, input, audio, math and collision helpers are available without imports; existing general-purpose features remain available.

```hacha
tipo Jugador
	pos Vec2
	vel Vec2

var jugador = Jugador {vel: {x: 40}}

fn actualizar(dt num)
	jugador.pos = jugador.pos + jugador.vel * dt

fn pintar()
	graficos.rectangulo_v(jugador.pos, Vec2 {x: 10, y: 10}, .Rojo)
```

```console
npm run build
bin/hacha ejecutar examples/juego.hacha
bin/hacha construir examples/juego.hacha -o juego.exe
```

On Windows use `bin\hacha.exe`. Go 1.25+ is required; `HACHA_GO` can select the executable. The first build downloads pinned Ebitengine v2.10.1 dependencies. Linux/macOS require Ebitengine's native development dependencies. Builds use an isolated temporary module and embed all assets. They do not modify your project's Go module. `ejecutar` builds and runs; `construir` builds without opening a window.

The [playable example](examples/juego.hacha) includes keyboard movement, a sprite, collision, text and sound. Use WASD or arrow keys to collect the yellow target. See [the game API guide](docs/juegos.md) for signatures and lifecycle rules.

Game entries require `actualizar(dt num)` and `pintar()`, with optional `iniciar()`. Call `juego.configuracion(...)` inside `iniciar` to override the defaults: 320×180, scale 1, title "Hacha", and 60 TPS. The compiler creates the loop and entry point. A game cannot also declare `inicio`. Built-in type/namespace names are reserved, so older interfaces named `Fuente` should be renamed (the examples use `Proveedor`). `Vec2` uses value semantics; user structures continue to use reference semantics. Vector math uses methods such as `pos.normalizado()`; rectangles expose `interseca` and `contiene`. Drawing uses scalar, `_v`, and `_rect` variants; rotations are optional radians. The math namespace is `mate`. Vectors and other structures accept named or positional literals, such as `Vec2 {x: 10, y: 10}` and `Vec2 {10, 10}`.

`hacha compilar archivo.hacha -o salida.go` still emits formatted Go. Game output receives syntax validation; `ejecutar` and `construir` perform full Go compilation. General programs retain generated-Go type checking and their traditional entry point:

```hacha
fn sumar(a num, b num) num a + b

fn inicio()
	imprimir("hola desde Hacha")
```

Generated Go omits unused local bindings and compiler temporaries. Initializers still execute when they can have effects or fail, preserving evaluation order without dummy `_ = variable` statements.

Strings support strict concatenation, `${expression}` interpolation, and Unicode-aware methods:

```hacha
var nombre = "Ana"
imprimir("Hola " + nombre)
imprimir("${nombre}, tienes ${20 + 1} años")
imprimir(nombre.mayusculas())
imprimir(nombre.subcadena(0, 2) o "")
```

Interpolation accepts strings, numbers, and booleans. Methods cover length, search, prefixes and suffixes, casing, trimming, replacement, splitting, safe character access, and safe substring access. Positions count Unicode code points. See [the string example](examples/cadenas.hacha) and the method table in [specs.md](specs.md).

## Modules and imports

Each `.hacha` file is a module. Place `usar` declarations before other declarations:

```hacha
usar herramientas
usar modelos como m
usar interno/base_de_datos como bd
usar ../compartido/fechas

fn inicio()
	var usuario = m.Usuario {nombre: "Ana"}
	imprimir(herramientas.describir(usuario))
```

Paths are unquoted, use `/`, omit `.hacha`, and resolve relative to the importing file. The default namespace is the final path segment; `como` changes it. Namespace aliases must be unique and cannot conflict with top-level declarations. Local variables and parameters may shadow aliases in expressions.

Functions, types, interfaces, enums, fields, methods and variants are public. Imports do not re-export other imports, execute initialization code, or require use. Access imported declarations with `m.crear()`, `m.Usuario`, `m.Caja<num>`, or `m.Evento.Texto("hola")`. Imports of the same physical file share type identity; types with the same name from different files remain distinct.

The compiler loads all reachable modules and emits one formatted, checked Go file. Only the entry file may declare `fn inicio()`; importing a file that declares it is an error. Missing files, duplicate imports, alias conflicts and import cycles are errors. Paths may use `../` to leave the entry directory. There is no manifest or remote package resolution.

Top-level `var` and `const` declarations define public module globals. Mutable globals can be reassigned from functions; constants are limited to numeric, string, and boolean constant expressions. Imported values use the normal namespace syntax, such as `config.limite`. Forward references are supported and initialization cycles are rejected.

```hacha
const limite num = 10
var contador = 0

fn incrementar()
	contador = contador + 1
```

```console
hacha compilar examples/modulos/inicio.hacha -o modulos.go
go run modulos.go
```

The [multi-file example](examples/modulos/inicio.hacha) combines imports, interfaces, generics, default arguments and enums. The editor analyzes unsaved dependencies, refreshes dependent diagnostics, completes qualified names and imported members, shows hover documentation, and supports go-to-definition on imports and symbols.

`usar` is now reserved; rename any older function or member with that name.

## Interfaces and generics

```hacha
interfaz Describible
	fn describir() cadena

tipo Usuario
	nombre cadena
	fn describir() cadena @nombre

tipo Caja<T>
	valor T
	fn obtener() T @valor

fn identidad<T>(valor T) T valor
fn describir<T Describible>(valor T) cadena valor.describir()
```

Interfaces work structurally: any type with the required method signatures implements an interface automatically. Interface bodies may embed other interfaces; an empty `interfaz Cualquiera` accepts every value-bearing type. Interface calls use the signature's parameter names and require all non-variadic arguments. Concrete methods can still have their own defaults. Ordinary interfaces cannot be absent; use `I?` for absence.

Structs support Go-style embedding by placing a bare struct type inside a `tipo`:

```hacha
tipo Persona
	nombre cadena
	fn saludar() imprimir(@nombre)

tipo Empleado
	Persona
	puesto cadena

fn inicio()
	var empleado = Empleado {Persona: {nombre: "Ana"}, puesto: "Ingeniera"}
	empleado.saludar()
	empleado.nombre = "Luis"
```

Embedded fields and methods are promoted recursively, including through `@` inside methods and for structural interface conformance. Direct members shadow promoted members; the unique member at the shallowest depth wins. Multiple matches at that depth make the selector ambiguous, even if they reach the same declaration. Use an explicit path such as `empleado.Persona.nombre` to select the embedded value.

Literal keys must name direct fields: initialize `Persona`, not its promoted `nombre`. Omitted embeddings receive fresh recursive defaults; required nested fields and required cycles follow the ordinary struct rules. Explicitly supplied objects retain their references. `Caja<num>` can be embedded with the implicit field name `Caja`; two instantiations of `Caja` cannot be embedded together. Only declared structs can be embedded, not interfaces, enums, primitives, lists, wrappers, or bare type parameters. See [the embedding example](examples/embebidos.hacha).

Functions, structs, enums and interfaces support `<T>` parameters with optional interface constraints (`<T Describible>`). Function calls infer types from explicit arguments or accept all type arguments explicitly, such as `identidad<num>(1)`. Type uses require arguments, such as `Caja<Usuario> {valor: usuario}`. Methods inherit the struct's parameters. A field typed directly as `T` always requires initialization; `[T]` and `T?` have their usual defaults. Generic arithmetic, equality, type unions and independently generic methods are not supported.

`valor como Usuario` safely tests an interface value and returns `Usuario?`. Interface `casos` accepts type patterns, binds the narrowed value, chooses the first matching arm and requires a final `_`:

```hacha
fn mostrar(valor Describible)
	casos valor |dato|
		Usuario => imprimir(dato.nombre)
		_ => imprimir(valor.describir())
```

See [the runnable example](examples/interfaces_genericos.hacha) and [the language specification](specs.md). Completion and hover show interface methods, constraint methods and instantiated generic signatures.

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

Use exhaustive `casos`, optional `si valor |payload|` bindings, lazy `o` fallbacks, or `capturar |error|` recovery. `intentar` propagates one layer of absence or a compatible error; `retornar` exits explicitly. Access to the contained value always requires extraction. Discarded wrapper expressions and unread local result variables (`T!E`) are errors. Optional locals (`T?`) may be declared, copied, and left unused.

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

Lists have built-in methods such as `valores.longitud()`, `valores.buscar_indice(40)`, `valores.obtener(2)`, and mutating calls such as `valores.agregar(40)`, `valores.insertar(0, 10)`, and `valores.invertir()`. Search and safe access return optionals; insertion and deletion return `bool` for invalid-index handling. See `examples/listas.hacha` and the full method table in [specs.md](specs.md).

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
