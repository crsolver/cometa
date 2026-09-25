package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cometa/internal/compiler"
	"cometa/internal/lspserver"
	"cometa/internal/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
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
	if len(args) == 0 || (args[0] != "compilar" && args[0] != "ejecutar" && args[0] != "construir") {
		return fmt.Errorf("uso: cometa <compilar|ejecutar|construir> <archivo.cometa> [-o salida] | lsp")
	}
	var input, output string
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "-o":
			index++
			if index >= len(args) || output != "" {
				return fmt.Errorf("uso: cometa compilar <archivo.cometa> [-o <archivo.go>]")
			}
			output = args[index]
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
		if args[0] == "construir" {
			output = strings.TrimSuffix(input, filepath.Ext(input))
			if runtime.GOOS == "windows" {
				output += ".exe"
			}
		}
	}
	generated, err := compiler.CompileProject(input, nil)
	if err != nil {
		return err
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
