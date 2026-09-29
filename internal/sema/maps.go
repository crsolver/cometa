package sema

import (
	"cometa/internal/ast"
	"cometa/internal/diagnostic"
)

func mapKeyAllowed(t Type) bool { return t.Kind == String || t.Kind == Integer || t.Kind == Boolean }

// MapMethods shares concrete signatures between checking and editor tooling.
func MapMethods(t Type) map[string]FuncInfo {
	methods := map[string]FuncInfo{}
	add := func(name string, key bool, result Type) {
		f := FuncInfo{Decl: &ast.FuncDecl{Name: name}, Return: result}
		if key {
			f.Decl.Params = []ast.Param{{Name: "clave"}}
			f.Params = []Type{*t.Key}
		}
		methods[name] = f
	}
	add("longitud", false, Type{Kind: Integer})
	add("esta_vacia", false, Type{Kind: Boolean})
	add("contiene", true, Type{Kind: Boolean})
	add("obtener", true, Type{Kind: Optional, Elem: t.Elem})
	add("eliminar", true, Type{Kind: Boolean})
	add("claves", false, Type{Kind: Slice, Elem: t.Key})
	add("valores", false, Type{Kind: Slice, Elem: t.Elem})
	add("copiar", false, t)
	add("vaciar", false, Type{Kind: Void})
	return methods
}

func MapMethodDocumentation(name string) string {
	return map[string]string{
		"longitud":   "Devuelve la cantidad de entradas.",
		"esta_vacia": "Indica si el mapa está vacío.",
		"contiene":   "Indica si existe la clave.",
		"obtener":    "Devuelve el valor, o .Ninguno si la clave no existe.",
		"eliminar":   "Elimina la entrada y devuelve si existía; modifica todos los aliases.",
		"claves":     "Devuelve una lista independiente de claves, sin orden garantizado.",
		"valores":    "Devuelve una lista independiente de valores, sin orden garantizado.",
		"copiar":     "Devuelve una copia superficial independiente del mapa.",
		"vaciar":     "Elimina todas las entradas; modifica todos los aliases.",
	}[name]
}

func (c *checker) checkMapCall(call *ast.CallExpr, member *ast.MemberExpr, t Type) (Type, error) {
	f, ok := MapMethods(t)[member.Name]
	if !ok {
		return Type{}, c.fail(member.Pos, "el método %q no existe en %s%s", member.Name, t.String(), diagnostic.Hint(member.Name, sortedKeys(MapMethods(t))))
	}
	result, err := c.bindArguments(call, f)
	if err == nil {
		c.model.MapCalls[call] = member.Name
	}
	return result, err
}

func (c *checker) checkMapLiteral(expr *ast.MapLiteralExpr, expected *Type) (Type, error) {
	var key, value *Type
	if expected != nil && expected.Kind != Interface {
		if expected.Kind != Map {
			return Type{}, c.fail(expr.Pos, "se esperaba %s, no un mapa", expected.String())
		}
		key, value = expected.Key, expected.Elem
	}
	if len(expr.Entries) == 0 && key == nil {
		return Type{}, c.fail(expr.Pos, "no se puede inferir el tipo de un mapa vacío")
	}
	broken := false
	for _, entry := range expr.Entries {
		k, err := c.checkExprExpected(entry.Key, key)
		if err != nil {
			c.report(err)
			broken = true
		} else if k.Kind == Never {
			// An explicit exit prevents this entry from being constructed.
		} else if !mapKeyAllowed(k) {
			c.report(c.fail(entry.Key.Position(), "una clave de mapa debe ser cadena, entero o bool, no %s", k.String()))
			broken = true
		} else if key == nil {
			inferred := k
			key = &inferred
		} else if !k.Equal(*key) {
			c.report(c.fail(entry.Key.Position(), "la clave debe ser %s, no %s", key.String(), k.String()))
			broken = true
		}
		v, err := c.checkExprExpected(entry.Value, value)
		if err != nil {
			c.report(err)
			broken = true
			continue
		}
		if v.Kind == Never {
			continue
		}
		if v.Kind == Void {
			c.report(c.fail(entry.Value.Position(), "una entrada de mapa requiere un valor"))
			broken = true
			continue
		}
		if value == nil {
			inferred := v
			value = &inferred
		} else if !c.model.Assignable(v, *value) {
			c.report(c.fail(entry.Value.Position(), "el valor debe ser %s, no %s", value.String(), v.String()))
			broken = true
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	if key == nil || value == nil {
		return Type{Kind: Never}, nil
	}
	return Type{Kind: Map, Key: key, Elem: value}, nil
}
