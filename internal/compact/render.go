package compact

import (
	"strconv"
	"strings"
)

const legend = "# → call made by the method above, ⇒ outcome, Type#n{…} first occurrence of a value, " +
	"#n the same value again, =#n+{f: v} like #n except these fields (∅ = gone), " +
	"^n reference within one captured value, ×N repeated N times"

var (
	entryKeys   = []string{"this", "args"}
	resultKeys  = []string{"ret", "exception"}
	outcomeKeys = []string{"call.target", "call.args", "ret", "exception"}
)

type group struct {
	root  *call
	count int
}

func Render(recs []*object) string {
	var hits []*object
	for _, r := range recs {
		if r.text("kind") == "hit" {
			hits = append(hits, r)
		}
	}
	lines := []string{legend}
	lines = append(lines, summary(recs, hits)...)
	roots := build(hits)
	groups := distinct(roots)
	lines = append(lines, "executions: "+strconv.Itoa(len(roots))+", distinct: "+strconv.Itoa(len(groups)))
	r := &renderer{f: newFormatter(true)}
	for _, g := range groups {
		body := fold(r.render(g.root))
		if g.count > 1 {
			body[0] += "   ×" + strconv.Itoa(g.count)
		}
		lines = append(lines, "")
		lines = append(lines, body...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func summary(recs, hits []*object) []string {
	var order []string
	counts := map[string]int{}
	pids := map[string]bool{}
	for _, h := range hits {
		k := h.text("probe") + "@" + h.text("point")
		if counts[k] == 0 {
			order = append(order, k)
		}
		counts[k]++
		pids[h.text("pid")] = true
	}
	parts := make([]string, len(order))
	for i, k := range order {
		parts[i] = k + " " + strconv.Itoa(counts[k])
	}
	head := "hits: " + strings.Join(parts, ", ")
	if len(pids) > 1 {
		head += "  jvms: " + strconv.Itoa(len(pids))
	}
	return append([]string{head}, notes(recs)...)
}

func notes(recs []*object) []string {
	var out []string
	for _, r := range recs {
		switch {
		case r.text("kind") == "end":
			out = append(out, endLine(r))
		case r.text("kind") == "diag" && r.text("level") != "info":
			out = append(out, strings.TrimSpace(r.text("level")+": "+r.text("probe")+" "+r.text("msg")))
		}
	}
	return out
}

func endLine(r *object) string {
	s := "end: reason=" + r.text("reason")
	if code := r.text("code"); code != "" {
		s += " code=" + code
	}
	if d := r.text("dropped"); d != "" && d != "0" {
		s += " dropped=" + d
	}
	return s
}

func distinct(roots []*call) []*group {
	canon := &renderer{f: newFormatter(false)}
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

func shortMethod(m string) string {
	name, args, _ := strings.Cut(m, "(")
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		name = parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return name + "(" + args
}

func methodName(m string) string {
	name, _, _ := strings.Cut(shortMethod(m), "(")
	return name
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
