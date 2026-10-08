package sema

import (
	"github.com/crsolver/cometa/internal/ast"
)

type FieldInfo struct {
	Decl *ast.Field
	Type Type
}

type FuncInfo struct {
	EmbeddedPath []string
	TypeParams   []Type
	Constraints  []Type
	TypeArgs     []Type
	Decl         *ast.FuncDecl
	Params       []Type
	Return       Type
}

type TypeInfo struct {
	Decl    *ast.TypeDecl
	Fields  map[string]FieldInfo
	Methods map[string]FuncInfo
}

type GlobalInfo struct {
	Decl     *ast.GlobalDecl
	Type     Type
	Constant bool
}

// BuiltinKind says which part of the language implements a built-in call.
type BuiltinKind int

const (
	ListMethod   BuiltinKind = iota + 1 // lista.agregar(x)
	MapMethod                           // mapa.claves()
	StringMethod                        // texto.mayusculas()
	NumberMethod                        // n.formato(2)
	StructCopy                          // the copiar() every tipo has
	Conversion                          // entero(x), decimal(x)
)

// Builtin is a call that no declaration implements.
type Builtin struct {
	Kind BuiltinKind
	// Name is the method, or the type a Conversion converts to.
	Name string
	// Type is the result of a Conversion and the type a StructCopy copies.
	Type Type
}

type Model struct {
	// FormatsValues is set when an interpolation holds a value that only the
	// imprimir runtime can spell (a list, a structure, an enum...).
	FormatsValues    bool
	NumericCoercions map[ast.Expr]Type
	Assets           map[*ast.CallExpr]Asset
	Stdlib           StdlibModel
	Interfaces       map[string]*InterfaceInfo
	TypeParams       map[ast.Decl][]Type
	Constraints      map[string]Type
	RawTypes         map[ast.Expr]Type
	TypeRefs         map[*ast.TypeRef]Type
	Wraps            map[ast.Expr]Type
	Calls            map[*ast.CallExpr]CallInfo
	// Builtins holds the calls the language implements itself. A call is
	// here, in Stdlib.Calls, or is a call to a declared function.
	Builtins      map[*ast.CallExpr]Builtin
	EnumCompares  map[*ast.BinaryExpr]EnumCompare
	LocalNames    map[string]bool
	Enums         map[string]*EnumInfo
	Constructors  map[ast.Expr]ConstructorInfo
	Types         map[string]*TypeInfo
	Functions     map[string]FuncInfo
	Globals       map[string]*GlobalInfo
	GlobalRefs    map[*ast.IdentExpr]*GlobalInfo
	ExprTypes     map[ast.Expr]Type
	ExpectedTypes map[ast.Expr]Type
	PatternTypes  map[*ast.MatchArm]Type
	VarTypes      map[*ast.VarDeclStmt]Type
}
