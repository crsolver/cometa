package ast

// Pos identifies a one-based location in a Hacha source file.
type Pos struct {
	Filename string // Set by project analysis; empty for standalone analysis.
	Line     int
	Column   int
}

type Node interface {
	Position() Pos
}

type Program struct {
	NativeModules []string // Explicit bundled imports, collected across the project.
	// InvalidNames preserves declarations whose syntax could not be recovered.
	InvalidNames map[string]Pos
	Imports      []*ImportDecl
	Decls        []Decl
}

// BadStmt marks a damaged source region; it must never reach code generation.
type BadStmt struct {
	Pos  Pos
	Name string
}

func (*BadStmt) stmtNode()       {}
func (s *BadStmt) Position() Pos { return s.Pos }

type ImportDecl struct {
	Pos      Pos
	PathPos  Pos
	AliasPos Pos
	Path     string
	Alias    string
}

type EnumDecl struct {
	TypeParams []TypeParam
	Pos        Pos
	NamePos    Pos
	Name       string
	Variants   []*VariantDecl
}

func (*EnumDecl) declNode()       {}
func (d *EnumDecl) Position() Pos { return d.Pos }

type VariantDecl struct {
	Pos     Pos
	Name    string
	Payload *TypeRef
}

type MatchExpr struct {
	Pos        Pos
	Value      Expr
	Binding    string
	BindingPos Pos
	Arms       []*MatchArm
}

func (*MatchExpr) exprNode()       {}
func (e *MatchExpr) Position() Pos { return e.Pos }

type MatchArm struct {
	Invalid       bool
	TypePattern   *TypeRef
	QualifierType *TypeRef
	Pos           Pos
	Qualifier     string
	QualifierPos  Pos
	NamePos       Pos
	Pattern       string
	Body          []Stmt
}

// ContextualVariantExpr resolves its enum from the expected expression type.
type ContextualVariantExpr struct {
	Pos     Pos
	NamePos Pos
	Name    string
}

func (*ContextualVariantExpr) exprNode()       {}
func (e *ContextualVariantExpr) Position() Pos { return e.Pos }

type MatchStmt struct{ Match *MatchExpr }

func (*MatchStmt) stmtNode()       {}
func (s *MatchStmt) Position() Pos { return s.Match.Pos }

type Decl interface {
	Node
	declNode()
}

// GlobalDecl declares a package-level mutable variable or constant.
type GlobalDecl struct {
	Pos      Pos
	NamePos  Pos
	Name     string
	Type     *TypeRef
	Value    Expr
	Constant bool
}

func (*GlobalDecl) declNode()       {}
func (d *GlobalDecl) Position() Pos { return d.Pos }

type TypeRef struct {
	Args      []TypeRef
	Wrapper   string
	Payload   *TypeRef
	ErrorType *TypeRef
	Pos       Pos
	Name      string
	Element   *TypeRef
}

func (t TypeRef) IsSlice() bool { return t.Element != nil }

type TypeDecl struct {
	TypeParams []TypeParam
	Pos        Pos
	NamePos    Pos
	Name       string
	Fields     []*Field
	Methods    []*FuncDecl
}

func (*TypeDecl) declNode()       {}
func (d *TypeDecl) Position() Pos { return d.Pos }

type Field struct {
	Embedded bool
	Pos      Pos
	Name     string
	Type     TypeRef
}

func (f *Field) Position() Pos { return f.Pos }

type FuncDecl struct {
	TypeParams []TypeParam
	Pos        Pos
	NamePos    Pos
	Name       string
	Params     []Param
	ReturnType *TypeRef
	Body       []Stmt
	Receiver   string
}

func (*FuncDecl) declNode()       {}
func (d *FuncDecl) Position() Pos { return d.Pos }

type Param struct {
	Default  Expr
	Variadic bool
	Pos      Pos
	Name     string
	Type     TypeRef
}

type Stmt interface {
	Node
	stmtNode()
}

type ExprStmt struct {
	Pos  Pos
	Expr Expr
}

func (*ExprStmt) stmtNode()       {}
func (s *ExprStmt) Position() Pos { return s.Pos }

type AssignStmt struct {
	Pos    Pos
	Target Expr
	Value  Expr
}

func (*AssignStmt) stmtNode()       {}
func (s *AssignStmt) Position() Pos { return s.Pos }

type VarDeclStmt struct {
	Pos     Pos
	NamePos Pos
	Name    string
	Type    *TypeRef
	Value   Expr
}

func (*VarDeclStmt) stmtNode()       {}
func (s *VarDeclStmt) Position() Pos { return s.Pos }

type IfStmt struct {
	Pos      Pos
	Branches []IfBranch
	Else     []Stmt
}

func (*IfStmt) stmtNode()       {}
func (s *IfStmt) Position() Pos { return s.Pos }

type IfBranch struct {
	Binding    string
	BindingPos Pos
	Pos        Pos
	Condition  Expr
	Body       []Stmt
}

// RepeatStmt represents an infinite loop (Iterable is nil), list iteration,
// or a numeric range (Iterable is the start and RangeEnd is the exclusive end).
type RepeatStmt struct {
	Pos        Pos
	Iterable   Expr
	RangeEnd   Expr
	ElementPos Pos
	Element    string
	IndexPos   Pos
	Index      string
	Body       []Stmt
}

func (*RepeatStmt) stmtNode()       {}
func (s *RepeatStmt) Position() Pos { return s.Pos }

type ContinueStmt struct{ Pos Pos }

type ScopeStmt struct {
	Pos Pos
	Value Expr
	Body []Stmt
}
func (*ScopeStmt) stmtNode() {}
func (s *ScopeStmt) Position() Pos { return s.Pos }

func (*ContinueStmt) stmtNode()       {}
func (s *ContinueStmt) Position() Pos { return s.Pos }

type BreakStmt struct{ Pos Pos }

func (*BreakStmt) stmtNode()       {}
func (s *BreakStmt) Position() Pos { return s.Pos }

type Expr interface {
	Node
	exprNode()
}

type IdentExpr struct {
	Pos  Pos
	Name string
}

func (*IdentExpr) exprNode()       {}
func (e *IdentExpr) Position() Pos { return e.Pos }

type ReceiverExpr struct {
	Pos     Pos
	NamePos Pos
	Name    string
}

func (*ReceiverExpr) exprNode()       {}
func (e *ReceiverExpr) Position() Pos { return e.Pos }

type LiteralExpr struct {
	Pos   Pos
	Kind  string
	Value string
}

func (*LiteralExpr) exprNode()       {}
func (e *LiteralExpr) Position() Pos { return e.Pos }

// InterpolatedStringExpr is a string whose Parts alternate between decoded
// literal text and embedded expressions. Literal parts are represented by
// string values; expression parts retain their original source positions.
type InterpolatedStringExpr struct {
	Pos   Pos
	Parts []InterpolatedStringPart
}

type InterpolatedStringPart struct {
	Text string
	Expr Expr
}

func (*InterpolatedStringExpr) exprNode()       {}
func (e *InterpolatedStringExpr) Position() Pos { return e.Pos }

type UnaryExpr struct {
	Pos      Pos
	Operator string
	Value    Expr
}

func (*UnaryExpr) exprNode()       {}
func (e *UnaryExpr) Position() Pos { return e.Pos }

type BinaryExpr struct {
	Pos      Pos
	Left     Expr
	Operator string
	Right    Expr
}

func (*BinaryExpr) exprNode()       {}
func (e *BinaryExpr) Position() Pos { return e.Pos }

type CallExpr struct {
	ArgInfo []ArgumentInfo // Parallel to Args, retaining source order.
	Pos     Pos
	Callee  Expr
	Args    []Expr
}

type ArgumentInfo struct {
	Pos    Pos
	Name   string
	Spread bool
}

func (*CallExpr) exprNode()       {}
func (e *CallExpr) Position() Pos { return e.Pos }

type MemberExpr struct {
	Pos     Pos
	Object  Expr
	NamePos Pos
	Name    string
}

func (*MemberExpr) exprNode()       {}
func (e *MemberExpr) Position() Pos { return e.Pos }

type IndexExpr struct {
	Pos    Pos
	Object Expr
	Index  Expr
}

func (*IndexExpr) exprNode()       {}
func (e *IndexExpr) Position() Pos { return e.Pos }

type StructLiteralExpr struct {
	Type     *TypeRef
	Pos      Pos
	TypeName string
	Fields   []FieldValue
	Values   []Expr
}

func (*StructLiteralExpr) exprNode()       {}
func (e *StructLiteralExpr) Position() Pos { return e.Pos }

type FieldValue struct {
	Pos   Pos
	Name  string
	Value Expr
}

type ListLiteralExpr struct {
	Pos      Pos
	Elements []Expr
}

func (*ListLiteralExpr) exprNode()       {}
func (e *ListLiteralExpr) Position() Pos { return e.Pos }

type IfExpr struct {
	Pos      Pos
	Branches []IfExprBranch
	Else     Expr
}

func (*IfExpr) exprNode()       {}
func (e *IfExpr) Position() Pos { return e.Pos }

type IfExprBranch struct {
	Binding    string
	BindingPos Pos
	Pos        Pos
	Condition  Expr
	Value      Expr
}

// ReturnExpr terminates the enclosing function, including inside value blocks.
type ReturnExpr struct {
	Pos   Pos
	Value Expr
}

func (*ReturnExpr) exprNode()       {}
func (e *ReturnExpr) Position() Pos { return e.Pos }

type TryExpr struct {
	Pos   Pos
	Value Expr
}

func (*TryExpr) exprNode()       {}
func (e *TryExpr) Position() Pos { return e.Pos }

type RecoverExpr struct {
	Pos        Pos
	Value      Expr
	Error      bool
	Binding    string
	BindingPos Pos
	Body       []Stmt
}

func (*RecoverExpr) exprNode()       {}
func (e *RecoverExpr) Position() Pos { return e.Pos }

type BlockExpr struct {
	Pos  Pos
	Body []Stmt
}

func (*BlockExpr) exprNode()       {}
func (e *BlockExpr) Position() Pos { return e.Pos }

type TypeParam struct {
	Pos        Pos
	Name       string
	Constraint *TypeRef
}

type InterfaceDecl struct {
	Pos        Pos
	NamePos    Pos
	Name       string
	TypeParams []TypeParam
	Methods    []*FuncDecl
	Embeds     []TypeRef
}

func (*InterfaceDecl) declNode()       {}
func (d *InterfaceDecl) Position() Pos { return d.Pos }

type InstantiateExpr struct {
	Pos  Pos
	Name string
	Args []TypeRef
}

func (*InstantiateExpr) exprNode()       {}
func (e *InstantiateExpr) Position() Pos { return e.Pos }

type AssertExpr struct {
	Pos    Pos
	Value  Expr
	Target TypeRef
}

func (*AssertExpr) exprNode()       {}
func (e *AssertExpr) Position() Pos { return e.Pos }
