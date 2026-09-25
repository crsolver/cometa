package codegen

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

// Analyze the lowered program so source bindings and evaluation-order
// temporaries follow the same rules. Parser objects distinguish lexical scopes,
// including shadowing. Repeat because removing a binding can make its inputs dead.
func eliminateUnusedLocals(source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for {
		reads := map[*ast.Object]int{}
		locals := map[*ast.Object]bool{}
		ignored := map[*ast.Ident]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.DeclStmt:
				if decl, ok := n.Decl.(*ast.GenDecl); ok && decl.Tok == token.VAR {
					for _, spec := range decl.Specs {
						for _, id := range spec.(*ast.ValueSpec).Names {
							locals[id.Obj] = true
						}
					}
				}
			case *ast.ValueSpec:
				for _, id := range n.Names {
					ignored[id] = true
				}
			case *ast.Field:
				for _, id := range n.Names {
					ignored[id] = true
				}
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					if id, ok := lhs.(*ast.Ident); ok {
						if n.Tok == token.DEFINE {
							locals[id.Obj] = true
						}
						if n.Tok == token.DEFINE || n.Tok == token.ASSIGN {
							ignored[id] = true
						}
					}
				}
				if discardIdentifier(n) {
					ignored[n.Rhs[0].(*ast.Ident)] = true
				}
			case *ast.RangeStmt:
				if id, ok := n.Key.(*ast.Ident); ok {
					ignored[id] = true
					if n.Tok == token.DEFINE {
						locals[id.Obj] = true
					}
				}
				if id, ok := n.Value.(*ast.Ident); ok {
					ignored[id] = true
					if n.Tok == token.DEFINE {
						locals[id.Obj] = true
					}
				}
			case *ast.Ident:
				if n.Obj != nil && !ignored[n] {
					reads[n.Obj]++
				}
			}
			return true
		})
		changed := false
		unused := func(id *ast.Ident) bool { return id.Obj != nil && locals[id.Obj] && reads[id.Obj] == 0 }
		var rewrite func(ast.Stmt) ast.Stmt
		rewrite = func(stmt ast.Stmt) ast.Stmt {
			switch s := stmt.(type) {
			case *ast.AssignStmt:
				if discardIdentifier(s) {
					changed = true
					return nil
				}
				if len(s.Lhs) > 1 && (s.Tok == token.DEFINE || s.Tok == token.ASSIGN) {
					hasNew := false
					for i, lhs := range s.Lhs {
						if id, ok := lhs.(*ast.Ident); ok && unused(id) {
							s.Lhs[i] = ast.NewIdent("_")
							changed = true
						}
						id, ok := s.Lhs[i].(*ast.Ident)
						hasNew = hasNew || (ok && id.Name != "_" && id.Obj != nil && id.Obj.Decl == s)
					}
					if s.Tok == token.DEFINE && !hasNew {
						s.Tok = token.ASSIGN
					}
				}
				// Lowering emits single-result declarations and assignments.
				if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
					if id, ok := s.Lhs[0].(*ast.Ident); ok && (unused(id) || id.Name == "_") {
						if id.Name != "_" {
							changed = true
							s.Lhs[0] = ast.NewIdent("_")
							s.Tok = token.ASSIGN
						}
						if discardable(s.Rhs[0]) {
							changed = true
							return nil
						}
					}
				}
			case *ast.DeclStmt:
				decl, ok := s.Decl.(*ast.GenDecl)
				if !ok || decl.Tok != token.VAR || len(decl.Specs) != 1 {
					break
				}
				v := decl.Specs[0].(*ast.ValueSpec)
				if len(v.Names) != 1 || !unused(v.Names[0]) {
					break
				}
				changed = true
				if len(v.Values) == 0 || discardable(v.Values[0]) {
					return nil
				}
				// Retain the declared type: untyped constants and interface
				// conversions must keep their original Go typing context.
				v.Names[0] = ast.NewIdent("_")
			case *ast.RangeStmt:
				if id, ok := s.Key.(*ast.Ident); ok && unused(id) {
					s.Key = ast.NewIdent("_")
					changed = true
				}
				if id, ok := s.Value.(*ast.Ident); ok && unused(id) {
					s.Value = nil
					changed = true
				}
				if id, ok := s.Key.(*ast.Ident); ok && id.Name == "_" && s.Value == nil {
					s.Key = nil
					s.Tok = token.ILLEGAL
					changed = true
				}
			case *ast.TypeSwitchStmt:
				if a, ok := s.Assign.(*ast.AssignStmt); ok && unused(a.Lhs[0].(*ast.Ident)) {
					s.Assign = &ast.ExprStmt{X: a.Rhs[0]}
					changed = true
				}
			}
			return stmt
		}
		// Statement-list edits preserve the object identities used by this
		// iteration's usage counts; later iterations remove newly dead inputs.
		ast.Inspect(file, func(n ast.Node) bool {
			var list *[]ast.Stmt
			switch n := n.(type) {
			case *ast.BlockStmt:
				list = &n.List
			case *ast.CaseClause:
				list = &n.Body
			case *ast.CommClause:
				list = &n.Body
			case *ast.IfStmt:
				if n.Init != nil {
					n.Init = rewrite(n.Init)
				}
			case *ast.ForStmt:
				if n.Init != nil {
					n.Init = rewrite(n.Init)
				}
				if n.Post != nil {
					n.Post = rewrite(n.Post)
				}
			case *ast.SwitchStmt:
				if n.Init != nil {
					n.Init = rewrite(n.Init)
				}
			case *ast.TypeSwitchStmt:
				if n.Init != nil {
					n.Init = rewrite(n.Init)
				}
			case *ast.LabeledStmt:
				if next := rewrite(n.Stmt); next != nil {
					n.Stmt = next
				} else {
					n.Stmt = &ast.EmptyStmt{}
				}
			}
			if list != nil {
				kept := (*list)[:0]
				for _, stmt := range *list {
					if next := rewrite(stmt); next != nil {
						kept = append(kept, next)
					}
				}
				*list = kept
			}
			return true
		})
		if !changed {
			break
		}
	}
	var output bytes.Buffer
	err = format.Node(&output, fset, file)
	return output.Bytes(), err
}

func discardIdentifier(s *ast.AssignStmt) bool {
	if s.Tok != token.ASSIGN || len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return false
	}
	lhs, ok := s.Lhs[0].(*ast.Ident)
	if !ok || lhs.Name != "_" {
		return false
	}
	_, ok = s.Rhs[0].(*ast.Ident)
	return ok
}

// Be conservative: calls, indexing, dereferences, assertions and arithmetic
// can have effects or panic even when their result is unused.
func discardable(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return discardable(e.X)
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && id.Obj == nil && len(e.Args) == 1 {
			switch id.Name {
			case "float64", "int64", "int", "bool", "string":
				return discardable(e.Args[0])
			}
		}
	case *ast.CompositeLit:
		for _, element := range e.Elts {
			if kv, ok := element.(*ast.KeyValueExpr); ok {
				if _, field := kv.Key.(*ast.Ident); !field && !discardable(kv.Key) {
					return false
				}
				if !discardable(kv.Value) {
					return false
				}
			} else if !discardable(element) {
				return false
			}
		}
		return true
	case *ast.UnaryExpr:
		return e.Op == token.AND && discardable(e.X)
	}
	return false
}
