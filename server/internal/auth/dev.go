package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"backend/internal/account"
)

// DevProviderName is the provider recorded on an Account made by a dev login.
// It is not "google", so a dev Account can never collide with a real one.
const DevProviderName = "dev"

// DevProvider turns a name from a URL into an Identity, with no browser and no
// network. It exists so that one person can be several Users at once while
// testing a shared Session.
//
// The routes that use it are registered only when DEV_LOGIN_SECRET is set, so
// a build with that variable unset cannot reach this code at all.
type DevProvider struct{}

// Exchange treats the OAuth code as the User's name. The same name always
// gives the same Account, because rejoining a Session as the same User is
// exactly what these logins are for.
func (DevProvider) Exchange(_ context.Context, code string) (account.Identity, error) {
	if code == "" {
		return account.Identity{}, errors.New("dev login needs a user name")
	}
	return account.Identity{
		Provider:       DevProviderName,
		ProviderUserID: code,
		Name:           code,
	}, nil
}

// DevStart begins a dev login for the User named in ?u. It does the same work
// as Start, but it reads the name from the request instead of asking a
// provider where to send the browser.
func DevStart(secret, appBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		who := c.Query("u")
		if who == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name the user with ?u=, for example ?u=ada"})
			return
		}

		state := randomToken()
		setCookie(c, StateCookieName, state, stateCookieMaxAge)

		next := appBaseURL + "/api/auth/dev/callback?" + url.Values{
			"code":  {who},
			"state": {state},
			"k":     {secret},
		}.Encode()
		c.Redirect(http.StatusFound, next)
	}
}

// RequireDevSecret drops any request that does not carry the dev secret in ?k.
// It answers 404 rather than 401, so a scan of a deployed build cannot tell
// that these routes exist.
func RequireDevSecret(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := []byte(c.Query("k"))
		if subtle.ConstantTimeCompare(got, []byte(secret)) != 1 {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
