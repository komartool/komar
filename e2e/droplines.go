package e2e

import (
	"os"
	"regexp"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"
)

func cmdDropLines(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("drop-lines does not support negation")
	}
	if len(args) != 3 {
		ts.Fatalf("usage: drop-lines <regexp> <in> <out>")
	}
	re, err := regexp.Compile(args[0])
	ts.Check(err)
	var kept []string
	for _, line := range strings.SplitAfter(ts.ReadFile(args[1]), "\n") {
		if line != "" && !re.MatchString(line) {
			kept = append(kept, line)
		}
	}
	ts.Check(os.WriteFile(ts.MkAbs(args[2]), []byte(strings.Join(kept, "")), 0o600))
}
