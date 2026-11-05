package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type member struct {
	key   string
	value any
}

type object []member

func (o object) get(key string) (any, bool) {
	for _, m := range o {
		if m.key == key {
			return m.value, true
		}
	}
	return nil, false
}

func (o object) set(key string, value any) {
	for i := range o {
		if o[i].key == key {
			o[i].value = value
			return
		}
	}
}

func (o object) without(key string) object {
	out := make(object, 0, len(o))
	for _, m := range o {
		if m.key != key {
			out = append(out, m)
		}
	}
	return out
}

func decodeOrdered(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after JSON value")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	switch tok {
	case json.Delim('{'):
		return decodeObject(dec)
	case json.Delim('['):
		return decodeArray(dec)
	}
	return tok, nil
}

func decodeObject(dec *json.Decoder) (object, error) {
	obj := object{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode: %w", err)
		}
		key, _ := tok.(string)
		v, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		obj = append(obj, member{key: key, value: v})
	}
	_, err := dec.Token()
	return obj, wrap(err)
}

func decodeArray(dec *json.Decoder) ([]any, error) {
	arr := []any{}
	for dec.More() {
		v, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	_, err := dec.Token()
	return arr, wrap(err)
}

func wrap(err error) error {
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

func encodeOrdered(buf *bytes.Buffer, v any) {
	switch x := v.(type) {
	case object:
		buf.WriteByte('{')
		for i, m := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeString(buf, m.key)
			buf.WriteByte(':')
			encodeOrdered(buf, m.value)
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodeOrdered(buf, item)
		}
		buf.WriteByte(']')
	case string:
		writeString(buf, x)
	case nil:
		buf.WriteString("null")
	default:
		fmt.Fprint(buf, x)
	}
}

func writeString(buf *bytes.Buffer, s string) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Truncate(buf.Len() - 1)
}
