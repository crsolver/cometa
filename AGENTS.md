# Cometa project handoff

This repository implements **Cometa**, a statically typed, indentation-based programming language with Spanish syntax. The compiler and language server are written in Go. The same executable supports compilation and LSP mode.

## Current status

- The compiler MVP is implemented and tested.
- The LSP server is implemented with `github.com/owenrumney/go-lsp v0.2.5`.
- The VS Code extension contains both TextMate syntax support and a TypeScript language client.
- From `vscode-extension/`, `npm run build` compiles the TypeScript client to `out/extension.js` and the Go executable to `bin/cometa.exe` on Windows (`bin/cometa` elsewhere).
- VS Code F5 launch is configured as **Run Cometa Extension** and opens the `examples` directory in an Extension Development Host.
- This directory is a Git repository; preserve existing uncommitted changes.

## Commands

```powershell
# Install VS Code client dependencies
cd vscode-extension
npm install

# Build the TypeScript client and Go executable
npm run build

# Screenshot a game (hidden window) for visual review
.\bin\cometa.exe captura ..\examples\pincel\07_sprites.cometa -o ..\sprites.png --escala 4

# Compile Cometa to Go
.\bin\cometa.exe compilar ..\examples\basico\09_enums.cometa -o ..\enums.go

# Start the language server manually
.\bin\cometa.exe lsp

# Form used by vscode-languageclient; also intentionally supported
.\bin\cometa.exe lsp --stdio

# Return to the repository root for Go commands
cd ..

# Run tests
& 'C:\Program Files\Go\bin\go.exe' test ./...

# Static checks
& 'C:\Program Files\Go\bin\go.exe' vet ./...
```

The machine has two Go installations. `C:\msys64\mingw64\bin\go.exe` reports Go 1.25.5 but points at a Go 1.26.3 GOROOT and fails with version-mismatch errors. Prefer `C:\Program Files\Go\bin\go.exe`. The global Go build cache can be unwritable in sandboxed sessions; `vscode-extension/scripts/build-server.js` automatically uses `.cache/go-build` and `.cache/go-tmp`. For direct Go commands, set `GOCACHE` and `GOTMPDIR` to absolute workspace-local directories if access errors occur.

## Standard library and Pincel

- `std/mate/curvas` exposes 31 pure scalar easing functions under `curvas`: `lineal` plus ten families with `_entrada`, `_salida`, `_entrada_salida`. Inputs clamp to [0, 1] with exact endpoints; elastic/back output overshoot is preserved. No tween state or Ebitengine dependency. See `docs/curvas.md`, `examples/pincel/08_curvas.cometa`, and `internal/stdlib/curvas_runtime.txt`.

- Library APIs require explicit per-file imports: `std/mate`, `std/azar`, and `std/pincel` (core: `ejecutar`, `Juego`) plus individual `std/pincel/{graficos,color,entrada,audio,ventana,tiempo,recursos,rejilla,datos}` modules; `std/pincel` does not import submodules. Aliases use `como`; local `./std/` paths remain filesystem imports.
- Types are qualified: `mate.Vec2`, `mate.Rect`, `color.Color`, etc. Contextual literals/constants remain supported. Former library names are not globally reserved.
- Games start from `inicio()` through `pincel.ejecutar(instancia, ...configuracion) !`. The object implements `pincel.Juego` with `pub fn actualizar(dt decimal)` and `pub fn pintar()` methods. No automatic callbacks or `pincel.configuracion`.
- Drawing phase checks live in the library runtime. Resource embedding still requires direct global constructors with literal module-relative paths.
- `internal/stdlib` owns native declarations, linkage identities, and embedded implementations. Runtime dependency selection emits only the imported closure; math/random do not pull in Ebitengine.
- Standard declarations are navigable at `cometa-std:///` URIs. The extension provider reads them using `cometa biblioteca std/...`.
- `std/pincel/retro` bundles CC0 DUNGEON.mode atlases for `texto`, `icono`, and indexed `glifo`, with integer positions/scales and tintable transparent glyphs. Dungeon-437 supplies Unicode text; Dungeon-mode supplies named icons. Assets and implementations live in `internal/stdlib/assets` and `retro_runtime.txt`; only retro imports emit atlas data. `pincel.ejecutar(..., pixelado = verdadero)` disables screen filtering; integer window/camera scales preserve crisp pixels. See `examples/pincel/06_retro.cometa`; `internal/compiler/testdata/programas/{retro,dungeon2,escape_retro}.cometa` are compile fixtures.

- `std/pincel/lienzo` builds editable `graficos.Imagen` values in code: `desde_texto`/`hoja_desde_texto` text-grid sprites (single-character `[cadena: Color]` keys; `.`/space transparent), `nuevo`, integer non-antialiased `pixel`/`rect`/`linea`/`circulo`/`rellenar`, alpha-blended mirrored `pegar`, `leer_pixel` (`Color?`) and `guardar(...) !` PNG export. `_hgImagen` keeps straight-alpha CPU pixels (`image.NRGBA`) and uploads lazily through `gpu()`, re-uploading when `dirty`; loaded images are CPU-backed too. Runtime lives in `internal/stdlib/lienzo_runtime.txt`. `cometa captura <juego> [-o png] [--cuadros N,M,...] [--escala N] [--entrada guion.txt]` (`cmd/cometa/captura.go`) appends an `init` setting `_hgcapturaRutas`/`_hgcapturaCuadros` (one PNG per tick, `<salida>_<tick>.png` when several) to generated Go, runs with a hidden window, and saves the logical screen from `_hgGame.Draw`; `Update` stalls while a shot is pending so each PNG shows exactly its tick. `--entrada` scripts (`60 +D`, `90 -D`, `100 Enter`, `120 raton x y`, `RatonIzquierdo`) become `_hgguion` events (codes from `stdlib.InputCode`: `int(ebiten.Key)`, mouse `-1-int(button)`) applied by `_hgavanzarGuion` at the start of each tick; with `_hgguionActivo` the entrada helpers in `runtime.txt` (and the UI adapter, which goes through them) read only the simulated state. Text typing and gamepads are not scriptable. See `docs/lienzo.md` and `examples/pincel/07_sprites.cometa`. The fixture `internal/compiler/testdata/programas/huerto.cometa` is a top-down farming game with all art generated by lienzo. It bakes the ground into one canvas and repaints the cells queued in `Granja.pendientes`. Its Ebitengine-free simulation `huerto/granja.cometa` is covered by `internal/compiler/huerto_example_test.go`, including a hidden-window capture run.

## Language decisions

- Hashmaps use `[K: V]`, literals `[clave: valor]`, and contextual empty `[:]`. Keys are `cadena`, `entero`, or `bool`; generic value types are supported, generic keys are rejected. Reads return `V?`, writes accept `V`. Maps share entries across aliases/parameters, have fresh writable defaults, and `copiar()` is shallow and independent. `repetir (mapa) |valor, clave|` is unordered. Methods: `longitud`, `esta_vacia`, `contiene`, `obtener`, `eliminar`, `claves`, `valores`, `copiar`, `vaciar`. Native Go maps use flow lowering for ordered evaluation and comma-ok lookups; map writes must never take an entry address. Built-in map operations do not satisfy interface method requirements. See `examples/basico/07_mapas.cometa`, `internal/sema/maps.go`, and `internal/codegen/maps.go`.

- Source extension and VS Code language ID: `.cometa` / `cometa`.
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
- Declarations and struct members are private to their file by default. `pub` exposes top-level functions/globals/types and individual fields, methods, and embeddings. Enum variants and interface requirements inherit type visibility; `pub` is invalid on them, locals, imports, parameters, or `inicio`. Public APIs may expose inferred private types. Cross-module promotion requires an accessible complete path; only public concrete methods through public embedding paths satisfy interfaces. External literals cannot initialize private fields, even positionally.
- Generated types, fields, methods, and ordinary functions retain exported Go names as an implementation detail. Internal visibility marker methods preserve Cometa interface satisfaction during dynamic assertions and type switches.
- A top-level `fn inicio()` becomes Go `func main()` and cannot accept parameters or return a value.
- A function with a declared result implicitly returns its final expression. A final multiline conditional must produce the declared type on every path.
- `si` works as a statement and as a value. Value conditionals require `sino`, and every branch must have the same type.
- `enum` variants have zero or one explicitly typed payload. Constructors use `Evento.Cargar` or `Evento.Texto("hola")`, or `.Cargar` / `.Texto("hola")` with an expected enum type. Case labels require `Evento.Variante`, `.Variante`, or `_`; bare labels are invalid. Enum and struct payloads preserve reference semantics.
- `casos` is exhaustive as both a statement and an expression. Arm labels are indented; `=>` introduces one inline statement/expression or an indented block. Optional `|e|` binds only explicit payload arms, never unit variants or `_`. Value arms produce a common type. Loop control cannot escape a value match to an outer loop.
- Receiver fields and methods require `@`; bare identifiers do not fall back to receiver members.
- Structs embed declared structs with a bare type line (`Persona`, `Caja<entero>`). The implicit field/literal key is the base type name. Fields and methods promote at the shallowest depth in one namespace; direct members shadow promotions and equal-depth paths are ambiguous. Promoted methods satisfy interfaces. Literals accept direct keys only; embeddings retain reference, fresh-default and required-cycle rules. `internal/sema/embedding.go` shares resolution with LSP; promoted calls retain the receiver path for default helpers. See `internal/compiler/testdata/programas/embebidos.cometa`.
- `imprimir(valor)` maps to `fmt.Println(valor)`.
- Optionals use `T?`; results use `T!` (string error) or `T!E`; `!`/`!E` mean success without a payload. Constructors are `.Alguno`/`.Ninguno` and `.Ok`/`.Error`, with one implicit wrapping step under an expected type. Extraction requires `casos`, optional `si` binding, `o`, `atrapar`, or `intentar`. `o` supplies a default for optionals and valued results (discarding the error); `atrapar` accepts only results and requires `|error|` when the result has a payload. `retornar` exits explicitly; propagation exits the enclosing Cometa function. No user-defined generics are required.
- Omitted scalar/list/optional fields have valid defaults. Struct fields default to fresh recursive instances, but result/enum fields require explicit initialization through every required nested field. Required struct cycles are rejected; optionals and lists allow recursion. Ordinary declared references cannot be nil in Cometa.
- Discarded wrapper expressions and unread local result (`T!E`) variables are errors. Optional (`T?`) locals may be declared, copied, and left unused; payload access still requires extraction. Parameter defaults cannot contain `retornar` or `intentar`.
- `usar ruta [como alias]` imports a file module relative to the importing file, with `/` separators and implicit `.cometa`. Imports precede declarations, expose public declarations through a namespace, and do not re-export imports. Only the root may declare `inicio`. Duplicate physical imports and cycles are rejected. Local values can shadow aliases. See `examples/basico/12_modulos.cometa`.
- List iteration and infinite loops use `repetir`; `continuar` and `romper` control the nearest loop. Standalone null values, explicit references, generic operator constraints, remote package resolution, and direct executable generation remain outside the MVP.

## Architecture

- `cmd/cometa/nuevo.go`: `cometa nuevo` also writes `AGENTS.md` (embedded from `cmd/cometa/plantillas/agentes_base.md`, plus `agentes_pincel.md` for `--juego`) and a `CLAUDE.md` importing it; `--sin-agentes` skips both. The guide sets the teacher-not-coder persona and summarises the language; keep it in sync with language changes (`TestAgentGuideExamplesCompile` compiles its ```cometa blocks).
- `cmd/cometa/main.go`: CLI dispatch. Supports `compilar`, `lsp`, and `lsp --stdio`.
- `internal/lexer`: line-aware lexer with `INDENT`/`DEDENT` tokens and positioned errors.
- `internal/parser`: handwritten parser producing the AST.
- `internal/ast`: declarations, statements, expressions, types, and source positions.
- `internal/sema`: symbol collection, type checking, receiver checks, call validation, and implicit-return validation. `generics.go`, `interfaces.go` and `recursion.go` handle substitution, structural assignability, constraints, safe inspection and expanding-instantiation rejection.
- `internal/codegen`: exported Go name mapping, conditional lowering, `go/format`, and generated-Go type validation.
- `internal/codegen/generics.go`: native Go interfaces/generics, exact method bridges for defaults, and interface type switches.
- `internal/codegen/flow.go`: statement-level lowering for wrappers and explicit exits; preserves evaluation order and laziness without returning from generated expression helper functions.
- `internal/compiler`: standalone `Analyze`/`Compile` retain single-source behavior. `AnalyzeProject`/`CompileProject` accept a canonical-path source loader, load the import graph, bind unique declaration identities on a separate AST and check/generate one program. Original module ASTs and declaration links support tooling.
- `pkg/cometa`: the only stable Go API for other modules (`Check`, `Compile`, `Options{Files, Loader, LineDirectives}`, `*Error{Diagnostics}`). It wraps `AnalyzeProject`/`CompileProject` and returns plain values only; never expose `internal` types through it. `Files` maps in-memory projects onto a fixed virtual directory (`<tmp>/cometa-memoria/proyecto`, never read) and rewrites diagnostics and //line paths back to the caller's keys. Asset embedding reads through the project's `SourceLoader`, so in-memory projects can supply images, fonts and sounds.
- `internal/lspserver/modules.go`: filesystem URI conversion, unsaved source overlays, dependency diagnostics, watched-file changes, qualified completion/hover and definition navigation.
- `internal/lspserver`: go-lsp handler, incremental document store, live diagnostics, UTF-16 position conversion, and hierarchical document symbols.
- `vscode-extension/src/extension.ts`: VS Code language client. It launches the configured/default executable with `lsp` and explicit stdio transport.
- `vscode-extension/scripts/build-server.js`: cross-platform Go build helper and local-cache setup.

## Testing support (std/pruebas, cometa probar)

- `std/pruebas` (`afirmar`, `igual`, `casi_igual`, `fallar`; runtime in `internal/stdlib/pruebas_runtime.txt`) has no Ebitengine dependency. Failures are panics of `_hgpruebaFallo`, which exposes `PruebaFallo()`. `igual` picks `Entero/Decimal/Cadena/Bool` runtime variants by argument type in `sema/game.go` (like `mate.absoluto`).
- `cometa probar <pruebas.cometa> [--json] [--tiempo S] [--filtro x]` (`cmd/cometa/probar.go`) analyzes the project, finds top-level `fn prueba_*`, compiles the generated Go plus a `harness.go` (written through `goBuild` in `build.go`), runs it with a timeout and reads its JSONL event log (`COMETA_PROBAR_SALIDA`). Exit codes: 0 pass, 1 test failure, 2 compile error, 3 timeout, 4 internal. See `docs/pruebas.md`. Visual (Pincel) assertions are not implemented yet; see `MEJORAS.md`.

## Errors, run diagnostics and releases

- `ejecutar`/`construir`/`captura` compile with `compiler.CompileProjectForRun`, which emits `//line file.cometa:N` before each statement (`codegen.Options.LineDirectives`; `compilar` output has none, so goldens are unaffected). Go panics and `go build` errors then carry Cometa lines. `cmd/cometa/errors.go` filters the game's stderr (`crashFilter`), translates the panic into Spanish with the source line, and classifies `go build` failures (no network, old Go, missing native libs, internal codegen bugs). Missing Go is detected in `goExecutable`; a slow first build prints a notice after 4 s.
- "¿Quisiste decir…?" comes from `diagnostic.Suggest/Hint` (bind.go names, sema members/methods/variants); `diagnostic.Foreign` maps keywords from other languages (`while`, `if`, `print`…). Parser errors that start with "se esperaba" get ", pero se encontró …" plus `+=`/`++`/reserved-word hints (`parser.foundSuffix`).
- `imprimir` spells enum variants and results (`Evento.Texto("hola")`, `Ok(1)`, `Error("x")`) using `_hsenumNombres`, registered per enum in generated `init` functions only when `imprimir` is used.
- `cometa --version` reports `main.version`, injected by GoReleaser (`.goreleaser.yaml`). `.github/workflows/ci.yml` runs vet/tests on three OSes and the extension build; `release.yml` publishes archives and per-platform `.vsix` on `v*` tags. Neither workflow has been run yet.

## Complete Windows bundle

- `cmd/cometa/bundle.go`: if `go/bin/go(.exe)` sits next to `cometa` (or under `COMETA_HOME`), builds use that toolchain offline (`GOMODCACHE=modcache/`, `GOPROXY=off`, `-trimpath`, `CGO_ENABLED=0`, build cache seeded from `gocache/` into the user cache dir on first use). Without it, the system Go is used as before. `COMETA_SEMBRAR=1` writes caches straight into the bundle.
- `scripts/bundle-windows.ps1` builds `dist/cometa_<v>_windows_amd64_completo.zip` (~200 MB: Go, Ebitengine modules, prewarmed cache, `THIRD_PARTY_NOTICES.txt`); run it manually (needs internet once) to make the offline install package; releases publish only the normal binary. Seeding builds real examples through `cometa construir` so cache keys match.

## Language ergonomics added in the beta push

- `x op= v` (`+= -= *= /= %=`) is desugared in the parser to `x = x op v` with a second, independently parsed copy of the target; targets containing calls are rejected so nothing runs twice, and `AssignStmt.Compound` lets sema handle map entries: `checker.defaultCompoundMapRead` wraps the read copy as `m[k] o cero` (0, 0.0 or ""), so a missing key counts as the zero value; other value types are rejected.
- `mientras cond` is desugared in the parser to an infinite `repetir` whose first statement is `si !cond romper`, so sema/codegen have no new loop kind (`continuar` re-tests the condition).
- `casos` over `entero`/`cadena`/`bool` uses `MatchArm.Literals` (parser: `enum.go`), `internal/sema/scalar_match.go` and `emitScalarMatch`/the scalar branch of `flowMatch` (Go `switch`). Arms may also name top-level `const`s with a plain literal value (a lone identifier parses as a type pattern; `checker.constantLabel` converts it, and lists go through `parseLabel`; duplicates are detected by value). A final `_` is required except for a complete `bool` match; decimals and `|x|` bindings are rejected.
- `cadena(x)` is parsed as the interpolation `"${x}"`. `a_entero()`/`a_decimal()` are string methods (`StringCalls`); `formato(n)` lives on numbers (`internal/sema/numbers.go`, `Model.NumberCalls`, `flowNumberCall`).
- Every user `tipo` has a built-in shallow `copiar()` (`Model.StructCopies`); a user-declared `copiar` wins. `enum == .Variante` (unit variants only) is recorded in `Model.EnumCompares` and lowered to a tag comparison.
- `num` is no longer a keyword; using it as a type still reports "el tipo num fue eliminado".

## Pincel additions (beta push)

- New stdlib code lives in `internal/stdlib/extra_runtime.txt` (embedded and tree-shaken like the other runtimes): `azar.semilla` (a private PCG generator), `datos` (JSON key-value store of strings under `os.UserConfigDir()/cometa/<dir>-<file>/`; codegen defines `_hgdatosCarpeta` when `std/pincel/datos` is imported, see `dataFolder`; `datos.juego(nombre)` overrides it at run time and drops the cache; `guardar/leer_entero|decimal` store numbers as text), standard-layout gamepads (`BotonMando`/`EjeMando`), `graficos.Hoja` (`graficos.hoja(imagen, w, h)` wraps any `Imagen`, so no asset-plumbing changes) and `rejilla.Rejilla` (integer grid, methods on a pointer type, `choca` for solid-cell collision, `dibujar` culls off-screen cells). Vec2/Rect helpers, `mate` extras, `graficos.medir_texto`, `retro.ancho_texto`, `pincel.salir` and extra `Tecla` constants live in `runtime.txt`/`catalog.go`.
- `graficos.fuente_predeterminada()` returns a lazily parsed Go Regular font (`golang.org/x/image/font/gofont/goregular`, already an Ebitengine dependency). Because runtime selection is per module, every `std/pincel/graficos` import emits the `goregular` import; tests must not add it again.
- `Camara2D` has value methods returning copies (`a_mundo`, `a_pantalla`, `siguiendo`, `limitada`, `sacudida`, `visible`); `tiempo.Temporizador` is a pointer type like `Rejilla`; `ventana.cursor/icono/es_pantalla_completa`. All live in `extra_runtime.txt`; `TestCamaraYTemporizador` covers the window-free parts. Catalog parameter names cannot be keywords (`repetir`).
- Every catalog entry needs `FunctionDocs`/`MethodDocs`/`TypeDocs` (`docs_test.go` enforces it) and new type names must be added to both `typeName` regexes (`modules.go`, `ui.go`), `TypeModules`, `Fields` and `IsValue` when they are value types.
- Real Go type-checking of game runtime code only happens in `cometa construir/ejecutar`; `compilar` validates syntax only. After touching runtime `.txt` files, build a scratch game with the new API. Console-only pieces (rejilla, azar, datos, mate) are covered by `internal/compiler/pincel_extras_test.go` without Ebitengine.

## LSP and VS Code details

- `vscode-languageclient v10.1.1` appends `--stdio` when `TransportKind.stdio` is configured. Do not remove support for `cometa lsp --stdio`; doing so causes immediate server exit, `EPIPE`, and a VS Code restart loop.
- LSP diagnostics run on open, every change, and save, and are cleared on close or after a successful reanalysis.
- The frontend collects recoverable lexer, parser, binding and semantic diagnostics, ordered by file and position and capped at 100 errors plus a truncation notice. `internal/diagnostic` preserves typed errors through `Unwrap() []error`; consumers must flatten aggregates before converting positions. Partial ASTs/models are for tooling only; compilation never generates Go after frontend errors.
- Recovery skips damaged statements/members at indentation boundaries. Invalid declarations and local bindings suppress dependent errors; independent statements, declarations and modules are still analyzed. Some whole-program checks and unread-result checks are deferred when their prerequisites are incomplete.
- The server advertises incremental text synchronization and hierarchical document symbols.
- Document synchronization and edits use UTF-16 positions through go-lsp's document store. Compiler rune columns are converted to UTF-16 for diagnostics.
- VS Code setting `cometa.server.path` overrides the bundled/default server path. Relative paths resolve from the first workspace folder; `${workspaceFolder}` is supported.
- Check startup failures under **Output -> Cometa Language Server**.
- The extension requires VS Code 1.106 or newer.

## Testing expectations

- Run `npm run build` from `vscode-extension/` after modifying the VS Code client, manifest, build helper, CLI entrypoint, or LSP dependencies.
- Run `go test ./...` after modifying Go code. Existing tests cover lexer indentation, parser shape, semantic failures, golden Go output, CLI behavior, LSP capability negotiation, diagnostic publication/clearing, UTF-16 columns, and document symbols.
- Run `go vet ./...` before handing off substantial Go changes.
- A binary-level LSP smoke test should initialize `vscode-extension/bin/cometa.exe lsp --stdio`, await the initialize response, send shutdown, await its response, then send exit. Sending shutdown and exit without awaiting the response can create a false cancellation failure.
- `internal/compiler/testdata/programas/interfaces_genericos.cometa` demonstrates interfaces, generic structs/enums, constraints and safe inspection.
- `internal/compiler/testdata/programas/usuario.cometa` demonstrates enums, matching and payload references. The original struct regression fixture is `internal/compiler/testdata/usuario.cometa`, paired with `usuario.go.golden` in that directory. Enum runtime tests also execute generated Go.

## Website (separate repo)

- The site lives in its own repository, `crsolver/cometadev` (https://github.com/crsolver/cometadev), published at https://crsolver.github.io/cometadev/. Locally it is the sibling folder `../webpage` (not inside this repo). It is an Astro 7 site: landing page plus `/tour/`, `/biblioteca/<modulo>` (generated), `/guias/`, `/referencia/`, 404 and sitemap. See its README for details; all internal links go through `url()` because of the `/cometadev` base path.
- The site holds a **snapshot** of this repo's content: `docs/*.md`, `specs.md`, `examples/`, `vscode-extension/syntaxes/cometa.tmLanguage.json` and the generated library reference. After changing any of those (or `internal/stdlib`), run `npm run sync` in the website repo (defaults to `../cometa`; pass another path or set `COMETA_REPO`), then commit and push it there; pushing to `main` redeploys.
- `cmd/gendocs` writes `webpage/src/data/biblioteca.json` (relative to this repo's root; the folder no longer exists here and is recreated on demand) from `internal/stdlib` (Functions, Methods, Fields, Constants, `*Docs` maps). The website's sync script copies that file. New stdlib symbols appear automatically after a sync; add `FunctionDocs`/`MethodDocs`/`TypeDocs` so they have prose. `cmd/gendocs/main_test.go` checks the catalog is fully covered.
- The docs/specs links `x.md`, `../specs.md`, `../examples/...` are rewritten by the website (`src/lib/enlaces.mjs`), and repo links use `https://github.com/crsolver/cometa`.
- The Pages workflow lives in the website repo (`.github/workflows/pages.yml` there); this repo has none.

## Repository hygiene

- Use `apply_patch` for source edits.
- Preserve user-created or generated files unless their ownership is certain. A root-level `usuario.go` may exist from manual testing.
- `experiments/`, `bin/`, `out/`, `node_modules/`, `.cache/`, `.gocache/`, `.gomodcache/`, and `.gotmp/` are intentionally ignored.
- `README.md` is a short Spanish landing page (install, first program, links); language reference belongs in `specs.md` and `docs/`. Keep `specs.md`, `README.md`, VS Code metadata, examples, and tests synchronized with language changes.
- `examples/` is for learners: short, commented, runnable programs in `basico/` (console) and `pincel/` (games), indexed by `examples/README.md`. `TestNumericExamples` compiles every `examples/*/*.cometa`; deeper files are imported modules. Keep large showcase games out of it.
- `MEJORAS.md` is the backlog of language/stdlib rough edges found while writing Cometa code. Add an entry (observed, problem, possible fix) whenever you hit one; remove it when fixed.
- `experiments/` is ignored by Git and holds local experiments and showcase games (huerto, plataformas, escape_retro, etc.). Never make tests or docs depend on it; copy programs that tests need into `internal/compiler/testdata/programas/`.
