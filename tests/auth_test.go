package tests

import (
	"net/http"
	"strings"
	"testing"
)

// ADR-013, API-009: registration issues a distinct, one-time API key per agent.
func TestAPIRegisterAgentIssuesKey(t *testing.T) {
	server := newServer(t)
	anon := client{t: t, server: server}
	first := anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": "Alpha", "type": "agent"}).
		wantStatus(t, http.StatusCreated).body
	second := anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": "Alpha", "type": "human"}).
		wantStatus(t, http.StatusCreated).body

	for _, issued := range []map[string]any{first, second} {
		if id, _ := issued["agentId"].(string); !strings.HasPrefix(id, "agt_") {
			t.Fatalf("agentId = %v", issued["agentId"])
		}
		if key, _ := issued["apiKey"].(string); !strings.HasPrefix(key, "c4_") || len(key) < 40 {
			t.Fatalf("apiKey = %v", issued["apiKey"])
		}
		if issued["displayName"] != "Alpha" {
			t.Fatalf("displayName = %v", issued["displayName"])
		}
	}
	if first["agentId"] == second["agentId"] || first["apiKey"] == second["apiKey"] {
		t.Fatalf("two registrations share an identity: %v %v", first, second)
	}
	if second["type"] != "human" {
		t.Fatalf("type = %v, want human", second["type"])
	}

	anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": "", "type": "agent"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": "Robot", "type": "alien"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
}

// BDR-013 AC-1..3 and BDR-007 AC-3: missing or unrecognised credentials are
// rejected on create, join and move, and nothing changes.
func TestAPIRejectsUnrecognisedCredentials(t *testing.T) {
	a := newArena(t)
	a.start(t)
	forged := client{t: t, server: a.server, token: "c4_" + strings.Repeat("A", 43)}
	madeUp := client{t: t, server: a.server, token: "token-alpha"}
	path := "/api/v1/games/" + a.gameID

	for _, c := range []client{forged, madeUp, a.anon} {
		c.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent"}).
			wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
		c.do(http.MethodPost, path+"/moves", map[string]any{"column": 3, "expectedVersion": 0}).
			wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
		c.do(http.MethodPost, path+"/resign", nil).
			wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
	}

	started := a.gameID
	waiting := a.create(t)
	a.gameID = started
	forged.do(http.MethodPost, "/api/v1/games/"+waiting+"/join",
		map[string]any{"disc": "red", "displayName": "Forger", "type": "agent"}).
		wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
	open := a.anon.do(http.MethodGet, "/api/v1/games/"+waiting, nil).wantStatus(t, http.StatusOK).body
	if len(open["openSlots"].([]any)) != 2 {
		t.Fatalf("forged join took a slot: %v", open["openSlots"])
	}

	list := a.anon.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK).body
	if n := len(list["games"].([]any)); n != 2 {
		t.Fatalf("unauthenticated create added a game: %d games", n)
	}
	state := a.state(t)
	if number(state["version"]) != 0 || state["currentTurn"] != "red" {
		t.Fatalf("rejected moves changed the game: %v", state)
	}
}

// BDR-013 AC-4: key strings never appear in any client-visible output.
func TestAPIKeysNeverLeak(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	path := "/api/v1/games/" + a.gameID

	outputs := []response{
		a.anon.do(http.MethodGet, path, nil),
		a.anon.do(http.MethodGet, "/api/v1/games", nil),
		a.red.do(http.MethodPost, path+"/moves", map[string]any{"column": 4, "expectedVersion": 7}),
		a.other.do(http.MethodPost, path+"/resign", nil),
		a.anon.do(http.MethodGet, "/api/v1/openapi.json", nil),
	}
	for _, out := range outputs {
		for _, key := range []string{a.red.token, a.yellow.token, a.other.token} {
			if strings.Contains(out.raw, key) {
				t.Fatalf("an API key appears in output: %s", out.raw)
			}
		}
	}
}
