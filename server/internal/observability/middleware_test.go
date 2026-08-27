package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

func TestMiddlewareReportsAPanicAndStillAnswers(t *testing.T) {
	sent := run(t, func(c *gin.Context) { panic("the shell fell over") })

	if len(sent) != 1 {
		t.Fatalf("Sentry got %d events, want 1", len(sent))
	}
	if got := sent[0].Message; got != "the shell fell over" {
		t.Errorf("event message %q, want %q", got, "the shell fell over")
	}
}

func TestCaptureReportsAHandledError(t *testing.T) {
	sent := run(t, func(c *gin.Context) {
		Capture(c, errors.New("could not start a Session"))
		c.Status(http.StatusInternalServerError)
	})

	if len(sent) != 1 {
		t.Fatalf("Sentry got %d events, want 1", len(sent))
	}
	if len(sent[0].Exception) == 0 || sent[0].Exception[0].Value != "could not start a Session" {
		t.Errorf("event carries %+v, want the handled error", sent[0].Exception)
	}
}

// run sends one request through the Sentry middleware to route, and returns
// what Sentry would have been told. It binds its own hub to the request, so it
// leaves the process-wide hub that Init writes to alone.
func run(t *testing.T, route gin.HandlerFunc) []*sentry.Event {
	t.Helper()
	gin.SetMode(gin.TestMode)

	post := &postbox{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@example.test/1",
		Transport: post,
	})
	if err != nil {
		t.Fatalf("sentry.NewClient: %v", err)
	}
	hub := sentry.NewHub(client, sentry.NewScope())

	r := gin.New()
	r.Use(gin.Recovery(), func(c *gin.Context) {
		c.Request = c.Request.WithContext(sentry.SetHubOnContext(c.Request.Context(), hub))
		c.Next()
	}, Middleware(), route)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	// A panic must still leave the caller with an answer, not a dead socket.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want %d", w.Code, http.StatusInternalServerError)
	}
	return post.taken()
}

// postbox stands in for Sentry's network transport and keeps every event.
type postbox struct {
	mu   sync.Mutex
	sent []*sentry.Event
}

func (p *postbox) SendEvent(event *sentry.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, event)
}

func (p *postbox) taken() []*sentry.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sent
}

func (p *postbox) Configure(sentry.ClientOptions)        {}
func (p *postbox) Flush(time.Duration) bool              { return true }
func (p *postbox) FlushWithContext(context.Context) bool { return true }
func (p *postbox) Close()                                {}
