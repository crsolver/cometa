package sema

import "cometa/internal/ast"

// An edge tracks how a declaration's type parameter flows into another's.
// A cycle containing a constructor would require infinitely many Go instances.
type typeEdge struct {
	from, to string
	grows    bool
	pos      ast.Pos
}

func (c *checker) recordExpansion(params, args []Type, pos ast.Pos) {
	var visit func(Type, string, bool)
	visit = func(t Type, target string, grows bool) {
		if t.Kind == TypeParameter {
			c.expansions = append(c.expansions, typeEdge{paramKey(t), target, grows, pos})
			return
		}
		if t.Elem != nil {
			visit(*t.Elem, target, true)
		}
		if t.Key != nil {
			visit(*t.Key, target, true)
		}
		if t.Err != nil {
			visit(*t.Err, target, true)
		}
		for _, a := range t.Args {
			visit(a, target, true)
		}
	}
	for i, p := range params {
		if i < len(args) {
			visit(args[i], paramKey(p), false)
		}
	}
}

func (c *checker) checkExpansions() error {
	graph := map[string][]string{}
	for _, e := range c.expansions {
		graph[e.from] = append(graph[e.from], e.to)
	}
	for _, e := range c.expansions {
		if !e.grows {
			continue
		}
		seen := map[string]bool{}
		work := []string{e.to}
		for len(work) > 0 {
			n := work[len(work)-1]
			work = work[:len(work)-1]
			if n == e.from {
				return c.fail(e.pos, "instanciación genérica recursiva que expande sus argumentos de tipo")
			}
			if !seen[n] {
				seen[n] = true
				work = append(work, graph[n]...)
			}
		}
	}
	return nil
}
