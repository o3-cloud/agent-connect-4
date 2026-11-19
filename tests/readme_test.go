package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readmeSection returns the README text under heading, up to the next "## ".
func readmeSection(t *testing.T, heading string) string {
	t.Helper()
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	start := strings.Index(text, "\n"+heading+"\n")
	if start < 0 {
		t.Fatalf("README has no %q section", heading)
	}
	section := text[start+len(heading)+2:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	return section
}

var shBlock = regexp.MustCompile("(?s)```sh\n(.*?)```")

// readme-cli-walkthrough-one-player, BDR-019, BDR-030: a new user who pastes
// the README's CLI blocks, in order, against a running server gets no errors,
// and the walkthrough game ends completed. The blocks run as written with
// `set -eu`, against a freshly built bin/connect4.
func TestREADMECLIWalkthroughRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("builds bin/connect4")
	}
	blocks := shBlock.FindAllStringSubmatch(readmeSection(t, "## Playing with the `connect4` CLI"), -1)
	if len(blocks) == 0 {
		t.Fatal("the CLI section has no sh blocks")
	}
	script := "set -eu\n"
	for _, block := range blocks {
		script += block[1]
	}

	dir := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "bin", "connect4"), "../cmd/connect4")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build connect4: %v\n%s", err, out)
	}
	server := newServer(t)
	run := exec.Command("sh", "-c", script)
	run.Dir = dir
	run.Env = append(os.Environ(), "CONNECT4_SERVER="+server.URL)
	run.Env = append(run.Env, "CONNECT4_API_KEY=") // the walkthrough must set its own keys
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("README CLI walkthrough failed: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	if !strings.Contains(string(out), `"event":"game_completed"`) {
		t.Fatalf("the walkthrough game did not end completed; output:\n%s", out)
	}
}
