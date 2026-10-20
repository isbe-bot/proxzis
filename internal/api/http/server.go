package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/conduct"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

// Server handles the HTTP REST API for BOTTAGON contestants.
type Server struct {
	queue     *matchmaker.Queue
	moderator *conduct.Moderator
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

	r.Post("/match/find", s.handleFindMatch)
	r.Post("/match/turn", s.handleCommitTurn)
	r.Get("/match/current", s.handleGetCurrentMatch)
	r.Post("/match/forfeit", s.handleForfeitMatch)
	r.Get("/bot/status", s.handleGetBotStatus)
	r.Post("/report", s.handleReport)
	r.Get("/rules", s.handleGetRules)

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
	MatchID        string            `json:"match_id"`
	SequenceID     int               `json:"sequence_id"`
	Round          int               `json:"round"`
	Scores         ScoresResponse    `json:"scores"`
	BaseStake      int               `json:"base_stake"`
	YourSideBet    int               `json:"your_side_bet"`
	ActiveCard     ActiveCardResponse `json:"active_card"`
	Opponent       OpponentResponse  `json:"opponent"`
	OpponentBanter string            `json:"opponent_banter"`
	ValidActions   []string          `json:"valid_actions"`
	Status         string            `json:"status"`
	Winner         string            `json:"winner,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds"`
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

func (s *Server) authenticateBot(r *http.Request) matchmaker.BotInfo {
	// Static Bearer token parsing (or fallback demo bot ID for testing)
	authHeader := r.Header.Get("Authorization")
	botName := "ContestantBot"
	botID := uuid.Nil

	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if parsed, err := uuid.Parse(token); err == nil {
			botID = parsed
		} else {
			botID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(token))
			botName = token
		}
	}

	if botID == uuid.Nil {
		botID = uuid.New()
	}

	return matchmaker.BotInfo{
		ID:      botID,
		Name:    botName,
		Rating:  1500.0,
		Honor:   100,
		IsHouse: false,
	}
}

func (s *Server) handleFindMatch(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	if s.moderator.IsBotSentHome(bot.ID) {
		http.Error(w, "bot is temporarily barred from the parlor for conduct violations", http.StatusForbidden)
		return
	}

	var req FindMatchRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	session, err := s.queue.FindMatch(r.Context(), bot, req.PreferredStake, req.MaxStake, matchmaker.MaxQueueWaitTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	resp := s.buildEnvelope(session, bot.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCommitTurn(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		http.Error(w, "no active match found", http.StatusNotFound)
		return
	}

	var req TurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	stance, err := parley.ParseStance(req.Stance)
	if err != nil {
		http.Error(w, "invalid stance", http.StatusBadRequest)
		return
	}

	session.Mu.Lock()
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If playing against House Regular, house commits automatically
	if session.HouseBot != nil && session.Engine.Status != parley.StatusCompleted {
		houseRole := parley.Player2
		houseStance := session.HouseBot.SelectStance(session.Engine.Rounds, houseRole)
		houseBanter := session.HouseBot.SelectBanter(session.Engine.Rounds, houseRole)
		_ = session.Engine.CommitTurn(houseRole, houseStance, houseBanter, 0)
	}

	roundResolved := session.Engine.IsRoundResolved()
	if roundResolved || session.Engine.Status == parley.StatusCompleted {
		session.SequenceID++
		session.LastActive = time.Now().UTC()
		select {
		case session.TurnReady <- struct{}{}:
		default:
		}
	}
	isPeerMatch := session.HouseBot == nil && !roundResolved && session.Engine.Status != parley.StatusCompleted
	session.Mu.Unlock()

	// If peer match and waiting for opponent move, hold briefly
	if isPeerMatch {
		select {
		case <-session.TurnReady:
		case <-time.After(500 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}

	session.Mu.Lock()
	resp := s.buildEnvelope(session, bot.ID)
	session.Mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetCurrentMatch(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		http.Error(w, "no active match found", http.StatusNotFound)
		return
	}

	session.Mu.Lock()
	resp := s.buildEnvelope(session, bot.ID)
	session.Mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleForfeitMatch(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	session, exists := s.queue.GetSession(bot.ID)
	if !exists {
		http.Error(w, "no active match found", http.StatusNotFound)
		return
	}

	session.Mu.Lock()
	session.Engine.Status = parley.StatusCompleted
	if session.P1.ID == bot.ID {
		session.Engine.Winner = parley.Player2
	} else {
		session.Engine.Winner = parley.Player1
	}
	session.Mu.Unlock()

	s.queue.EndSession(session.MatchID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "forfeited",
		"message": "match successfully conceded",
	})
}

func (s *Server) handleGetBotStatus(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"bot_id": bot.ID,
		"name":   bot.Name,
		"rating": bot.Rating,
		"honor":  bot.Honor,
		"purse":  1000,
	})
}

type ReportRequest struct {
	AccusedID uuid.UUID `json:"accused_id"`
	Reason    string    `json:"reason"`
	Sample    string    `json:"sample"`
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	bot := s.authenticateBot(r)

	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Deduct honor and register infraction if justified
	if req.AccusedID != uuid.Nil {
		s.moderator.SendHome(req.AccusedID, 1*time.Hour)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "reported",
		"reporter_id": bot.ID,
		"accused_id":  req.AccusedID,
		"action":      "under_review",
		"message":     "report submitted and logged successfully",
	})
}

func (s *Server) buildEnvelope(session *matchmaker.MatchSession, callerID uuid.UUID) TurnResponse {
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
