package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const masked = "<masked>"

var (
	maskedStrings = []string{"komar", "jvm"}
	maskedTimes   = []string{"start", "ts"}
)

func cmdJSONLNorm(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("jsonl-norm does not support negation")
	}
	var durations []string
	if len(args) == 4 && args[0] == "-duration" {
		durations, args = strings.Split(args[1], ","), args[2:]
	}
	if len(args) != 2 {
		ts.Fatalf("usage: jsonl-norm [-duration key,key] <in.jsonl> <out.jsonl>")
	}
	out, err := normalize(ts.ReadFile(args[0]), durations)
	ts.Check(err)
	ts.Check(os.WriteFile(ts.MkAbs(args[1]), []byte(out), 0o600))
}

func normalize(content string, durations []string) (string, error) {
	n := &normalizer{calls: map[string]int{}, pids: map[string]int{}, threads: map[string]int{}, durations: durations}
	var out strings.Builder
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for line := 1; sc.Scan(); line++ {
		v, err := decodeOrdered(sc.Bytes())
		if err != nil {
			return "", fmt.Errorf("line %d: %w", line, err)
		}
		rec, ok := v.(object)
		if !ok {
			return "", fmt.Errorf("line %d: not a JSON object", line)
		}
		rec, err = n.record(rec)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", line, err)
		}
		var buf bytes.Buffer
		encodeOrdered(&buf, rec)
		out.Write(buf.Bytes())
		out.WriteByte('\n')
	}
	return out.String(), sc.Err()
}

type normalizer struct {
	calls     map[string]int
	pids      map[string]int
	threads   map[string]int
	lastSeq   float64
	durations []string
}

func (n *normalizer) record(rec object) (object, error) {
	if err := n.seq(rec); err != nil {
		return nil, err
	}
	rec = rec.without("seq").without("t")
	if err := maskFields(rec); err != nil {
		return nil, err
	}
	pid := fmt.Sprint(field(rec, "pid"))
	if _, ok := rec.get("pid"); ok {
		rec.set("pid", renumber(n.pids, pid))
	}
	for _, key := range []string{"call", "parent"} {
		if v, ok := rec.get(key); ok && v != nil {
			rec.set(key, renumber(n.calls, pid+"/"+fmt.Sprint(v)))
		}
	}
	if thread, ok := field(rec, "thread").(object); ok {
		thread.set("id", renumber(n.threads, pid+"/"+fmt.Sprint(field(thread, "id"))))
	}
	return rec, n.maskDurations(rec)
}

func (n *normalizer) seq(rec object) error {
	v, ok := rec.get("seq")
	if !ok {
		return nil
	}
	seq, err := number(v)
	if err != nil || seq <= n.lastSeq {
		return fmt.Errorf("seq %v does not increase (previous %v)", v, n.lastSeq)
	}
	n.lastSeq = seq
	return nil
}

func (n *normalizer) maskDurations(rec object) error {
	what, ok := field(rec, "what").(object)
	if !ok {
		return nil
	}
	for _, key := range n.durations {
		v, ok := what.get(key)
		if !ok {
			continue
		}
		if d, err := number(v); err != nil || d < 0 {
			return fmt.Errorf("what.%s is not a non-negative number: %v", key, v)
		}
		what.set(key, "<duration>")
	}
	return nil
}

func maskFields(rec object) error {
	for _, key := range maskedStrings {
		if _, ok := rec.get(key); ok {
			rec.set(key, masked)
		}
	}
	for _, key := range maskedTimes {
		v, ok := rec.get(key)
		if !ok {
			continue
		}
		s, _ := v.(string)
		if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
			return fmt.Errorf("%s is not an ISO-8601 time: %v", key, v)
		}
		rec.set(key, masked)
	}
	return nil
}

func field(o object, key string) any {
	v, _ := o.get(key)
	return v
}

func renumber(ids map[string]int, key string) json.Number {
	if _, ok := ids[key]; !ok {
		ids[key] = len(ids) + 1
	}
	return json.Number(fmt.Sprint(ids[key]))
}

func number(v any) (float64, error) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("not a number: %v", v)
	}
	f, err := n.Float64()
	if err != nil {
		return 0, fmt.Errorf("not a number: %w", err)
	}
	return f, nil
}
