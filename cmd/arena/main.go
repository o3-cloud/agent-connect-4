// Command arena is the Connect Four Arena game server.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connect4arena/internal/app"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check whether the service is healthy")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if *healthcheck {
		client := http.Client{Timeout: 2 * time.Second}
		response, err := client.Get("http://127.0.0.1:" + port + "/healthz")
		if err != nil || response.StatusCode != http.StatusNoContent {
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}

	cfg, err := app.ConfigFromEnv(os.Getenv)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	// NFR-006: JSON lines on standard output; Arena Bot's records join them.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg.Logger = logger
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.New(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("server started", "event", "server_started", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
