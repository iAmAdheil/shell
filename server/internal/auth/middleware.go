package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/login"
)

// CookieName is the auth cookie a browser sends back on every request.
const CookieName = "shell_login"

// accountIDKey is where RequireLogin leaves the logged-in Account's ID.
const accountIDKey = "accountID"

// RequireLogin rejects a request that carries no valid auth cookie.
func RequireLogin(logins *login.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(CookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}

		l, ok := logins.Get(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}

		// Re-issue the cookie so the browser's copy lives as long as the
		// server's. Without this the cookie would expire 30 days after login,
		// however active the User has been.
		setCookie(c, CookieName, l.Token, int(login.TTL.Seconds()))

		c.Set(accountIDKey, l.AccountID)
		c.Next()
	}
}

// AccountID returns the logged-in Account's ID. Only call it from a handler
// that RequireLogin guards.
func AccountID(c *gin.Context) string {
	id, _ := c.Get(accountIDKey)
	s, _ := id.(string)
	return s
}
