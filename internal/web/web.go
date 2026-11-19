// Package web serves the browser UI (UI-*, ADR-011, ADR-012): a dashboard to
// start a game against Arena Bot (ADR-014) and a game page whose board is an
// htmx fragment. Rendering and rules stay on the server; the page only shows
// what the game service reports and offers column controls to the human who
// holds the turn (UI-003, UI-005).
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"unicode/utf8"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
)

//go:embed templates/*.html
var templateFS embed.FS

// static holds the vendored htmx release and, after `make build`, the
// Tailwind CSS (ADR-011, ADR-012).
//
//go:embed static
var staticFS embed.FS

var pages = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// sessionCookie holds the browser player's API key (ADR-013). It is HttpOnly,
// so page scripts never see it, and SameSite=Lax, so other sites cannot post
// moves with it.
const sessionCookie = "c4_session"

type handler struct {
	service  *game.Service
	registry *identity.Registry
	house    func(gameID string)
}

// Register adds the browser routes to mux. house starts Arena Bot for a game.
func Register(mux *http.ServeMux, service *game.Service, registry *identity.Registry, house func(gameID string)) {
	h := handler{service: service, registry: registry, house: house}
	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", h.dashboard)
	mux.HandleFunc("POST /play", h.play)
	mux.HandleFunc("GET /games/{gameId}", h.gamePage)
	mux.HandleFunc("GET /games/{gameId}/board", h.board)
	mux.HandleFunc("POST /games/{gameId}/moves/{column}", h.move)
}

func (h handler) viewer(r *http.Request) (identity.Agent, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return identity.Agent{}, false
	}
	return h.registry.Authenticate(c.Value)
}

func (h handler) dashboard(w http.ResponseWriter, r *http.Request) {
	name := ""
	if agent, ok := h.viewer(r); ok {
		name = agent.DisplayName
	}
	render(w, http.StatusOK, "dashboard", map[string]any{"Name": name})
}

// play is BDR-026 AC-1: create a human_vs_agent game, take the chosen disc,
// and let Arena Bot take the other.
func (h handler) play(w http.ResponseWriter, r *http.Request) {
	name, disc := r.PostFormValue("name"), engine.Disc(r.PostFormValue("disc"))
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 || (disc != engine.Red && disc != engine.Yellow) {
		render(w, http.StatusBadRequest, "dashboard", map[string]any{
			"Name": name, "Error": "Enter a name of 1 to 64 characters and choose red or yellow.",
		})
		return
	}
	agent, ok := h.viewer(r)
	if !ok {
		var key string
		var err error
		if agent, key, err = h.registry.Register(name, "human"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: key, Path: "/", HttpOnly: true,
			SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: 30 * 24 * 3600})
	}
	created, err := h.service.Create(game.ModeHumanVsAgent)
	if err == nil {
		_, err = h.service.Join(created.ID, agent.ID, disc, name, game.PlayerHuman)
	}
	if err != nil {
		http.Error(w, "could not start the game: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.house(created.ID)
	http.Redirect(w, r, "/games/"+created.ID, http.StatusSeeOther)
}

func (h handler) gamePage(w http.ResponseWriter, r *http.Request) {
	g, err := h.service.Get(r.PathValue("gameId"))
	if err != nil {
		http.Error(w, "No game has this identifier.", http.StatusNotFound)
		return
	}
	render(w, http.StatusOK, "game", h.boardView(r, g))
}

func (h handler) board(w http.ResponseWriter, r *http.Request) {
	g, err := h.service.Get(r.PathValue("gameId"))
	if err != nil {
		http.Error(w, "No game has this identifier.", http.StatusNotFound)
		return
	}
	render(w, http.StatusOK, "board", h.boardView(r, g))
}

// move is BDR-026 AC-2 and AC-3: a click submits a move only when the viewer
// holds the turn and the column is legal; otherwise nothing is sent and the
// current board comes back.
func (h handler) move(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("gameId")
	g, err := h.service.Get(id)
	if err != nil {
		http.Error(w, "No game has this identifier.", http.StatusNotFound)
		return
	}
	column, err := strconv.Atoi(r.PathValue("column"))
	if agent, ok := h.viewer(r); ok && err == nil && canMove(g, agent.ID) && legal(g, column) {
		_, _ = h.service.Move(id, agent.ID, column, g.Version) // a lost race simply re-renders the latest state
		g, _ = h.service.Get(id)
	}
	render(w, http.StatusOK, "board", h.boardView(r, g))
}

func canMove(g game.Game, playerID string) bool {
	if g.Status != game.StatusInProgress {
		return false
	}
	p := playerFor(g, g.CurrentTurn)
	return p != nil && p.ID == playerID && p.Type == game.PlayerHuman
}

func legal(g game.Game, column int) bool {
	for _, c := range g.LegalMoves {
		if c == column {
			return true
		}
	}
	return false
}

func playerFor(g game.Game, disc engine.Disc) *game.Player {
	switch disc {
	case engine.Red:
		return g.Red
	case engine.Yellow:
		return g.Yellow
	}
	return nil
}

type cellView struct {
	Row, Col int
	Disc     string
}

type columnView struct {
	Index    int
	Playable bool
}

type playerView struct {
	Disc, Name, Type string
	You, ToMove      bool
}

type boardView struct {
	ID, Status, StatusText string
	Version                int
	Live, Spectating       bool
	Cells                  []cellView
	Columns                []columnView
	Players                []playerView
}

func (h handler) boardView(r *http.Request, g game.Game) boardView {
	agent, signedIn := h.viewer(r)
	isPlayer := func(p *game.Player) bool { return signedIn && p != nil && p.ID == agent.ID }
	v := boardView{ID: g.ID, Status: string(g.Status), Version: g.Version, Live: g.Status != game.StatusCompleted,
		Spectating: !isPlayer(g.Red) && !isPlayer(g.Yellow)}

	for r := 0; r < engine.Rows; r++ {
		for c := 0; c < engine.Columns; c++ {
			v.Cells = append(v.Cells, cellView{Row: r, Col: c, Disc: string(g.Board[r][c])})
		}
	}
	mine := signedIn && canMove(g, agent.ID)
	for c := 0; c < engine.Columns; c++ {
		v.Columns = append(v.Columns, columnView{Index: c, Playable: mine && legal(g, c)})
	}
	for _, disc := range []engine.Disc{engine.Red, engine.Yellow} {
		pv := playerView{Disc: string(disc), Name: "Waiting for a player", ToMove: g.Status == game.StatusInProgress && g.CurrentTurn == disc}
		if p := playerFor(g, disc); p != nil {
			pv.Name, pv.Type, pv.You = p.DisplayName, string(p.Type), isPlayer(p)
		}
		v.Players = append(v.Players, pv)
	}

	name := func(disc engine.Disc) string {
		if p := playerFor(g, disc); p != nil {
			return p.DisplayName
		}
		return string(disc)
	}
	switch g.Status {
	case game.StatusWaiting:
		v.StatusText = "Waiting for an opponent to join"
	case game.StatusInProgress:
		switch {
		case mine:
			v.StatusText = "Your turn"
		case v.Spectating:
			v.StatusText = "Spectating: " + name(g.CurrentTurn) + " (" + string(g.CurrentTurn) + ") to move"
		default:
			v.StatusText = name(g.CurrentTurn) + "'s turn"
		}
	case game.StatusCompleted:
		switch {
		case g.Winner == engine.Empty:
			v.StatusText = "Draw"
		case isPlayer(playerFor(g, g.Winner)):
			v.StatusText = "You win"
		default:
			v.StatusText = name(g.Winner) + " wins"
		}
		if g.ResultReason == game.ReasonResignation {
			v.StatusText += " by resignation"
		}
	}
	return v
}

func render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pages.ExecuteTemplate(w, name, data)
}
