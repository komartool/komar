package compact

import "strings"

type renderer struct {
	f   *formatter
	out []string
}

func (r *renderer) render(c *call) []string {
	r.out = nil
	r.call(c, 0, false)
	return r.out
}

func (r *renderer) call(c *call, depth int, inline bool) {
	v := &callView{r: r, c: c, depth: depth, pad: strings.Repeat("  ", depth)}
	start := len(r.out)
	for i := 0; i < len(c.events); {
		i = v.step(i)
	}
	if !v.opened && !inline {
		head := v.pad + r.f.names.method(c.method) + "  (no entry record)"
		r.out = append(r.out[:start], append([]string{head}, r.out[start:]...)...)
	}
}

func (r *renderer) keys(what *object, skip []string, base *object) string {
	if what == nil {
		return ""
	}
	var parts []string
	for i, k := range what.keys {
		if contains(skip, k) {
			continue
		}
		if prev, ok := base.get(k); ok && canonical(prev) == canonical(what.vals[i]) {
			continue
		}
		parts = append(parts, k+"="+r.f.value(what.vals[i]))
	}
	return strings.Join(parts, " ")
}

func (r *renderer) args(v any) string {
	if list, ok := v.([]any); ok {
		return r.f.join(list)
	}
	return r.f.value(v)
}

type callView struct {
	r      *renderer
	c      *call
	depth  int
	pad    string
	entry  *object
	opened bool
}

func (v *callView) emit(line string) {
	v.r.out = append(v.r.out, line)
}

func (v *callView) step(i int) int {
	e := v.c.events[i]
	if e.child != nil {
		v.r.call(e.child, v.depth+1, false)
		return i + 1
	}
	switch e.hit.point {
	case "entry":
		v.onEntry(e.hit)
	case "exit", "exception":
		v.onOutcome(e.hit)
	case "before":
		return v.onBefore(i)
	case "after":
		v.emit(v.pad + "  → " + v.callSite(e.hit, nil))
	default:
		v.emit(withRest(v.pad+"  L"+e.hit.line, v.r.keys(e.hit.what, nil, nil)))
	}
	return i + 1
}

func (v *callView) onEntry(h *hit) {
	line := v.pad + v.r.f.names.method(v.c.method)
	if v.opened {
		line = v.pad + "(entry again)"
	}
	v.opened = true
	v.entry = h.what
	var parts []string
	if this, ok := h.what.get("this"); ok {
		parts = append(parts, "this="+v.r.f.value(this))
	}
	if args, ok := h.what.get("args"); ok {
		parts = append(parts, "args=("+v.r.args(args)+")")
	}
	if rest := v.r.keys(h.what, entryKeys, nil); rest != "" {
		parts = append(parts, rest)
	}
	if len(parts) > 0 {
		line += "  " + strings.Join(parts, "  ")
	}
	v.emit(line)
}

func (v *callView) onOutcome(h *hit) {
	res := "returned"
	if h.point == "exception" || h.what.has("ret") {
		res = outcome(v.r.f, h)
	}
	v.emit(withRest(v.pad+"  ⇒ "+res, v.r.keys(h.what, resultKeys, v.entry)))
}

func (v *callView) onBefore(i int) int {
	evs := v.c.events
	before := evs[i].hit
	j := i + 1
	for j < len(evs) && evs[j].child != nil {
		j++
	}
	var after *hit
	if j < len(evs) && isOutcome(evs[j].hit, before) {
		after = evs[j].hit
	}
	v.emit(v.pad + "  → " + v.callSite(before, after))
	for k := i + 1; k < j; k++ {
		v.r.call(evs[k].child, v.depth+1, true)
	}
	if after != nil {
		return j + 1
	}
	return j
}

func (v *callView) callSite(first, after *hit) string {
	f := v.r.f
	s := f.names.callName(first.callee) + "("
	if args, ok := first.what.get("call.args"); ok {
		s += v.r.args(args)
	}
	s += ") ⇒ "
	switch {
	case after != nil:
		s += outcome(f, after)
	case first.point == "before":
		s += "?"
	default:
		s += outcome(f, first)
	}
	rest := v.r.keys(first.what, outcomeKeys, nil)
	if after != nil {
		rest = strings.TrimSpace(rest + " " + v.r.keys(after.what, resultKeys, first.what))
	}
	return withRest(s, rest)
}

func outcome(f *formatter, h *hit) string {
	if h.point == "exception" {
		if exc, ok := h.what.get("exception"); ok {
			return "threw " + f.value(exc)
		}
		return "threw"
	}
	if ret, ok := h.what.get("ret"); ok {
		return f.value(ret)
	}
	return "ok"
}

func isOutcome(h *hit, before *hit) bool {
	return h != nil && h.probe == before.probe && h.callee == before.callee && (h.point == "after" || h.point == "exception")
}

func withRest(line, rest string) string {
	if rest == "" {
		return line
	}
	return line + "  " + rest
}
