package spec

import (
	"gopkg.in/yaml.v3"
)

const (
	PointEntry     = "entry"
	PointExit      = "exit"
	PointException = "exception"

	Constructor      = "new"
	constructorAlias = "<init>"
)

var (
	whereKeys    = []string{"class", "subtypes", "method", "params", "at", "line", "call"}
	methodPoints = []string{PointEntry, PointExit, PointException}
	ctorPoints   = []string{PointEntry, PointExit}
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
	for _, key := range []string{"line", "call"} {
		if values[key] != nil {
			c.add(values[key], "probe %q: %q probes are not implemented yet", id, key)
		}
	}
	if params := values["params"]; params != nil {
		w.Params, w.HasParam = c.stringList(params, id, "params"), true
	}
	w.At = c.points(values["at"], id, w.Method == Constructor)
	return w
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

func (c *checker) points(n *yaml.Node, id string, ctor bool) []string {
	allowed := methodPoints
	if ctor {
		allowed = ctorPoints
	}
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
