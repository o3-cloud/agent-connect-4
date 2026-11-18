package tests

import (
	"bytes"
	"strings"
	"testing"

	"connect4arena/internal/cli"
)

func TestRootCommandPrintsHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "with --help flag", args: []string{"--help"}},
		{name: "with no arguments", args: []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := cli.NewRootCommand()
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(test.args)

			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			help := stdout.String()
			for _, want := range []string{"connect4", "Connect Four Arena", "Usage:"} {
				if !strings.Contains(help, want) {
					t.Errorf("help output missing %q:\n%s", want, help)
				}
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRootCommandRejectsUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := cli.NewRootCommand()
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"bogus"})

	if err := command.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want unknown command error")
	}
}
