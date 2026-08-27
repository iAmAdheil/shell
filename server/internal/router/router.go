package router

import (
	"github.com/gin-gonic/gin"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/handler"
	"backend/internal/login"
)

// Deps are the collaborators the routes need.
type Deps struct {
	Accounts account.Store
	Logins   *login.Store
	Google   auth.IdentityProvider

	// WebBaseURL is where a finished login sends the browser back to.
	WebBaseURL string
}

func New(d Deps) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)

		api.GET("/auth/google/start", auth.Start(d.Google))
		api.GET("/auth/google/callback", auth.Callback(d.Google, d.Accounts, d.Logins, d.WebBaseURL))
		api.POST("/auth/logout", auth.Logout(d.Logins))

		guarded := api.Group("")
		guarded.Use(auth.RequireLogin(d.Logins))
		guarded.GET("/me", handler.Me(d.Accounts))
	}

	return r
}
