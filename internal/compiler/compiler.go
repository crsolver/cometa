package compiler

import (
	"hacha/internal/ast"
	"hacha/internal/codegen"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
)

// Analyze runs the source-language frontend without generating Go. It is used
// by editor tooling that needs an AST and semantic model for an in-memory file.
func Analyze(filename string, source []byte) (*ast.Program, *sema.Model, error) {
	tokens, err := lexer.Lex(filename, string(source))
	if err != nil {
		return nil, nil, err
	}
	program, err := parser.Parse(filename, tokens)
	if err != nil {
		return nil, nil, err
	}
	if len(program.Imports) != 0 {
		return program, nil, &sema.Error{Filename: filename, Pos: program.Imports[0].Pos, Message: "usar requiere AnalyzeProject con un cargador de módulos"}
	}
	model, err := sema.Check(filename, program)
	if err != nil {
		return program, nil, err
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
