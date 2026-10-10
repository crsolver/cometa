package sema

import (
	"sort"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

func (c *checker) signature(function *ast.FuncDecl) (FuncInfo, error) {
	info := FuncInfo{Decl: function, Return: Type{Kind: Void}}
	broken := false
	seen := map[string]bool{}
	optional := false
	for index, param := range function.Params {
		err := func() error {
			if param.Default != nil {
				if param.Variadic {
					return c.fail(param.Pos, "un parámetro variádico no acepta un valor predeterminado")
				}
				optional = true
			} else if optional && !param.Variadic {
				return c.fail(param.Pos, "un parámetro obligatorio no puede seguir a uno con valor predeterminado")
			}
			if param.Variadic && index != len(function.Params)-1 {
				return c.fail(param.Pos, "el parámetro variádico debe ser el último")
			}
			if seen[param.Name] {
				return c.fail(param.Pos, "el parámetro %q está duplicado", param.Name)
			}
			seen[param.Name] = true
			paramType, err := c.resolveType(param.Type)
			if err != nil {
				return err
			}
			if param.Variadic {
				element := paramType
				paramType = Type{Kind: Slice, Elem: &element}
			}
			info.Params = append(info.Params, paramType)
			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}

	}
	if function.ReturnType != nil {
		result, err := c.resolveType(*function.ReturnType)
		if err != nil {
			return FuncInfo{}, err
		}
		info.Return = result
	}
	if broken {
		return FuncInfo{}, errInvalid
	}
	return info, nil
}

func (c *checker) resolveBuiltinType(ref ast.TypeRef) (Type, error) {
	if ref.Wrapper != "" {
		payload, err := c.resolveType(*ref.Payload)
		if err != nil {
			return Type{}, err
		}
		t := Type{Kind: Optional, Elem: &payload}
		if ref.Wrapper == "!" {
			t.Kind = Result
			e, err := c.resolveType(*ref.ErrorType)
			if err != nil {
				return Type{}, err
			}
			t.Err = &e
		}
		return t, nil
	}
	if ref.Key != nil {
		key, err := c.resolveType(*ref.Key)
		if err != nil {
			return Type{}, err
		}
		if !mapKeyAllowed(key) {
			return Type{}, c.fail(ref.Key.Pos, "una clave de mapa debe ser cadena, entero o bool, no %s", key.String())
		}
		element, err := c.resolveType(*ref.Element)
		if err != nil {
			return Type{}, err
		}
		if element.Kind == Void {
			return Type{}, c.fail(ref.Pos, "un mapa requiere valores")
		}
		return Type{Kind: Map, Key: &key, Elem: &element}, nil
	}
	if ref.IsSlice() {
		element, err := c.resolveType(*ref.Element)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: Slice, Elem: &element}, nil
	}
	switch ref.Name {
	case "$unidad":
		return Type{Kind: Void}, nil
	case "num":
		return Type{}, c.fail(ref.Pos, "el tipo num fue eliminado; use entero o decimal")
	case "entero":
		return Type{Kind: Integer}, nil
	case "decimal":
		return Type{Kind: Decimal}, nil
	case "cadena":
		return Type{Kind: String}, nil
	case "bool":
		return Type{Kind: Boolean}, nil
	default:
		if c.model.Enums[ref.Name] != nil {
			return Type{Kind: Enum, Name: ref.Name}, nil
		}
		if _, exists := c.model.Types[ref.Name]; !exists {
			return Type{}, c.fail(ref.Pos, "el tipo %q no existe", ref.Name)
		}
		return Type{Kind: Named, Name: ref.Name}, nil
	}
}

func (c *checker) checkFunction(function *ast.FuncDecl, receiver *TypeInfo) error {
	before := c.damaged
	for _, p := range function.Params {
		if stdlib.Reserved(p.Name) {
			return c.fail(p.Pos, "nombre de parámetro reservado: %s", p.Name)
		}
	}
	var scope ast.Decl = function
	if receiver != nil {
		scope = receiver.Decl
	}
	defer c.enterDeclaration(scope, receiver)()
	for index, param := range function.Params {
		c.model.LocalNames[param.Name] = true
		expected := c.mustResolve(function, index)
		if param.Default != nil {
			c.inDefault = true
			actual, err := c.checkExprExpected(param.Default, &expected)
			c.inDefault = false
			if err != nil {
				c.report(err)
				c.damaged++
			} else if !c.model.Assignable(actual, expected) {
				c.report(c.fail(param.Default.Position(), "el valor predeterminado de %q debe ser %s, no %s%s", param.Name, expected.String(), actual.String(), c.unmet(actual, expected)))
				c.damaged++
			}
		}
		c.vars[param.Name] = expected
	}
	var expected Type
	if receiver == nil {
		expected = c.model.Functions[function.Name].Return
	} else {
		expected = receiver.Methods[function.Name].Return
	}
	c.returnType = expected
	if expected.Kind != Void {
		var declared *ast.Pos
		if function.ReturnType != nil {
			declared = &function.ReturnType.Pos
		}
		actual, err := c.checkValueBlockAt(function.Body, &expected, declared)
		if err != nil {
			return err
		}
		if !c.model.Assignable(actual, expected) {
			return c.fail(function.Pos, "la función %q produce %s, pero declara %s%s", function.Name, actual.String(), expected.String(), c.unmet(actual, expected))
		}
	} else {
		for _, stmt := range function.Body {
			c.checkStmt(stmt)
		}
	}
	var unread []*ast.VarDeclStmt
	for binding, read := range c.reads {
		if !read {
			unread = append(unread, binding)
		}
	}
	sort.Slice(unread, func(i, j int) bool {
		if unread[i].Pos.Line != unread[j].Pos.Line {
			return unread[i].Pos.Line < unread[j].Pos.Line
		}
		return unread[i].Pos.Column < unread[j].Pos.Column
	})
	if c.damaged == before {
		for _, binding := range unread {
			c.report(c.fail(binding.Pos, "el resultado %q debe usarse o manejarse explícitamente", binding.Name))
		}
	}
	return nil
}

func (c *checker) checkFieldDefaults(d *ast.TypeDecl) error {
	defer c.enterDeclaration(d, nil)()
	info := c.model.Types[d.Name]
	for _, field := range d.Fields {
		if field.Default == nil {
			continue
		}
		expected := info.Fields[field.Name].Type
		c.inDefault = true
		actual, err := c.checkExprExpected(field.Default, &expected)
		c.inDefault = false
		if err != nil {
			c.report(err)
			c.damaged++
			continue
		}
		if !c.model.Assignable(actual, expected) {
			c.report(c.fail(field.Default.Position(), "el valor predeterminado de %q debe ser %s, no %s%s", field.Name, expected.String(), actual.String(), c.unmet(actual, expected)))
			c.damaged++
		}
	}
	return nil
}

func (c *checker) mustResolve(function *ast.FuncDecl, index int) Type {
	t, _ := c.resolveType(function.Params[index].Type)
	if function.Params[index].Variadic {
		return Type{Kind: Slice, Elem: &t}
	}
	return t
}
