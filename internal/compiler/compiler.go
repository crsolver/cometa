package compiler

import (
	"fmt"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/codegen"
	"github.com/crsolver/cometa/internal/sema"
)

// Analyze runs the frontend over one in-memory file. It is AnalyzeSource for
// callers that only need the syntax tree and the semantic model.
func Analyze(filename string, source []byte) (*ast.Program, *sema.Model, error) {
	p, err := AnalyzeSource(filename, source)
	return p.Program, p.Model, err
}

// AnalyzeSource analyzes one in-memory file as a project of a single module,
// so a lone file goes through exactly the same passes as a project. The file
// may import the standard library; importing other files needs AnalyzeProject
// and a loader. Diagnostics name the file as filename spells it.
func AnalyzeSource(filename string, source []byte) (*Project, error) {
	root, err := CanonicalPath(filename)
	if err != nil {
		return &Project{Program: &ast.Program{InvalidNames: map[string]ast.Pos{}}}, err
	}
	p, err := AnalyzeProject(filename, func(path string) ([]byte, error) {
		if path == root {
			return source, nil
		}
		return nil, fmt.Errorf("usar archivos locales requiere AnalyzeProject con un cargador de módulos")
	})
	if p.Root != nil {
		p.Root.Display = filename
	}
	return p, p.DisplayPaths(err)
}

// Compile translates one in-memory file into Go.
func Compile(filename string, source []byte) ([]byte, error) {
	p, err := AnalyzeSource(filename, source)
	if err != nil {
		return nil, err
	}
	generated, err := p.Generate(codegen.Options{})
	return generated, p.DisplayPaths(err)
}
