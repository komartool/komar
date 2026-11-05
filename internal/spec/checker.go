package spec

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

var (
	topKeys   = []string{"version", "calls", "limits", "stop", "jvm", "probes"}
	probeKeys = []string{"id", "where", "when", "what", "limits"}
)

const supportedVersion = 1

type checker struct {
	problems []Problem
	warnings []Problem
}

func (c *checker) add(n *yaml.Node, format string, args ...any) {
	c.problems = append(c.problems, Problem{Line: n.Line, Col: n.Column, Msg: fmt.Sprintf(format, args...)})
}

func (c *checker) warn(n *yaml.Node, format string, args ...any) {
	c.warnings = append(c.warnings, Problem{Line: n.Line, Col: n.Column, Msg: fmt.Sprintf(format, args...)})
}

func (c *checker) document(doc *yaml.Node) *Spec {
	s := &Spec{Limits: DefaultLimits()}
	if len(doc.Content) == 0 {
		c.problems = append(c.problems, Problem{Line: 1, Col: 1, Msg: `spec is empty; it needs a "probes" list`})
		return s
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		c.add(root, `spec must be a map with a "probes" list`)
		return s
	}
	top := c.mapping(root, topKeys, "")
	c.version(top["version"])
	s.Calls = c.boolean(top["calls"], "calls")
	if s.Calls {
		c.warn(top["calls"], `"calls: true" is not implemented yet; method calls in expressions are rejected`)
	}
	s.Limits = c.limits(top["limits"], s.Limits)
	s.Stop = c.stop(top["stop"])
	c.notYet(top["jvm"], "jvm")
	if top["probes"] == nil {
		c.add(root, `missing required key "probes"`)
		return s
	}
	s.Probes = c.probes(top["probes"], s.Limits)
	return s
}

func (c *checker) version(n *yaml.Node) {
	if n == nil {
		return
	}
	if v, err := strconv.Atoi(n.Value); err != nil || v != supportedVersion {
		c.add(n, "version %q is not supported; this komar reads version %d", n.Value, supportedVersion)
	}
}

func (c *checker) boolean(n *yaml.Node, key string) bool {
	if n == nil {
		return false
	}
	v, err := strconv.ParseBool(n.Value)
	if n.Kind != yaml.ScalarNode || err != nil {
		c.add(n, "%q must be true or false", key)
	}
	return v
}

func (c *checker) notYet(n *yaml.Node, key string) {
	if n != nil {
		c.warn(n, "%q is not implemented yet and is ignored", key)
	}
}

func (c *checker) mapping(n *yaml.Node, known []string, where string) map[string]*yaml.Node {
	values := map[string]*yaml.Node{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if !contains(known, key.Value) {
			c.add(key, "unknown key %q%s%s", key.Value, where, suggestion(key.Value, known))
			continue
		}
		values[key.Value] = value
	}
	return values
}

func (c *checker) probes(n *yaml.Node, limits Limits) []Probe {
	if n.Kind != yaml.SequenceNode {
		c.add(n, `"probes" must be a list`)
		return nil
	}
	seen := map[string]bool{}
	probes := make([]Probe, 0, len(n.Content))
	for _, item := range n.Content {
		p, ok := c.probe(item, limits)
		if !ok {
			continue
		}
		if seen[p.ID] {
			c.add(item, "duplicate probe id %q", p.ID)
		}
		seen[p.ID] = true
		probes = append(probes, p)
	}
	return probes
}

func (c *checker) probe(n *yaml.Node, limits Limits) (Probe, bool) {
	if n.Kind != yaml.MappingNode {
		c.add(n, "a probe must be a map with id and where")
		return Probe{}, false
	}
	values := c.mapping(n, probeKeys, " in probe")
	id := values["id"]
	if id == nil || id.Kind != yaml.ScalarNode || id.Value == "" {
		c.add(n, `probe has no "id"`)
		return Probe{}, false
	}
	p := Probe{ID: id.Value, Limits: c.limits(values["limits"], limits)}
	if values["where"] == nil {
		c.add(n, "probe %q has no \"where\"", p.ID)
	} else {
		p.Where = c.where(values["where"], p.ID)
	}
	p.When = c.when(values["when"], p.ID)
	p.What = c.what(values["what"], p.ID)
	return p, true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
