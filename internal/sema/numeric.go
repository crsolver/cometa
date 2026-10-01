package sema

import "github.com/crsolver/cometa/internal/ast"

func (t Type) Numeric() bool { return t.Kind == Integer || t.Kind == Decimal }

// promoteNumeric records a single scalar widening at the expression boundary.
// Containers and interface dynamic types never participate in this operation.
func (c *checker) promoteNumeric(expr ast.Expr) {
	if c.model.ExprTypes[expr].Kind == Integer {
		t := Type{Kind: Decimal}
		c.model.NumericCoercions[expr] = t
		c.model.ExprTypes[expr] = t
	}
}

func (c *checker) promoteBlock(body []ast.Stmt) {
	if len(body) == 0 {
		return
	}
	switch s := body[len(body)-1].(type) {
	case *ast.ExprStmt:
		c.promoteNumeric(s.Expr)
	case *ast.IfStmt:
		for _, b := range s.Branches {
			c.promoteBlock(b.Body)
		}
		c.promoteBlock(s.Else)
	case *ast.MatchStmt:
		for _, a := range s.Match.Arms {
			c.promoteBlock(a.Body)
		}
		c.model.ExprTypes[s.Match] = Type{Kind: Decimal}
	}
}
