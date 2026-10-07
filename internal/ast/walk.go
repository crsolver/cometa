package ast

// Inspect traverses the syntax under root in source order, parents first.
//
// root is a *Program, a declaration, a statement, an expression or a list of
// statements. visit receives every declaration, statement and expression, and
// also each *Field, *MatchArm and *TypeRef (nested type references included).
// When visit returns false the children of that node are skipped.
//
// Only syntax is followed, never semantic links such as the declaration a
// call resolves to. Every node type must be handled here: a type that is
// missing would silently hide its children from every analysis built on
// Inspect, so TestInspectVisitsEveryNode fails until it is added.
func Inspect(root any, visit func(node any) bool) {
	w := walker{visit: visit}
	switch root := root.(type) {
	case nil:
	case *Program:
		if root != nil {
			for _, decl := range root.Decls {
				w.decl(decl)
			}
		}
	case []Stmt:
		w.stmts(root)
	case Decl:
		w.decl(root)
	case Stmt:
		w.stmt(root)
	case Expr:
		w.expr(root)
	case *Field:
		w.field(root)
	case *TypeRef:
		w.typeRef(root)
	default:
		panic("ast.Inspect: unsupported root")
	}
}

type walker struct {
	visit func(any) bool
}

func (w walker) typeParams(params []TypeParam) {
	for i := range params {
		w.typeRef(params[i].Constraint)
	}
}

func (w walker) typeRef(t *TypeRef) {
	if t == nil || !w.visit(t) {
		return
	}
	w.typeRef(t.Key)
	for i := range t.Args {
		w.typeRef(&t.Args[i])
	}
	w.typeRef(t.Payload)
	w.typeRef(t.ErrorType)
	w.typeRef(t.Element)
}

func (w walker) field(f *Field) {
	if f == nil || !w.visit(f) {
		return
	}
	w.expr(f.Default)
	w.typeRef(&f.Type)
}

func (w walker) function(d *FuncDecl) {
	if d == nil || !w.visit(d) {
		return
	}
	w.typeParams(d.TypeParams)
	for i := range d.Params {
		w.expr(d.Params[i].Default)
		w.typeRef(&d.Params[i].Type)
	}
	w.typeRef(d.ReturnType)
	w.stmts(d.Body)
}

func (w walker) decl(decl Decl) {
	switch d := decl.(type) {
	case nil:
	case *FuncDecl:
		w.function(d)
	case *TypeDecl:
		if d == nil || !w.visit(d) {
			return
		}
		w.typeParams(d.TypeParams)
		for _, field := range d.Fields {
			w.field(field)
		}
		for _, method := range d.Methods {
			w.function(method)
		}
	case *EnumDecl:
		if d == nil || !w.visit(d) {
			return
		}
		w.typeParams(d.TypeParams)
		for _, variant := range d.Variants {
			if variant != nil {
				w.typeRef(variant.Payload)
			}
		}
	case *InterfaceDecl:
		if d == nil || !w.visit(d) {
			return
		}
		w.typeParams(d.TypeParams)
		for _, method := range d.Methods {
			w.function(method)
		}
		for i := range d.Embeds {
			w.typeRef(&d.Embeds[i])
		}
	case *GlobalDecl:
		if d == nil || !w.visit(d) {
			return
		}
		w.typeRef(d.Type)
		w.expr(d.Value)
	default:
		panic("ast.Inspect: unknown declaration")
	}
}

func (w walker) stmts(body []Stmt) {
	for _, stmt := range body {
		w.stmt(stmt)
	}
}

func (w walker) stmt(stmt Stmt) {
	switch s := stmt.(type) {
	case nil:
	case *BadStmt:
		if s != nil {
			w.visit(s)
		}
	case *ContinueStmt:
		if s != nil {
			w.visit(s)
		}
	case *BreakStmt:
		if s != nil {
			w.visit(s)
		}
	case *MatchStmt:
		if s == nil || !w.visit(s) {
			return
		}
		if s.Match != nil {
			w.expr(s.Match)
		}
	case *ExprStmt:
		if s == nil || !w.visit(s) {
			return
		}
		w.expr(s.Expr)
	case *AssignStmt:
		if s == nil || !w.visit(s) {
			return
		}
		w.expr(s.Target)
		w.expr(s.Value)
	case *VarDeclStmt:
		if s == nil || !w.visit(s) {
			return
		}
		w.typeRef(s.Type)
		w.expr(s.Value)
	case *IfStmt:
		if s == nil || !w.visit(s) {
			return
		}
		for i := range s.Branches {
			w.expr(s.Branches[i].Condition)
			w.stmts(s.Branches[i].Body)
		}
		w.stmts(s.Else)
	case *ScopeStmt:
		if s == nil || !w.visit(s) {
			return
		}
		w.expr(s.Value)
		w.stmts(s.Body)
	case *RepeatStmt:
		if s == nil || !w.visit(s) {
			return
		}
		w.expr(s.Iterable)
		w.expr(s.RangeEnd)
		w.stmts(s.Body)
	default:
		panic("ast.Inspect: unknown statement")
	}
}

func (w walker) exprs(list []Expr) {
	for _, expr := range list {
		w.expr(expr)
	}
}

func (w walker) expr(expr Expr) {
	switch e := expr.(type) {
	case nil:
	case *IdentExpr:
		if e != nil {
			w.visit(e)
		}
	case *ReceiverExpr:
		if e != nil {
			w.visit(e)
		}
	case *LiteralExpr:
		if e != nil {
			w.visit(e)
		}
	case *ContextualVariantExpr:
		if e != nil {
			w.visit(e)
		}
	case *MatchExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
		for _, arm := range e.Arms {
			if arm == nil || !w.visit(arm) {
				continue
			}
			w.typeRef(arm.TypePattern)
			w.typeRef(arm.QualifierType)
			w.exprs(arm.Literals)
			w.stmts(arm.Body)
		}
	case *MapLiteralExpr:
		if e == nil || !w.visit(e) {
			return
		}
		for _, entry := range e.Entries {
			w.expr(entry.Key)
			w.expr(entry.Value)
		}
	case *RangeLabel:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Start)
		w.expr(e.End)
	case *InterpolatedStringExpr:
		if e == nil || !w.visit(e) {
			return
		}
		for _, part := range e.Parts {
			w.expr(part.Expr)
		}
	case *UnaryExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
	case *BinaryExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Left)
		w.expr(e.Right)
	case *CallExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Callee)
		w.exprs(e.Args)
	case *MemberExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Object)
	case *IndexExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Object)
		w.expr(e.Index)
	case *StructLiteralExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.typeRef(e.Type)
		for _, field := range e.Fields {
			w.expr(field.Value)
		}
		w.exprs(e.Values)
	case *ListLiteralExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.exprs(e.Elements)
	case *IfExpr:
		if e == nil || !w.visit(e) {
			return
		}
		for _, branch := range e.Branches {
			w.expr(branch.Condition)
			w.expr(branch.Value)
		}
		w.expr(e.Else)
	case *ReturnExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
	case *TryExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
	case *RecoverExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
		w.stmts(e.Body)
	case *BlockExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.stmts(e.Body)
	case *InstantiateExpr:
		if e == nil || !w.visit(e) {
			return
		}
		for i := range e.Args {
			w.typeRef(&e.Args[i])
		}
	case *AssertExpr:
		if e == nil || !w.visit(e) {
			return
		}
		w.expr(e.Value)
		w.typeRef(&e.Target)
	default:
		panic("ast.Inspect: unknown expression")
	}
}
