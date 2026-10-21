package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/isbe-bot/bottagon/internal/catalog"
)

func mountCatalog(r chi.Router, store catalog.Store) {
	h := catalogHandler{store: store}
	r.Get("/bots", h.listBots)
	r.Post("/bots", h.createBot)
	r.Get("/bots/{id}", h.getBot)
	r.Get("/matches", h.listMatches)
	r.Get("/matches/{id}", h.getMatch)
	r.Get("/exhibition", h.exhibition)
}

type catalogHandler struct {
	store catalog.Store
}

func (h catalogHandler) listBots(w http.ResponseWriter, r *http.Request) {
	limit, offset := pageQuery(r)
	role := r.URL.Query().Get("role")
	switch role {
	case "", "all", "house", "contender":
	default:
		writeCatalogError(w, http.StatusBadRequest, "invalid role")
		return
	}
	if role == "" {
		role = "all"
	}
	page, err := h.store.ListBots(r.Context(), catalog.BotQuery{Role: role, Q: clipQuery(r), Limit: limit, Offset: offset})
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusOK, page)
}

func (h catalogHandler) getBot(w http.ResponseWriter, r *http.Request) {
	bot, err := h.store.GetBot(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, catalog.ErrNotFound) {
		writeCatalogError(w, http.StatusNotFound, "bot not found")
		return
	}
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusOK, bot)
}

func (h catalogHandler) listMatches(w http.ResponseWriter, r *http.Request) {
	limit, offset := pageQuery(r)
	outcome := r.URL.Query().Get("outcome")
	switch outcome {
	case "", "all", "draw", "house", "contender":
	default:
		writeCatalogError(w, http.StatusBadRequest, "invalid outcome")
		return
	}
	stake := 0
	if raw := r.URL.Query().Get("stake"); raw != "" && raw != "all" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			writeCatalogError(w, http.StatusBadRequest, "invalid stake")
			return
		}
		stake = value
	}
	page, err := h.store.ListMatches(r.Context(), catalog.MatchQuery{
		Q: clipQuery(r), Outcome: outcome, Stake: stake, Bot: r.URL.Query().Get("bot"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusOK, page)
}

func (h catalogHandler) getMatch(w http.ResponseWriter, r *http.Request) {
	match, err := h.store.GetMatch(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, catalog.ErrNotFound) {
		writeCatalogError(w, http.StatusNotFound, "match not found")
		return
	}
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusOK, match)
}

func (h catalogHandler) exhibition(w http.ResponseWriter, r *http.Request) {
	item, err := h.store.Exhibition(r.Context())
	if errors.Is(err, catalog.ErrNotFound) {
		writeCatalogError(w, http.StatusNotFound, "exhibition not found")
		return
	}
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusOK, item)
}

func (h catalogHandler) createBot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Persona string `json:"persona"`
		Website string `json:"website"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeCatalogError(w, http.StatusBadRequest, "invalid registration")
		return
	}
	if strings.TrimSpace(body.Website) != "" {
		writeCatalogError(w, http.StatusBadRequest, "invalid registration")
		return
	}
	bot, err := h.store.CreateBot(r.Context(), catalog.CreateBot{Name: body.Name, Email: body.Email, Persona: body.Persona})
	var invalid *catalog.ValidationError
	if errors.As(err, &invalid) {
		status := http.StatusBadRequest
		if invalid.Conflict {
			status = http.StatusConflict
		}
		writeCatalogJSON(w, status, map[string]any{"errors": invalid.Fields})
		return
	}
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "archive unavailable")
		return
	}
	writeCatalogJSON(w, http.StatusCreated, bot)
}

func pageQuery(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if raw := r.URL.Query().Get("page"); raw != "" && r.URL.Query().Get("offset") == "" {
		page, _ := strconv.Atoi(raw)
		if page > 1 {
			limit, _ = catalog.ClampPage(limit, 0)
			offset = (page - 1) * limit
		}
	}
	return limit, offset
}

func clipQuery(r *http.Request) string {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 120 {
		q = q[:120]
	}
	return q
}

func writeCatalogJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeCatalogError(w http.ResponseWriter, status int, message string) {
	writeJSONError(w, status, message)
}
