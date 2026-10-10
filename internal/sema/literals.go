package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/stdlib"
)

func (c *checker) checkStructLiteral(expr *ast.StructLiteralExpr, expected *Type) (Type, error) {
	if c.invalid[expr.TypeName] {
		return Type{}, errInvalid
	}
	var result Type
	if expr.TypeName != "" {
		if _, exists := c.model.Types[expr.TypeName]; !exists {
			return Type{}, c.fail(expr.Pos, "el tipo %q no existe", expr.TypeName)
		}
		ref := ast.TypeRef{Pos: expr.Pos, Name: expr.TypeName}
		if expr.Type != nil {
			ref = *expr.Type
		}
		var err error
		result, err = c.resolveType(ref)
		if err != nil {
			return Type{}, err
		}
	} else if expected != nil && expected.Kind == Named {
		result = *expected
	} else {
		return Type{}, c.fail(expr.Pos, "no se puede inferir el tipo del literal de estructura")
	}
	if expected != nil && !c.model.Assignable(result, *expected) {
		return Type{}, c.fail(expr.Pos, "el literal es %s, pero se esperaba %s%s", result.String(), expected.String(), c.unmet(result, *expected))
	}
	if stdlib.IsOpaque(result.Name) {
		return Type{}, c.fail(expr.Pos, "%s es un tipo opaco; use su constructor incorporado", result.Name)
	}
	info := c.model.StructInfo(result)
	broken := false
	seen := map[string]bool{}
	if len(expr.Values) > len(info.Decl.Fields) {
		extra := expr.Values[len(info.Decl.Fields)]
		return Type{}, c.fail(extra.Position(), "el literal de %s tiene más valores posicionales que campos directos", result.String())
	}
	for index, value := range expr.Values {
		err := func() error {
			fieldDecl := info.Decl.Fields[index]
			if !ast.Accessible(fieldDecl.Public, fieldDecl.Pos, value.Position()) {
				return c.fail(value.Position(), "el campo %q es privado", fieldDecl.Name)
			}
			field := info.Fields[fieldDecl.Name]
			seen[fieldDecl.Name] = true
			actual, err := c.checkExprExpected(value, &field.Type)
			if err != nil {
				return err
			}
			if !c.model.Assignable(actual, field.Type) {
				return c.fail(value.Position(), "el campo %q debe ser %s, no %s%s", fieldDecl.Name, field.Type.String(), actual.String(), c.unmet(actual, field.Type))
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	for _, value := range expr.Fields {
		err := func() error {
			if seen[value.Name] {
				return c.fail(value.Pos, "el campo %q está duplicado", value.Name)
			}
			seen[value.Name] = true
			field, exists := info.Fields[value.Name]
			if !exists {
				return c.fail(value.Pos, "el campo %q no existe en %s%s", value.Name, result.Name, diagnostic.Hint(value.Name, sortedKeys(info.Fields)))
			}
			if !ast.Accessible(field.Decl.Public, field.Decl.Pos, value.Pos) {
				return c.fail(value.Pos, "el campo %q es privado", value.Name)
			}
			actual, err := c.checkExprExpected(value.Value, &field.Type)
			if err != nil {
				return err
			}
			if !c.model.Assignable(actual, field.Type) {
				return c.fail(value.Value.Position(), "el campo %q debe ser %s, no %s%s", value.Name, field.Type.String(), actual.String(), c.unmet(actual, field.Type))
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	for _, field := range info.Decl.Fields {
		err := func() error {
			if !seen[field.Name] && field.Default == nil {
				if c.model.Types[result.Name].Fields[field.Name].Type.Kind == TypeParameter {
					return c.fail(expr.Pos, "el campo %s requiere inicialización explícita", field.Name)
				}
				if missing := c.missingDefault(info.Fields[field.Name].Type, field.Name); missing != "" {
					return c.fail(expr.Pos, "el campo %s requiere inicialización explícita", missing)
				}
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	return result, nil
}

func (c *checker) checkListLiteral(expr *ast.ListLiteralExpr, expected *Type) (Type, error) {
	broken := false
	var element *Type
	if expected != nil && expected.Kind != Interface {
		if expected.Kind != Slice {
			return Type{}, c.fail(expr.Pos, "se esperaba %s, no una lista", expected.String())
		}
		element = expected.Elem
	}
	if len(expr.Elements) == 0 {
		if element == nil {
			return Type{}, c.fail(expr.Pos, "no se puede inferir el tipo de una lista vacía")
		}
		return Type{Kind: Slice, Elem: element}, nil
	}
	for index, value := range expr.Elements {
		actual, err := c.checkExprExpected(value, element)
		if err != nil {
			c.report(err)
			broken = true
			continue
		}
		if element == nil {
			if actual.Kind == Void {
				return Type{}, c.fail(value.Position(), "un elemento de lista requiere un valor")
			}
			inferred := actual
			element = &inferred
		} else if !c.model.Assignable(actual, *element) {
			if expected == nil && actual.Kind == Decimal && element.Kind == Integer {
				*element = Type{Kind: Decimal}
				for _, prior := range expr.Elements[:index] {
					c.promoteNumeric(prior)
				}
				continue
			}
			c.report(c.fail(value.Position(), "el elemento debe ser %s, no %s%s", element.String(), actual.String(), c.unmet(actual, *element)))
			broken = true
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	return Type{Kind: Slice, Elem: element}, nil
}
