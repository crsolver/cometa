package sema

import (
	"cometa/internal/ast"
	"cometa/internal/diagnostic"
)

var stringMethodOrder = []string{
	"longitud", "esta_vacia", "contiene", "buscar_indice", "empieza_con",
	"termina_con", "mayusculas", "minusculas", "recortar", "reemplazar",
	"dividir", "obtener", "subcadena", "a_entero", "a_decimal",
}

func StringMethods() map[string]FuncInfo {
	stringType := Type{Kind: String}
	number := Type{Kind: Integer}
	optionalNumber := Type{Kind: Optional, Elem: &number}
	optionalString := Type{Kind: Optional, Elem: &stringType}
	decimal := Type{Kind: Decimal}
	optionalDecimal := Type{Kind: Optional, Elem: &decimal}
	methods := map[string]FuncInfo{}
	add := func(name string, params []ast.Param, types []Type, result Type) {
		methods[name] = FuncInfo{Decl: &ast.FuncDecl{Name: name, Params: params}, Params: types, Return: result}
	}
	add("longitud", nil, nil, number)
	add("esta_vacia", nil, nil, Type{Kind: Boolean})
	add("contiene", []ast.Param{{Name: "valor"}}, []Type{stringType}, Type{Kind: Boolean})
	add("buscar_indice", []ast.Param{{Name: "valor"}}, []Type{stringType}, optionalNumber)
	add("empieza_con", []ast.Param{{Name: "prefijo"}}, []Type{stringType}, Type{Kind: Boolean})
	add("termina_con", []ast.Param{{Name: "sufijo"}}, []Type{stringType}, Type{Kind: Boolean})
	add("mayusculas", nil, nil, stringType)
	add("minusculas", nil, nil, stringType)
	add("recortar", nil, nil, stringType)
	add("reemplazar", []ast.Param{{Name: "buscar"}, {Name: "reemplazo"}}, []Type{stringType, stringType}, stringType)
	add("dividir", []ast.Param{{Name: "separador"}}, []Type{stringType}, Type{Kind: Slice, Elem: &stringType})
	add("obtener", []ast.Param{{Name: "indice"}}, []Type{number}, optionalString)
	add("subcadena", []ast.Param{{Name: "inicio"}, {Name: "fin"}}, []Type{number, number}, optionalString)
	add("a_entero", nil, nil, optionalNumber)
	add("a_decimal", nil, nil, optionalDecimal)
	return methods
}

func StringMethodNames() []string { return append([]string(nil), stringMethodOrder...) }

func StringMethodDocumentation(name string) string {
	return map[string]string{
		"longitud":      "Devuelve la cantidad de puntos de código Unicode.",
		"esta_vacia":    "Indica si la cadena está vacía.",
		"contiene":      "Indica si la cadena contiene el texto indicado.",
		"buscar_indice": "Devuelve el primer índice Unicode, o .Ninguno si no existe.",
		"empieza_con":   "Indica si la cadena comienza con el prefijo.",
		"termina_con":   "Indica si la cadena termina con el sufijo.",
		"mayusculas":    "Convierte la cadena a mayúsculas Unicode.",
		"minusculas":    "Convierte la cadena a minúsculas Unicode.",
		"recortar":      "Elimina el espacio Unicode de ambos extremos.",
		"reemplazar":    "Reemplaza todas las coincidencias no superpuestas.",
		"dividir":       "Divide la cadena conservando las partes vacías.",
		"obtener":       "Devuelve el punto de código del índice, o .Ninguno.",
		"subcadena":     "Devuelve el rango [inicio, fin), o .Ninguno si no es válido.",
		"a_entero":      "Convierte el texto a entero, o devuelve .Ninguno si no es un entero válido (sin espacios; usa recortar() antes).",
		"a_decimal":     "Convierte el texto a decimal (con punto, por ejemplo \"3.5\"), o devuelve .Ninguno si no es un número finito válido.",
	}[name]
}

func (c *checker) checkStringCall(call *ast.CallExpr, member *ast.MemberExpr) (Type, error) {
	signature, exists := StringMethods()[member.Name]
	if !exists {
		return Type{}, c.fail(member.Pos, "el método %q no existe en cadena%s", member.Name, diagnostic.Hint(member.Name, sortedKeys(StringMethods())))
	}
	result, err := c.bindArguments(call, signature)
	if err == nil {
		c.model.StringCalls[call] = member.Name
	}
	return result, err
}
