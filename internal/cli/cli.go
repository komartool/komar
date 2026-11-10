package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/mxprshn/komar/internal/celcheck"
	"github.com/mxprshn/komar/internal/compact"
	"github.com/mxprshn/komar/internal/launch"
	"github.com/mxprshn/komar/internal/runtime/jvm"
	"github.com/mxprshn/komar/internal/server"
	"github.com/mxprshn/komar/internal/spec"
	"github.com/mxprshn/komar/internal/writer"
)

const (
	Version      = "0.1.0"
	format       = 1
	agentGrace   = 3 * time.Second
	stopGrace    = 3 * time.Second
	reasonExit   = "exit"
	reasonStop   = "stop"
	reasonSignal = "signal"
	exitOK       = 0
	exitUsage    = 2
	exitSpec     = 2
	exitStart    = 3
	exitFailure  = 4
)

type runArgs struct {
	spec    string
	out     string
	formats formats
	program []string
}

func Main(args []string, stderr io.Writer) int {
	ra, err := parse(args)
	if err != nil {
		say(stderr, "komar: %v\nusage: komar run <spec.yaml> -o <name> [-f jsonl,view] -- <program> [args...]\n", err)
		return exitUsage
	}
	s, err := load(ra.spec)
	if err != nil {
		return reportSpecError(stderr, err)
	}
	for _, w := range s.Warnings {
		say(stderr, "komar: warning: %s:%d:%d: %s\n", s.Path, w.Line, w.Col, w.Msg)
	}
	return run(ra, s, stderr)
}

func load(path string) (*spec.Spec, error) {
	s, err := spec.Load(path)
	var invalid *spec.InvalidError
	if s == nil || (err != nil && !errors.As(err, &invalid)) {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	res, cerr := celcheck.Check(s)
	if cerr != nil {
		return nil, fmt.Errorf("check expressions: %w", cerr)
	}
	s.Warnings = append(s.Warnings, res.Warnings...)
	if invalid == nil && len(res.Errors) == 0 {
		return s, nil
	}
	if invalid == nil {
		invalid = &spec.InvalidError{Path: s.Path}
	}
	invalid.Problems = append(invalid.Problems, res.Errors...)
	return nil, invalid
}

func say(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func parse(args []string) (runArgs, error) {
	if len(args) == 0 || args[0] != "run" {
		return runArgs{}, errors.New(`expected command "run"`)
	}
	ra := runArgs{formats: allFormats}
	rest := args[1:]
	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]
		var err error
		switch {
		case arg == "--":
			ra.program, rest = rest, nil
		case arg == "-o":
			ra.out, rest, err = value(rest, "-o needs a path")
			ra.out = outputBase(ra.out)
		case arg == "-f":
			var list string
			list, rest, err = value(rest, "-f needs a list of formats")
			if err == nil {
				ra.formats, err = parseFormats(list)
			}
		case ra.spec == "" && len(arg) > 0 && arg[0] != '-':
			ra.spec = arg
		default:
			err = fmt.Errorf("unexpected argument %q", arg)
		}
		if err != nil {
			return runArgs{}, err
		}
	}
	return ra, validate(ra)
}

func value(rest []string, missing string) (string, []string, error) {
	if len(rest) == 0 || rest[0] == "--" {
		return "", rest, errors.New(missing)
	}
	return rest[0], rest[1:], nil
}

func validate(ra runArgs) error {
	switch {
	case ra.spec == "":
		return errors.New("missing spec file")
	case ra.out == "":
		return errors.New("missing -o <name>")
	case len(ra.program) == 0:
		return errors.New("missing program after --")
	}
	return nil
}

func reportSpecError(stderr io.Writer, err error) int {
	var invalid *spec.InvalidError
	if !errors.As(err, &invalid) {
		say(stderr, "komar: %v\nkomar: nothing started\n", err)
		return exitSpec
	}
	for _, p := range invalid.Problems {
		say(stderr, "komar: %s:%d:%d: %s\n", invalid.Path, p.Line, p.Col, p.Msg)
	}
	noun := "errors"
	if len(invalid.Problems) == 1 {
		noun = "error"
	}
	say(stderr, "komar: %d %s, nothing started\n", len(invalid.Problems), noun)
	return exitSpec
}

func run(ra runArgs, s *spec.Spec, stderr io.Writer) int {
	dump, err := dumpPath(ra)
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	if !ra.formats.jsonl {
		defer func() { _ = os.Remove(dump) }()
	}
	out, err := writer.Create(dump)
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	code := session(ra, s, out, stderr)
	if err := out.Close(); err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	if ra.formats.view {
		if err := compact.Write(dump, ra.out+extView, probeIDs(s)); err != nil {
			say(stderr, "komar: warning: %v\n", err)
		}
	}
	return code
}

func probeIDs(s *spec.Spec) []string {
	ids := make([]string, len(s.Probes))
	for i, p := range s.Probes {
		ids[i] = p.ID
	}
	return ids
}

func session(ra runArgs, s *spec.Spec, out *writer.Writer, stderr io.Writer) int {
	srv, err := server.Listen(out, s.Stop.Hits)
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	defer srv.Shutdown(agentGrace)
	addr := srv.Addr()
	js, err := jvm.Prepare(addr.IP.String(), addr.Port, s)
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	defer func() { _ = js.Close() }()
	if err := out.Write(runRecord(ra, s)); err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	for _, w := range s.Warnings {
		msg := fmt.Sprintf("%s:%d:%d: %s", s.Path, w.Line, w.Col, w.Msg)
		if err := out.Write(diagRec{Kind: "diag", Level: "warn", Probe: w.Probe, Msg: msg}); err != nil {
			say(stderr, "komar: %v\n", err)
			return exitFailure
		}
	}
	proc, err := launch.Start(ra.program, js.Env(os.Environ()))
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	return finish(proc, srv, out, s.Stop, stderr)
}

func finish(proc *launch.Process, srv *server.Server, out *writer.Writer, stop spec.Stop, stderr io.Writer) int {
	reason, stopped := await(proc, srv, stop)
	code, err := proc.Result()
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	srv.Shutdown(agentGrace)
	totals := srv.Totals()
	if totals.Invalid > 0 {
		say(stderr, "komar: warning: %d malformed records from agents were skipped\n", totals.Invalid)
	}
	for _, id := range sortedKeys(totals.Unplaced) {
		if err := out.Write(diagRec{Kind: "diag", Level: "error", Probe: id, Msg: totals.Unplaced[id]}); err != nil {
			say(stderr, "komar: %v\n", err)
			return exitFailure
		}
	}
	var endCode *int
	if !stopped {
		endCode = &code
	}
	if err := out.Write(endRecord(reason, endCode, totals)); err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	if stopped {
		say(stderr, "komar: stop condition reached, program terminated\n")
	} else {
		say(stderr, "komar: program exited with code %d\n", code)
	}
	return exitOK
}

func await(proc *launch.Process, srv *server.Server, stop spec.Stop) (string, bool) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	var timeout <-chan time.Time
	if stop.After > 0 {
		timer := time.NewTimer(stop.After)
		defer timer.Stop()
		timeout = timer.C
	}
	reason := reasonExit
	for {
		select {
		case <-proc.Done():
			return reason, false
		case sig := <-signals:
			reason = reasonSignal
			proc.Signal(sig)
		case <-timeout:
			proc.Terminate(stopGrace)
			return reasonStop, true
		case <-srv.LimitReached():
			proc.Terminate(stopGrace)
			return reasonStop, true
		}
	}
}

type diagRec struct {
	Kind  string `json:"kind"`
	Level string `json:"level"`
	Probe string `json:"probe,omitempty"`
	Msg   string `json:"msg"`
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type runRec struct {
	Kind   string   `json:"kind"`
	Format int      `json:"format"`
	Komar  string   `json:"komar"`
	Spec   string   `json:"spec"`
	Cmd    []string `json:"cmd"`
	Start  string   `json:"start"`
}

type endRec struct {
	Kind       string           `json:"kind"`
	Reason     string           `json:"reason"`
	Code       *int             `json:"code"`
	Hits       map[string]int   `json:"hits"`
	Dropped    int64            `json:"dropped"`
	WhenErrors map[string]int64 `json:"when_errors"`
}

func runRecord(ra runArgs, s *spec.Spec) runRec {
	return runRec{
		Kind:   "run",
		Format: format,
		Komar:  Version,
		Spec:   s.Path,
		Cmd:    ra.program,
		Start:  time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func endRecord(reason string, code *int, t server.Totals) endRec {
	return endRec{
		Kind: "end", Reason: reason, Code: code,
		Hits: t.Hits, Dropped: t.Dropped, WhenErrors: t.WhenErrors,
	}
}
