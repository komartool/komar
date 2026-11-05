package e2e

import (
	"errors"
	"os/exec"
	"strconv"

	"github.com/rogpeppe/go-internal/testscript"
)

func cmdExecStatus(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("exec-status does not support negation")
	}
	if len(args) < 2 {
		ts.Fatalf("usage: exec-status <code> <command> [args...]")
	}
	want, err := strconv.Atoi(args[0])
	ts.Check(err)
	got := 0
	if err := ts.Exec(args[1], args[2:]...); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			ts.Fatalf("%s: %v", args[1], err)
		}
		got = exitErr.ExitCode()
	}
	if got != want {
		ts.Fatalf("%s exited with %d, want %d", args[1], got, want)
	}
}
