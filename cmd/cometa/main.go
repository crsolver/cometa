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

func run(args []string) error {
	useBundledGo()
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version" || args[0] == "-v") {
		fmt.Println("cometa", version)
		return nil
	}
	if len(args) > 0 && args[0] == "probar" {
		return runProbar(args[1:])
	}
	if len(args) > 0 && args[0] == "nuevo" {
		return runNew(args[1:])
	}
	if len(args) == 2 && args[0] == "biblioteca" {
		source, ok := stdlib.Source(args[1])
		if !ok {
			return fmt.Errorf("módulo estándar desconocido %q", args[1])
		}
		fmt.Print(source)
		return nil
	}
	if isLSPCommand(args) {
		return lspserver.Run(context.Background())
	}
	if len(args) == 0 || (args[0] != "compilar" && args[0] != "ejecutar" && args[0] != "construir" && args[0] != "captura") {
		return fmt.Errorf("uso: cometa <comando> ...\n  nuevo     <nombre> [--juego] [--sin-agentes]  crea un proyecto de ejemplo (consola o juego)\n  ejecutar <archivo.cometa>              compila y ejecuta el programa\n  construir <archivo.cometa> [-o salida]  genera un ejecutable\n  compilar  <archivo.cometa> [-o salida.go]  genera el código Go\n  captura   <archivo.cometa> [-o captura.png] [--cuadros N,M,...] [--escala N] [--entrada guion.txt]\n  probar    <pruebas.cometa> [--json] [--tiempo S] [--filtro x]  ejecuta las funciones prueba_*\n  biblioteca <std/módulo>                 muestra las declaraciones de un módulo estándar\n  lsp                                     servidor de lenguaje para editores\n  --version                               muestra la versión")
	}
	var input, output, script string
	var frames []int
	scale := 1
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "-o":
			index++
			if index >= len(args) || output != "" {
				return fmt.Errorf("uso: cometa compilar <archivo.cometa> [-o <archivo.go>]")
			}
			output = args[index]
		case "--cuadros", "--escala", "--entrada":
			flag := args[index]
			index++
			if args[0] != "captura" || index >= len(args) {
				return errCaptureUsage
			}
			switch flag {
			case "--cuadros":
				var err error
				if frames, err = parseFrames(args[index]); err != nil {
					return err
				}
			case "--escala":
				value, _ := strconv.Atoi(args[index])
				if value <= 0 || value > 64 {
					return errCaptureUsage
				}
				scale = value
			default:
				script = args[index]
			}
		default:
			if strings.HasPrefix(args[index], "-") || input != "" {
				return fmt.Errorf("argumento inesperado %q", args[index])
			}
			input = args[index]
		}
	}
	if input == "" {
		return fmt.Errorf("falta el archivo .cometa de entrada")
	}
	if filepath.Ext(input) != ".cometa" {
		return fmt.Errorf("el archivo de entrada debe tener extensión .cometa")
	}
	if args[0] == "ejecutar" && output != "" {
		return fmt.Errorf("ejecutar no acepta -o")
	}
	if output == "" {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".go"
		if args[0] == "captura" {
			output = strings.TrimSuffix(input, filepath.Ext(input)) + ".png"
		}
		if args[0] == "construir" {
			output = strings.TrimSuffix(input, filepath.Ext(input))
			if runtime.GOOS == "windows" {
				output += ".exe"
			}
		}
	}
	var events []scriptEvent
	if script != "" {
		var err error
		if events, err = readScript(script); err != nil {
			return err
		}
	}
	compile := compiler.CompileProject
	if args[0] != "compilar" {
		compile = compiler.CompileProjectForRun
	}
	generated, err := compile(input, nil)
	if err != nil {
		return err
	}
	if args[0] == "captura" {
		return captureProgram(generated, output, frames, scale, events)
	}
	if args[0] != "compilar" {
		return buildProgram(generated, output, args[0] == "ejecutar")
	}
	if err = os.WriteFile(output, generated, 0o644); err != nil {
		return fmt.Errorf("no se pudo escribir %s: %w", output, err)
	}
	return nil
}

func isLSPCommand(args []string) bool {
	return len(args) == 1 && args[0] == "lsp" ||
		len(args) == 2 && args[0] == "lsp" && args[1] == "--stdio"
}
