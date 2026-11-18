package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func gamePath(id string) string { return "/games/" + url.PathEscape(id) }

// exactArgs is cobra.ExactArgs with a SPEC.md §16 INVALID_ARGUMENT error.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return &apiError{Code: "INVALID_ARGUMENT", Message: err.Error() + "; usage: " + cmd.UseLine()}
		}
		return nil
	}
}

func registerCommand(o *options) *cobra.Command {
	var name, kind string
	cmd := &cobra.Command{
		Use:   "register --name <displayName>",
		Short: "Get an API key for a new player identity (shown once)",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := o.client().do(http.MethodPost, "/agents", map[string]string{"displayName": name, "type": kind})
			if err != nil {
				return err
			}
			o.print(cmd, doc, func(w io.Writer, d document) {
				_, _ = fmt.Fprintf(w, "Registered %s as %s. The key is shown only once:\nexport CONNECT4_API_KEY=%s\n",
					d.str("displayName"), d.str("agentId"), d.str("apiKey"))
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name shown to other players (required)")
	cmd.Flags().StringVar(&kind, "type", "agent", "agent or human")
	return cmd
}

func createCommand(o *options) *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a game and print its id",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := o.client().do(http.MethodPost, "/games", map[string]string{"mode": mode})
			if err != nil {
				return err
			}
			o.print(cmd, doc, func(w io.Writer, d document) { _, _ = fmt.Fprintln(w, d.str("gameId")) })
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "agent_vs_agent", "agent_vs_agent or human_vs_agent")
	return cmd
}

func joinCommand(o *options) *cobra.Command {
	var disc, name, kind string
	cmd := &cobra.Command{
		Use:   "join <gameId> --disc red|yellow --name <displayName>",
		Short: "Take a player slot in a game",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := o.client().do(http.MethodPost, gamePath(args[0])+"/join",
				map[string]string{"disc": disc, "displayName": name, "type": kind})
			if err != nil {
				return err
			}
			o.print(cmd, doc, printGame)
			return nil
		},
	}
	cmd.Flags().StringVar(&disc, "disc", "", "red or yellow (required)")
	cmd.Flags().StringVar(&name, "name", "", "display name shown to other players (required)")
	cmd.Flags().StringVar(&kind, "type", "agent", "agent or human")
	return cmd
}

func stateCommand(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "state <gameId>",
		Short: "Show a game's board and status",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := o.client().do(http.MethodGet, gamePath(args[0]), nil)
			if err != nil {
				return err
			}
			o.print(cmd, doc, printGame)
			return nil
		},
	}
}

func moveCommand(o *options) *cobra.Command {
	var column, expected int
	var key string
	cmd := &cobra.Command{
		Use:   "move <gameId> --column <0-6>",
		Short: "Drop your disc into a column",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("column") {
				return &apiError{Code: "INVALID_ARGUMENT", Message: "--column is required; usage: " + cmd.UseLine()}
			}
			c := o.client()
			if !cmd.Flags().Changed("expected-version") {
				state, err := c.do(http.MethodGet, gamePath(args[0]), nil)
				if err != nil {
					return err
				}
				expected = state.num("version")
			}
			body := map[string]any{"column": column, "expectedVersion": expected}
			if key != "" {
				body["idempotencyKey"] = key
			}
			doc, err := c.do(http.MethodPost, gamePath(args[0])+"/moves", body)
			if err != nil {
				return err
			}
			o.print(cmd, doc, func(w io.Writer, d document) {
				move, _ := d.m["move"].(map[string]any)
				_, _ = fmt.Fprintf(w, "Move %v: %v in column %v (row %v).", move["moveNumber"], move["disc"], move["column"], move["row"])
				if d.str("status") == "completed" {
					_, _ = fmt.Fprintf(w, " Game over: %s.\n", outcome(d))
				} else {
					_, _ = fmt.Fprintf(w, " %s to move.\n", d.str("nextTurn"))
				}
			})
			return nil
		},
	}
	cmd.Flags().IntVar(&column, "column", 0, "column from 0 to 6 (required)")
	cmd.Flags().IntVar(&expected, "expected-version", 0, "game version the move is based on (default: the current version)")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "retry-safe key: resending the same key returns the first result")
	return cmd
}

func historyCommand(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "history <gameId>",
		Short: "List a game's moves in order",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := o.client().do(http.MethodGet, gamePath(args[0])+"/moves", nil)
			if err != nil {
				return err
			}
			o.print(cmd, doc, func(w io.Writer, d document) {
				moves, _ := d.m["moves"].([]any)
				for _, m := range moves {
					move, _ := m.(map[string]any)
					_, _ = fmt.Fprintf(w, "%v. %v column %v\n", move["moveNumber"], move["disc"], move["column"])
				}
				if d.str("status") == "completed" {
					_, _ = fmt.Fprintf(w, "Result: %s\n", outcome(d))
				}
			})
			return nil
		},
	}
}

func resignCommand(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "resign <gameId>",
		Short: "Resign an in-progress game; your opponent wins",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := o.client().do(http.MethodPost, gamePath(args[0])+"/resign", nil)
			if err != nil {
				return err
			}
			o.print(cmd, doc, printGame)
			return nil
		},
	}
}

func outcome(d document) string {
	if winner := d.str("winner"); winner != "" {
		return winner + " wins (" + d.str("resultReason") + ")"
	}
	return "draw"
}

// printGame draws the board (row 0 at the top) and a status line.
func printGame(w io.Writer, d document) {
	rows, _ := d.m["board"].([]any)
	for _, r := range rows {
		cells, _ := r.([]any)
		marks := make([]string, len(cells))
		for i, cell := range cells {
			switch cell {
			case "red":
				marks[i] = "R"
			case "yellow":
				marks[i] = "Y"
			default:
				marks[i] = "."
			}
		}
		_, _ = fmt.Fprintln(w, strings.Join(marks, " "))
	}
	_, _ = fmt.Fprintln(w, "0 1 2 3 4 5 6")
	status := d.str("status")
	switch status {
	case "in_progress":
		_, _ = fmt.Fprintf(w, "%s  %s: in_progress, %s to move, version %d\n", d.str("gameId"), d.str("mode"), d.str("currentTurn"), d.num("version"))
	case "completed":
		_, _ = fmt.Fprintf(w, "%s  %s: completed, %s, version %d\n", d.str("gameId"), d.str("mode"), outcome(d), d.num("version"))
	default:
		_, _ = fmt.Fprintf(w, "%s  %s: %s, version %d\n", d.str("gameId"), d.str("mode"), status, d.num("version"))
	}
}
