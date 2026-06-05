// Package httpapi exposes the Apolaki assistant over HTTP: a streaming chat
// endpoint (topic-gate -> retrieve -> personalize -> assemble -> generate ->
// safety -> log), a feedback hook, health, and a local browser test page.
package httpapi

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apolaki/solar-assistant/internal/generator"
	"github.com/apolaki/solar-assistant/internal/personalizer"
	"github.com/apolaki/solar-assistant/internal/prompt"
	"github.com/apolaki/solar-assistant/internal/retriever"
	"github.com/apolaki/solar-assistant/internal/safety"
	"github.com/apolaki/solar-assistant/internal/topicgate"
)

//go:embed index.html
var indexHTML []byte

// Retriever is the subset of *retriever.Retriever the API needs.
type Retriever interface {
	Search(ctx context.Context, query string, tenantID *string, audience string, k int) ([]retriever.Chunk, error)
}

// Logger is the subset of *chatlog.Logger the API needs.
type Logger interface {
	StartConversation(ctx context.Context, tenantID, userID *string, mode, channel string) (string, error)
	LogTurn(ctx context.Context, convID string, tenantID *string, question, answer, model string, chunkIDs []string, latencyMs int) (string, string, error)
	RecordFeedback(ctx context.Context, messageID string, tenantID, userID *string, rating string, solved *bool, note string) error
}

// Pinger is the health-check subset of the DB pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps are the collaborators and config for the HTTP API.
type Deps struct {
	Retriever    Retriever
	Generator    generator.Client
	Personalizer personalizer.Personalizer
	Logger       Logger
	Pool         Pinger
	GenModel     string // model label recorded with each turn
	JWTSecret    string // empty => permissive dev auth
	TopK         int    // retrieval depth (default 4)
}

type server struct {
	d Deps
}

// Handler builds the http.Handler with all routes mounted. /assistant/* is
// wrapped in auth middleware; GET / serves the test page unauthenticated.
func Handler(d Deps) http.Handler {
	if d.TopK <= 0 {
		d.TopK = 4
	}
	if d.Personalizer == nil {
		d.Personalizer = personalizer.Noop{}
	}
	s := &server{d: d}
	mux := http.NewServeMux()
	mux.Handle("/assistant/chat", authMiddleware(d.JWTSecret, http.HandlerFunc(s.chat)))
	mux.Handle("/assistant/feedback", authMiddleware(d.JWTSecret, http.HandlerFunc(s.feedback)))
	mux.HandleFunc("/assistant/health", s.health)
	mux.HandleFunc("/", s.page)
	return mux
}

func (s *server) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{"status": "ok"}
	if s.d.Pool != nil {
		if err := s.d.Pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "db_unreachable"
		}
	}
	writeJSON(w, status)
}

type chatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	Mode           string `json:"mode"` // customer (default) | buyer | installer
}

func (s *server) chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	tenantID, userID := identity(ctx)
	m := prompt.ModeByName(req.Mode)
	start := time.Now()

	emit := func(event string, payload any) {
		b, _ := json.Marshal(payload)
		if event != "" {
			fmt.Fprintf(w, "event: %s\n", event)
		}
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	// Layer 1: topic gate — off-topic never reaches the model.
	if !topicgate.IsSolarRelated(req.Message) {
		redirect := topicgate.Redirect()
		emit("", map[string]string{"token": redirect})
		s.finish(ctx, &req, m.Name, tenantID, userID, redirect, "topic-gate", nil, start, false, emit)
		return
	}

	chunks, err := s.d.Retriever.Search(ctx, req.Message, tenantID, m.Audience, s.d.TopK)
	if err != nil {
		emit("error", map[string]string{"error": "retrieval failed"})
		return
	}

	// Layer 2/3: grounding + safety.
	decision := safety.Check(chunks, req.Message)
	if decision.Escalate {
		msg := safety.Escalation()
		emit("", map[string]string{"token": msg})
		s.finish(ctx, &req, m.Name, tenantID, userID, msg, "escalation", chunks, start, true, emit)
		return
	}

	sys, user := prompt.AssembleFor(m, req.Message, chunks)
	if pc, perr := s.d.Personalizer.Context(ctx, deref(userID)); perr == nil && pc != "" {
		user = "USER CONTEXT: " + pc + "\n\n" + user
	}

	var answer strings.Builder
	streamErr := s.d.Generator.Stream(ctx, sys, user, func(tok string) {
		answer.WriteString(tok)
		emit("", map[string]string{"token": tok})
	})
	if streamErr != nil {
		emit("error", map[string]string{"error": "generation failed"})
		return
	}
	if decision.Disclaimer != "" {
		tail := "\n\n" + decision.Disclaimer
		answer.WriteString(tail)
		emit("", map[string]string{"token": tail})
	}
	s.finish(ctx, &req, m.Name, tenantID, userID, answer.String(), s.d.GenModel, chunks, start, false, emit)
}

// finish logs the turn and emits the terminal done event (conversation id,
// assistant message id, source titles). Logging failures are non-fatal.
func (s *server) finish(ctx context.Context, req *chatRequest, mode string, tenantID, userID *string,
	answer, model string, chunks []retriever.Chunk, start time.Time, escalated bool, emit func(string, any)) {
	convID := req.ConversationID
	var msgID string
	if s.d.Logger != nil {
		if convID == "" {
			if id, err := s.d.Logger.StartConversation(ctx, tenantID, userID, mode, "web"); err == nil {
				convID = id
			}
		}
		if convID != "" {
			if _, aid, err := s.d.Logger.LogTurn(ctx, convID, tenantID, req.Message, answer, model,
				chunkIDs(chunks), int(time.Since(start).Milliseconds())); err == nil {
				msgID = aid
			}
		}
	}
	emit("done", map[string]any{
		"conversation_id": convID,
		"message_id":      msgID,
		"sources":         sourceTitles(chunks),
		"escalated":       escalated,
	})
}

type feedbackRequest struct {
	MessageID string `json:"message_id"`
	Rating    string `json:"rating"`
	Solved    *bool  `json:"solved"`
	Note      string `json:"note"`
}

func (s *server) feedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.MessageID == "" || (req.Rating != "up" && req.Rating != "down") {
		http.Error(w, "message_id and rating (up|down) are required", http.StatusBadRequest)
		return
	}
	tenantID, userID := identity(r.Context())
	if s.d.Logger != nil {
		if err := s.d.Logger.RecordFeedback(r.Context(), req.MessageID, tenantID, userID,
			req.Rating, req.Solved, req.Note); err != nil {
			http.Error(w, "could not record feedback", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func chunkIDs(chunks []retriever.Chunk) []string {
	if len(chunks) == 0 {
		return nil
	}
	ids := make([]string, 0, len(chunks))
	for _, c := range chunks {
		ids = append(ids, c.ChunkID)
	}
	return ids
}

func sourceTitles(chunks []retriever.Chunk) []string {
	titles := make([]string, 0, len(chunks))
	for _, c := range chunks {
		titles = append(titles, c.Title)
	}
	return titles
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
