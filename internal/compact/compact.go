package compact

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Ext = ".komar"

func PathFor(dump string) string {
	return strings.TrimSuffix(dump, filepath.Ext(dump)) + Ext
}

func Write(dump, out string) error {
	data, err := os.ReadFile(dump)
	if err != nil {
		return fmt.Errorf("read dump: %w", err)
	}
	if err := os.WriteFile(out, []byte(Render(parseLines(data))), 0o600); err != nil {
		return fmt.Errorf("write compact view: %w", err)
	}
	return nil
}

func parseLines(data []byte) []*object {
	var recs []*object
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), len(data)+1)
	for sc.Scan() {
		dec := json.NewDecoder(bytes.NewReader(sc.Bytes()))
		dec.UseNumber()
		v, err := decode(dec)
		if o, ok := v.(*object); ok && err == nil {
			recs = append(recs, o)
		}
	}
	return recs
}

func decode(dec *json.Decoder) (any, error) {
	tok, err := token(dec)
	if err != nil {
		return nil, err
	}
	switch tok {
	case json.Delim('{'):
		return decodeObject(dec)
	case json.Delim('['):
		return decodeList(dec)
	}
	return tok, nil
}

func decodeObject(dec *json.Decoder) (*object, error) {
	o := &object{}
	for dec.More() {
		tok, err := token(dec)
		if err != nil {
			return nil, err
		}
		v, err := decode(dec)
		if err != nil {
			return nil, err
		}
		key, _ := tok.(string)
		o.add(key, v)
	}
	_, err := token(dec)
	return o, err
}

func decodeList(dec *json.Decoder) ([]any, error) {
	items := []any{}
	for dec.More() {
		v, err := decode(dec)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	_, err := token(dec)
	return items, err
}

func token(dec *json.Decoder) (json.Token, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	return tok, nil
}
