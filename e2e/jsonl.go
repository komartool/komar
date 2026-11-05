package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const masked = "<masked>"

var maskedKeys = map[string]bool{"start": true, "komar": true, "jvm": true, "ts": true, "t": true}

func cmdJSONLCmp(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("jsonl-cmp does not support negation")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: jsonl-cmp <actual.jsonl> <expected.jsonl>")
	}
	actual, err := normalize(ts.ReadFile(args[0]))
	ts.Check(err)
	expected, err := canonical(ts.ReadFile(args[1]))
	ts.Check(err)
	if actual != expected {
		ts.Fatalf("%s differs from %s\n--- actual (normalized)\n%s\n--- expected\n%s", args[0], args[1], actual, expected)
	}
}

func normalize(content string) (string, error) {
	n := &normalizer{pids: map[float64]int{}}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line, err := n.record(sc.Bytes())
		if err != nil {
			return "", fmt.Errorf("line %d: %w", len(out)+1, err)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), sc.Err()
}

func canonical(content string) (string, error) {
	var out []string
	for i, line := range strings.Split(strings.TrimSpace(content), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return "", fmt.Errorf("expected line %d: %w", i+1, err)
		}
		enc, err := encode(rec)
		if err != nil {
			return "", err
		}
		out = append(out, enc)
	}
	return strings.Join(out, "\n"), nil
}

type normalizer struct {
	pids    map[float64]int
	lastSeq float64
}

func (n *normalizer) record(raw []byte) (string, error) {
	var rec map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&rec); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	if err := n.seq(rec); err != nil {
		return "", err
	}
	if err := mask(rec); err != nil {
		return "", err
	}
	if pid, ok := rec["pid"].(float64); ok {
		rec["pid"] = n.pid(pid)
	}
	return encode(rec)
}

func (n *normalizer) seq(rec map[string]any) error {
	seq, ok := rec["seq"].(float64)
	if !ok {
		return nil
	}
	if seq <= n.lastSeq {
		return fmt.Errorf("seq %v does not increase (previous %v)", seq, n.lastSeq)
	}
	n.lastSeq = seq
	delete(rec, "seq")
	return nil
}

func (n *normalizer) pid(pid float64) int {
	if id, ok := n.pids[pid]; ok {
		return id
	}
	n.pids[pid] = len(n.pids) + 1
	return n.pids[pid]
}

func mask(rec map[string]any) error {
	for key := range maskedKeys {
		value, ok := rec[key]
		if !ok {
			continue
		}
		if key == "start" || key == "ts" {
			s, _ := value.(string)
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return fmt.Errorf("%s is not an ISO-8601 time: %v", key, value)
			}
		}
		rec[key] = masked
	}
	return nil
}

func encode(rec map[string]any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(rec); err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}
