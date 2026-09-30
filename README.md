<p align="center">
  <img src="docs/logo.svg" alt="Logo de Cometa" width="120">
</p>

<h1 align="center">Cometa</h1>

Cometa es un lenguaje de programación con sintaxis en español, pensado para aprender y para hacer juegos 2D por diversión. Los bloques se marcan con tabuladores, los tipos son estáticos y el programa se traduce a Go.

```cometa
usar std/mate
usar std/pincel
usar std/pincel/graficos

tipo Partida
	pos mate.Vec2
	pub fn actualizar(dt decimal)
		@pos.x += 40 * dt
	pub fn pintar()
		graficos.rectangulo_v(@pos, {10, 10}, .Rojo)

fn inicio()
	pincel.ejecutar(Partida {}, titulo = "Mi juego") atrapar |error|
		imprimir(error)
```

## Instalar

1. Descarga `cometa` de la página de versiones y ponlo en tu `PATH`.
2. Instala [Go 1.25 o superior](https://go.dev/dl/): Cometa usa Go para compilar tus programas. Puedes indicar otro ejecutable con la variable `COMETA_GO`.
3. Opcional: instala la extensión de VS Code (`.vsix` de la misma página) para tener colores, errores mientras escribes y autocompletado.
4. En Linux y macOS, los juegos necesitan las [dependencias de Ebitengine](https://ebitengine.org/en/documents/install.html).

```console
cometa --version
```

## Tu primer programa

```console
cometa nuevo hola
cd hola
cometa ejecutar principal.cometa
```

Para un juego, usa `cometa nuevo mi_juego --juego`. La primera vez se descargan las dependencias del juego (hace falta internet); después es rápido. La guía completa está en [docs/empezar.md](docs/empezar.md).

## Aprender más

| Quiero… | Lee |
| --- | --- |
| Ver programas cortos y comentados, de menor a mayor dificultad | [examples/](examples/README.md) |
| Hacer un juego | [docs/juegos.md](docs/juegos.md) |
| Dibujar mis propios sprites con código | [docs/lienzo.md](docs/lienzo.md) |
| Animar con curvas de suavizado | [docs/curvas.md](docs/curvas.md) |
| Generar terrenos con ruido | [docs/ruido.md](docs/ruido.md) |
| Crear menús y paneles | [docs/ui.md](docs/ui.md) |
| Consultar todas las reglas del lenguaje | [specs.md](specs.md) |

## Comandos

| Comando | Qué hace |
| --- | --- |
| `cometa nuevo <nombre> [--juego] [--sin-agentes]` | Crea un proyecto listo para ejecutar, con `AGENTS.md` para asistentes de IA (que actúan como profesor). |
| `cometa ejecutar <archivo>` | Compila y ejecuta. |
| `cometa construir <archivo> -o <salida>` | Genera un ejecutable con todos los recursos incluidos. |
| `cometa captura <juego> -o <png>` | Ejecuta un juego con la ventana oculta y guarda una captura. |
| `cometa compilar <archivo> -o <salida.go>` | Genera el código Go. |
| `cometa lsp` | Inicia el servidor de lenguaje para el editor. |

## Un vistazo al lenguaje

```cometa
enum Estado
	Menu
	Jugando
	Fin

fn describir(estado Estado) cadena
	casos estado
		.Menu => "menú"
		.Jugando => "jugando"
		.Fin => "fin"

fn inicio()
	var puntos = 0
	mientras puntos < 3
		puntos += 1
	var texto = "42".a_entero() o 0
	imprimir("${describir(.Jugando)}: ${puntos + texto}")
```

Además tiene listas, mapas `[cadena: entero]`, opcionales (`T?`), resultados (`T!`), interfaces, genéricos, módulos con `usar` y `casos` sobre enums, números y textos. Todo está descrito en [specs.md](specs.md).

## Desarrollo del compilador

El repositorio contiene el compilador y el servidor de lenguaje (Go) y la extensión de VS Code (TypeScript). Consulta [AGENTS.md](AGENTS.md) para las convenciones del proyecto.

```console
go test ./...
go vet ./...
cd vscode-extension
npm install
npm run build
```

`npm run build` compila el cliente y deja el ejecutable en `vscode-extension/bin/`. Para probar la extensión, abre el repositorio en VS Code, pulsa `F5` y elige **Run Cometa Extension**. Si el servidor está en otra ruta, configura `cometa.server.path`; los fallos de arranque aparecen en **Salida → Cometa Language Server**.

Licencia: [MIT](LICENSE).

## Pendiente

Las asperezas conocidas del lenguaje y la biblioteca están en [MEJORAS.md](MEJORAS.md).
