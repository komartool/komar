package celcheck

import (
	"fmt"
	"komar/internal/spec"
	"regexp"
	"strings"

	"cel.dev/cel-go/cel"
)

const (
	RootThis      = "this"
	RootArgs      = "args"
	RootVars      = "vars"
	RootRet       = "ret"
	RootException = "exception"
	RootDuration  = "duration"
)

var (
	roots      = []string{RootThis, RootArgs, RootVars, RootRet, RootException, RootDuration}
	undeclared = regexp.MustCompile(`undeclared reference to '([A-Za-z_][A-Za-z0-9_]*)'`)
)

type Result struct {
	Errors   []spec.Problem
	Warnings []spec.Problem
}

type checker struct {
	env      *cel.Env
	builtins map[string]bool
	res      Result
}

func Check(s *spec.Spec) (Result, error) {
	opts := make([]cel.EnvOption, 0, len(roots))
	for _, r := range roots {
		opts = append(opts, cel.Variable(r, cel.DynType))
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return Result{}, fmt.Errorf("create CEL environment: %w", err)
	}
	c := &checker{env: env, builtins: map[string]bool{}}
	for name := range env.Functions() {
		c.builtins[name] = true
	}
	for i := range s.Probes {
		c.probe(&s.Probes[i])
	}
	return c.res, nil
}

func (c *checker) probe(p *spec.Probe) {
	for i := range p.What {
		c.expr(p.ID, fmt.Sprintf("what %q", p.What[i].Key), &p.What[i].Expr)
	}
	if p.When == nil {
		return
	}
	ast, ok := c.expr(p.ID, "when", p.When)
	if !ok {
		return
	}
	t := ast.OutputType()
	if !t.IsExactType(cel.BoolType) && !t.IsExactType(cel.DynType) {
		c.fail(p.ID, p.When, 0, 0, fmt.Sprintf("when must be a boolean condition, got %s", t))
		return
	}
	c.neverFires(p)
}

func (c *checker) neverFires(p *spec.Probe) {
	ctor := p.Where.Method == spec.Constructor
	for _, point := range p.Where.At {
		for _, root := range p.When.Roots {
			if !Available(root, point, ctor) {
				c.res.Warnings = append(c.res.Warnings, spec.Problem{
					Line: p.When.Line, Col: p.When.Col, Probe: p.ID,
					Msg: fmt.Sprintf("probe %q: when uses %s, not available at %s; %s will never fire", p.ID, root, point, point),
				})
				break
			}
		}
	}
}

func Available(root, point string, ctor bool) bool {
	switch root {
	case RootRet:
		return point == spec.PointExit
	case RootException:
		return point == spec.PointException
	case RootDuration:
		return point == spec.PointExit || point == spec.PointException
	case RootThis:
		return !ctor || point != spec.PointEntry
	}
	return true
}

func (c *checker) expr(id, label string, e *spec.Expr) (*cel.Ast, bool) {
	parsed, iss := c.env.Parse(e.Text)
	if iss.Err() != nil {
		c.issues(id, label, e, iss)
		return nil, false
	}
	r := collect(parsed.NativeRep().Expr(), c.builtins)
	for _, m := range r.methods {
		c.fail(id, e, 0, 0, fmt.Sprintf("%s: %s() calls a program method; method calls are not implemented yet, read the field instead", label, m))
	}
	if len(r.methods) > 0 {
		return nil, false
	}
	checked, iss := c.env.Check(parsed)
	if iss.Err() != nil {
		c.issues(id, label, e, iss)
		return nil, false
	}
	e.Roots, e.Vars = r.rootList(), r.varList()
	return checked, true
}

func (c *checker) issues(id, label string, e *spec.Expr, iss *cel.Issues) {
	for _, err := range iss.Errors() {
		msg := strings.TrimSuffix(err.Message, " (in container '')")
		if m := undeclared.FindStringSubmatch(msg); m != nil {
			msg += fmt.Sprintf("; program names are under vars, e.g. vars.%s", m[1])
		}
		c.fail(id, e, err.Location.Line(), err.Location.Column(), label+": "+msg)
	}
}

func (c *checker) fail(id string, e *spec.Expr, line, col int, msg string) {
	p := spec.Problem{Line: e.Line, Col: e.Col, Probe: id, Msg: fmt.Sprintf("probe %q: %s", id, msg)}
	if line > 0 {
		p.Line, p.Col = position(e, line, col)
	}
	c.res.Errors = append(c.res.Errors, p)
}

func position(e *spec.Expr, line, col int) (int, int) {
	switch {
	case e.Block:
		return e.Line + line, col + 1
	case line > 1:
		return e.Line + line - 1, col + 1
	case e.Quoted:
		return e.Line, e.Col + col + 1
	}
	return e.Line, e.Col + col
}
