package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/compiler"
	"github.com/crsolver/cometa/internal/lspserver"
	"github.com/crsolver/cometa/internal/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var code exitCode
		if errors.As(err, &code) {
			os.Exit(int(code))
		}
		if err != errReported {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

// version is set at release time with -ldflags "-X main.version=v0.1.0".
var version = "dev"

// A command of the cometa executable. help is its line in the general usage
// text; run receives the arguments after the command name and parses its own
// flags.
type command struct {
	name string
	help string
	run  func(args []string) error
}

// commands is in the order the usage text lists them. It is filled in init
// because some commands print that text, which reads this list.
var commands []command

func init() {
	commands = []command{
		{"nuevo", "nuevo     <nombre> [--juego] [--sin-agentes]  crea un proyecto de ejemplo (consola o juego)", runNew},
		{"ejecutar", "ejecutar <archivo.cometa>              compila y ejecuta el programa", programCommand("ejecutar")},
		{"construir", "construir <archivo.cometa> [-o salida]  genera un ejecutable", programCommand("construir")},
		{"compilar", "compilar  <archivo.cometa> [-o salida.go]  genera el código Go", programCommand("compilar")},
		{"captura", "captura   <archivo.cometa> [-o captura.png] [--cuadros N,M,...] [--escala N] [--entrada guion.txt] [--datos carpeta]", programCommand("captura")},
		{"probar", "probar    <pruebas.cometa> [--json] [--tiempo S] [--filtro x] [--datos carpeta]  ejecuta las funciones prueba_*", runProbar},
		{"mapa", "mapa                                    abre el editor de mapas en el navegador", runMapa},
		{"biblioteca", "biblioteca <std/módulo>                 muestra las declaraciones de un módulo estándar", runBiblioteca},
		{"lsp", "lsp                                     servidor de lenguaje para editores", runLSP},
	}
}

func usage() error {
	var text strings.Builder
	text.WriteString("uso: cometa <comando> ...")
	for _, c := range commands {
		text.WriteString("\n  " + c.help)
	}
	text.WriteString("\n  --version                               muestra la versión")
	return errors.New(text.String())
}

func run(args []string) error {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version" || args[0] == "-v") {
		fmt.Println("cometa", version)
		return nil
	}
	if len(args) > 0 {
		for _, c := range commands {
			if c.name == args[0] {
				return c.run(args[1:])
			}
		}
	}
	return usage()
}

func runBiblioteca(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	source, ok := stdlib.Source(args[0])
	if !ok {
		return fmt.Errorf("módulo estándar desconocido %q", args[0])
	}
	fmt.Print(source)
	return nil
}

func runLSP(args []string) error {
	if !isLSPCommand(append([]string{"lsp"}, args...)) {
		return usage()
	}
	return lspserver.Run(context.Background())
}

// isLSPCommand accepts `lsp` and the `lsp --stdio` that vscode-languageclient
// sends; any other transport is not supported.
func isLSPCommand(args []string) bool {
	return len(args) == 1 && args[0] == "lsp" ||
		len(args) == 2 && args[0] == "lsp" && args[1] == "--stdio"
}

// programOptions are the arguments of the commands that compile one program:
// compilar, ejecutar, construir and captura.
type programOptions struct {
	input, output string
	// Only for captura.
	frames []int
	scale  int
	script string
	data   string
}

func parseProgramArgs(name string, args []string) (programOptions, error) {
	options := programOptions{scale: 1}
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; arg {
		case "-o":
			index++
			if index >= len(args) || options.output != "" {
				return options, fmt.Errorf("uso: cometa %s <archivo.cometa> [-o <salida>]", name)
			}
			options.output = args[index]
		case "--cuadros", "--escala", "--entrada", "--datos":
			if name != "captura" {
				return options, fmt.Errorf("argumento inesperado %q: solo cometa captura lo admite", arg)
			}
			index++
			if index >= len(args) {
				return options, errCaptureUsage
			}
			switch arg {
			case "--cuadros":
				frames, err := parseFrames(args[index])
				if err != nil {
					return options, err
				}
				options.frames = frames
			case "--escala":
				value, _ := strconv.Atoi(args[index])
				if value <= 0 || value > 64 {
					return options, errCaptureUsage
				}
				options.scale = value
			case "--datos":
				options.data = args[index]
			default:
				options.script = args[index]
			}
		default:
			if strings.HasPrefix(arg, "-") || options.input != "" {
				return options, fmt.Errorf("argumento inesperado %q", arg)
			}
			options.input = arg
		}
	}
	if options.input == "" {
		return options, fmt.Errorf("falta el archivo .cometa de entrada")
	}
	if filepath.Ext(options.input) != ".cometa" {
		return options, fmt.Errorf("el archivo de entrada debe tener extensión .cometa")
	}
	if name == "ejecutar" && options.output != "" {
		return options, fmt.Errorf("ejecutar no acepta -o")
	}
	if options.output == "" {
		stem := strings.TrimSuffix(options.input, filepath.Ext(options.input))
		switch name {
		case "captura":
			options.output = stem + ".png"
		case "construir":
			options.output = stem
			if runtime.GOOS == "windows" {
				options.output += ".exe"
			}
		default:
			options.output = stem + ".go"
		}
	}
	return options, nil
}

func programCommand(name string) func(args []string) error {
	return func(args []string) error {
		options, err := parseProgramArgs(name, args)
		if err != nil {
			return err
		}
		var events []scriptEvent
		if options.script != "" {
			if events, err = readScript(options.script); err != nil {
				return err
			}
		}
		// Programs that will run carry //line directives, so a crash or a build
		// error points at the Cometa source.
		compile := compiler.CompileProjectForRun
		if name == "compilar" {
			compile = compiler.CompileProject
		}
		generated, err := compile(options.input, nil)
		if err != nil {
			return err
		}
		switch name {
		case "captura":
			return captureProgram(generated, options.output, options.frames, options.scale, events, options.data)
		case "compilar":
			if err = os.WriteFile(options.output, generated, 0o644); err != nil {
				return fmt.Errorf("no se pudo escribir %s: %w", options.output, err)
			}
			return nil
		default:
			return buildProgram(generated, options.output, name == "ejecutar")
		}
	}
}
