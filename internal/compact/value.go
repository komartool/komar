package compact

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

const (
	minShared = 40
	minDelta  = 160
)

var plainKey = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$.]*$`)

type formatter struct {
	share  bool
	names  *names
	labels map[string]int
	last   map[string]recent
}

type recent struct {
	fields *object
	key    string
}

func newFormatter(share bool, n *names) *formatter {
	return &formatter{share: share, names: n, labels: map[string]int{}, last: map[string]recent{}}
}

func (f *formatter) value(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(x)
	case json.Number:
		return x.String()
	case string:
		return strconv.Quote(x)
	case []any:
		return f.list(x)
	case *object:
		return f.object(x)
	}
	return "?"
}

func (f *formatter) list(items []any) string {
	render := func() string { return "[" + f.join(items) + "]" }
	if !f.share {
		return render()
	}
	key := canonical(items)
	if len(key) < minShared {
		return render()
	}
	if n, ok := f.labels[key]; ok {
		return "#" + strconv.Itoa(n)
	}
	return f.label(key) + render()
}

func (f *formatter) join(items []any) string {
	parts := make([]string, len(items))
	for i, x := range items {
		parts[i] = f.value(x)
	}
	return strings.Join(parts, ", ")
}

func (f *formatter) object(o *object) string {
	if s, ok := f.marker(o); ok {
		return s
	}
	if base, ok := proxyOf(o.text("@type")); ok {
		return f.names.typ(base) + "{proxy}"
	}
	typ := f.names.typ(o.text("@type"))
	cut, partial := o.get("@truncated")
	if _, isList := cut.([]any); partial && !isList {
		return typ + "{…}"
	}
	id := ""
	if _, ok := o.get("@id"); ok {
		id = "^" + o.text("@id")
	}
	fields := o.fields()
	if len(fields.keys) == 0 {
		return typ + id + "{}"
	}
	more := ""
	if partial {
		more = ", …"
	}
	render := func() string { return "{" + f.members(fields) + more + "}" }
	if !f.share {
		return typ + id + render()
	}
	key := canonical(o)
	if len(key) < minShared {
		return typ + id + render()
	}
	if n, ok := f.labels[key]; ok {
		return typ + "#" + strconv.Itoa(n)
	}
	if !partial {
		render = f.delta(o.text("@type"), key, fields, render)
	}
	return typ + f.label(key) + id + render()
}

func (f *formatter) marker(o *object) (string, bool) {
	if _, ok := o.get("@ref"); ok {
		return "^" + o.text("@ref"), true
	}
	if _, ok := o.get("@error"); ok {
		return "<error: " + o.text("@error") + ">", true
	}
	if _, ok := o.get("@prefix"); ok {
		return strconv.Quote(o.text("@prefix")+"…") + "(" + o.text("@size") + " chars)", true
	}
	if k, ok := o.get("@key"); ok {
		v, _ := o.get("@value")
		return f.value(k) + ": " + f.value(v), true
	}
	if items, ok := o.get("@items"); ok {
		return f.items(o, items), true
	}
	return "", false
}

func (f *formatter) items(o *object, items any) string {
	list, _ := items.([]any)
	s := f.list(list)
	size, err := strconv.Atoi(o.text("@size"))
	if err == nil && size > len(list) {
		s += "…+" + strconv.Itoa(size-len(list))
	}
	return s
}

func (f *formatter) members(fields *object) string {
	parts := make([]string, len(fields.keys))
	for i, k := range fields.keys {
		parts[i] = memberName(k) + ": " + f.value(fields.vals[i])
	}
	return strings.Join(parts, ", ")
}

func (f *formatter) label(key string) string {
	n := len(f.labels) + 1
	f.labels[key] = n
	return "#" + strconv.Itoa(n)
}

func (f *formatter) delta(typ, key string, fields *object, full func() string) func() string {
	if typ == "" || len(key) < minDelta {
		return full
	}
	prev, ok := f.last[typ]
	f.last[typ] = recent{fields: fields, key: key}
	if !ok {
		return full
	}
	changed, gone := diff(prev.fields, fields)
	if 2*(len(changed.keys)+len(gone)) > len(fields.keys) {
		return full
	}
	base := strconv.Itoa(f.labels[prev.key])
	return func() string {
		parts := []string{}
		if len(changed.keys) > 0 {
			parts = append(parts, f.members(changed))
		}
		for _, k := range gone {
			parts = append(parts, memberName(k)+": ∅")
		}
		return "=#" + base + "+{" + strings.Join(parts, ", ") + "}"
	}
}

func diff(old, cur *object) (*object, []string) {
	changed := &object{}
	for i, k := range cur.keys {
		prev, ok := old.get(k)
		if !ok || canonical(prev) != canonical(cur.vals[i]) {
			changed.add(k, cur.vals[i])
		}
	}
	var gone []string
	for _, k := range old.keys {
		if _, ok := cur.get(k); !ok {
			gone = append(gone, k)
		}
	}
	return changed, gone
}

func memberName(k string) string {
	if plainKey.MatchString(k) {
		return k
	}
	return strconv.Quote(k)
}
