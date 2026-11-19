package tests

import (
	"bytes"
	"net/http"
	"slices"
	"strings"
	"testing"

	"connect4arena/internal/cli"
)

// openAPI is the published document, with helpers to look things up the way
// an agent that knows nothing else would.
type openAPI struct {
	t   *testing.T
	doc map[string]any
}

func fetchOpenAPI(t *testing.T, c client) openAPI {
	t.Helper()
	r := c.do(http.MethodGet, "/api/v1/openapi.json", nil).wantStatus(t, http.StatusOK)
	return openAPI{t: t, doc: r.body}
}

func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }

// operation returns the method, path and definition of operationID.
func (o openAPI) operation(operationID string) (string, string, map[string]any) {
	o.t.Helper()
	for path, ops := range obj(o.doc["paths"]) {
		for method, op := range obj(ops) {
			if obj(op)["operationId"] == operationID {
				return strings.ToUpper(method), path, obj(op)
			}
		}
	}
	o.t.Fatalf("OpenAPI document has no operation %q", operationID)
	return "", "", nil
}

// schema resolves a schema or a local $ref to one.
func (o openAPI) schema(s map[string]any) map[string]any {
	if ref, ok := s["$ref"].(string); ok {
		return obj(obj(obj(o.doc["components"])["schemas"])[strings.TrimPrefix(ref, "#/components/schemas/")])
	}
	return s
}

func (o openAPI) requestSchema(op map[string]any) map[string]any {
	return o.schema(obj(obj(obj(obj(op["requestBody"])["content"])["application/json"])["schema"]))
}

func (o openAPI) responseSchema(op map[string]any, status string) map[string]any {
	return o.schema(obj(obj(obj(obj(obj(op["responses"])[status])["content"])["application/json"])["schema"]))
}

// enumValue returns want if the schema property lists it in its enum.
func (o openAPI) enumValue(s map[string]any, property, want string) string {
	o.t.Helper()
	for _, v := range obj(obj(s["properties"])[property])["enum"].([]any) {
		if v == want {
			return want
		}
	}
	o.t.Fatalf("schema property %s does not offer %q", property, want)
	return ""
}

func (o openAPI) wantProperties(s map[string]any, names ...string) {
	o.t.Helper()
	for _, name := range names {
		if _, ok := obj(s["properties"])[name]; !ok {
			o.t.Fatalf("schema lacks property %q: %v", name, s)
		}
	}
}

var bdr022Operations = map[string]bool{ // operationId -> requires a bearer key
	"registerAgent": false, "listGames": false, "createGame": true, "getGame": false,
	"joinGame": true, "submitMove": true, "getGameHistory": false, "resignGame": true,
}

// BDR-022 scenario 1, AC-1: every public operation, its inputs, responses and
// the error shape are in the document.
func TestOpenAPIDescribesEveryOperation(t *testing.T) {
	a := newArena(t)
	o := fetchOpenAPI(t, a.anon)
	if v, _ := o.doc["openapi"].(string); !strings.HasPrefix(v, "3.") {
		t.Fatalf("openapi = %v, want 3.x", o.doc["openapi"])
	}
	info, _ := obj(o.doc["info"])["description"].(string)
	for _, want := range []string{"registerAgent", "Bearer", "createGame", "joinGame", "getGame", "submitMove", "expectedVersion", "/mcp"} {
		if !strings.Contains(info, want) {
			t.Errorf("info.description does not explain %q:\n%s", want, info)
		}
	}

	errorBody := o.schema(obj(obj(o.schema(map[string]any{"$ref": "#/components/schemas/ErrorResponse"})["properties"])["error"]))
	o.wantProperties(errorBody, "code", "message", "retryable", "gameId", "currentVersion", "retryAfterMs")
	codes := obj(obj(errorBody["properties"])["code"])["enum"].([]any)
	for _, code := range []string{"INVALID_ARGUMENT", "UNAUTHENTICATED", "FORBIDDEN", "GAME_NOT_FOUND", "NOT_YOUR_TURN",
		"STALE_STATE", "GAME_NOT_STARTED", "GAME_COMPLETED", "INVALID_COLUMN", "COLUMN_FULL", "RATE_LIMITED", "INTERNAL"} {
		if !slices.Contains(codes, any(code)) {
			t.Errorf("error code enum lacks %s", code)
		}
	}

	for id, authed := range bdr022Operations {
		method, path, op := o.operation(id)
		if op["summary"] == nil || op["description"] == nil {
			t.Errorf("%s %s (%s) lacks a summary or description", method, path, id)
		}
		if got := op["security"] != nil; got != authed {
			t.Errorf("%s declares security = %v, want %v", id, got, authed)
		}
		if method == http.MethodPost && id != "resignGame" && len(o.requestSchema(op)) == 0 {
			t.Errorf("%s has no request body schema", id)
		}
		responses := obj(op["responses"])
		ok := false
		for status := range responses {
			if strings.HasPrefix(status, "2") && len(o.responseSchema(op, status)) > 0 {
				ok = true
			}
		}
		if !ok {
			t.Errorf("%s has no 2xx response schema", id)
		}
		statuses := []string{"429", "500"}
		if authed {
			statuses = append(statuses, "401")
		}
		for _, status := range statuses {
			s := obj(obj(obj(obj(responses[status])["content"])["application/json"])["schema"])
			if s["$ref"] != "#/components/schemas/ErrorResponse" {
				t.Errorf("%s response %s does not use ErrorResponse: %v", id, status, responses[status])
			}
		}
		if obj(obj(obj(responses["429"])["headers"])["Retry-After"]) == nil {
			t.Errorf("%s 429 response does not document Retry-After", id)
		}
	}
}

// BDR-022 scenario 2, AC-2: help names every command, and each command's
// help shows its arguments and flags.
func TestCLIHelpListsEveryCommandAndItsFlags(t *testing.T) {
	help := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		command := cli.NewRootCommand()
		command.SetOut(&out)
		command.SetErr(&out)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatalf("connect4 %v: %v", args, err)
		}
		return out.String()
	}
	root := help("--help")
	usage := map[string][]string{
		"register": {"--name"},
		"create":   {"--mode"},
		"join":     {"<gameId>", "--disc", "--name"},
		"state":    {"<gameId>"},
		"move":     {"<gameId>", "--column", "--expected-version", "--idempotency-key"},
		"history":  {"<gameId>"},
		"watch":    {"<gameId>", "--interval"},
		"resign":   {"<gameId>"},
	}
	for command, wants := range usage {
		if !strings.Contains(root, "\n  "+command+" ") {
			t.Errorf("connect4 --help does not list %s:\n%s", command, root)
		}
		page := help(command, "--help")
		for _, want := range append(wants, "--json", "--server", "--api-key") {
			if !strings.Contains(page, want) {
				t.Errorf("connect4 %s --help lacks %s:\n%s", command, want, page)
			}
		}
	}
}

// BDR-022 AC-4: an agent that knows only the OpenAPI document registers,
// creates, joins and finishes a game. Every path, field and enum value it uses
// is looked up in the document first.
func TestAgentPlaysUsingOnlyOpenAPIDocument(t *testing.T) {
	server := newServer(t)
	anon := client{t: t, server: server}
	o := fetchOpenAPI(t, anon)

	call := func(c client, id string, params map[string]string, body map[string]any) map[string]any {
		t.Helper()
		method, path, op := o.operation(id)
		for name, value := range params {
			path = strings.ReplaceAll(path, "{"+name+"}", value)
		}
		if body != nil {
			s := o.requestSchema(op)
			for _, required := range s["required"].([]any) {
				if _, ok := body[required.(string)]; !ok {
					t.Fatalf("%s: agent omitted required field %v", id, required)
				}
			}
			o.wantProperties(s, mapKeys(body)...)
		}
		var payload any
		if body != nil {
			payload = body
		}
		r := c.do(method, path, payload)
		if r.status/100 != 2 {
			t.Fatalf("%s %s: %d %s", method, path, r.status, r.raw)
		}
		return r.body
	}

	_, _, registerOp := o.operation("registerAgent")
	registered := o.responseSchema(registerOp, "201")
	o.wantProperties(registered, "apiKey")
	regBody := o.requestSchema(registerOp)
	players := map[string]client{}
	for _, name := range []string{"Doc-Red", "Doc-Yellow"} {
		out := call(anon, "registerAgent", nil, map[string]any{"displayName": name, "type": o.enumValue(regBody, "type", "agent")})
		players[name] = client{t: t, server: server, token: out["apiKey"].(string)}
	}

	_, _, createOp := o.operation("createGame")
	created := call(players["Doc-Red"], "createGame", nil,
		map[string]any{"mode": o.enumValue(o.requestSchema(createOp), "mode", "agent_vs_agent")})
	o.wantProperties(o.responseSchema(createOp, "201"), "gameId")
	gameID := created["gameId"].(string)
	ids := map[string]string{"gameId": gameID}

	_, _, joinOp := o.operation("joinGame")
	joinBody := o.requestSchema(joinOp)
	for name, disc := range map[string]string{"Doc-Red": "red", "Doc-Yellow": "yellow"} {
		call(players[name], "joinGame", ids, map[string]any{
			"disc": o.enumValue(joinBody, "disc", disc), "displayName": name, "type": o.enumValue(joinBody, "type", "agent"),
		})
	}

	_, _, getOp := o.operation("getGame")
	o.wantProperties(o.responseSchema(getOp, "200"), "status", "currentTurn", "legalMoves", "version", "winner", "resultReason")
	byDisc := map[string]client{"red": players["Doc-Red"], "yellow": players["Doc-Yellow"]}
	for range 42 {
		state := call(anon, "getGame", ids, nil)
		if state["status"] == "completed" {
			break
		}
		legal := ints(state["legalMoves"])
		call(byDisc[state["currentTurn"].(string)], "submitMove", ids,
			map[string]any{"column": legal[0], "expectedVersion": number(state["version"])})
	}
	final := call(anon, "getGame", ids, nil)
	if final["status"] != "completed" || final["resultReason"] == nil {
		t.Fatalf("game did not complete: %v", final)
	}
	if history := call(anon, "getGameHistory", ids, nil); len(history["moves"].([]any)) != number(final["moveNumber"]) {
		t.Fatalf("history has %d moves, game reports %v", len(history["moves"].([]any)), final["moveNumber"])
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
