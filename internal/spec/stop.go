package spec

import (
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Stop struct {
	After time.Duration
	Hits  int
}

var stopKeys = []string{"after", "hits"}

func (c *checker) stop(n *yaml.Node) Stop {
	var s Stop
	if n == nil {
		return s
	}
	if n.Kind != yaml.MappingNode {
		c.add(n, `"stop" must be a map with "after" and/or "hits"`)
		return s
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		switch key.Value {
		case "after":
			s.After = c.duration(value)
		case "hits":
			s.Hits = c.positive(value, "stop.hits")
		case "max_hits":
			c.add(key, `unknown key "max_hits" in stop: the run total is "hits"; the per-probe cap "max_hits" belongs in "limits"`)
		default:
			c.add(key, "unknown key %q in stop%s", key.Value, suggestion(key.Value, stopKeys))
		}
	}
	return s
}

func (c *checker) duration(n *yaml.Node) time.Duration {
	d, err := time.ParseDuration(n.Value)
	if n.Kind != yaml.ScalarNode || err != nil || d <= 0 {
		c.add(n, "stop.after %q must be a positive duration such as 30s, 2m or 500ms", n.Value)
		return 0
	}
	return d
}

func (c *checker) positive(n *yaml.Node, key string) int {
	v, err := strconv.Atoi(n.Value)
	if n.Kind != yaml.ScalarNode || err != nil || v < 1 {
		c.add(n, "%s must be a positive integer", key)
		return 0
	}
	return v
}
