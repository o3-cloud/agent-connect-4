// Command connect4 is the Connect Four Arena CLI.
package main

import (
	"fmt"
	"os"

	"connect4arena/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
