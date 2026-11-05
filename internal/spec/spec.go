package spec

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Path   string
	Probes []Probe
}

type Probe struct {
	ID string
}

type Problem struct {
	Line int
	Col  int
	Msg  string
}

type InvalidError struct {
	Path     string
	Problems []Problem
}

func (e *InvalidError) Error() string {
	lines := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		lines = append(lines, fmt.Sprintf("%s:%d:%d: %s", e.Path, p.Line, p.Col, p.Msg))
	}
	return strings.Join(lines, "\n")
}

var (
	topKeys   = []string{"version", "calls", "limits", "stop", "jvm", "probes"}
	probeKeys = []string{"id", "where", "when", "what", "limits"}
)

func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, &InvalidError{Path: path, Problems: []Problem{{Line: 1, Col: 1, Msg: yamlMessage(err)}}}
	}
	c := &checker{}
	s := c.document(&doc)
	if len(c.problems) > 0 {
		return nil, &InvalidError{Path: path, Problems: c.problems}
	}
	s.Path = path
	return s, nil
}

func yamlMessage(err error) string {
	return strings.TrimPrefix(err.Error(), "yaml: ")
}

type checker struct {
	problems []Problem
}

func (c *checker) add(n *yaml.Node, format string, args ...any) {
	c.problems = append(c.problems, Problem{Line: n.Line, Col: n.Column, Msg: fmt.Sprintf(format, args...)})
}

func (c *checker) document(doc *yaml.Node) *Spec {
	s := &Spec{}
	if len(doc.Content) == 0 {
		c.problems = append(c.problems, Problem{Line: 1, Col: 1, Msg: `spec is empty; it needs a "probes" list`})
		return s
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		c.add(root, "spec must be a map with a \"probes\" list")
		return s
	}
	probes := c.mapping(root, topKeys, "")
	if probes["probes"] == nil {
		c.add(root, `missing required key "probes"`)
		return s
	}
	s.Probes = c.probes(probes["probes"])
	return s
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

func (c *checker) probes(n *yaml.Node) []Probe {
	if n.Kind != yaml.SequenceNode {
		c.add(n, `"probes" must be a list`)
		return nil
	}
	seen := map[string]bool{}
	probes := make([]Probe, 0, len(n.Content))
	for _, item := range n.Content {
		p, ok := c.probe(item)
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

func (c *checker) probe(n *yaml.Node) (Probe, bool) {
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
	where := values["where"]
	switch {
	case where == nil:
		c.add(n, "probe %q has no \"where\"", id.Value)
	case where.Kind != yaml.MappingNode:
		c.add(where, "probe %q: \"where\" must be a map", id.Value)
	}
	return Probe{ID: id.Value}, true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
