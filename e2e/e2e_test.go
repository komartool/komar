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
		Dir:           "testdata",
		UpdateScripts: os.Getenv("KOMAR_E2E_UPDATE") == "1",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"jsonl-norm":  cmdJSONLNorm,
			"exec-status": cmdExecStatus,
			"drop-lines":  cmdDropLines,
		},
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", os.Getenv("HOME"))
			return nil
		},
	})
}
