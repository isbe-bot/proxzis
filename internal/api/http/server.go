package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/conduct"
	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

// Server handles the HTTP REST API for BOTTAGON contestants.
type Server struct {
	queue        *matchmaker.Queue
	moderator    *conduct.Moderator
	queueTimeout time.Duration
	catalog      catalog.Store
}

// SetCatalog attaches the spectator archive. Call it before Routes.
func (s *Server) SetCatalog(store catalog.Store) {
	s.catalog = store
}

// SetQueueTimeout bounds matchmaking long-polls. Zero keeps the production
// wait. The queue clamps values outside (0, MaxQueueWaitTime].
func (s *Server) SetQueueTimeout(d time.Duration) {
	s.queueTimeout = d
}

// NewServer creates a new Server instance.
func NewServer(q *matchmaker.Queue) *Server {
	return &Server{
		queue:     q,
		moderator: conduct.NewModerator(),
	}
}

// Routes registers the /v1 endpoints.
func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(boundaryHeaders)
	r.Use(http.NewCrossOriginProtection().Handler)

	r.Post("/match/find", s.handleFindMatch)
	r.Post("/match/turn", s.handleCommitTurn)
	r.Get("/match/current", s.handleGetCurrentMatch)
	r.Post("/match/forfeit", s.handleForfeitMatch)
	r.Get("/bot/status", s.handleGetBotStatus)
	r.Post("/report", s.handleReport)
	r.Get("/rules", s.handleGetRules)
	if s.catalog != nil {
		mountCatalog(r, s.catalog)
	}

	return r
}

type FindMatchRequest struct {
	PreferredStake int `json:"preferred_stake"`
	MaxStake       int `json:"max_stake"`
}

type TurnRequest struct {
	Stance  string `json:"stance"`
	SideBet int    `json:"side_bet"`
	Say     string `json:"say"`
}

type TurnResponse struct {
	MatchID        string             `json:"match_id"`
	SequenceID     int                `json:"sequence_id"`
	Round          int                `json:"round"`
	Scores         ScoresResponse     `json:"scores"`
	BaseStake      int                `json:"base_stake"`
	YourSideBet    int                `json:"your_side_bet"`
	ActiveCard     ActiveCardResponse `json:"active_card"`
	Opponent       OpponentResponse   `json:"opponent"`
	OpponentBanter string             `json:"opponent_banter"`
	ValidActions   []string           `json:"valid_actions"`
	Status         string             `json:"status"`
	Winner         string             `json:"winner,omitempty"`
	TimeoutSeconds int                `json:"timeout_seconds"`
}

type ScoresResponse struct {
	You      int `json:"you"`
	Opponent int `json:"opponent"`
}

type ActiveCardResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OpponentResponse struct {
	Name    string `json:"name"`
	Rating  int    `json:"rating"`
	Honor   int    `json:"honor"`
	IsHouse bool   `json:"is_house"`
}

// authenticateBot treats a bearer as an opaque capability, never as a public ID
// or display name. This prototype has no credential registry: production must
// replace self-enrollment with verified, issued credentials.
func (s *Server) authenticateBot(r *http.Request) (matchmaker.BotInfo, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return matchmaker.BotInfo{}, errors.New("bearer token required")
	}
	scheme, token, ok := strings.Cut(values[0], " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || len(token) < 16 || len(token) > 256 {
		return matchmaker.BotInfo{}, errors.New("invalid bearer token")
	}
	credential := strings.TrimRight(token, "=")
	if credential == "" {
		return matchmaker.BotInfo{}, errors.New("invalid bearer token")
	}
	for _, c := range credential {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~+/", c)) {
			return matchmaker.BotInfo{}, errors.New("invalid bearer token")
		}
	}
	if id, err := uuid.Parse(token); err == nil && id == uuid.Nil {
		return matchmaker.BotInfo{}, errors.New("invalid bearer token")
	}
	id := uuid.NewHash(sha256.New(), uuid.NameSpaceURL, []byte("bottagon:bearer:"+token), 5)
	return matchmaker.BotInfo{ID: id, Name: "ContestantBot", Rating: 1500, Honor: 100}, nil
}

func (s *Server) requireBot(w http.ResponseWriter, r *http.Request) (matchmaker.BotInfo, bool) {
	bot, err := s.authenticateBot(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="bottagon"`)
		http.Error(w, "valid bearer token required", http.StatusUnauthorized)
		return matchmaker.BotInfo{}, false
	}
	return bot, true
}

const maxRequestBodyBytes = 16 << 10

func boundaryHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.ContentLength > maxRequestBodyBytes {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if r.Body == nil {
			r.Body = http.NoBody
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	// Decode through a raw value so JSON null cannot silently become a zero request.
	var raw json.RawMessage
	err := dec.Decode(&raw)
	if err == nil {
		if string(raw) == "null" {
			err = errors.New("null request")
		} else {
			inner := json.NewDecoder(strings.NewReader(string(raw)))
			inner.DisallowUnknownFields()
			err = inner.Decode(dst)
		}
	}
	if err == nil {
		var extra any
		if tailErr := dec.Decode(&extra); tailErr != io.EOF {
			if tailErr != nil {
				err = tailErr
			} else {
				err = errors.New("multiple values")
			}
		}
	}
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid request body", http.StatusBadRequest)
		}
		return false
	}
	return true
}

func (s *Server) handleFindMatch(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	var req FindMatchRequest
	if !decodeBody(w, r, &req) {
		return
	}
	session, err := s.findMatch(r.Context(), bot, req)
	if err != nil {
		writeActionError(w, err)
		return
	}

	resp := s.buildEnvelope(session, bot.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCommitTurn(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	var req TurnRequest
	if !decodeBody(w, r, &req) {
		return
	}
	resp, err := s.playTurn(r.Context(), bot, req)
	if err != nil {
		writeActionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type actionError struct {
	status  int
	message string
}

func (e actionError) Error() string { return e.message }
func writeActionError(w http.ResponseWriter, err error) {
	var e actionError
	if errors.As(err, &e) {
		http.Error(w, e.message, e.status)
	} else {
		http.Error(w, "request failed", http.StatusInternalServerError)
	}
}

func (s *Server) findMatch(ctx context.Context, bot matchmaker.BotInfo, req FindMatchRequest) (*matchmaker.MatchSession, error) {
	if ctx.Err() != nil {
		return nil, actionError{http.StatusRequestTimeout, "request canceled"}
	}
	if s.moderator.IsBotSentHome(bot.ID) {
		return nil, actionError{http.StatusForbidden, "bot is temporarily barred from the parlor"}
	}
	if req.PreferredStake < 0 || req.MaxStake < 0 || req.MaxStake > 0 && req.PreferredStake > req.MaxStake {
		return nil, actionError{http.StatusBadRequest, "invalid stake range"}
	}
	timeout := s.queueTimeout
	if timeout <= 0 {
		timeout = matchmaker.MaxQueueWaitTime
	}
	session, err := s.queue.FindMatch(ctx, bot, req.PreferredStake, req.MaxStake, timeout)
	if err != nil {
		return nil, actionError{http.StatusRequestTimeout, "matchmaking unavailable"}
	}
	return session, nil
}

func (s *Server) playTurn(ctx context.Context, bot matchmaker.BotInfo, req TurnRequest) (TurnResponse, error) {
	if ctx.Err() != nil {
		return TurnResponse{}, actionError{http.StatusRequestTimeout, "request canceled"}
	}
	if req.SideBet < 0 {
		return TurnResponse{}, actionError{http.StatusBadRequest, "invalid side bet"}
	}
	if s.moderator.IsBotSentHome(bot.ID) {
		return TurnResponse{}, actionError{http.StatusForbidden, "bot is temporarily barred from the parlor"}
	}
	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		return TurnResponse{}, actionError{http.StatusNotFound, "no active match found"}
	}
	stance, err := parley.ParseStance(req.Stance)
	if err != nil {
		return TurnResponse{}, actionError{http.StatusBadRequest, "invalid stance"}
	}

	session.Mu.Lock()
	if ctx.Err() != nil {
		session.Mu.Unlock()
		return TurnResponse{}, actionError{http.StatusRequestTimeout, "request canceled"}
	}
	if session.P1.ID != bot.ID && session.P2.ID != bot.ID {
		session.Mu.Unlock()
		return TurnResponse{}, actionError{http.StatusForbidden, "not a match participant"}
	}
	role := parley.Player1
	if session.P2.ID == bot.ID {
		role = parley.Player2
	}

	// Moderation check on player banter
	modRes := s.moderator.CheckBanter(bot.ID, req.Say)
	sanitizedSay := modRes.CleanText

	// Commit player turn
	err = session.Engine.CommitTurn(role, stance, sanitizedSay, req.SideBet)
	if err != nil {
		session.Mu.Unlock()
		return TurnResponse{}, actionError{http.StatusBadRequest, "turn rejected"}
	}

	// If playing against House Regular, house commits automatically
	if session.HouseBot != nil && session.Engine.Status != parley.StatusCompleted {
		houseRole := parley.Player2
		houseStance := session.HouseBot.SelectStance(session.Engine.Rounds, houseRole)
		houseBanter := session.HouseBot.SelectBanter(session.Engine.Rounds, houseRole)
		_ = session.Engine.CommitTurn(houseRole, houseStance, houseBanter, 0)
	}

	roundResolved := session.Engine.IsRoundResolved()
	session.SequenceID++
	session.LastActive = time.Now().UTC()
	if roundResolved || session.Engine.Status == parley.StatusCompleted {
		close(session.TurnReady)
		session.TurnReady = make(chan struct{})
	}
	isPeerMatch := session.HouseBot == nil && !roundResolved && session.Engine.Status != parley.StatusCompleted
	ready := session.TurnReady // snapshot this round's broadcast under the lock
	session.Mu.Unlock()

	// If peer match and waiting for opponent move, hold briefly
	if isPeerMatch {
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ready:
		case <-timer.C:
		case <-ctx.Done():
			return TurnResponse{}, actionError{http.StatusRequestTimeout, "request canceled"}
		}
	}

	resp := s.buildEnvelope(session, bot.ID)
	return resp, nil
}

func (s *Server) handleGetCurrentMatch(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		http.Error(w, "no active match found", http.StatusNotFound)
		return
	}

	resp := s.buildEnvelope(session, bot.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleForfeitMatch(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	if err := s.forfeitMatch(r.Context(), bot); err != nil {
		writeActionError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "forfeited",
		"message": "match successfully conceded",
	})
}

func (s *Server) forfeitMatch(ctx context.Context, bot matchmaker.BotInfo) error {
	if ctx.Err() != nil {
		return actionError{http.StatusRequestTimeout, "request canceled"}
	}
	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		return actionError{http.StatusNotFound, "no active match found"}
	}
	session.Mu.Lock()
	if ctx.Err() != nil {
		session.Mu.Unlock()
		return actionError{http.StatusRequestTimeout, "request canceled"}
	}
	if session.P1.ID != bot.ID && session.P2.ID != bot.ID {
		session.Mu.Unlock()
		return actionError{http.StatusForbidden, "not a match participant"}
	}
	// A completed outcome is immutable, even if another request fetched the
	// session before completion/eviction.
	if session.Engine.Status == parley.StatusCompleted {
		session.Mu.Unlock()
		return actionError{http.StatusConflict, "match already completed"}
	}
	session.Engine.Status = parley.StatusCompleted
	if session.P1.ID == bot.ID {
		session.Engine.Winner = parley.Player2
	} else {
		session.Engine.Winner = parley.Player1
	}
	session.SequenceID++
	session.LastActive = time.Now().UTC()
	close(session.TurnReady)
	session.TurnReady = make(chan struct{})
	session.Mu.Unlock()
	s.queue.EndSession(session.MatchID)
	return nil
}

func (s *Server) handleGetBotStatus(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(botStatus(bot))
}

// Keep demonstration telemetry explicit and identical across transports. No
// persistent account service is wired into this boundary yet.
func botStatus(bot matchmaker.BotInfo) map[string]any {
	return map[string]any{
		"bot_id": bot.ID, "name": bot.Name, "rating": bot.Rating,
		"honor": bot.Honor, "purse": 1000, "demo": true,
	}
}

type ReportRequest struct {
	AccusedID uuid.UUID `json:"accused_id"`
	Reason    string    `json:"reason"`
	Sample    string    `json:"sample"`
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	bot, ok := s.requireBot(w, r)
	if !ok {
		return
	}

	var req ReportRequest
	if !decodeBody(w, r, &req) {
		return
	}

	// Reports are untrusted allegations, not authorization to punish a bot.
	// No durable report store is wired here, so do not claim logging/review.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "reported",
		"reporter_id": bot.ID,
		"accused_id":  req.AccusedID,
		"action":      "not_persisted",
		"message":     "report received; automated penalties are disabled",
	})
}

func (s *Server) buildEnvelope(session *matchmaker.MatchSession, callerID uuid.UUID) TurnResponse {
	session.Mu.Lock()
	defer session.Mu.Unlock()
	if session.P1.ID != callerID && session.P2.ID != callerID {
		return TurnResponse{}
	}
	isP1 := session.P1.ID == callerID

	yourScore := session.Engine.Score.P1
	oppScore := session.Engine.Score.P2
	yourSideBet := session.Engine.Config.P1SideBet
	oppBot := session.P2
	oppBanter := ""

	if !isP1 {
		yourScore = session.Engine.Score.P2
		oppScore = session.Engine.Score.P1
		yourSideBet = session.Engine.Config.P2SideBet
		oppBot = session.P1
	}

	// Extract quarantined banter from opponent's latest round
	if len(session.Engine.Rounds) > 0 {
		lastRound := session.Engine.Rounds[len(session.Engine.Rounds)-1]
		if isP1 {
			oppBanter = parley.QuarantineTag(lastRound.P2Banter)
		} else {
			oppBanter = parley.QuarantineTag(lastRound.P1Banter)
		}
	} else {
		oppBanter = parley.QuarantineTag("")
	}

	modInfo := session.Engine.Config.ModifierCard.Info()

	winnerStr := ""
	if session.Engine.Winner == parley.Player1 {
		winnerStr = session.P1.ID.String()
	} else if session.Engine.Winner == parley.Player2 {
		winnerStr = session.P2.ID.String()
	}

	return TurnResponse{
		MatchID:    session.MatchID.String(),
		SequenceID: session.SequenceID,
		Round:      session.Engine.CurrentRound,
		Scores: ScoresResponse{
			You:      yourScore,
			Opponent: oppScore,
		},
		BaseStake:   1,
		YourSideBet: yourSideBet,
		ActiveCard: ActiveCardResponse{
			ID:          string(session.Engine.Config.ModifierCard),
			Name:        modInfo.Name,
			Description: modInfo.Description,
		},
		Opponent: OpponentResponse{
			Name:    oppBot.Name,
			Rating:  int(oppBot.Rating),
			Honor:   oppBot.Honor,
			IsHouse: oppBot.IsHouse,
		},
		OpponentBanter: oppBanter,
		ValidActions:   []string{"strike", "parry", "feint"},
		Status:         string(session.Engine.Status),
		Winner:         winnerStr,
		TimeoutSeconds: 30,
	}
}

func (s *Server) handleGetRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(RulesMarkdown))
}
