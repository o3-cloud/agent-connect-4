// Command arena is the Connect Four Arena game server.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"connect4arena/internal/app"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check whether the service is healthy")
	flag.Parse()

	// BDR-033: every setting, PORT included, is checked before anything is
	// logged, and a bad one stops the server with exit 2 and its name.
	// Docker reserves health check exit code 2, so -healthcheck exits 1.
	cfg, err := app.ConfigFromEnv(os.Getenv)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		if *healthcheck {
			os.Exit(1)
		}
		os.Exit(2)
	}
	addr := ":" + strconv.Itoa(cfg.Port)

	if *healthcheck {
		client := http.Client{Timeout: 2 * time.Second}
		response, err := client.Get("http://127.0.0.1" + addr + "/healthz")
		if err != nil || response.StatusCode != http.StatusNoContent {
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}

	// NFR-006: JSON lines on standard output; Arena Bot's records join them.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg.Logger = logger
	// ADR-015: with ARENA_JOURNAL_PATH, restore every game and agent first. A
	// journal that cannot be read or replayed stops the server, naming the
	// line or event.
	handler, closeJournal, err := app.Open(cfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	// Bind first, so server_started is logged only for a server that listens.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		_ = closeJournal()
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Info("server started", "event", "server_started", "addr", server.Addr)
	err = server.Serve(listener)
	_ = closeJournal()
	if err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
