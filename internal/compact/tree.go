package compact

type hit struct {
	probe  string
	point  string
	method string
	callee string
	line   string
	what   *object
}

type event struct {
	hit   *hit
	child *call
}

type call struct {
	method string
	events []event
}

func build(hits []*object) []*call {
	calls := map[string]*call{}
	var roots []*call
	for _, r := range hits {
		h := toHit(r)
		key := r.text("pid") + "/" + r.text("call")
		c, ok := calls[key]
		if !ok {
			c = &call{method: h.method}
			calls[key] = c
			if p, ok := calls[r.text("pid")+"/"+r.text("parent")]; ok {
				p.events = append(p.events, event{child: c})
			} else {
				roots = append(roots, c)
			}
		}
		c.events = append(c.events, event{hit: h})
	}
	return roots
}

func toHit(r *object) *hit {
	what, _ := r.get("what")
	w, _ := what.(*object)
	return &hit{
		probe:  r.text("probe"),
		point:  r.text("point"),
		method: r.text("method"),
		callee: r.text("callee"),
		line:   r.text("line"),
		what:   w,
	}
}
