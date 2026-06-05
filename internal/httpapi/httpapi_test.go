package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

type fakeRetriever struct {
	chunks    []retriever.Chunk
	gotTenant *string
}

func (f *fakeRetriever) Search(ctx context.Context, q string, tenant *string, aud string, k int) ([]retriever.Chunk, error) {
	f.gotTenant = tenant
	return f.chunks, nil
}

type fakeGenerator struct {
	called bool
	toks   []string
}

func (f *fakeGenerator) Stream(ctx context.Context, sys, user string, onToken func(string)) error {
	f.called = true
	for _, t := range f.toks {
		onToken(t)
	}
	return nil
}

type fakeLogger struct {
	feedbackCalled bool
	loggedAnswer   string
	gotTenant      *string
}

func (f *fakeLogger) StartConversation(ctx context.Context, tenant, user *string, mode, channel string) (string, error) {
	return "conv-1", nil
}
func (f *fakeLogger) LogTurn(ctx context.Context, convID string, tenant *string, q, a, model string, ids []string, ms int) (string, string, error) {
	f.loggedAnswer = a
	f.gotTenant = tenant
	return "user-1", "asst-1", nil
}
func (f *fakeLogger) RecordFeedback(ctx context.Context, msgID string, tenant, user *string, rating string, solved *bool, note string) error {
	f.feedbackCalled = true
	return nil
}

func newTestServer(r Retriever, g *fakeGenerator, l Logger) http.Handler {
	return Handler(Deps{Retriever: r, Generator: g, Logger: l, GenModel: "sea-lion-9b"})
}

func post(h http.Handler, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestChatHappyPathStreamsAndLogs(t *testing.T) {
	gen := &fakeGenerator{toks: []string{"Pwede", " mag-solar"}}
	log := &fakeLogger{}
	h := newTestServer(&fakeRetriever{chunks: []retriever.Chunk{
		{ChunkID: "11111111-1111-1111-1111-111111111111", Title: "ROI", Content: "tipid", Score: 0.2},
	}}, gen, log)

	rec := post(h, "/assistant/chat", `{"message":"magkano matitipid sa solar?"}`, nil)
	body := rec.Body.String()

	if !gen.called {
		t.Fatal("generator should have been called for a grounded solar question")
	}
	if !strings.Contains(body, `"token":"Pwede"`) || !strings.Contains(body, "mag-solar") {
		t.Fatalf("streamed tokens missing: %q", body)
	}
	if !strings.Contains(body, "event: done") || !strings.Contains(body, `"message_id":"asst-1"`) {
		t.Fatalf("done event / message id missing: %q", body)
	}
	if !strings.Contains(body, "ROI") {
		t.Fatalf("source title missing in done event: %q", body)
	}
	if log.loggedAnswer != "Pwede mag-solar" {
		t.Fatalf("logged answer wrong: %q", log.loggedAnswer)
	}
}

func TestChatOffTopicSkipsGenerator(t *testing.T) {
	gen := &fakeGenerator{toks: []string{"should not run"}}
	h := newTestServer(&fakeRetriever{}, gen, &fakeLogger{})

	rec := post(h, "/assistant/chat", `{"message":"sino panalo sa NBA finals?"}`, nil)
	if gen.called {
		t.Fatal("generator must NOT be called for off-topic messages")
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "solar") {
		t.Fatalf("expected a Taglish solar redirect: %q", rec.Body.String())
	}
}

func TestChatEscalatesWhenNoGroundedSource(t *testing.T) {
	gen := &fakeGenerator{toks: []string{"should not run"}}
	h := newTestServer(&fakeRetriever{chunks: []retriever.Chunk{
		{Title: "Far", Content: "x", Score: 0.95},
	}}, gen, &fakeLogger{})

	rec := post(h, "/assistant/chat", `{"message":"may tanong ako tungkol sa solar permit"}`, nil)
	if gen.called {
		t.Fatal("generator must NOT be called when retrieval has no grounded source")
	}
	if !strings.Contains(rec.Body.String(), `"escalated":true`) {
		t.Fatalf("expected escalation done event: %q", rec.Body.String())
	}
}

func TestChatRejectsEmptyMessage(t *testing.T) {
	h := newTestServer(&fakeRetriever{}, &fakeGenerator{}, &fakeLogger{})
	rec := post(h, "/assistant/chat", `{"message":"   "}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty message, got %d", rec.Code)
	}
}

func TestFeedbackValidatesAndRecords(t *testing.T) {
	log := &fakeLogger{}
	h := newTestServer(&fakeRetriever{}, &fakeGenerator{}, log)

	bad := post(h, "/assistant/feedback", `{"message_id":"asst-1"}`, nil)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing rating, got %d", bad.Code)
	}
	if log.feedbackCalled {
		t.Fatal("feedback should not be recorded on invalid input")
	}

	ok := post(h, "/assistant/feedback", `{"message_id":"asst-1","rating":"down"}`, nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", ok.Code)
	}
	if !log.feedbackCalled {
		t.Fatal("valid feedback should be recorded")
	}
}

func TestPermissiveAuthPassesTenantHeader(t *testing.T) {
	ret := &fakeRetriever{chunks: []retriever.Chunk{{Title: "ROI", Content: "x", Score: 0.2}}}
	h := newTestServer(ret, &fakeGenerator{toks: []string{"ok"}}, &fakeLogger{})

	post(h, "/assistant/chat", `{"message":"magkano ang solar panel?"}`,
		map[string]string{"X-Tenant-Id": "tenant-xyz"})
	if ret.gotTenant == nil || *ret.gotTenant != "tenant-xyz" {
		t.Fatalf("expected tenant from header to reach retriever, got %v", ret.gotTenant)
	}
}
