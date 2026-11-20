// Package journal is the arena's append-only event log (ADR-015): one JSON
// object per line, each with seq, at, type and data, written with fsync. The
// log is the source of truth; state on start is whatever replaying it gives.
package journal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Event is one journal line.
type Event struct {
	Seq  int64           `json:"seq"`
	At   time.Time       `json:"at"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Journal appends events to one file. It is safe for concurrent use.
type Journal struct {
	mu   sync.Mutex
	file *os.File
	seq  int64
	size int64
	err  error // set when the file could not be restored after a failed write
}

// Open opens or creates the journal at path and returns every event in it.
// A torn last line (no trailing newline, or not valid JSON) is what a crash
// mid-append leaves behind, so it is truncated away and logged as
// journal_truncated. Anything wrong before the last line is an error naming
// the line: the journal never silently skips an event. logger may be nil.
func Open(path string, logger *slog.Logger) (*Journal, []Event, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open journal: %w", err)
	}
	events, size, err := read(file, path, logger)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if _, err := file.Seek(size, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("open journal: %w", err)
	}
	j := &Journal{file: file, size: size}
	if len(events) > 0 {
		j.seq = events[len(events)-1].Seq
	}
	return j, events, nil
}

// read parses the journal and returns its events and the length of its intact
// prefix, truncating a torn last line.
func read(file *os.File, path string, logger *slog.Logger) ([]Event, int64, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, fmt.Errorf("read journal: %w", err)
	}
	var events []Event
	var offset int64
	reader := bufio.NewReader(bytes.NewReader(raw))
	for line := 1; ; line++ {
		text, readErr := reader.ReadBytes('\n')
		if len(text) == 0 && readErr == io.EOF {
			return events, offset, nil
		}
		last := readErr == io.EOF || int(offset)+len(text) == len(raw)
		event, err := parse(text, int64(len(events))+1)
		torn := readErr == io.EOF || errors.Is(err, errNotJSON)
		if err == nil && readErr == io.EOF {
			err = errors.New("no trailing newline")
		}
		if err != nil {
			if !last || !torn {
				return nil, 0, fmt.Errorf("journal %s line %d: %w", path, line, err)
			}
			if err := file.Truncate(offset); err != nil {
				return nil, 0, fmt.Errorf("truncate torn journal line %d: %w", line, err)
			}
			if err := file.Sync(); err != nil {
				return nil, 0, fmt.Errorf("truncate torn journal line %d: %w", line, err)
			}
			logger.Warn("journal truncated", "event", "journal_truncated", "path", path, "line", line,
				"bytes", len(text), "reason", err.Error())
			return events, offset, nil
		}
		events = append(events, event)
		offset += int64(len(text))
	}
}

// errNotJSON marks a line that is not a JSON object at all, as a write cut
// short by a crash leaves it.
var errNotJSON = errors.New("not a valid event")

func parse(text []byte, wantSeq int64) (Event, error) {
	var event Event
	if err := json.Unmarshal(bytes.TrimSuffix(text, []byte("\n")), &event); err != nil {
		return Event{}, fmt.Errorf("%w: %w", errNotJSON, err)
	}
	switch {
	case event.Seq != wantSeq:
		return Event{}, fmt.Errorf("seq %d, want %d", event.Seq, wantSeq)
	case event.Type == "":
		return Event{}, errors.New("event has no type")
	case event.At.IsZero():
		return Event{}, errors.New("event has no time")
	case len(event.Data) == 0:
		return Event{}, errors.New("event has no data")
	}
	return event, nil
}

// Append writes one event and returns once it is on disk (fsync). If the
// write fails, the partial line is cut off again so later appends stay
// readable; if even that fails, every later Append returns the error.
func (j *Journal) Append(eventType string, at time.Time, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("journal %s: %w", eventType, err)
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.err != nil {
		return j.err
	}
	line, err := json.Marshal(Event{Seq: j.seq + 1, At: at.UTC(), Type: eventType, Data: payload})
	if err != nil {
		return fmt.Errorf("journal %s: %w", eventType, err)
	}
	line = append(line, '\n')
	if _, err := j.file.Write(line); err != nil {
		return j.rollback(fmt.Errorf("journal %s: %w", eventType, err))
	}
	if err := j.file.Sync(); err != nil {
		return j.rollback(fmt.Errorf("journal %s: %w", eventType, err))
	}
	j.seq++
	j.size += int64(len(line))
	return nil
}

func (j *Journal) rollback(cause error) error {
	if err := j.file.Truncate(j.size); err != nil {
		j.err = fmt.Errorf("journal unusable after a failed write: %w", errors.Join(cause, err))
		return j.err
	}
	if _, err := j.file.Seek(j.size, io.SeekStart); err != nil {
		j.err = fmt.Errorf("journal unusable after a failed write: %w", errors.Join(cause, err))
		return j.err
	}
	return cause
}

// Close closes the journal file. Appends after Close fail.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.err == nil {
		j.err = errors.New("journal is closed")
	}
	return j.file.Close()
}
