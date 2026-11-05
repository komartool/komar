package cli

import (
	"errors"
	"fmt"
	"io"
	"komar/internal/celcheck"
	"komar/internal/launch"
	"komar/internal/runtime/jvm"
	"komar/internal/server"
	"komar/internal/spec"
	"komar/internal/writer"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	Version     = "0.1.0"
	format      = 1
	agentGrace  = 3 * time.Second
	exitOK      = 0
	exitUsage   = 2
	exitSpec    = 2
	exitStart   = 3
	exitFailure = 4
)

type runArgs struct {
	spec    string
	out     string
	program []string
}

func Main(args []string, stderr io.Writer) int {
	ra, err := parse(args)
	if err != nil {
		say(stderr, "komar: %v\nusage: komar run <spec.yaml> -o <out.jsonl> -- <program> [args...]\n", err)
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
	var ra runArgs
	rest := args[1:]
	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]
		switch {
		case arg == "--":
			ra.program, rest = rest, nil
		case arg == "-o":
			if len(rest) == 0 {
				return runArgs{}, errors.New("-o needs a path")
			}
			ra.out, rest = rest[0], rest[1:]
		case ra.spec == "" && len(arg) > 0 && arg[0] != '-':
			ra.spec = arg
		default:
			return runArgs{}, fmt.Errorf("unexpected argument %q", arg)
		}
	}
	return ra, validate(ra)
}

func validate(ra runArgs) error {
	switch {
	case ra.spec == "":
		return errors.New("missing spec file")
	case ra.out == "":
		return errors.New("missing -o <out.jsonl>")
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
	out, err := writer.Create(ra.out)
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	code := session(ra, s, out, stderr)
	if err := out.Close(); err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	return code
}

func session(ra runArgs, s *spec.Spec, out *writer.Writer, stderr io.Writer) int {
	srv, err := server.Listen(out)
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
	proc, err := launch.Start(ra.program, js.Env(os.Environ()))
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitStart
	}
	return finish(proc, srv, out, stderr)
}

func finish(proc *launch.Process, srv *server.Server, out *writer.Writer, stderr io.Writer) int {
	reason := forwardSignals(proc)
	code, err := proc.Wait()
	if err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	srv.Shutdown(agentGrace)
	totals := srv.Totals()
	if totals.Invalid > 0 {
		say(stderr, "komar: warning: %d malformed records from agents were skipped\n", totals.Invalid)
	}
	if err := out.Write(endRecord(reason(), code, totals)); err != nil {
		say(stderr, "komar: %v\n", err)
		return exitFailure
	}
	say(stderr, "komar: program exited with code %d\n", code)
	return exitOK
}

func forwardSignals(proc *launch.Process) func() string {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	got := make(chan struct{})
	go func() {
		for sig := range signals {
			select {
			case <-got:
			default:
				close(got)
			}
			proc.Signal(sig)
		}
	}()
	return func() string {
		signal.Stop(signals)
		select {
		case <-got:
			return "signal"
		default:
			return "exit"
		}
	}
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
	Code       int              `json:"code"`
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

func endRecord(reason string, code int, t server.Totals) endRec {
	return endRec{
		Kind: "end", Reason: reason, Code: code,
		Hits: t.Hits, Dropped: t.Dropped, WhenErrors: t.WhenErrors,
	}
}
