// Package observability reports server errors and request timings to Sentry.
//
// The Sentry SDK keeps its client in one process-wide hub, so this package
// holds process-wide state instead of a value the caller passes around. A
// second, injected copy would report nothing.
package observability

import (
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// flushWait is how long Close waits for queued events on the way out.
const flushWait = 2 * time.Second

// Config is what Sentry needs to report from this server.
type Config struct {
	// DSN names the Sentry project to report to. An empty DSN turns Sentry
	// off, which is what a local run without a Sentry account wants.
	DSN string

	// Environment separates one deployment's events from another's, for
	// example "development" or "production".
	Environment string

	// TracesSampleRate is the share of requests that get a timing report, from
	// 0.0 to 1.0. Errors are always reported, whatever this rate is.
	TracesSampleRate float64
}

// Init starts Sentry. An empty DSN turns Sentry off and is not an error.
func Init(cfg Config) error {
	if cfg.DSN == "" {
		return nil
	}

	return sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.DSN,
		Environment: cfg.Environment,

		// Every default here is off, and that is deliberate. A Session carries
		// whatever a User types into a shell, and every request carries the
		// auth cookie. Leaving SendDefaultPII and DataCollection unset keeps
		// cookies, request bodies and caller IP addresses out of Sentry.

		EnableTracing: cfg.TracesSampleRate > 0,
		TracesSampler: sampler(cfg.TracesSampleRate),
	})
}

// Enabled reports whether Init found a DSN and started Sentry.
func Enabled() bool {
	return sentry.CurrentHub().Client() != nil
}

// Close sends whatever is still queued. Call it on the way out of main.
func Close() {
	sentry.Flush(flushWait)
}

// Middleware reports a panic in any route, and times the request. It also
// leaves a Sentry hub on the gin context for Capture to find.
func Middleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{
		// gin.Default already recovers a panic and answers with a 500, so
		// this middleware reports the panic and then hands it back.
		Repanic: true,
	})
}

// Capture reports one handled error against the request that hit it. Use it
// where a handler answers with a 5xx: those are the failures nobody sees.
func Capture(c *gin.Context, err error) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
	}
}

// SetUser tags every later event on this request with the logged-in Account.
// Only the ID goes to Sentry, never the User's name or their avatar.
func SetUser(c *gin.Context, accountID string) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.Scope().SetUser(sentry.User{ID: accountID})
	}
}

// quiet routes report no timings. A health check says nothing, and a Session
// socket stays open for as long as the Session lasts, so its "request
// duration" would measure the Session, not the server.
var quiet = map[string]bool{
	"GET /api/health":            true,
	"GET /api/sessions/:code/ws": true,
}

func sampler(rate float64) sentry.TracesSampler {
	return func(ctx sentry.SamplingContext) float64 {
		if quiet[ctx.Span.Name] {
			return 0
		}
		return rate
	}
}
