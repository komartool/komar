package compact

import (
	"strconv"
	"strings"
)

const maxAlso = 3

type diagGroup struct {
	first   string
	classes []string
	count   int
}

func summary(recs, hits []*object, probes []string) []string {
	var out []string
	end := findEnd(recs)
	if end != nil {
		out = append(out, endLine(end))
	}
	if idle := idleProbes(hits, probes); len(idle) > 0 {
		line := "no hits: " + strings.Join(idle, ", ")
		if end != nil && end.text("reason") == "stop" {
			line += " (the run was stopped before they fired)"
		}
		out = append(out, line)
	}
	out = append(out, hitsLine(hits))
	return append(out, diags(recs)...)
}

func findEnd(recs []*object) *object {
	for _, r := range recs {
		if r.text("kind") == "end" {
			return r
		}
	}
	return nil
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

func idleProbes(hits []*object, probes []string) []string {
	fired := map[string]bool{}
	for _, h := range hits {
		fired[h.text("probe")] = true
	}
	var idle []string
	for _, p := range probes {
		if !fired[p] {
			idle = append(idle, p)
		}
	}
	return idle
}

func hitsLine(hits []*object) string {
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
	line := "hits: " + strings.Join(parts, ", ")
	if len(pids) > 1 {
		line += "  jvms: " + strconv.Itoa(len(pids))
	}
	return line
}

func diags(recs []*object) []string {
	var order []string
	groups := map[string]*diagGroup{}
	for _, r := range recs {
		if r.text("kind") != "diag" || r.text("level") == "info" {
			continue
		}
		text := r.text("level") + ": " + strings.TrimSpace(r.text("probe")+" "+r.text("msg"))
		class := r.text("class")
		key := text
		if class != "" {
			key = strings.ReplaceAll(text, class, "\x00")
		}
		g, ok := groups[key]
		if !ok {
			g = &diagGroup{first: text}
			groups[key] = g
			order = append(order, key)
		} else if class != "" {
			g.classes = append(g.classes, class)
		}
		g.count++
	}
	out := make([]string, len(order))
	for i, k := range order {
		out[i] = groups[k].line()
	}
	return out
}

func (g *diagGroup) line() string {
	if g.count == 1 {
		return g.first
	}
	if len(g.classes) == 0 {
		return g.first + "   ×" + strconv.Itoa(g.count)
	}
	also := g.classes
	more := ""
	if len(also) > maxAlso {
		also = also[:maxAlso]
		more = ", …"
	}
	return g.first + "   ×" + strconv.Itoa(g.count) + " classes, also " + strings.Join(also, ", ") + more
}
