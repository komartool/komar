package e2e

import (
	"komar/internal/cli"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"komar": func() { os.Exit(cli.Main(os.Args[1:], os.Stderr)) },
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"jsonl-cmp":   cmdJSONLCmp,
			"exec-status": cmdExecStatus,
		},
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", os.Getenv("HOME"))
			return nil
		},
	})
}
