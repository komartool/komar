package celcheck

import (
	"sort"

	"cel.dev/cel-go/common/ast"
)

type refs struct {
	builtins map[string]bool
	roots    map[string]bool
	vars     map[string]bool
	methods  []string
}

func collect(e ast.Expr, builtins map[string]bool) *refs {
	r := &refs{builtins: builtins, roots: map[string]bool{}, vars: map[string]bool{}}
	r.walk(e, map[string]bool{})
	return r
}

func (r *refs) walk(e ast.Expr, scope map[string]bool) {
	switch e.Kind() {
	case ast.IdentKind:
		if !scope[e.AsIdent()] {
			r.roots[e.AsIdent()] = true
		}
	case ast.SelectKind:
		r.selection(e.AsSelect(), scope)
	case ast.CallKind:
		r.call(e.AsCall(), scope)
	case ast.ListKind:
		r.all(e.AsList().Elements(), scope)
	case ast.MapKind:
		for _, entry := range e.AsMap().Entries() {
			r.walk(entry.AsMapEntry().Key(), scope)
			r.walk(entry.AsMapEntry().Value(), scope)
		}
	case ast.StructKind:
		for _, field := range e.AsStruct().Fields() {
			r.walk(field.AsStructField().Value(), scope)
		}
	case ast.ComprehensionKind:
		r.comprehension(e.AsComprehension(), scope)
	case ast.LiteralKind, ast.UnspecifiedExprKind:
	}
}

func (r *refs) selection(s ast.SelectExpr, scope map[string]bool) {
	op := s.Operand()
	if op.Kind() == ast.IdentKind && op.AsIdent() == RootVars && !scope[RootVars] {
		r.vars[s.FieldName()] = true
	}
	r.walk(op, scope)
}

func (r *refs) call(c ast.CallExpr, scope map[string]bool) {
	if c.IsMemberFunction() {
		if !r.builtins[c.FunctionName()] {
			r.methods = append(r.methods, c.FunctionName())
		}
		r.walk(c.Target(), scope)
	}
	r.all(c.Args(), scope)
}

func (r *refs) comprehension(c ast.ComprehensionExpr, scope map[string]bool) {
	r.walk(c.IterRange(), scope)
	r.walk(c.AccuInit(), scope)
	inner := make(map[string]bool, len(scope)+3)
	for k := range scope {
		inner[k] = true
	}
	inner[c.IterVar()] = true
	inner[c.AccuVar()] = true
	if c.HasIterVar2() {
		inner[c.IterVar2()] = true
	}
	r.walk(c.LoopCondition(), inner)
	r.walk(c.LoopStep(), inner)
	r.walk(c.Result(), inner)
}

func (r *refs) all(list []ast.Expr, scope map[string]bool) {
	for _, e := range list {
		r.walk(e, scope)
	}
}

func (r *refs) rootList() []string {
	return sorted(r.roots)
}

func (r *refs) varList() []string {
	return sorted(r.vars)
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
