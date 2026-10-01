package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
)

// NumberMethods lists the built-in methods of entero and decimal values.
func NumberMethods() map[string]FuncInfo {
	return map[string]FuncInfo{
		"formato": {
			Decl:   &ast.FuncDecl{Name: "formato", Params: []ast.Param{{Name: "decimales"}}},
			Params: []Type{{Kind: Integer}},
			Return: Type{Kind: String},
		},
	}
}

func NumberMethodDocumentation(name string) string {
	return map[string]string{
		"formato": "Escribe el número con exactamente esa cantidad de decimales (de 0 a 20), por ejemplo 3.14159.formato(2) es \"3.14\".",
	}[name]
}

func (c *checker) checkNumberCall(call *ast.CallExpr, member *ast.MemberExpr) (Type, error) {
	signature, exists := NumberMethods()[member.Name]
	if !exists {
		return Type{}, c.fail(member.Pos, "el método %q no existe en números%s", member.Name, diagnostic.Hint(member.Name, sortedKeys(NumberMethods())))
	}
	result, err := c.bindArguments(call, signature)
	if err == nil {
		c.model.NumberCalls[call] = member.Name
	}
	return result, err
}
