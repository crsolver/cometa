package compiler

import (
	"fmt"
	"hacha/internal/ast"
	"hacha/internal/codegen"
	"hacha/internal/diagnostic"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
)

// Analyze runs the source-language frontend without generating Go. It is used
// by editor tooling that needs an AST and semantic model for an in-memory file.
func Analyze(filename string, source []byte) (*ast.Program, *sema.Model, error) {
	var errors diagnostic.List
	tokens, err := lexer.Lex(filename, string(source))
	errors.Add(err)
	program, err := parser.Parse(filename, tokens)
	errors.Add(err)
	if len(program.Imports) != 0 {
		root, err := CanonicalPath(filename)
		if err != nil {
			return program, nil, err
		}
		p, err := AnalyzeProject(filename, func(path string) ([]byte, error) {
			if path == root {
				return source, nil
			}
			return nil, fmt.Errorf("usar archivos locales requiere AnalyzeProject con un cargador de módulos")
		})
		return p.Program, p.Model, err
	}
	model, err := sema.Check(filename, program)
	errors.Add(err)
	if err := errors.Err(); err != nil {
		return program, model, err
	}
	if err = loadAssets(filename, model); err != nil {
		return program, model, err
	}
	return program, model, nil
}

func Compile(filename string, source []byte) ([]byte, error) {
	program, model, err := Analyze(filename, source)
	if err != nil {
		return nil, err
	}
	return codegen.Generate(filename, program, model)
}
