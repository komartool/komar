package spec

import (
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	PointEntry     = "entry"
	PointExit      = "exit"
	PointException = "exception"
	PointLine      = "line"
	PointBefore    = "before"
	PointAfter     = "after"

	Constructor      = "new"
	constructorAlias = "<init>"
)

var (
	whereKeys    = []string{"class", "subtypes", "method", "params", "at", "line", "call"}
	callKeys     = []string{"class", "subtypes", "method", "params"}
	methodPoints = []string{PointEntry, PointExit, PointException}
	ctorPoints   = []string{PointEntry, PointExit}
	callPoints   = []string{PointBefore, PointAfter, PointException}
)

func (c *checker) where(n *yaml.Node, id string) Where {
	if n.Kind != yaml.MappingNode {
		c.add(n, "probe %q: \"where\" must be a map", id)
		return Where{}
	}
	values := c.mapping(n, whereKeys, " in where")
	w := Where{
		Class:    c.requiredString(n, values["class"], id, "class"),
		Subtypes: c.boolean(values["subtypes"], "subtypes"),
		Method:   c.requiredString(n, values["method"], id, "method"),
	}
	if w.Method == constructorAlias {
		w.Method = Constructor
	}
	if params := values["params"]; params != nil {
		w.Params, w.HasParam = c.stringList(params, id, "params"), true
	}
	if values["line"] != nil && values["call"] != nil {
		c.add(values["call"], "probe %q: a probe has one kind; use \"line\" or \"call\", not both", id)
		return w
	}
	if call := values["call"]; call != nil {
		w.Call = c.callTarget(call, id)
		w.At = c.points(values["at"], id, callPoints, false)
		return w
	}
	if line := values["line"]; line != nil {
		w.Lines = c.lines(line, id)
		w.At = []string{PointLine}
		if at := values["at"]; at != nil {
			c.add(at, "probe %q: \"at\" is not allowed on a line probe; it fires at each listed line", id)
		}
		return w
	}
	allowed := methodPoints
	if w.Method == Constructor {
		allowed = ctorPoints
	}
	w.At = c.points(values["at"], id, allowed, w.Method == Constructor)
	return w
}

func (c *checker) callTarget(n *yaml.Node, id string) *CallTarget {
	if n.Kind != yaml.MappingNode {
		c.add(n, "probe %q: \"call\" must be a map with class and method", id)
		return nil
	}
	values := c.mapping(n, callKeys, " in call")
	t := &CallTarget{
		Class:    c.requiredString(n, values["class"], id, "class"),
		Subtypes: c.boolean(values["subtypes"], "subtypes"),
		Method:   c.requiredString(n, values["method"], id, "method"),
	}
	if t.Method == Constructor || t.Method == constructorAlias {
		c.add(values["method"], "probe %q: calls to constructors are not implemented yet", id)
	}
	if params := values["params"]; params != nil {
		t.Params, t.HasParam = c.stringList(params, id, "params"), true
	}
	return t
}

func (c *checker) lines(n *yaml.Node, id string) []int {
	items := n.Content
	if n.Kind == yaml.ScalarNode {
		items = []*yaml.Node{n}
	} else if n.Kind != yaml.SequenceNode || len(items) == 0 {
		c.add(n, "probe %q: \"line\" must be a line number or a non-empty list of line numbers", id)
		return nil
	}
	var out []int
	for _, item := range items {
		v, err := strconv.Atoi(item.Value)
		if item.Kind != yaml.ScalarNode || err != nil || v < 1 {
			c.add(item, "probe %q: line %q must be a positive integer", id, item.Value)
			continue
		}
		out = append(out, v)
	}
	return out
}

func (c *checker) requiredString(parent, n *yaml.Node, id, key string) string {
	if n == nil {
		c.add(parent, "probe %q: where has no %q", id, key)
		return ""
	}
	if n.Kind != yaml.ScalarNode || n.Value == "" {
		c.add(n, "probe %q: %q must be a non-empty string", id, key)
		return ""
	}
	return n.Value
}

func (c *checker) stringList(n *yaml.Node, id, key string) []string {
	if n.Kind != yaml.SequenceNode {
		c.add(n, "probe %q: %q must be a list", id, key)
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode || item.Value == "" {
			c.add(item, "probe %q: %q items must be non-empty strings", id, key)
			continue
		}
		out = append(out, item.Value)
	}
	return out
}

func (c *checker) points(n *yaml.Node, id string, allowed []string, ctor bool) []string {
	if n == nil {
		return append([]string(nil), allowed...)
	}
	items := n.Content
	if n.Kind == yaml.ScalarNode {
		items = []*yaml.Node{n}
	} else if n.Kind != yaml.SequenceNode {
		c.add(n, "probe %q: \"at\" must be a list of points", id)
		return nil
	}
	if len(items) == 0 {
		c.add(n, "probe %q: \"at\" is empty", id)
		return nil
	}
	return c.pointList(items, allowed, id, ctor)
}

func (c *checker) pointList(items []*yaml.Node, allowed []string, id string, ctor bool) []string {
	var at []string
	for _, item := range items {
		switch {
		case ctor && item.Value == PointException:
			c.add(item, "probe %q: \"exception\" on constructors is not implemented yet", id)
		case !contains(allowed, item.Value):
			c.add(item, "probe %q: unknown point %q; allowed: %v%s", id, item.Value, allowed, suggestion(item.Value, allowed))
		case !contains(at, item.Value):
			at = append(at, item.Value)
		}
	}
	return at
}
