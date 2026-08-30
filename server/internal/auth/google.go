package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"backend/internal/account"
)

// userInfoURL returns the profile of whoever the access token belongs to.
const userInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// GoogleProvider is the real Google implementation of IdentityProvider.
type GoogleProvider struct {
	cfg *oauth2.Config
}

// NewGoogleProvider builds the provider. redirectURL must match one of the
// Authorized redirect URIs on the Google Cloud OAuth client.
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{cfg: &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"openid", "profile", "email"},
	}}
}

func (p *GoogleProvider) AuthCodeURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange turns the one-use code from the callback into the User's profile.
func (p *GoogleProvider) Exchange(ctx context.Context, code string) (account.Identity, error) {
	tok, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return account.Identity{}, fmt.Errorf("exchange code with google: %w", err)
	}

	resp, err := p.cfg.Client(ctx, tok).Get(userInfoURL)
	if err != nil {
		return account.Identity{}, fmt.Errorf("read google profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return account.Identity{}, fmt.Errorf("google profile returned %s", resp.Status)
	}

	// sub is Google's stable ID for this person. It does not change when they
	// rename themselves or change email, so it is what an Account is keyed by.
	var profile struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return account.Identity{}, fmt.Errorf("decode google profile: %w", err)
	}
	if profile.Sub == "" {
		return account.Identity{}, fmt.Errorf("google profile has no subject ID")
	}

	name := profile.Name
	if name == "" {
		name = profile.Email
	}

	return account.Identity{
		Provider:       "google",
		ProviderUserID: profile.Sub,
		Name:           name,
		AvatarURL:      profile.Picture,
	}, nil
}
