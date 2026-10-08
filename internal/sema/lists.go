package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
)

var listMutators = map[string]bool{
	"agregar": true, "extender": true, "insertar": true,
	"eliminar": true, "invertir": true,
	"ordenar": true, "ordenar_por": true,
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
	descending := ast.Param{Name: "descendente", Default: &ast.LiteralExpr{Kind: "bool", Value: "falso"}}
	add("ordenar", []ast.Param{descending}, []Type{{Kind: Boolean}}, Type{Kind: Void})
	// checkListCall replaces the key type with the one of the argument.
	keys := Type{Kind: Slice, Elem: &Type{Kind: Decimal}}
	add("ordenar_por", []ast.Param{{Name: "claves"}, descending}, []Type{keys, {Kind: Boolean}}, Type{Kind: Void})
	return methods
}

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
		"ordenar":       "Ordena de menor a mayor una lista de entero, decimal o cadena en el mismo almacenamiento; con descendente = verdadero, de mayor a menor.",
		"ordenar_por":   "Ordena esta lista según una lista de claves (entero, decimal o cadena) de la misma longitud: el elemento i se coloca donde quedaría claves[i]. Los empates conservan su orden y las claves no se modifican.",
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

func sortable(t Type) bool { return t.Kind == Integer || t.Kind == Decimal || t.Kind == String }

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
	if member.Name == "ordenar" && !sortable(*list.Elem) {
		return Type{}, c.fail(member.Pos, "el método \"ordenar\" solo admite listas de entero, decimal o cadena, no %s; usa ordenar_por con una lista de claves", list.String())
	}
	if member.Name == "ordenar_por" {
		// The keys may be [entero], [decimal] or [cadena]: take the type of the
		// argument when it is one of them, so binding checks the rest as usual.
		for i, arg := range call.Args {
			named := i < len(call.ArgInfo) && call.ArgInfo[i].Name != ""
			if named && call.ArgInfo[i].Name != "claves" || !named && i != 0 {
				continue
			}
			if empty, ok := arg.(*ast.ListLiteralExpr); ok && len(empty.Elements) == 0 {
				continue // `[]` takes the declared key type
			}
			keys, err := c.checkExpr(arg)
			if err != nil {
				return Type{}, err
			}
			if keys.Kind != Slice || !sortable(*keys.Elem) {
				return Type{}, c.fail(arg.Position(), "las claves de ordenar_por deben ser una lista de entero, decimal o cadena, no %s", keys.String())
			}
			signature.Params = []Type{keys, signature.Params[1]}
		}
	}
	result, err := c.bindArguments(call, signature)
	if err == nil {
		c.model.Builtins[call] = Builtin{Kind: ListMethod, Name: member.Name}
	}
	return result, err
}
