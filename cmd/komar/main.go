package main

import (
	"komar/internal/cli"
	"os"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stderr))
}
