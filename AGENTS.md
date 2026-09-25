# Hacha project handoff

This repository implements **Hacha**, a statically typed, indentation-based programming language with Spanish syntax. The compiler and language server are written in Go. The same executable supports compilation and LSP mode.

## Current status

- The compiler MVP is implemented and tested.
- The LSP server is implemented with `github.com/owenrumney/go-lsp v0.2.5`.
- The VS Code extension contains both TextMate syntax support and a TypeScript language client.
- `npm run build` compiles the TypeScript client to `out/extension.js` and the Go executable to `bin/hacha.exe` on Windows (`bin/hacha` elsewhere).
- VS Code F5 launch is configured as **Run Hacha Extension** and opens the `examples` directory in an Extension Development Host.
- This directory is a Git repository; preserve existing uncommitted changes.

## Commands

```powershell
# Install VS Code client dependencies
npm install

# Build the TypeScript client and Go executable
npm run build

# Compile Hacha to Go
.\bin\hacha.exe compilar .\examples\usuario.hacha -o .\usuario.go

# Start the language server manually
.\bin\hacha.exe lsp

# Form used by vscode-languageclient; also intentionally supported
.\bin\hacha.exe lsp --stdio

# Run tests
& 'C:\Program Files\Go\bin\go.exe' test ./...

# Static checks
& 'C:\Program Files\Go\bin\go.exe' vet ./...
```

The machine has two Go installations. `C:\msys64\mingw64\bin\go.exe` reports Go 1.25.5 but points at a Go 1.26.3 GOROOT and fails with version-mismatch errors. Prefer `C:\Program Files\Go\bin\go.exe`. The global Go build cache can be unwritable in sandboxed sessions; `scripts/build-server.js` automatically uses `.cache/go-build` and `.cache/go-tmp`. For direct Go commands, set `GOCACHE` and `GOTMPDIR` to absolute workspace-local directories if access errors occur.

## Standard library and Pincel

- `std/mate/curvas` exposes 31 pure scalar easing functions under `curvas`: `lineal` plus ten families with `_entrada`, `_salida`, `_entrada_salida`. Inputs clamp to [0, 1] with exact endpoints; elastic/back output overshoot is preserved. No tween state or Ebitengine dependency. See `docs/curvas.md`, `examples/curvas.hacha`, and `internal/stdlib/curvas_runtime.txt`.

- Library APIs require explicit per-file imports: `std/mate`, `std/azar`, and individual `std/pincel/{juego,graficos,color,entrada,audio,ventana,tiempo,recursos}` modules. Aliases use `como`; local `./std/` paths remain filesystem imports.
- Types are qualified: `mate.Vec2`, `mate.Rect`, `color.Color`, etc. Contextual literals/constants remain supported. Former library names are not globally reserved.
- Games start from `inicio()` through `juego.ejecutar(instancia, ...configuracion) !`. The object implements `juego.Juego` with `actualizar(dt decimal)` and `pintar()` methods. No automatic callbacks or `juego.configuracion`.
- Drawing phase checks live in the library runtime. Resource embedding still requires direct global constructors with literal module-relative paths.
- `internal/stdlib` owns native declarations, linkage identities, and embedded implementations. Runtime dependency selection emits only the imported closure; math/random do not pull in Ebitengine.
- Standard declarations are navigable at `hacha-std:///` URIs. The extension provider reads them using `hacha biblioteca std/...`.
- `std/pincel/retro` bundles CC0 DUNGEON.mode atlases for `texto`, `icono`, and indexed `glifo`, with integer positions/scales and tintable transparent glyphs. Dungeon-437 supplies Unicode text; Dungeon-mode supplies named icons. Assets and implementations live in `internal/stdlib/assets` and `retro_runtime.txt`; only retro imports emit atlas data. `juego.ejecutar(..., pixelado = verdadero)` disables screen filtering; integer window/camera scales preserve crisp pixels. See `examples/retro.hacha` and `examples/dungeon2.hacha`.

## Language decisions

- Source extension and VS Code language ID: `.hacha` / `hacha`.
- Indentation is significant and must use tabs. Blank and comment-only lines do not affect indentation.
- Authoritative syntax is the indentation-based form in `specs.md`; legacy `fin`, declaration braces and declaration colons are not supported. `=>` is reserved for `casos` arms.
- Built-in types map as follows:
  - `entero` -> Go `int64`
  - `decimal` -> Go `float64`
  - `cadena` -> Go `string`
  - `bool` -> Go `bool`
- `num` is removed. Integer-to-decimal widening is implicit and recorded in `Model.NumericCoercions`; existing containers remain invariant. Integer division truncates toward zero; `%` is integer-only. Indices, lengths, loop bounds, RGBA channels and game dimensions/TPS are integers. Game callbacks use `actualizar(dt decimal)`. `entero(decimal)` truncates with finite/range checks; rounding helpers return integers.
- Declared structures and enums have reference semantics: `Usuario` -> `*Usuario`; `[Usuario]` -> `[]*Usuario`. Interfaces and type parameters do not add pointers; `Caja<Usuario>` -> `*Caja[*Usuario]`.
- `interfaz` declares implicit structural interfaces, with method signatures and embedded interfaces; a bodyless declaration is an empty interface. Interface fields require initialization. Interface signatures have no defaults; named calls use the static interface parameter names. Concrete default methods expose exact public signatures and internal default helpers.
- Functions, structs, enums and interfaces accept `<T>` parameters and `<T Interfaz>` constraints. Methods inherit struct parameters. Bodies are checked against constraints even when unused. Calls infer from explicit argument types only; unresolved inference requires all type arguments. Fields declared directly as `T` require explicit initialization in every instantiation. Generic arithmetic, equality, unions and independent method type parameters are deferred.
- `valor como Tipo` safely inspects an interface and returns `Tipo?`. Interface `casos` uses type patterns, first-match ordering and a required final `_`. Lists, generic structs/enums and existing wrappers remain invariant.
- Generated types, fields, methods, and ordinary functions are exported in Go.
- A top-level `fn inicio()` becomes Go `func main()` and cannot accept parameters or return a value.
- A function with a declared result implicitly returns its final expression. A final multiline conditional must produce the declared type on every path.
- `si` works as a statement and as a value. Value conditionals require `sino`, and every branch must have the same type.
- `enum` variants have zero or one explicitly typed payload. Constructors use `Evento.Cargar` or `Evento.Texto("hola")`, or `.Cargar` / `.Texto("hola")` with an expected enum type. Case labels require `Evento.Variante`, `.Variante`, or `_`; bare labels are invalid. Enum and struct payloads preserve reference semantics.
- `casos` is exhaustive as both a statement and an expression. Arm labels are indented; `=>` introduces one inline statement/expression or an indented block. Optional `|e|` binds only explicit payload arms, never unit variants or `_`. Value arms produce a common type. Loop control cannot escape a value match to an outer loop.
- Receiver fields and methods require `@`; bare identifiers do not fall back to receiver members.
- Structs embed declared structs with a bare type line (`Persona`, `Caja<entero>`). The implicit field/literal key is the base type name. Fields and methods promote at the shallowest depth in one namespace; direct members shadow promotions and equal-depth paths are ambiguous. Promoted methods satisfy interfaces. Literals accept direct keys only; embeddings retain reference, fresh-default and required-cycle rules. `internal/sema/embedding.go` shares resolution with LSP; promoted calls retain the receiver path for default helpers. See `examples/embebidos.hacha`.
- `imprimir(valor)` maps to `fmt.Println(valor)`.
- Optionals use `T?`; results use `T!` (string error) or `T!E`; `!`/`!E` mean success without a payload. Constructors are `.Alguno`/`.Ninguno` and `.Ok`/`.Error`, with one implicit wrapping step under an expected type. Extraction requires `casos`, optional `si` binding, `o`, `capturar`, or `intentar`. `retornar` exits explicitly; propagation exits the enclosing Hacha function. No user-defined generics are required.
- Omitted scalar/list/optional fields have valid defaults. Struct fields default to fresh recursive instances, but result/enum fields require explicit initialization through every required nested field. Required struct cycles are rejected; optionals and lists allow recursion. Ordinary declared references cannot be nil in Hacha.
- Discarded wrapper expressions and unread local result (`T!E`) variables are errors. Optional (`T?`) locals may be declared, copied, and left unused; payload access still requires extraction. Parameter defaults cannot contain `retornar` or `intentar`.
- `usar ruta [como alias]` imports a file module relative to the importing file, with `/` separators and implicit `.hacha`. Imports precede declarations, expose public declarations through a namespace, and do not re-export imports. Only the root may declare `inicio`. Duplicate physical imports and cycles are rejected. Local values can shadow aliases. See `examples/modulos/inicio.hacha`.
- List iteration and infinite loops use `repetir`; `continuar` and `romper` control the nearest loop. Standalone null values, explicit references, generic operator constraints, remote package resolution, and direct executable generation remain outside the MVP.

## Architecture

- `cmd/hacha/main.go`: CLI dispatch. Supports `compilar`, `lsp`, and `lsp --stdio`.
- `internal/lexer`: line-aware lexer with `INDENT`/`DEDENT` tokens and positioned errors.
- `internal/parser`: handwritten parser producing the AST.
- `internal/ast`: declarations, statements, expressions, types, and source positions.
- `internal/sema`: symbol collection, type checking, receiver checks, call validation, and implicit-return validation. `generics.go`, `interfaces.go` and `recursion.go` handle substitution, structural assignability, constraints, safe inspection and expanding-instantiation rejection.
- `internal/codegen`: exported Go name mapping, conditional lowering, `go/format`, and generated-Go type validation.
- `internal/codegen/generics.go`: native Go interfaces/generics, exact method bridges for defaults, and interface type switches.
- `internal/codegen/flow.go`: statement-level lowering for wrappers and explicit exits; preserves evaluation order and laziness without returning from generated expression helper functions.
- `internal/compiler`: standalone `Analyze`/`Compile` retain single-source behavior. `AnalyzeProject`/`CompileProject` accept a canonical-path source loader, load the import graph, bind unique declaration identities on a separate AST and check/generate one program. Original module ASTs and declaration links support tooling.
- `internal/lspserver/modules.go`: filesystem URI conversion, unsaved source overlays, dependency diagnostics, watched-file changes, qualified completion/hover and definition navigation.
- `internal/lspserver`: go-lsp handler, incremental document store, live diagnostics, UTF-16 position conversion, and hierarchical document symbols.
- `src/extension.ts`: VS Code language client. It launches the configured/default executable with `lsp` and explicit stdio transport.
- `scripts/build-server.js`: cross-platform Go build helper and local-cache setup.

## LSP and VS Code details

- `vscode-languageclient v10.1.1` appends `--stdio` when `TransportKind.stdio` is configured. Do not remove support for `hacha lsp --stdio`; doing so causes immediate server exit, `EPIPE`, and a VS Code restart loop.
- LSP diagnostics run on open, every change, and save, and are cleared on close or after a successful reanalysis.
- The frontend collects recoverable lexer, parser, binding and semantic diagnostics, ordered by file and position and capped at 100 errors plus a truncation notice. `internal/diagnostic` preserves typed errors through `Unwrap() []error`; consumers must flatten aggregates before converting positions. Partial ASTs/models are for tooling only; compilation never generates Go after frontend errors.
- Recovery skips damaged statements/members at indentation boundaries. Invalid declarations and local bindings suppress dependent errors; independent statements, declarations and modules are still analyzed. Some whole-program checks and unread-result checks are deferred when their prerequisites are incomplete.
- The server advertises incremental text synchronization and hierarchical document symbols.
- Document synchronization and edits use UTF-16 positions through go-lsp's document store. Compiler rune columns are converted to UTF-16 for diagnostics.
- VS Code setting `hacha.server.path` overrides the bundled/default server path. Relative paths resolve from the first workspace folder; `${workspaceFolder}` is supported.
- Check startup failures under **Output -> Hacha Language Server**.
- The extension requires VS Code 1.106 or newer.

## Testing expectations

- Run `npm run build` after modifying the VS Code client, manifest, build helper, CLI entrypoint, or LSP dependencies.
- Run `go test ./...` after modifying Go code. Existing tests cover lexer indentation, parser shape, semantic failures, golden Go output, CLI behavior, LSP capability negotiation, diagnostic publication/clearing, UTF-16 columns, and document symbols.
- Run `go vet ./...` before handing off substantial Go changes.
- A binary-level LSP smoke test should initialize `bin/hacha.exe lsp --stdio`, await the initialize response, send shutdown, await its response, then send exit. Sending shutdown and exit without awaiting the response can create a false cancellation failure.
- `examples/interfaces_genericos.hacha` demonstrates interfaces, generic structs/enums, constraints and safe inspection.
- `examples/usuario.hacha` demonstrates enums, matching and payload references. The original struct regression fixture is `internal/compiler/testdata/usuario.hacha`, paired with `usuario.go.golden` in that directory. Enum runtime tests also execute generated Go.

## Repository hygiene

- Use `apply_patch` for source edits.
- Preserve user-created or generated files unless their ownership is certain. A root-level `usuario.go` may exist from manual testing.
- `bin/`, `out/`, `node_modules/`, `.cache/`, `.gocache/`, `.gomodcache/`, and `.gotmp/` are intentionally ignored.
- Keep `specs.md`, `README.md`, VS Code metadata, examples, and tests synchronized with language changes.
