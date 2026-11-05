package spec

import (
	"gopkg.in/yaml.v3"
)

func exprOf(n *yaml.Node) Expr {
	return Expr{
		Text:   n.Value,
		Line:   n.Line,
		Col:    n.Column,
		Quoted: n.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle) != 0,
		Block:  n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0,
	}
}

func (c *checker) when(n *yaml.Node, id string) *Expr {
	if n == nil {
		return nil
	}
	if n.Kind != yaml.ScalarNode || n.Value == "" {
		c.add(n, "probe %q: \"when\" must be a CEL expression", id)
		return nil
	}
	e := exprOf(n)
	return &e
}

func (c *checker) what(n *yaml.Node, id string) []Item {
	if n == nil {
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		c.add(n, "probe %q: \"what\" must be a list", id)
		return nil
	}
	seen := map[string]bool{}
	items := make([]Item, 0, len(n.Content))
	for _, node := range n.Content {
		item, ok := c.item(node, id)
		if !ok {
			continue
		}
		if seen[item.Key] {
			c.add(node, "probe %q: duplicate what key %q", id, item.Key)
			continue
		}
		seen[item.Key] = true
		items = append(items, item)
	}
	return items
}

func (c *checker) item(n *yaml.Node, id string) (Item, bool) {
	switch {
	case n.Kind == yaml.ScalarNode && n.Value != "":
		return Item{Key: n.Value, Expr: exprOf(n)}, true
	case n.Kind == yaml.MappingNode && len(n.Content) == 2 && n.Content[0].Value != "" &&
		n.Content[1].Kind == yaml.ScalarNode && n.Content[1].Value != "":
		return Item{Key: n.Content[0].Value, Expr: exprOf(n.Content[1])}, true
	}
	c.add(n, "probe %q: a what item is an expression or a one-entry map name: expression", id)
	return Item{}, false
}
