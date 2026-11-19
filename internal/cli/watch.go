package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// watchEvent is one line of `connect4 watch --json`, shaped like a LIVE-001
// game event: the game, the event type, the state version and its data.
type watchEvent struct {
	Event        string          `json:"event"`
	GameID       string          `json:"gameId"`
	Version      int             `json:"version"`
	Status       string          `json:"status,omitempty"`
	Move         json.RawMessage `json:"move,omitempty"`
	Winner       *string         `json:"winner,omitempty"`
	ResultReason *string         `json:"resultReason,omitempty"`
}

func watchCommand(o *options) *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "watch <gameId>",
		Short: "Follow a game, printing each new move, until it is over",
		Long: "watch prints every move accepted after it starts, in order, and the result once the game\n" +
			"is completed, then exits 0. With --json it prints one JSON event per line: a \"watching\"\n" +
			"event with the starting version, a \"move\" event per move and a final \"game_completed\" event.",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if interval <= 0 {
				return &apiError{Code: "INVALID_ARGUMENT", Message: "--interval must be greater than zero"}
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return o.watch(ctx, cmd.OutOrStdout(), args[0], interval)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", time.Second, "how often to poll the server for new moves")
	return cmd
}

// watch polls the game's move history (API-006) and reports what is new. The
// history is authoritative, so a missed poll can never skip or reorder moves.
func (o *options) watch(ctx context.Context, w io.Writer, gameID string, interval time.Duration) error {
	c := o.client()
	path := gamePath(gameID) + "/moves"
	history, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	seen := len(movesOf(history))
	o.emit(w, watchEvent{Event: "watching", GameID: history.str("gameId"), Version: history.num("version"), Status: history.str("status")},
		fmt.Sprintf("Watching %s (%s, version %d)", history.str("gameId"), history.str("status"), history.num("version")))

	for {
		moves := movesOf(history)
		for _, raw := range moves[seen:] {
			var move map[string]any
			_ = json.Unmarshal(raw, &move)
			number, _ := move["moveNumber"].(float64)
			o.emit(w, watchEvent{Event: "move", GameID: history.str("gameId"), Version: int(number), Move: raw},
				fmt.Sprintf("%v. %v column %v", move["moveNumber"], move["disc"], move["column"]))
		}
		seen = len(moves)

		if history.str("status") == "completed" {
			event := watchEvent{Event: "game_completed", GameID: history.str("gameId"), Version: history.num("version"),
				Status: "completed", Winner: optional(history.str("winner")), ResultReason: optional(history.str("resultReason"))}
			o.emit(w, event, "Result: "+outcome(history))
			return nil
		}

		select {
		case <-ctx.Done():
			return &apiError{Code: "INTERNAL", Message: "watch interrupted"}
		case <-time.After(interval):
		}
		if history, err = c.do(http.MethodGet, path, nil); err != nil {
			return err
		}
	}
}

// emit prints one event: a JSON line with --json, otherwise the text.
func (o *options) emit(w io.Writer, event watchEvent, text string) {
	if o.json {
		line, _ := json.Marshal(event)
		_, _ = fmt.Fprintln(w, string(line))
		return
	}
	_, _ = fmt.Fprintln(w, text)
}

func movesOf(d document) []json.RawMessage {
	var body struct {
		Moves []json.RawMessage `json:"moves"`
	}
	_ = json.Unmarshal(d.raw, &body)
	return body.Moves
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
