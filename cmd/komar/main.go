package main

import (
	"os"

	"github.com/mxprshn/komar/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stderr))
}
