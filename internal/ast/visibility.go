package ast

func IsPublic(node Decl) bool {
	switch d := node.(type) {
	case *FuncDecl:
		return d.Public
	case *GlobalDecl:
		return d.Public
	case *TypeDecl:
		return d.Public
	case *EnumDecl:
		return d.Public
	case *InterfaceDecl:
		return d.Public
	}
	return false
}

func SetPublic(node Decl, public bool) {
	switch d := node.(type) {
	case *FuncDecl:
		d.Public = public
	case *GlobalDecl:
		d.Public = public
	case *TypeDecl:
		d.Public = public
	case *EnumDecl:
		d.Public = public
	case *InterfaceDecl:
		d.Public = public
	}
}

func Accessible(public bool, declaration, use Pos) bool {
	return public || declaration.Filename == use.Filename
}
