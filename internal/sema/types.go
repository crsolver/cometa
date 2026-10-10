package sema

import (
	"strings"
)

type Kind int

const (
	Invalid Kind = iota
	Void
	Decimal
	Integer
	String
	Boolean
	Named
	Slice
	Map
	Enum
	Optional
	Result
	Never
	Interface
	TypeParameter
)

type Type struct {
	Key   *Type
	Args  []Type
	Owner string // Declaration identity for a type parameter.
	Kind  Kind
	Name  string
	Elem  *Type
	Err   *Type
}

func (t Type) String() string {
	switch t.Kind {
	case Optional:
		if t.Elem.Kind == Result {
			return "(" + t.Elem.String() + ")?"
		}
		return t.Elem.String() + "?"
	case Result:
		payload := t.Elem.String()
		if t.Elem.Kind == Void {
			payload = ""
		}
		if t.Elem.Kind == Result {
			payload = "(" + payload + ")"
		}
		if t.Err.Kind == String {
			return payload + "!"
		}
		errorType := t.Err.String()
		if t.Err.Wrapped() {
			errorType = "(" + errorType + ")"
		}
		return payload + "!" + errorType
	case Never:
		return "no retorna"
	case Void:
		return "sin valor"
	case Decimal:
		return "decimal"
	case Integer:
		return "entero"
	case String:
		return "cadena"
	case Boolean:
		return "bool"
	case Named, Enum, Interface, TypeParameter:
		name := t.Name
		if len(t.Args) > 0 {
			var args []string
			for _, arg := range t.Args {
				args = append(args, arg.String())
			}
			name += "<" + strings.Join(args, ", ") + ">"
		}
		return name
	case Slice:
		return "[" + t.Elem.String() + "]"
	case Map:
		return "[" + t.Key.String() + ": " + t.Elem.String() + "]"
	default:
		return "tipo inválido"
	}
}

func (t Type) Equal(other Type) bool {
	if t.Kind != other.Kind || t.Name != other.Name || t.Owner != other.Owner || len(t.Args) != len(other.Args) {
		return false
	}
	for i := range t.Args {
		if !t.Args[i].Equal(other.Args[i]) {
			return false
		}
	}
	if t.Kind == Map && (t.Key == nil || other.Key == nil || !t.Key.Equal(*other.Key)) {
		return false
	}
	if t.Kind == Slice || t.Kind == Map || t.Wrapped() {
		return t.Elem != nil && other.Elem != nil && t.Elem.Equal(*other.Elem) && (t.Kind != Result || t.Err.Equal(*other.Err))
	}
	return true
}
