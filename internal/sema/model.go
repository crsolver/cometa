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

type Model struct {
	// FormatsValues is set when an interpolation holds a value that only the
	// imprimir runtime can spell (a list, a structure, an enum...).
	FormatsValues    bool
	NumericCoercions map[ast.Expr]Type
	NumericCalls     map[*ast.CallExpr]Type
	Assets           map[*ast.CallExpr]Asset
	Game             GameModel
	Interfaces       map[string]*InterfaceInfo
	TypeParams       map[ast.Decl][]Type
	Constraints      map[string]Type
	RawTypes         map[ast.Expr]Type
	TypeRefs         map[*ast.TypeRef]Type
	Wraps            map[ast.Expr]Type
	Calls            map[*ast.CallExpr]CallInfo
	ListCalls        map[*ast.CallExpr]string
	MapCalls         map[*ast.CallExpr]string
	StringCalls      map[*ast.CallExpr]string
	NumberCalls      map[*ast.CallExpr]string
	StructCopies     map[*ast.CallExpr]Type // built-in copiar() calls on user structs
	EnumCompares     map[*ast.BinaryExpr]EnumCompare
	LocalNames       map[string]bool
	Enums            map[string]*EnumInfo
	Constructors     map[ast.Expr]ConstructorInfo
	Types            map[string]*TypeInfo
	Functions        map[string]FuncInfo
	Globals          map[string]*GlobalInfo
	GlobalRefs       map[*ast.IdentExpr]*GlobalInfo
	ExprTypes        map[ast.Expr]Type
	ExpectedTypes    map[ast.Expr]Type
	PatternTypes     map[*ast.MatchArm]Type
	VarTypes         map[*ast.VarDeclStmt]Type
}
