package router_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/login"
	"backend/internal/router"
)

var ada = account.Identity{
	Provider:       "google",
	ProviderUserID: "12345",
	Name:           "Ada Lovelace",
	AvatarURL:      "https://example.test/ada.png",
}

type harness struct {
	engine   *gin.Engine
	accounts *fakeAccounts
	google   *fakeProvider
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		accounts: newFakeAccounts(),
		google:   &fakeProvider{identity: ada},
	}
	h.engine = router.New(router.Deps{
		Accounts:   h.accounts,
		Logins:     login.NewStore(),
		Google:     h.google,
		WebBaseURL: "http://web.test",
	})
	return h
}

// do sends one request through the router, carrying the given cookies.
func (h *harness) do(method, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	h.engine.ServeHTTP(rec, req)
	return rec
}

// logIn runs the whole OAuth flow against the fake provider and returns the
// auth cookie a real browser would be left holding.
func (h *harness) logIn(t *testing.T) *http.Cookie {
	t.Helper()

	start := h.do(http.MethodGet, "/api/auth/google/start")
	state := cookie(t, start, auth.StateCookieName)

	back := h.do(http.MethodGet, "/api/auth/google/callback?code=any&state="+state.Value, state)
	return cookie(t, back, auth.CookieName)
}

// cookie pulls one named cookie out of a response, or fails the test.
func cookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %q cookie in response (status %d, cookies %v)", name, rec.Code, rec.Result().Cookies())
	return nil
}

func TestMeRejectsARequestWithNoCookie(t *testing.T) {
	rec := newHarness(t).do(http.MethodGet, "/api/me")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoggingInSetsAnHttpOnlyCookieThatUnlocksMe(t *testing.T) {
	h := newHarness(t)

	authCookie := h.logIn(t)

	if !authCookie.HttpOnly {
		t.Error("auth cookie is not HttpOnly, so page scripts can read it")
	}

	rec := h.do(http.MethodGet, "/api/me", authCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/me = %d, want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), ada.Name) {
		t.Errorf("body = %s, want it to name %q", rec.Body, ada.Name)
	}
}

func TestLoggingOutRejectsTheOldCookie(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)

	out := h.do(http.MethodPost, "/api/auth/logout", authCookie)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout = %d, want %d", out.Code, http.StatusNoContent)
	}
	if cleared := cookie(t, out, auth.CookieName); cleared.MaxAge >= 0 {
		t.Errorf("logout left the cookie with MaxAge %d, want a negative value to clear it", cleared.MaxAge)
	}

	rec := h.do(http.MethodGet, "/api/me", authCookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/me after logout = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMeRejectsACookieTheServerNeverIssued(t *testing.T) {
	h := newHarness(t)

	rec := h.do(http.MethodGet, "/api/me", &http.Cookie{Name: auth.CookieName, Value: "made-up"})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestTheAuthCookieLastsThirtyDays(t *testing.T) {
	h := newHarness(t)

	authCookie := h.logIn(t)

	const thirtyDays = 30 * 24 * 60 * 60
	if authCookie.MaxAge != thirtyDays {
		t.Errorf("MaxAge = %d, want %d (30 days)", authCookie.MaxAge, thirtyDays)
	}
}

func TestActivityRefreshesTheAuthCookie(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)

	rec := h.do(http.MethodGet, "/api/me", authCookie)

	const thirtyDays = 30 * 24 * 60 * 60
	refreshed := cookie(t, rec, auth.CookieName)
	if refreshed.Value != authCookie.Value {
		t.Errorf("refreshed cookie changed value, want the same token")
	}
	if refreshed.MaxAge != thirtyDays {
		t.Errorf("refreshed MaxAge = %d, want %d (30 days)", refreshed.MaxAge, thirtyDays)
	}
}

func TestLoggingInTwiceReusesTheSameAccount(t *testing.T) {
	h := newHarness(t)

	first := h.do(http.MethodGet, "/api/me", h.logIn(t)).Body.String()
	second := h.do(http.MethodGet, "/api/me", h.logIn(t)).Body.String()

	if first != second {
		t.Errorf("second login reported a different account:\n first  = %s\n second = %s", first, second)
	}
}

func TestACallbackWithoutAMatchingStateIsRejected(t *testing.T) {
	h := newHarness(t)
	start := h.do(http.MethodGet, "/api/auth/google/start")
	state := cookie(t, start, auth.StateCookieName)

	replayed := h.do(http.MethodGet, "/api/auth/google/callback?code=any&state=someone-elses-state", state)

	if replayed.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", replayed.Code, http.StatusBadRequest)
	}
	for _, c := range replayed.Result().Cookies() {
		if c.Name == auth.CookieName && c.Value != "" {
			t.Error("a callback with the wrong state still logged the visitor in")
		}
	}
}

func TestAFailedProviderExchangeDoesNotLogAnyoneIn(t *testing.T) {
	h := newHarness(t)
	h.google.err = errors.New("google said no")

	start := h.do(http.MethodGet, "/api/auth/google/start")
	state := cookie(t, start, auth.StateCookieName)
	back := h.do(http.MethodGet, "/api/auth/google/callback?code=any&state="+state.Value, state)

	if back.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", back.Code, http.StatusBadGateway)
	}
	for _, c := range back.Result().Cookies() {
		if c.Name == auth.CookieName && c.Value != "" {
			t.Error("a failed exchange still logged the visitor in")
		}
	}
}
