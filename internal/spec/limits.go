package spec

import (
	"strconv"

	"gopkg.in/yaml.v3"
)

type Limits struct {
	Depth   int `json:"depth"`
	Items   int `json:"items"`
	String  int `json:"string"`
	Fields  int `json:"fields"`
	MaxHits int `json:"max_hits"`
	Classes int `json:"classes"`
	CallMs  int `json:"call_ms"`
}

const (
	defaultDepth   = 3
	defaultItems   = 100
	defaultString  = 1000
	defaultFields  = 50
	defaultMaxHits = 1000
	defaultClasses = 50
	defaultCallMs  = 50
)

func DefaultLimits() Limits {
	return Limits{
		Depth:   defaultDepth,
		Items:   defaultItems,
		String:  defaultString,
		Fields:  defaultFields,
		MaxHits: defaultMaxHits,
		Classes: defaultClasses,
		CallMs:  defaultCallMs,
	}
}

var misplacedLimitKeys = map[string]string{"hits": `"hits" belongs in "stop"; the per-probe cap is "max_hits"`}

func (c *checker) limits(n *yaml.Node, base Limits) Limits {
	if n == nil {
		return base
	}
	if n.Kind != yaml.MappingNode {
		c.add(n, `"limits" must be a map`)
		return base
	}
	out := base
	fields := map[string]*int{
		"depth": &out.Depth, "items": &out.Items, "string": &out.String, "fields": &out.Fields,
		"max_hits": &out.MaxHits, "classes": &out.Classes, "call_ms": &out.CallMs,
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		target, ok := fields[key.Value]
		if !ok {
			c.unknownLimit(key)
			continue
		}
		v, err := strconv.Atoi(value.Value)
		if value.Kind != yaml.ScalarNode || err != nil || v < 1 {
			c.add(value, "limit %q must be a positive integer", key.Value)
			continue
		}
		*target = v
	}
	return out
}

func (c *checker) unknownLimit(key *yaml.Node) {
	if hint, ok := misplacedLimitKeys[key.Value]; ok {
		c.add(key, "unknown limit %q: %s", key.Value, hint)
		return
	}
	known := []string{"depth", "items", "string", "fields", "max_hits", "classes", "call_ms"}
	c.add(key, "unknown limit %q%s", key.Value, suggestion(key.Value, known))
}
