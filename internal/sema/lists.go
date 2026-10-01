package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
)

var listMethodOrder = []string{
	"longitud", "esta_vacia", "contiene", "buscar_indice", "obtener",
	"primero", "ultimo", "agregar", "extender", "insertar", "eliminar",
	"copiar", "invertir",
}

var listMutators = map[string]bool{
	"agregar": true, "extender": true, "insertar": true,
	"eliminar": true, "invertir": true,
}

// ListMethods returns the concrete built-in method signatures for a list type.
// The declarations are synthetic and exist so normal argument binding and
// editor signature rendering share one source of truth.
func ListMethods(list Type) map[string]FuncInfo {
	if list.Kind != Slice || list.Elem == nil {
		return nil
	}
	element := *list.Elem
	optionalElement := Type{Kind: Optional, Elem: &element}
	number := Type{Kind: Integer}
	optionalNumber := Type{Kind: Optional, Elem: &number}
	methods := map[string]FuncInfo{}
	add := func(name string, params []ast.Param, types []Type, result Type) {
		methods[name] = FuncInfo{
			Decl:   &ast.FuncDecl{Name: name, Params: params},
			Params: types,
			Return: result,
		}
	}
	add("longitud", nil, nil, number)
	add("esta_vacia", nil, nil, Type{Kind: Boolean})
	add("contiene", []ast.Param{{Name: "valor"}}, []Type{element}, Type{Kind: Boolean})
	add("buscar_indice", []ast.Param{{Name: "valor"}}, []Type{element}, optionalNumber)
	add("obtener", []ast.Param{{Name: "indice"}}, []Type{number}, optionalElement)
	add("primero", nil, nil, optionalElement)
	add("ultimo", nil, nil, optionalElement)
	add("agregar", []ast.Param{{Name: "valor"}}, []Type{element}, Type{Kind: Void})
	add("extender", []ast.Param{{Name: "otra"}}, []Type{list}, Type{Kind: Void})
	add("insertar", []ast.Param{{Name: "indice"}, {Name: "valor"}}, []Type{number, element}, Type{Kind: Boolean})
	add("eliminar", []ast.Param{{Name: "indice"}}, []Type{number}, Type{Kind: Boolean})
	add("copiar", nil, nil, list)
	add("invertir", nil, nil, Type{Kind: Void})
	return methods
}

func ListMethodNames() []string { return append([]string(nil), listMethodOrder...) }

func ListMethodDocumentation(name string) string {
	return map[string]string{
		"longitud":      "Devuelve la cantidad de elementos.",
		"esta_vacia":    "Indica si la lista no contiene elementos.",
		"contiene":      "Indica si la lista contiene un valor igual.",
		"buscar_indice": "Devuelve el primer índice del valor, o .Ninguno si no existe.",
		"obtener":       "Devuelve el elemento del índice, o .Ninguno si el índice no es válido.",
		"primero":       "Devuelve el primer elemento, o .Ninguno si la lista está vacía.",
		"ultimo":        "Devuelve el último elemento, o .Ninguno si la lista está vacía.",
		"agregar":       "Agrega un elemento y actualiza esta variable de lista.",
		"extender":      "Agrega los elementos de otra lista y actualiza esta variable de lista.",
		"insertar":      "Inserta un elemento y actualiza esta variable de lista; devuelve falso si el índice no es válido.",
		"eliminar":      "Elimina un elemento y actualiza esta variable de lista; devuelve falso si el índice no es válido.",
		"copiar":        "Devuelve una copia superficial con almacenamiento independiente.",
		"invertir":      "Invierte los elementos de esta lista en el mismo almacenamiento.",
	}[name]
}

func listElementComparable(t Type) bool {
	switch t.Kind {
	case Integer, Decimal, String, Boolean, Named:
		return true
	default:
		return false
	}
}

func assignableListReceiver(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.IdentExpr, *ast.ReceiverExpr, *ast.MemberExpr, *ast.IndexExpr:
		return true
	default:
		return false
	}
}

func (c *checker) checkListCall(call *ast.CallExpr, member *ast.MemberExpr, list Type) (Type, error) {
	methods := ListMethods(list)
	signature, exists := methods[member.Name]
	if !exists {
		return Type{}, c.fail(member.Pos, "el método %q no existe en %s%s", member.Name, list.String(), diagnostic.Hint(member.Name, sortedKeys(methods)))
	}
	if listMutators[member.Name] && !assignableListReceiver(member.Object) {
		return Type{}, c.fail(member.Object.Position(), "el método %q requiere una lista asignable", member.Name)
	}
	if (member.Name == "contiene" || member.Name == "buscar_indice") && !listElementComparable(*list.Elem) {
		return Type{}, c.fail(member.Pos, "el método %q no admite elementos de tipo %s", member.Name, list.Elem.String())
	}
	result, err := c.bindArguments(call, signature)
	if err == nil {
		c.model.ListCalls[call] = member.Name
	}
	return result, err
}
