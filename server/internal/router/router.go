package router

import (
	"github.com/gin-gonic/gin"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/handler"
	"backend/internal/login"
	"backend/internal/observability"
	"backend/internal/session"
)

// Deps are the collaborators the routes need.
type Deps struct {
	Accounts account.Store
	Logins   *login.Store
	Google   auth.IdentityProvider
	Sessions *session.Store

	// WebBaseURL is where a finished login sends the browser back to.
	WebBaseURL string
}

func New(d Deps) *gin.Engine {
	r := gin.Default()

	// Sentry wraps every route, so a panic in any of them is reported. With no
	// DSN it would report nothing, so it stays off the chain entirely.
	if observability.Enabled() {
		r.Use(observability.Middleware())
	}

	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)

		api.GET("/auth/google/start", auth.Start(d.Google))
		api.GET("/auth/google/callback", auth.Callback(d.Google, d.Accounts, d.Logins, d.WebBaseURL))
		api.POST("/auth/logout", auth.Logout(d.Logins))

		guarded := api.Group("")
		guarded.Use(auth.RequireLogin(d.Logins), tagAccount)
		guarded.GET("/me", handler.Me(d.Accounts))
		guarded.POST("/sessions", handler.CreateSession(d.Sessions))
		guarded.GET("/sessions/:code", handler.CheckSession(d.Sessions))
		guarded.GET("/sessions/:code/ws", handler.JoinSession(d.Sessions, d.Accounts))
	}

	return r
}

// tagAccount names the logged-in Account on every Sentry event from this
// request. It must run after RequireLogin, which is what finds that Account.
func tagAccount(c *gin.Context) {
	observability.SetUser(c, auth.AccountID(c))
}
