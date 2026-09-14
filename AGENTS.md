# Hacha project handoff

This repository implements **Hacha**, a statically typed, indentation-based programming language with Spanish syntax. The compiler and language server are written in Go. The same executable supports compilation and LSP mode.

## Current status

- The compiler MVP is implemented and tested.
- The LSP server is implemented with `github.com/owenrumney/go-lsp v0.2.5`.
- The VS Code extension contains both TextMate syntax support and a TypeScript language client.
- `npm run build` compiles the TypeScript client to `out/extension.js` and the Go executable to `bin/hacha.exe` on Windows (`bin/hacha` elsewhere).
- VS Code F5 launch is configured as **Run Hacha Extension** and opens the `examples` directory in an Extension Development Host.
- There is no Git repository initialized in this directory as of the last check.

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

## Language decisions

- Source extension and VS Code language ID: `.hacha` / `hacha`.
- Indentation is significant and must use tabs. Blank and comment-only lines do not affect indentation.
- Authoritative syntax is the indentation-based form in `specs.md`; legacy `fin`, declaration braces and declaration colons are not supported. `=>` is reserved for `casos` arms.
- Built-in types map as follows:
  - `num` -> Go `float64`
  - `cadena` -> Go `string`
  - `bool` -> Go `bool`
- Declared Hacha types always have reference semantics: `Usuario` -> `*Usuario`; `[Usuario]` -> `[]*Usuario`.
- Generated types, fields, methods, and ordinary functions are exported in Go.
- A top-level `fn inicio()` becomes Go `func main()` and cannot accept parameters or return a value.
- A function with a declared result implicitly returns its final expression. A final multiline conditional must produce the declared type on every path.
- `si` works as a statement and as a value. Value conditionals require `sino`, and every branch must have the same type.
- `enum` variants have zero or one explicitly typed payload. Constructors use `Evento.Cargar` or `Evento.Texto("hola")`, or `.Cargar` / `.Texto("hola")` with an expected enum type. Case labels require `Evento.Variante`, `.Variante`, or `_`; bare labels are invalid. Enum and struct payloads preserve reference semantics.
- `casos` is exhaustive as both a statement and an expression. Arm labels are indented; `=>` introduces one inline statement/expression or an indented block. Optional `|e|` binds only explicit payload arms, never unit variants or `_`. Value arms produce a common type. Loop control cannot escape a value match to an outer loop.
- Receiver fields and methods require `@`; bare identifiers do not fall back to receiver members.
- `imprimir(valor)` maps to `fmt.Println(valor)`.
- Optionals use `T?`; results use `T!` (string error) or `T!E`; `!`/`!E` mean success without a payload. Constructors are `.Alguno`/`.Ninguno` and `.Ok`/`.Error`, with one implicit wrapping step under an expected type. Extraction requires `casos`, optional `si` binding, `o`, `capturar`, or `intentar`. `retornar` exits explicitly; propagation exits the enclosing Hacha function. No user-defined generics are required.
- Omitted scalar/list/optional fields have valid defaults. Struct fields default to fresh recursive instances, but result/enum fields require explicit initialization through every required nested field. Required struct cycles are rejected; optionals and lists allow recursion. Ordinary declared references cannot be nil in Hacha.
- Discarded wrapper expressions and unread local wrapper variables are errors. Parameter defaults cannot contain `retornar` or `intentar`.
- List iteration and infinite loops use `repetir`; `continuar` and `romper` control the nearest loop. Imports, standalone null values, explicit references, user-defined generics, multi-file modules, and direct executable generation remain outside the MVP.

## Architecture

- `cmd/hacha/main.go`: CLI dispatch. Supports `compilar`, `lsp`, and `lsp --stdio`.
- `internal/lexer`: line-aware lexer with `INDENT`/`DEDENT` tokens and positioned errors.
- `internal/parser`: handwritten parser producing the AST.
- `internal/ast`: declarations, statements, expressions, types, and source positions.
- `internal/sema`: symbol collection, type checking, receiver checks, call validation, and implicit-return validation.
- `internal/codegen`: exported Go name mapping, conditional lowering, `go/format`, and generated-Go type validation.
- `internal/codegen/flow.go`: statement-level lowering for wrappers and explicit exits; preserves evaluation order and laziness without returning from generated expression helper functions.
- `internal/compiler`: `Analyze` runs the frontend for in-memory tooling; `Compile` additionally generates Go.
- `internal/lspserver`: go-lsp handler, incremental document store, live diagnostics, UTF-16 position conversion, and hierarchical document symbols.
- `src/extension.ts`: VS Code language client. It launches the configured/default executable with `lsp` and explicit stdio transport.
- `scripts/build-server.js`: cross-platform Go build helper and local-cache setup.

## LSP and VS Code details

- `vscode-languageclient v10.1.1` appends `--stdio` when `TransportKind.stdio` is configured. Do not remove support for `hacha lsp --stdio`; doing so causes immediate server exit, `EPIPE`, and a VS Code restart loop.
- LSP diagnostics run on open, every change, and save, and are cleared on close or after a successful reanalysis.
- The current compiler reports the first frontend error, so the LSP publishes at most one compiler diagnostic per analysis.
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
- `examples/usuario.hacha` demonstrates enums, matching and payload references. The original struct regression fixture is `internal/compiler/testdata/usuario.hacha`, paired with `usuario.go.golden` in that directory. Enum runtime tests also execute generated Go.

## Repository hygiene

- Use `apply_patch` for source edits.
- Preserve user-created or generated files unless their ownership is certain. A root-level `usuario.go` may exist from manual testing.
- `bin/`, `out/`, `node_modules/`, `.cache/`, `.gocache/`, `.gomodcache/`, and `.gotmp/` are intentionally ignored.
- Keep `specs.md`, `README.md`, VS Code metadata, examples, and tests synchronized with language changes.
