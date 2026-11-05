package spec

import (
	"gopkg.in/yaml.v3"
)

var builtins = []string{"args", "this", "ret", "exception", "duration"}

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
	var item Item
	exprNode := n
	switch {
	case n.Kind == yaml.ScalarNode:
		item = Item{Key: n.Value, Expr: n.Value}
	case n.Kind == yaml.MappingNode && len(n.Content) == 2 && n.Content[1].Kind == yaml.ScalarNode:
		item = Item{Key: n.Content[0].Value, Expr: n.Content[1].Value}
		exprNode = n.Content[1]
	default:
		c.add(n, "probe %q: a what item is an expression or a one-entry map name: expression", id)
		return Item{}, false
	}
	if !contains(builtins, item.Expr) {
		c.add(exprNode, "probe %q: what %q: only %v are implemented yet%s", id, item.Expr, builtins, suggestion(item.Expr, builtins))
		return Item{}, false
	}
	return item, true
}
