// Package ratelimit counts requests per client in fixed time windows
// (SPEC.md SEC-007). Each client has its own budget, so one noisy client
// never slows another.
package ratelimit

import (
	"sync"
	"time"
)

// sweepAt is the number of tracked clients above which expired windows are
// dropped, bounding memory.
const sweepAt = 10000

// Limiter allows up to limit requests per client in each window.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	clients map[string]*bucket
}

type bucket struct {
	start time.Time
	count int
}

// New returns a limiter allowing limit requests per window per client.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, clients: map[string]*bucket{}}
}

// Allow records a request from client. It reports whether the request is
// within the client's budget and, if not, how long until the window resets.
func (l *Limiter) Allow(client string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.clients[client]
	if b == nil || now.Sub(b.start) >= l.window {
		if b == nil && len(l.clients) >= sweepAt {
			l.sweep(now)
		}
		b = &bucket{start: now}
		l.clients[client] = b
	}
	if b.count >= l.limit {
		return false, b.start.Add(l.window).Sub(now)
	}
	b.count++
	return true, 0
}

func (l *Limiter) sweep(now time.Time) {
	for client, b := range l.clients {
		if now.Sub(b.start) >= l.window {
			delete(l.clients, client)
		}
	}
}
