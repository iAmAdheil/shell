package observability

import (
	"testing"

	"github.com/getsentry/sentry-go"
)

// Anyone can run this app without a Sentry account, so no DSN must leave the
// server working rather than failing to start.
func TestNoDSNLeavesSentryOff(t *testing.T) {
	if err := Init(Config{Environment: "test", TracesSampleRate: 1}); err != nil {
		t.Fatalf("Init with no DSN: %v", err)
	}
	if Enabled() {
		t.Fatal("Sentry reports itself as on, but no DSN was given")
	}
}

func TestSamplerDropsQuietRoutes(t *testing.T) {
	sample := sampler(1.0)

	for _, name := range []string{"GET /api/health", "GET /api/sessions/:code/ws"} {
		if got := sample(named(name)); got != 0 {
			t.Errorf("%s: sample rate %v, want 0", name, got)
		}
	}

	if got := sample(named("POST /api/sessions")); got != 1.0 {
		t.Errorf("POST /api/sessions: sample rate %v, want 1", got)
	}
}

func named(name string) sentry.SamplingContext {
	return sentry.SamplingContext{Span: &sentry.Span{Name: name}}
}
