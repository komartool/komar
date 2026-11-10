package compact

import (
	"encoding/json"
	"strings"
)

type object struct {
	keys []string
	vals []any
}

func (o *object) add(key string, v any) {
	o.keys = append(o.keys, key)
	o.vals = append(o.vals, v)
}

func (o *object) get(key string) (any, bool) {
	if o == nil {
		return nil, false
	}
	for i, k := range o.keys {
		if k == key {
			return o.vals[i], true
		}
	}
	return nil, false
}

func (o *object) has(key string) bool {
	_, ok := o.get(key)
	return ok
}

func (o *object) text(key string) string {
	v, _ := o.get(key)
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	}
	return ""
}

func (o *object) fields() *object {
	out := &object{}
	for i, k := range o.keys {
		if !strings.HasPrefix(k, "@") && !isEmpty(o.vals[i]) {
			out.add(k, o.vals[i])
		}
	}
	return out
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case []any:
		return len(x) == 0
	case *object:
		if len(x.keys) == 0 {
			return true
		}
		items, ok := x.get("@items")
		list, isList := items.([]any)
		return ok && len(x.keys) == 1 && isList && len(list) == 0
	}
	return false
}

func canonical(v any) string {
	var b strings.Builder
	writeCanonical(&b, v)
	return b.String()
}

func writeCanonical(b *strings.Builder, v any) {
	switch x := v.(type) {
	case *object:
		b.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, k)
			b.WriteByte(':')
			writeCanonical(b, x.vals[i])
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonical(b, item)
		}
		b.WriteByte(']')
	case string:
		writeString(b, x)
	default:
		data, _ := json.Marshal(x)
		b.Write(data)
	}
}

func writeString(b *strings.Builder, s string) {
	data, _ := json.Marshal(s)
	b.Write(data)
}
