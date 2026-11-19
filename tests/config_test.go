package tests

import (
	"bytes"
	"errors"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"connect4arena/internal/app"
)

// BDR-033, invalid-port-not-named-at-startup: PORT is validated with the
// other settings, so a bad value is reported by name before anything starts.
func TestPortConfigFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

	cfg, err := app.ConfigFromEnv(env(nil))
	if err != nil || cfg.Port != 8080 {
		t.Fatalf("default port = %d, %v; want 8080", cfg.Port, err)
	}
	cfg, err = app.ConfigFromEnv(env(map[string]string{"PORT": "9000"}))
	if err != nil || cfg.Port != 9000 {
		t.Fatalf("PORT=9000 gives %d, %v", cfg.Port, err)
	}
	for _, bad := range []string{"abc", "0", "-1", "65536", "70000", "80 80", "http"} {
		_, err := app.ConfigFromEnv(env(map[string]string{"PORT": bad}))
		want := `PORT must be a port number from 1 to 65535, got "` + bad + `"`
		if err == nil || err.Error() != want {
			t.Errorf("PORT=%q: err = %v, want %q", bad, err, want)
		}
	}
}

// buildArena builds bin/arena into a temporary directory.
func buildArena(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "arena")
	if out, err := exec.Command("go", "build", "-o", bin, "../cmd/arena").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// runArena runs the arena binary with env until it exits and returns its
// exit code, standard output and standard error.
func runArena(t *testing.T, bin string, env ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, stdout.String(), stderr.String()
	case errors.As(err, &exit):
		return exit.ExitCode(), stdout.String(), stderr.String()
	default:
		t.Fatalf("run arena: %v", err)
		return 0, "", ""
	}
}

// The server exits 2 naming PORT, like the other settings, and never logs
// server_started for an invalid port.
func TestArenaRejectsInvalidPortByName(t *testing.T) {
	if testing.Short() {
		t.Skip("builds bin/arena")
	}
	bin := buildArena(t)
	for _, bad := range []string{"abc", "70000"} {
		code, stdout, stderr := runArena(t, bin, "PORT="+bad)
		if code != 2 {
			t.Errorf("PORT=%s: exit %d, want 2", bad, code)
		}
		want := `PORT must be a port number from 1 to 65535, got "` + bad + `"`
		if !strings.Contains(stderr, want) {
			t.Errorf("PORT=%s: stderr = %q, want it to contain %q", bad, stderr, want)
		}
		if strings.Contains(stdout, "server_started") {
			t.Errorf("PORT=%s: logged server_started for a server that never listened:\n%s", bad, stdout)
		}
	}
}

// server_started is logged only once the listener is bound: a port already in
// use fails without it.
func TestArenaLogsServerStartedOnlyAfterBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("builds bin/arena")
	}
	bin := buildArena(t)
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = busy.Close() }()
	port := strconv.Itoa(busy.Addr().(*net.TCPAddr).Port)

	code, stdout, stderr := runArena(t, bin, "PORT="+port)
	if code == 0 {
		t.Fatalf("arena on busy port %s exited 0", port)
	}
	if strings.Contains(stdout, "server_started") {
		t.Errorf("logged server_started though port %s was in use:\n%s", port, stdout)
	}
	if !strings.Contains(stderr, port) {
		t.Errorf("stderr = %q, want it to name port %s", stderr, port)
	}
}
