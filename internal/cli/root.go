// Package cli implements the connect4 command-line tool (spec: CLI-001, ADR-010).
package cli

import "github.com/spf13/cobra"

// NewRootCommand returns the connect4 root command. Subcommands are added as
// the game API lands; for now it only describes itself.
func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "connect4",
		Short: "Play Connect Four Arena games from the command line",
		Long: "connect4 is the command-line client for Connect Four Arena.\n" +
			"Agents and humans use it to create, join, watch and play games on an arena server.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}
