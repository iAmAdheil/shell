package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/login"
	"backend/internal/observability"
	"backend/internal/router"
	"backend/internal/session"
	"backend/internal/shell"
)

func main() {
	// The server runs from server/, so the repo-root .env is one level up.
	_ = godotenv.Load("../.env", ".env")

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Sentry starts first, so it can report a failure in any later step.
	if err := observability.Init(cfg.sentry); err != nil {
		log.Fatalf("sentry: %v", err)
	}
	defer observability.Close()

	ctx := context.Background()
	accounts, err := account.OpenPostgres(ctx, cfg.databaseURL)
	if err != nil {
		log.Fatalf("accounts: %v (is `make db-up` running?)", err)
	}
	defer accounts.Close()

	runner, err := shell.NewDockerRunner()
	if err != nil {
		log.Fatalf("docker: %v (is Docker running?)", err)
	}

	// Sessions live only in this process. Ending them on the way out is what
	// stops a container outliving the server that started it (ticket 20).
	sessions := session.NewStore(runner)
	defer sessions.CloseAll()

	r := router.New(router.Deps{
		Accounts:   accounts,
		Logins:     login.NewStore(),
		Google:     auth.NewGoogleProvider(cfg.googleClientID, cfg.googleClientSecret, cfg.appBaseURL+"/api/auth/google/callback"),
		Sessions:   sessions,
		WebBaseURL: cfg.webBaseURL,
	})

	srv := &http.Server{Addr: ":8081", Handler: r}

	stopping := make(chan os.Signal, 1)
	signal.Notify(stopping, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("server listening on :8081")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-stopping
	log.Println("shutting down: ending every Session")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type config struct {
	databaseURL        string
	googleClientID     string
	googleClientSecret string
	appBaseURL         string
	webBaseURL         string
	sentry             observability.Config
}

// loadConfig reads the environment and reports every missing setting at once,
// so a first run does not turn into one restart per variable.
func loadConfig() (config, error) {
	var missing []string
	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}
	withDefault := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	rateWithDefault := func(key string, fallback float64) float64 {
		v, err := strconv.ParseFloat(withDefault(key, ""), 64)
		if err != nil || v < 0 || v > 1 {
			return fallback
		}
		return v
	}

	cfg := config{
		databaseURL:        required("DATABASE_URL"),
		googleClientID:     required("GOOGLE_CLIENT_ID"),
		googleClientSecret: required("GOOGLE_CLIENT_SECRET"),
		appBaseURL:         withDefault("APP_BASE_URL", "http://localhost:8081"),
		webBaseURL:         withDefault("WEB_BASE_URL", "http://localhost:5173"),

		// SENTRY_DSN is deliberately not required. Empty turns Sentry off, so
		// the app still runs for anyone without a Sentry account.
		sentry: observability.Config{
			DSN:              os.Getenv("SENTRY_DSN"),
			Environment:      withDefault("SENTRY_ENVIRONMENT", "development"),
			TracesSampleRate: rateWithDefault("SENTRY_TRACES_SAMPLE_RATE", 0.1),
		},
	}
	if len(missing) > 0 {
		return config{}, fmt.Errorf("missing environment settings %v: copy .env.example to .env and fill them in", missing)
	}
	return cfg, nil
}
