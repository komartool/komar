package compact

import (
	"strconv"
	"strings"
)

const legend = "# → call made by the method above, ⇒ outcome, Type#n{…} first occurrence of a value, " +
	"#n the same value again, =#n+{f: v} like #n except these fields (∅ = gone), " +
	"^n reference within one captured value, ×N repeated N times, " +
	"types are named in full on first use, {…} fields cut by limits, {proxy} a generated proxy or mock"

var (
	entryKeys   = []string{"this", "args"}
	resultKeys  = []string{"ret", "exception"}
	outcomeKeys = []string{"call.target", "call.args", "ret", "exception"}
)

type group struct {
	root  *call
	count int
}

func Render(dump []byte, probes []string) string {
	recs := parseLines(dump)
	var hits []*object
	for _, r := range recs {
		if r.text("kind") == "hit" {
			hits = append(hits, r)
		}
	}
	lines := []string{legend}
	lines = append(lines, summary(recs, hits, probes)...)
	roots := build(hits)
	groups := distinct(roots)
	lines = append(lines, "executions: "+strconv.Itoa(len(roots))+", distinct: "+strconv.Itoa(len(groups)))
	r := &renderer{f: newFormatter(true, newNames())}
	for _, g := range groups {
		body := fold(r.render(g.root))
		if g.count > 1 {
			body[0] += "   ×" + strconv.Itoa(g.count)
		}
		lines = append(lines, "")
		for _, line := range body {
			lines = append(lines, wrapLine(line, maxWidth)...)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func distinct(roots []*call) []*group {
	canon := &renderer{f: newFormatter(false, nil)}
	byText := map[string]*group{}
	var groups []*group
	for _, c := range roots {
		key := strings.Join(canon.render(c), "\n")
		if g, ok := byText[key]; ok {
			g.count++
			continue
		}
		g := &group{root: c, count: 1}
		byText[key] = g
		groups = append(groups, g)
	}
	return groups
}

func fold(lines []string) []string {
	var out []string
	for i := 0; i < len(lines); {
		n := 1
		for i+n < len(lines) && lines[i+n] == lines[i] {
			n++
		}
		line := lines[i]
		if n > 1 {
			line += "   ×" + strconv.Itoa(n)
		}
		out = append(out, line)
		i += n
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
