package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
	"connect4arena/internal/journal"
)

func openJournal(t *testing.T, path string, logger *slog.Logger) (*journal.Journal, []journal.Event) {
	t.Helper()
	j, events, err := journal.Open(path, logger)
	if err != nil {
		t.Fatalf("journal.Open: %v", err)
	}
	t.Cleanup(func() { _ = j.Close() })
	return j, events
}

func appendEvents(t *testing.T, j *journal.Journal, n int) {
	t.Helper()
	for i := range n {
		if err := j.Append("thing_happened", time.Unix(int64(i), 0).UTC(), map[string]int{"n": i}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
}

// ADR-015: one JSON object per line with seq, at, type and data; a reopened
// journal returns every event in order and continues the sequence.
func TestJournalAppendsAndReplays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	j, events := openJournal(t, path, nil)
	if len(events) != 0 {
		t.Fatalf("new journal has %d events", len(events))
	}
	appendEvents(t, j, 3)
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 3 || !strings.HasSuffix(string(raw), "\n") {
		t.Fatalf("journal file = %q, want three newline-terminated lines", raw)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"seq", "at", "type", "data"} {
		if _, ok := first[field]; !ok {
			t.Errorf("journal line has no %q field: %s", field, lines[0])
		}
	}

	j, events = openJournal(t, path, nil)
	if len(events) != 3 {
		t.Fatalf("reopened journal has %d events, want 3", len(events))
	}
	for i, e := range events {
		var data map[string]int
		if err := json.Unmarshal(e.Data, &data); err != nil {
			t.Fatal(err)
		}
		if e.Seq != int64(i+1) || e.Type != "thing_happened" || data["n"] != i || !e.At.Equal(time.Unix(int64(i), 0)) {
			t.Fatalf("event %d = %+v (data %v)", i, e, data)
		}
	}
	appendEvents(t, j, 1)
	_ = j.Close()
	_, events = openJournal(t, path, nil)
	if got := events[len(events)-1].Seq; got != 4 {
		t.Fatalf("appended after reopen with seq %d, want 4", got)
	}
}

// ADR-015 step 4: a torn last line (cut off mid-object, or not valid JSON) is
// truncated away and logged, every complete event is kept, and the next
// append continues from there.
func TestJournalDropsATornLastLine(t *testing.T) {
	for name, tail := range map[string]string{
		"no newline":   `{"seq":3,"at":"2026-11-20T00:00:00Z","ty`,
		"invalid json": "{\"seq\":3,\"at\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "arena.jsonl")
			j, _ := openJournal(t, path, nil)
			appendEvents(t, j, 2)
			_ = j.Close()
			intact, _ := os.ReadFile(path)
			if err := os.WriteFile(path, append(append([]byte{}, intact...), tail...), 0o600); err != nil {
				t.Fatal(err)
			}

			var logs bytes.Buffer
			j, events := openJournal(t, path, slog.New(slog.NewJSONHandler(&logs, nil)))
			if len(events) != 2 {
				t.Fatalf("restored %d events, want the 2 complete ones", len(events))
			}
			if !strings.Contains(logs.String(), `"event":"journal_truncated"`) || !strings.Contains(logs.String(), `"line":3`) {
				t.Fatalf("truncation not logged with its line: %s", logs.String())
			}
			if now, _ := os.ReadFile(path); !bytes.Equal(now, intact) {
				t.Fatalf("journal after recovery = %q, want the intact prefix %q", now, intact)
			}
			appendEvents(t, j, 1)
			_ = j.Close()
			_, events = openJournal(t, path, nil)
			if len(events) != 3 || events[2].Seq != 3 {
				t.Fatalf("after recovery and one append: %+v", events)
			}
		})
	}
}

// ADR-015 step 4: corruption anywhere but the last line stops start-up with an
// error naming the line; the server never silently skips an event.
func TestJournalRefusesCorruptionBeforeTheLastLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	j, _ := openJournal(t, path, nil)
	appendEvents(t, j, 3)
	_ = j.Close()
	raw, _ := os.ReadFile(path)
	lines := strings.SplitAfter(string(raw), "\n")

	for name, corrupt := range map[string]string{
		"invalid json": lines[0] + "not json\n" + lines[2],
		"sequence gap": lines[0] + lines[2],
		"missing type": lines[0] + `{"seq":2,"at":"2026-11-20T00:00:00Z","data":{}}` + "\n" + lines[2],
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
				t.Fatal(err)
			}
			j, _, err := journal.Open(path, nil)
			if err == nil {
				_ = j.Close()
				t.Fatal("opened a journal corrupted before its last line")
			}
			if !strings.Contains(err.Error(), "line 2") {
				t.Fatalf("error %q does not name line 2", err)
			}
		})
	}
}

func TestJournalOpenFailsForAnUnusablePath(t *testing.T) {
	if _, _, err := journal.Open(filepath.Join(t.TempDir(), "missing", "dir", "arena.jsonl"), nil); err == nil {
		t.Fatal("opened a journal in a directory that does not exist")
	}
}

// failingStore refuses every write after the first `allow` ones.
type failingStore struct{ allow int }

func (f *failingStore) Append(string, time.Time, any) error {
	if f.allow == 0 {
		return errors.New("disk full")
	}
	f.allow--
	return nil
}

// ADR-015 step 3: a change is applied only once it is journaled, so a failed
// write is an INTERNAL error that leaves the game exactly as it was.
func TestFailedJournalWriteChangesNothing(t *testing.T) {
	store := &failingStore{allow: 3} // create and two joins
	service := game.NewService(game.WithStore(store))
	g, err := service.Create(game.ModeAgentVsAgent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Join(g.ID, "agt_r", engine.Red, "R", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Join(g.ID, "agt_y", engine.Yellow, "Y", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	before, _ := service.Get(g.ID)

	_, err = service.MoveWithKey(g.ID, "agt_r", 3, 0, "k1")
	var gameErr *game.Error
	if !errors.As(err, &gameErr) || gameErr.Code != game.CodeInternal {
		t.Fatalf("move with a failing journal: err = %v, want INTERNAL", err)
	}
	if _, err := service.Resign(g.ID, "agt_y"); !errors.As(err, &gameErr) || gameErr.Code != game.CodeInternal {
		t.Fatalf("resign with a failing journal: err = %v, want INTERNAL", err)
	}
	if _, err := service.Create(game.ModeAgentVsAgent); err == nil {
		t.Fatal("create succeeded with a failing journal")
	}
	if after, _ := service.Get(g.ID); !reflect.DeepEqual(before, after) {
		t.Fatalf("a failed write changed the game:\nbefore %+v\nafter  %+v", before, after)
	}
	if n := len(service.List()); n != 1 {
		t.Fatalf("%d games listed, want 1", n)
	}

	store.allow = 1
	if result, err := service.MoveWithKey(g.ID, "agt_r", 3, 0, "k1"); err != nil || result.Move.Row != 5 || result.Version != 1 {
		t.Fatalf("retry once the journal works: %+v, %v; want the disc at the bottom, version 1", result, err)
	}

	registry := identity.NewRegistry().WithStore(&failingStore{})
	if _, key, err := registry.Register("A", "agent"); err == nil || key != "" {
		t.Fatalf("Register with a failing journal = %q, %v; want an error and no key", key, err)
	}
}
