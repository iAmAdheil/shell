package router_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"backend/internal/auth"
	"backend/internal/router"
)

// devSecret is the value a test sets DEV_LOGIN_SECRET to.
const devSecret = "test-secret"

// withDevLogin turns the dev login on for one harness.
func withDevLogin(d *router.Deps) { d.DevLoginSecret = devSecret }

// devLogIn signs in as the named User through the dev routes and returns the
// auth cookie a browser would hold. It follows the real redirect, so it proves
// that DevStart builds a URL the callback accepts.
func (h *harness) devLogIn(t *testing.T, who string) *http.Cookie {
	t.Helper()

	start := h.do(http.MethodGet, "/api/auth/dev/start?u="+url.QueryEscape(who)+"&k="+devSecret)
	if start.Code != http.StatusFound {
		t.Fatalf("GET dev start = %d, want %d (body %s)", start.Code, http.StatusFound, start.Body)
	}
	state := cookie(t, start, auth.StateCookieName)

	next, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}

	back := h.do(http.MethodGet, next.RequestURI(), state)
	if back.Code != http.StatusFound {
		t.Fatalf("GET dev callback = %d, want %d (body %s)", back.Code, http.StatusFound, back.Body)
	}
	return cookie(t, back, auth.CookieName)
}

// me reads the logged-in User's own Account.
func (h *harness) me(t *testing.T, authCookie *http.Cookie) (id, name string) {
	t.Helper()

	rec := h.do(http.MethodGet, "/api/me", authCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/me = %d, want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return body.ID, body.Name
}

// This is the guard that keeps the dev login out of production. Without the
// secret the routes are never registered, so there is no handler to reach.
func TestTheDevLoginDoesNotExistWithoutTheSecret(t *testing.T) {
	h := newHarness(t)

	for _, path := range []string{
		"/api/auth/dev/start?u=ada",
		"/api/auth/dev/start?u=ada&k=" + devSecret,
		"/api/auth/dev/callback?code=ada&state=x&k=" + devSecret,
	} {
		if rec := h.do(http.MethodGet, path); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func TestTheDevLoginRefusesAWrongSecret(t *testing.T) {
	h := newHarness(t, withDevLogin)

	for _, path := range []string{
		"/api/auth/dev/start?u=ada",
		"/api/auth/dev/start?u=ada&k=wrong",
		"/api/auth/dev/callback?code=ada&state=x&k=wrong",
	} {
		if rec := h.do(http.MethodGet, path); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func TestTheDevLoginNeedsAUserName(t *testing.T) {
	h := newHarness(t, withDevLogin)

	rec := h.do(http.MethodGet, "/api/auth/dev/start?k="+devSecret)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET dev start with no ?u = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTheDevLoginSignsInAsTheNamedUser(t *testing.T) {
	h := newHarness(t, withDevLogin)

	_, name := h.me(t, h.devLogIn(t, "ada"))
	if name != "ada" {
		t.Errorf("name = %q, want %q", name, "ada")
	}
}

// Rejoining a Session as the same User is what these logins are for, so the
// same name must always reach the same Account.
func TestTheDevLoginGivesOneNameOneAccount(t *testing.T) {
	h := newHarness(t, withDevLogin)

	first, _ := h.me(t, h.devLogIn(t, "ada"))
	again, _ := h.me(t, h.devLogIn(t, "ada"))
	if first != again {
		t.Errorf("second login as ada gave account %q, want the first one %q", again, first)
	}

	other, _ := h.me(t, h.devLogIn(t, "grace"))
	if other == first {
		t.Errorf("grace got ada's account %q, want a different one", other)
	}
}

// The point of the whole change: two Users in one Session, from one machine.
func TestTwoDevUsersShareOneSession(t *testing.T) {
	h := newHarness(t, withDevLogin)

	adaCookie := h.devLogIn(t, "ada")
	code := h.createSession(t, adaCookie)

	adaConn := h.connect(t, code, adaCookie)
	readRoster(t, adaConn)

	graceConn := h.connect(t, code, h.devLogIn(t, "grace"))

	both := []string{"ada", "grace"}
	if got := rosterNames(readRoster(t, adaConn)); !reflect.DeepEqual(got, both) {
		t.Errorf("Ada's roster = %v, want %v", got, both)
	}
	if got := rosterNames(readRoster(t, graceConn)); !reflect.DeepEqual(got, both) {
		t.Errorf("Grace's roster = %v, want %v", got, both)
	}
}
