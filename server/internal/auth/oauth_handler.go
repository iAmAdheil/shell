package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/account"
	"backend/internal/login"
)

// StateCookieName holds the one-use value that ties a callback back to the
// /start request that began it. Without it, anyone could replay a callback URL.
const StateCookieName = "shell_oauth_state"

// stateCookieMaxAge is how long a User has to finish the provider's consent
// screen. Ten minutes is long enough to read it and short enough to expire.
const stateCookieMaxAge = 600

// Start sends the User to the provider's consent screen.
func Start(provider IdentityProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := randomToken()
		setCookie(c, StateCookieName, state, stateCookieMaxAge)
		c.Redirect(http.StatusFound, provider.AuthCodeURL(state))
	}
}

// Callback receives the User back from the provider, finds or creates their
// Account, and leaves them holding an auth cookie.
func Callback(provider IdentityProvider, accounts account.Store, logins *login.Store, webBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		want, err := c.Cookie(StateCookieName)
		if err != nil || want == "" || c.Query("state") != want {
			c.JSON(http.StatusBadRequest, gin.H{"error": "login expired, please try again"})
			return
		}
		setCookie(c, StateCookieName, "", -1)

		id, err := provider.Exchange(c.Request.Context(), c.Query("code"))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "could not complete login"})
			return
		}

		a, err := accounts.FindOrCreate(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save account"})
			return
		}

		l := logins.Create(a.ID)
		setCookie(c, CookieName, l.Token, int(login.TTL.Seconds()))
		c.Redirect(http.StatusFound, webBaseURL)
	}
}

// Logout drops the Login the cookie points at and clears the cookie.
func Logout(logins *login.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(CookieName); err == nil {
			logins.Delete(token)
		}
		setCookie(c, CookieName, "", -1)
		c.Status(http.StatusNoContent)
	}
}

// setCookie writes an HTTP-only cookie scoped to the whole site.
func setCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
