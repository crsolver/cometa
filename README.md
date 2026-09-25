# Cometa

Cometa is a statically typed, general-purpose language with Spanish syntax, compiled to Go. Tabs define blocks. Its bundled standard library provides math and random helpers; Pincel provides optional 2D games, graphics, input, audio, and resources through explicit imports.

```cometa
usar std/mate
usar std/pincel/juego
usar std/pincel/graficos

tipo Partida
	pos mate.Vec2
	pub fn actualizar(dt decimal)
		@pos.x = @pos.x + 40 * dt
	pub fn pintar()
		graficos.rectangulo_v(@pos, {10, 10}, .Rojo)

fn inicio()
	juego.ejecutar(Partida {}, titulo = "Mi juego") capturar |error|
		imprimir(error)
```

```console
cd vscode-extension
npm install
npm run build
cd ..
vscode-extension/bin/cometa ejecutar examples/juego.cometa
vscode-extension/bin/cometa construir examples/juego.cometa -o juego.exe
```

On Windows use `vscode-extension\bin\cometa.exe`. Go 1.25+ is required; `COMETA_GO` can select the executable. The first Pincel build downloads pinned Ebitengine v2.10.1 dependencies; math-only programs do not depend on Ebitengine. Linux/macOS require Ebitengine's native development dependencies. Builds use an isolated temporary module and embed all assets. They do not modify your project's Go module. `ejecutar` builds and runs; `construir` builds without opening a window; `captura` runs a game hidden and saves its screen as PNG.

The [playable example](examples/juego.cometa) includes keyboard movement, a sprite, collision, text and sound. Use WASD or arrow keys to collect the yellow target. See [the game API guide](docs/juegos.md) for signatures and lifecycle rules.

For easing, import `std/mate/curvas` and use `mate.interpolar(a, b, curvas.cubica_entrada_salida(elapsed / duration))` with a positive duration. The module provides 31 pure scalar curves, clamps progress to `[0, 1]`, and preserves elastic/back overshoot. Import `std/mate` separately for interpolation. Neither module requires Ebitengine. See [the curve API](docs/curvas.md) and [runnable example](examples/curvas.cometa).

For procedural levels, `usar std/mate/ruido` provides seeded 2D smooth and fractal noise in `[0, 1]`, without graphics dependencies or shared random state. See [the noise API](docs/ruido.md) and [console terrain/cave example](examples/ruido.cometa).

Import individual Pincel modules with `usar std/pincel/juego`, `std/pincel/graficos`, `std/pincel/color`, `std/pincel/entrada`, `std/pincel/audio`, `std/pincel/ventana`, `std/pincel/tiempo`, `std/pincel/recursos`, `std/pincel/retro`, and `std/pincel/lienzo`. Math and random remain independent: `usar std/mate` and `usar std/azar`. Aliases use `como`; each source file imports its own dependencies.

Prototype without asset files using `retro.texto("¡Hola!", 8, 8)` and `retro.icono(.Corazon, 8, 24, color = .Rojo)`. The bundled CC0 DUNGEON.mode atlases provide 8×8 bitmap text, Spanish characters, and fantasy icons. Use `pixelado = verdadero` in `juego.ejecutar` with an integer window scale for crisp presentation. For a subtle CRT look, use `retro = verdadero`: it enables nearest-neighbor scaling plus soft scanlines, a faint RGB mask, and a gentle vignette across the whole image. This flag works with any Pincel game and needs no extra import. See [the retro guide](docs/juegos.md#juegos-retro-sin-recursos), [playable dungeon](examples/dungeon2.cometa), and [atlas gallery](examples/retro.cometa).

Draw your own pixel art in code with `usar std/pincel/lienzo`: `lienzo.desde_texto(["..oo..", ".oxxo."], ["o": .Negro, "x": .Rojo])` turns text grids into sprites, and canvases support integer `pixel`, `rect`, `linea`, `circulo`, `rellenar`, mirrored `pegar`, and `guardar` to PNG. `cometa captura juego.cometa -o juego.png --escala 4` runs a game with a hidden window and saves a screenshot for review. See [the canvas guide](docs/lienzo.md) and [code-drawn scene](examples/pixelart.cometa).

[La última luz](examples/escape_retro.cometa) is a three-floor, turn-based retro adventure with Spanish story dialogues and a fixed 24-color palette. Find each floor's key, fight skeletons, and restore the tower's light. Arrow keys move or attack, Space waits, Enter advances dialogue, and R restarts. Run it with `cometa ejecutar examples/escape_retro.cometa`.

[Bajo la tierra](examples/plataformas.cometa) is a 320×180 procedural platformer sandbox with 8×8 DUNGEON.mode blocks, surface terrain and connected caves across 6×3 fixed camera sections. It includes variable-height jumps, coyote time, jump buffering, mining, mouse-aimed shooting, slimes and cave bats. A/D or arrows move, Space jumps, left mouse shoots, and right mouse mines within reach. R returns to the refuge, N generates a new world, and H shows help. Run `vscode-extension/bin/cometa ejecutar examples/plataformas.cometa`; see [controls and world details](docs/juegos.md#bajo-la-tierra). Its [simulation module](examples/plataformas/mundo.cometa) runs without graphics and has automated gameplay checks.

[Huerto](examples/huerto.cometa) is a small top-down farming game in the style of Stardew Valley, with every sprite drawn in code through `lienzo`. Till the fenced field, plant turnips, carrots and pumpkins, water them daily (or let the rain do it), harvest and sell at the bin, then buy more seeds at the stand. The day/night clock ends at the house door. WASD or arrows walk, Space or left click use the selected tool, Q/E or the wheel switch tools, and H shows help. The ground is baked once into a canvas and repainted cell by cell as you work. Its [farm simulation](examples/huerto/granja.cometa) has automated gameplay checks. See [details](docs/juegos.md#huerto).

Games start from `inicio()` by calling `juego.ejecutar(instance, ...)`. The instance implements `juego.Juego` through `pub fn actualizar(dt decimal)` and `pub fn pintar()` methods. Configuration is passed to that call, which returns `!` and blocks until the window closes. Defaults remain 320×180, scale 1, title "Cometa", and 60 TPS. Initialize state before calling it. Top-level `actualizar`, `pintar`, and `iniciar` have no special meaning; `juego.configuracion` has been removed.

Library types require qualification: `mate.Vec2`, `mate.Rect`, and `color.Color`, for example. Contextual literals such as `{x: 10}` and constants such as `.Rojo` still work when their type is known. Library value semantics and vector operators are preserved; user structures retain reference semantics. Library names are no longer globally reserved. Drawing outside the active `pintar` phase fails at runtime.

`cometa compilar archivo.cometa -o salida.go` still emits formatted Go. Game output receives syntax validation; `ejecutar` and `construir` perform full Go compilation. General programs retain generated-Go type checking and their traditional entry point:

```cometa
fn sumar(a entero, b entero) entero a + b

fn inicio()
	imprimir("hola desde Cometa")
```

Generated Go omits unused local bindings and compiler temporaries. Initializers still execute when they can have effects or fail, preserving evaluation order without dummy `_ = variable` statements.

Strings support strict concatenation, `${expression}` interpolation, and Unicode-aware methods:

```cometa
var nombre = "Ana"
imprimir("Hola " + nombre)
imprimir("${nombre}, tienes ${20 + 1} años")
imprimir(nombre.mayusculas())
imprimir(nombre.subcadena(0, 2) o "")
```

Interpolation accepts strings, numbers, and booleans. Methods cover length, search, prefixes and suffixes, casing, trimming, replacement, splitting, safe character access, and safe substring access. Positions count Unicode code points. See [the string example](examples/cadenas.cometa) and the method table in [specs.md](specs.md).

## Hashmaps

Maps use `[K: V]` with `cadena`, `entero`, or `bool` keys. Index reads return `V?`; writes insert or replace entries:

```cometa
var puntos [cadena: entero] = [:]
puntos["Ana"] = 10
imprimir(puntos["Ana"] o 0)
repetir (puntos) |valor, clave|
	imprimir("${clave}: ${valor}")
```

Nonempty literals infer their types: `["Ana": 10, "Luis": 20]`. Maps share storage through assignments and parameters; `copiar()` makes an independent shallow copy. Iteration is unordered, with value then key. Methods include `longitud`, `esta_vacia`, `contiene`, `obtener`, `eliminar`, `claves`, `valores`, `copiar`, and `vaciar`. See [the map example](examples/mapas.cometa) and [the specification](specs.md#mapas).

## Modules and imports

Each `.cometa` file is a module. Place `usar` declarations before other declarations:

```cometa
usar herramientas
usar modelos como m
usar interno/base_de_datos como bd
usar ../compartido/fechas

fn inicio()
	var usuario = m.Usuario {nombre: "Ana"}
	imprimir(herramientas.describir(usuario))
```

Paths are unquoted, use `/`, omit `.cometa`, and resolve relative to the importing file. The default namespace is the final path segment; `como` changes it. Namespace aliases must be unique and cannot conflict with top-level declarations. Local variables and parameters may shadow aliases in expressions.

Declarations are private to their file by default. Use `pub fn`, `pub tipo`, `pub enum`, `pub interfaz`, `pub var`, or `pub const` to expose a module API. Struct fields, methods, and embedded fields independently require `pub` for external access. Enum variants and interface requirements inherit their type's visibility. Imports do not re-export other imports. Imports of the same physical file share type identity; types with the same name from different files remain distinct.

```cometa
pub tipo Usuario
	pub nombre cadena
	visitas entero
	pub fn describir() cadena @nombre

pub fn crear() Usuario Usuario {nombre: "Ana"}
```

All code in the declaring file can access private members. A public function may return a private type: consumers can use the inferred value and its public members, but cannot name that type. Promoted access requires every embedded field on the path to be accessible. Only public methods reached through public embedding paths satisfy interfaces, even inside the declaring file. Struct literals cannot supply private fields from other files, including by position; omitted fields still follow the usual default rules. Required private fields therefore need a factory in their defining module. `inicio` must remain private. `pub` is not valid on imports, locals, parameters, enum variants, or interface requirements.

The compiler loads all reachable modules and emits one formatted, checked Go file. Only the entry file may declare `fn inicio()`; importing a file that declares it is an error. Missing files, duplicate imports, alias conflicts and import cycles are errors. Paths may use `../` to leave the entry directory. There is no manifest or remote package resolution.

Top-level `var` and `const` declarations define private module globals; `pub var` and `pub const` expose them. Mutable globals can be reassigned from functions; constants are limited to numeric, string, and boolean constant expressions. Imported public values use the normal namespace syntax, such as `config.limite`. Forward references are supported and initialization cycles are rejected.

```cometa
const limite entero = 10
var contador = 0

fn incrementar()
	contador = contador + 1
```

```console
cometa compilar examples/modulos/inicio.cometa -o modulos.go
go run modulos.go
```

The [multi-file example](examples/modulos/inicio.cometa) combines imports, interfaces, generics, default arguments and enums. The editor analyzes unsaved dependencies, refreshes dependent diagnostics, completes qualified names and imported members, shows hover documentation, and supports go-to-definition on imports and symbols.

`usar` is now reserved; rename any older function or member with that name.

## Interfaces and generics

```cometa
interfaz Describible
	fn describir() cadena

tipo Usuario
	nombre cadena
	pub fn describir() cadena @nombre

tipo Caja<T>
	valor T
	pub fn obtener() T @valor

fn identidad<T>(valor T) T valor
fn describir<T Describible>(valor T) cadena valor.describir()
```

Interfaces work structurally: any type with the required method signatures implements an interface automatically. Interface bodies may embed other interfaces; an empty `interfaz Cualquiera` accepts every value-bearing type. Interface calls use the signature's parameter names and require all non-variadic arguments. Concrete methods can still have their own defaults. Ordinary interfaces cannot be absent; use `I?` for absence.

Structs support Go-style embedding by placing a bare struct type inside a `tipo`:

```cometa
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

Literal keys must name direct fields: initialize `Persona`, not its promoted `nombre`. Omitted embeddings receive fresh recursive defaults; required nested fields and required cycles follow the ordinary struct rules. Explicitly supplied objects retain their references. `Caja<entero>` can be embedded with the implicit field name `Caja`; two instantiations of `Caja` cannot be embedded together. Only declared structs can be embedded, not interfaces, enums, primitives, lists, wrappers, or bare type parameters. See [the embedding example](examples/embebidos.cometa).

Functions, structs, enums and interfaces support `<T>` parameters with optional interface constraints (`<T Describible>`). Function calls infer types from explicit arguments or accept all type arguments explicitly, such as `identidad<entero>(1)`. Type uses require arguments, such as `Caja<Usuario> {valor: usuario}`. Methods inherit the struct's parameters. A field typed directly as `T` always requires initialization; `[T]` and `T?` have their usual defaults. Generic arithmetic, equality, type unions and independently generic methods are not supported.

`valor como Usuario` safely tests an interface value and returns `Usuario?`. Interface `casos` accepts type patterns, binds the narrowed value, chooses the first matching arm and requires a final `_`:

```cometa
fn mostrar(valor Describible)
	casos valor |dato|
		Usuario => imprimir(dato.nombre)
		_ => imprimir(valor.describir())
```

See [the runnable example](examples/interfaces_genericos.cometa) and [the language specification](specs.md). Completion and hover show interface methods, constraint methods and instantiated generic signatures.

## Variadic parameters and named arguments

```cometa
fn sumar(base entero, valores ...entero) entero
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

Named arguments use `name = value` in any order. Positional arguments must come first; fixed parameters without defaults are required exactly once. A named variadic argument supplies the entire list (`valores = lista`, also `valores = lista...` when last). Expressions evaluate once in source order, with the receiver evaluated first. Functions and methods support both features; `imprimir` accepts `valor = ...`, while enum payload constructors remain positional. See `examples/argumentos.cometa` for a runnable example.

## Default parameter values

```cometa
fn saludar(nombre cadena = "mundo", saludo cadena = "hola " + nombre) cadena
	saludo

fn inicio()
	imprimir(saludar())                         // hola mundo
	imprimir(saludar("Ana"))                    // hola Ana
	imprimir(saludar(saludo = "buenos días"))    // buenos días
```

Functions and methods accept `name Type = expression`. Required parameters must precede defaulted parameters; a final variadic parameter is allowed but cannot declare a default. Positional arguments fill parameters from left to right; named arguments can skip defaults. Explicit zero, false, and empty values override defaults.

Omitted defaults evaluate once per call, in declaration order, after the receiver and all explicit arguments. Defaults are checked against the parameter type, even for unused functions, and can use earlier parameters, ordinary function calls, and `@` members in methods. They cannot reference themselves, later parameters, body locals, or caller locals. Lists and structs created by defaults are fresh on each evaluation; references to earlier parameters retain their usual sharing. See `examples/defaults.cometa`.

## Optionals and errors

Use `T?` for a value that may be absent, `T!` for a value or string error, and `T!E` for a typed error. `!` and `!E` represent operations with no success payload. A plain value implicitly becomes present/successful when the expected type allows one wrapping step. Absence and failure use `.Ninguno` and `.Error(error)` explicitly.

```cometa
fn buscar(existe bool) entero?
	si existe 42
	sino .Ninguno

fn cargar() entero! .Error("no disponible")

fn siguiente() entero!
	var valor = intentar cargar()
	valor + 1

fn inicio()
	imprimir(buscar(falso) o 0)
	imprimir(siguiente() capturar 0)
```

Use exhaustive `casos`, optional `si valor |payload|` bindings, lazy `o` fallbacks, or `capturar |error|` recovery. `intentar` propagates one layer of absence or a compatible error; `retornar` exits explicitly. Access to the contained value always requires extraction. Discarded wrapper expressions and unread local result variables (`T!E`) are errors. Optional locals (`T?`) may be declared, copied, and left unused.

Scalars, lists and optionals retain valid defaults. Omitted struct fields allocate fresh, recursively defaulted objects; result and enum fields require explicit initialization, including when nested. Required struct cycles are rejected: migrate a recursive `referido Usuario` to `referido Usuario?`. See [the executable example](examples/errores.cometa) and [the complete rules](specs.md).

## Try it

```console
go run ./cmd/cometa compilar examples/usuario.cometa -o usuario.go
go run usuario.go
```

## Language server

The same executable provides a Language Server Protocol server over standard input/output:

```console
cometa lsp
```

Configure an LSP-capable editor to launch that command for `.cometa` files using the `cometa` language ID. The server synchronizes incremental document changes, publishes compiler diagnostics while you type, provides a hierarchical document outline including enums and variants, completes fields and methods after `.` on typed variables or `@` inside methods, and shows inferred variable types and function signatures on hover. It also completes enum variants after `Evento.` and payload members inside `casos` arms. Consecutive `//` lines immediately above a function are included as its hover documentation.

## Test in VS Code

```console
cd vscode-extension
npm install
npm run build
```

Open this repository in VS Code, press `F5`, and choose **Run Cometa Extension**. The build task compiles the TypeScript client and places the Go server in `vscode-extension/bin/` before opening an Extension Development Host on the `examples` directory.

The compiler and editor report multiple recoverable frontend errors together, including type errors in valid parts of a file with syntax errors. Diagnostics are ordered by file and position, deduplicated, and limited to 100 errors plus a truncation notice. Recovery suppresses errors that depend on damaged declarations or expressions; some checks require those errors to be corrected first. Compilation, building and execution stop before producing output whenever frontend errors remain.

Open `usuario.cometa`, introduce errors on several independent lines, and check the **Problems** panel. Correcting one error removes its diagnostic while leaving the others visible. Replacing a leading tab with spaces also produces an indentation diagnostic. The **Outline** view should show the declarations whose syntax can still be recovered.

If the server executable lives elsewhere, set `cometa.server.path` in VS Code settings to its absolute path. Open **Output → Cometa Language Server** to inspect client or server startup failures.

The source language and current MVP boundaries are documented in [specs.md](specs.md).

Cometa also supports list loops with `repetir (lista) |elemento, indice|`, loop control through `continuar` and `romper`, and infinite inline loops such as `repetir imprimir("hola")`.

List literals can span multiple lines, with tab-indented elements and the closing `]` aligned with the opening line. Elements are comma-separated; a trailing comma is optional in both multiline and single-line lists.

Lists have built-in methods such as `valores.longitud()`, `valores.buscar_indice(40)`, `valores.obtener(2)`, and mutating calls such as `valores.agregar(40)`, `valores.insertar(0, 10)`, and `valores.invertir()`. Search and safe access return optionals; insertion and deletion return `bool` for invalid-index handling. See `examples/listas.cometa` and the full method table in [specs.md](specs.md).

Numeric ranges are available only in loops: `repetir (0..5) |i| imprimir(i)` prints `0` through `4`, and `repetir (5..0) |i| imprimir(i)` prints `5` through `1`. Bounds are `entero` expressions evaluated once, with an exclusive end and an automatic step of `1` or `-1`. Equal bounds produce no iterations. An optional second binding receives the index starting at zero.

## Enums and exhaustive matches

Variants optionally declare one explicit payload type. Struct payloads retain their existing references; primitives are stored by value.

```cometa
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
