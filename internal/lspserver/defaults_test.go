package lspserver

import (
	"testing"

	"hacha/internal/ast"
	"hacha/internal/compiler"
)

func TestDefaultParameterSignatureAndHover(t *testing.T) {
	source := "fn f(a num = 1, b num = a) num b\n"
	program, model, err := compiler.Analyze("defaults.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	function := program.Decls[0].(*ast.FuncDecl)
	if got := functionDetail(function); got != "fn f(a num = …, b num = …) num" {
		t.Fatalf("signature = %q", got)
	}
	info, ok := hoverInFunction(function, model.Functions["f"], model, function.Params[1].Default.Position(), []string{source})
	if !ok || info != variableHover("a", model.Functions["f"].Params[0]) {
		t.Fatalf("hover = %+v, found = %v", info, ok)
	}
}
