package spec

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Path     string
	Calls    bool
	Limits   Limits
	Probes   []Probe
	Warnings []Problem
}

type Probe struct {
	ID     string
	Where  Where
	When   *Expr
	What   []Item
	Limits Limits
}

type Expr struct {
	Text   string
	Line   int
	Col    int
	Quoted bool
	Block  bool
	Roots  []string
	Vars   []string
}

type Where struct {
	Class    string
	Subtypes bool
	Method   string
	Params   []string
	HasParam bool
	Lines    []int
	At       []string
}

type Item struct {
	Key  string
	Expr Expr
}

type Problem struct {
	Line  int
	Col   int
	Probe string
	Msg   string
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

func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, &InvalidError{Path: path, Problems: []Problem{{Line: 1, Col: 1, Msg: strings.TrimPrefix(err.Error(), "yaml: ")}}}
	}
	c := &checker{}
	s := c.document(&doc)
	s.Path = path
	s.Warnings = c.warnings
	if len(c.problems) > 0 {
		return s, &InvalidError{Path: path, Problems: c.problems}
	}
	return s, nil
}
