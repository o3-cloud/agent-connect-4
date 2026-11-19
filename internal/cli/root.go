// Package cli implements the connect4 command-line tool (CLI-001..007,
// ADR-010). Every command is a thin client of the REST API: with --json it
// prints exactly the server's JSON document on stdout, and on failure it exits
// with the SPEC.md §16 code for the error.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Default server address when neither --server nor CONNECT4_SERVER is set.
const defaultServer = "http://localhost:8080"

// options are the global flags shared by every command.
type options struct {
	server string
	apiKey string
	json   bool
	getenv func(string) string
}

// NewRootCommand returns the connect4 root command, configured from the
// process environment.
func NewRootCommand() *cobra.Command {
	return newRoot(&options{getenv: os.Getenv})
}

// Execute runs connect4 with args and returns the process exit code. getenv
// supplies CONNECT4_SERVER and CONNECT4_API_KEY.
func Execute(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	opts := &options{getenv: getenv}
	root := newRoot(opts)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	err := root.Execute()
	if err == nil {
		return 0
	}
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		// Cobra usage errors: unknown command or flag, wrong argument count.
		apiErr = &apiError{Code: "INVALID_ARGUMENT", Message: err.Error()}
	}
	if opts.json || wantsJSON(args) {
		_, _ = fmt.Fprintln(stdout, apiErr.document())
	} else {
		_, _ = fmt.Fprintf(stderr, "Error: %s: %s\n", apiErr.Code, apiErr.Message)
	}
	return apiErr.exitCode()
}

// wantsJSON reports whether --json was given, for errors raised before Cobra
// parsed the flags.
func wantsJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "--json=true" {
			return true
		}
	}
	return false
}

func newRoot(opts *options) *cobra.Command {
	root := &cobra.Command{
		Use:   "connect4",
		Short: "Play Connect Four Arena games from the command line",
		Long: "connect4 is the command-line client for Connect Four Arena.\n" +
			"Agents and humans use it to create, join, watch and play games on an arena server.\n\n" +
			"Get an API key with `connect4 register --name <name>`, then set CONNECT4_API_KEY.\n" +
			"Set CONNECT4_SERVER to point at a server other than " + defaultServer + ".",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	flags := root.PersistentFlags()
	flags.StringVar(&opts.server, "server", "", "arena server URL (default $CONNECT4_SERVER or "+defaultServer+")")
	flags.StringVar(&opts.apiKey, "api-key", "", "API key (default $CONNECT4_API_KEY)")
	flags.BoolVar(&opts.json, "json", false, "print only the server's JSON document on stdout")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &apiError{Code: "INVALID_ARGUMENT", Message: err.Error()}
	})
	root.AddCommand(registerCommand(opts), createCommand(opts), joinCommand(opts), stateCommand(opts),
		moveCommand(opts), historyCommand(opts), resignCommand(opts), watchCommand(opts))
	return root
}

func (o *options) client() *client {
	server := o.server
	if server == "" {
		server = o.getenv("CONNECT4_SERVER")
	}
	if server == "" {
		server = defaultServer
	}
	key := o.apiKey
	if key == "" {
		key = o.getenv("CONNECT4_API_KEY")
	}
	return &client{base: strings.TrimRight(server, "/") + "/api/v1", key: key}
}

// print writes the server's document as JSON, or as text via human.
func (o *options) print(cmd *cobra.Command, doc document, human func(io.Writer, document)) {
	if o.json {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(doc.raw))
		return
	}
	human(cmd.OutOrStdout(), doc)
}
