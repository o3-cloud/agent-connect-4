// Command connect4 is the Connect Four Arena CLI.
package main

import (
	"os"

	"connect4arena/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}
