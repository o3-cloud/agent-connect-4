package tests

import (
	"net/http"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// BDR-031 AC-1 with a journal (ADR-015, journal-fsync-serializes-all-games):
// 100 games played at once, each move journaled with fsync, keep the move
// p95 within 200 ms. Every game's moves share one journal file, so this fails
// if each append waits for every other game's fsync in turn.
func TestJournaledConcurrentGamesMeetMoveLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 100 games")
	}
	const games, movesPerGame = 100, 10
	server, _ := journalServer(t, filepath.Join(t.TempDir(), "arena.jsonl"))
	arenas := make([]*arena, games)
	for i := range arenas {
		arenas[i] = &arena{
			server: server,
			red:    register(t, server, "Red"),
			yellow: register(t, server, "Yellow"),
			anon:   client{t: t, server: server},
		}
		arenas[i].start(t)
	}

	var mu sync.Mutex
	var latencies []time.Duration
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, a := range arenas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			// Column v%7 alternates colours along the bottom row: no win in 10 moves.
			for v := range movesPerGame {
				mover := a.red
				if v%2 == 1 {
					mover = a.yellow
				}
				began := time.Now()
				status := a.move(t, mover, v%7, v, "").status
				took := time.Since(began)
				if status != http.StatusOK {
					t.Errorf("game %s move %d: status %d", a.gameID, v+1, status)
					return
				}
				mu.Lock()
				latencies = append(latencies, took)
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(latencies) != games*movesPerGame {
		t.Fatalf("%d moves accepted, want %d", len(latencies), games*movesPerGame)
	}
	slices.Sort(latencies)
	p50, p95 := latencies[len(latencies)/2], latencies[len(latencies)*95/100]
	t.Logf("%d journaled moves: p50 %s, p95 %s", len(latencies), p50, p95)
	if p95 > 200*time.Millisecond {
		t.Fatalf("move p95 = %s with a journal, want ≤ 200ms (BDR-031 AC-1)", p95)
	}
}
