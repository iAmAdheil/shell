package router_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// connect opens the Session WebSocket the way a browser would, carrying the
// auth cookie.
func (h *harness) connect(t *testing.T, code string, authCookie *http.Cookie) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(h.server.URL, "http") + "/api/sessions/" + code + "/ws"
	header := http.Header{}
	if authCookie != nil {
		header.Set("Cookie", authCookie.Name+"="+authCookie.Value)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial %s: %v (status %d)", url, err, status)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// readOutput waits for one terminal-output frame.
func readOutput(t *testing.T, conn *websocket.Conn) string {
	t.Helper()

	conn.SetReadDeadline(time.Now().Add(settle))
	kind, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if kind != websocket.BinaryMessage {
		t.Fatalf("got a %d frame, want a binary terminal frame", kind)
	}
	return string(data)
}

// connectExpectingRefusal dials the Session socket and requires it to fail.
func (h *harness) connectExpectingRefusal(t *testing.T, code string, authCookie *http.Cookie) {
	t.Helper()

	url := "ws" + strings.TrimPrefix(h.server.URL, "http") + "/api/sessions/" + code + "/ws"
	header := http.Header{"Cookie": {authCookie.Name + "=" + authCookie.Value}}

	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err == nil {
		conn.Close()
		t.Fatalf("Code %q was accepted, want it refused", code)
	}
}

func TestCreatingASessionNeedsALogin(t *testing.T) {
	h := newHarness(t)

	rec := h.do(http.MethodPost, "/api/sessions")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestALoggedInUserCanCreateASession(t *testing.T) {
	h := newHarness(t)

	code := h.createSession(t, h.logIn(t))

	if code == "" {
		t.Fatal("no Session Code came back")
	}
	if h.shells.Count() != 1 {
		t.Errorf("started %d containers, want 1", h.shells.Count())
	}
}

func TestVisitingWithNoCodeCreatesNoSession(t *testing.T) {
	h := newHarness(t)

	h.do(http.MethodGet, "/api/me", h.logIn(t))

	if h.shells.Count() != 0 {
		t.Errorf("started %d containers just by looking at the app, want 0", h.shells.Count())
	}
}

func TestJoiningASessionReplaysItsScrollback(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)
	h.shells.Last().Say("earlier output")

	conn := h.connect(t, code, authCookie)

	if got := readOutput(t, conn); !strings.Contains(got, "earlier output") {
		t.Errorf("on joining saw %q, want the earlier output replayed", got)
	}
}

func TestTypingInTheBrowserReachesTheShell(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)
	conn := h.connect(t, code, authCookie)

	if err := conn.WriteMessage(websocket.BinaryMessage, []byte("whoami\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	waitFor(t, "the shell to receive the keystrokes", func() bool {
		return h.shells.Last().Input() == "whoami\n"
	})
}

func TestShellOutputReachesTheBrowser(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)
	conn := h.connect(t, code, authCookie)

	h.shells.Last().Say("live from the container")

	if got := readOutput(t, conn); !strings.Contains(got, "live from the container") {
		t.Errorf("browser saw %q, want the live output", got)
	}
}

func TestTheBrowserCanResizeTheTerminal(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)
	conn := h.connect(t, code, authCookie)

	msg, err := json.Marshal(map[string]any{"type": "resize", "rows": 40, "cols": 120})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	waitFor(t, "the shell to be resized", func() bool {
		rows, cols := h.shells.Last().Size()
		return rows == 40 && cols == 120
	})
}

func TestDisconnectingDestroysTheContainer(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)
	conn := h.connect(t, code, authCookie)

	// Read one frame first, so the User is definitely attached before they
	// leave. Otherwise the test could close the socket before the server has
	// registered the watcher at all.
	h.shells.Last().Say("ready")
	readOutput(t, conn)

	conn.Close()

	waitFor(t, "the container to be destroyed", h.shells.Last().IsClosed)
}

func TestConnectingToAnUnknownCodeIsRefused(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)

	url := "ws" + strings.TrimPrefix(h.server.URL, "http") + "/api/sessions/nosuchcode/ws"
	header := http.Header{"Cookie": {authCookie.Name + "=" + authCookie.Value}}
	_, resp, err := websocket.DefaultDialer.Dial(url, header)

	if err == nil {
		t.Fatal("an unknown Session Code was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %v, want %d", resp, http.StatusNotFound)
	}
}

func TestConnectingWithoutALoginIsRefused(t *testing.T) {
	h := newHarness(t)
	code := h.createSession(t, h.logIn(t))

	url := "ws" + strings.TrimPrefix(h.server.URL, "http") + "/api/sessions/" + code + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)

	if err == nil {
		t.Fatal("a logged-out visitor was allowed into a Session")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want %d", resp, http.StatusUnauthorized)
	}
}

func TestAnEndedSessionCodeIsRefused(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)

	conn := h.connect(t, code, authCookie)
	h.shells.Last().Say("ready")
	readOutput(t, conn)
	conn.Close()
	waitFor(t, "the Session to end", h.shells.Last().IsClosed)

	rec := h.do(http.MethodGet, "/api/sessions/"+code, authCookie)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), "ended") {
		t.Errorf("body = %s, want it to say the Session has ended", rec.Body)
	}
}

func TestAnEndedSessionCodeNeverStartsANewSession(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)

	conn := h.connect(t, code, authCookie)
	h.shells.Last().Say("ready")
	readOutput(t, conn)
	conn.Close()
	waitFor(t, "the Session to end", h.shells.Last().IsClosed)

	h.do(http.MethodGet, "/api/sessions/"+code, authCookie)
	h.connectExpectingRefusal(t, code, authCookie)

	if h.shells.Count() != 1 {
		t.Errorf("started %d containers, want 1: an ended Code started a new Session", h.shells.Count())
	}
}

func TestACodeThatNeverExistedIsRefusedClearly(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)

	rec := h.do(http.MethodGet, "/api/sessions/nosuchcode", authCookie)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.Len() == 0 {
		t.Error("no message explaining why the Code was refused")
	}
	if h.shells.Count() != 0 {
		t.Errorf("started %d containers for an unknown Code, want 0", h.shells.Count())
	}
}

func TestARunningSessionCodeIsAccepted(t *testing.T) {
	h := newHarness(t)
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)

	rec := h.do(http.MethodGet, "/api/sessions/"+code, authCookie)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), code) {
		t.Errorf("body = %s, want it to name the Code %q", rec.Body, code)
	}
}

func TestCheckingACodeNeedsALogin(t *testing.T) {
	h := newHarness(t)
	code := h.createSession(t, h.logIn(t))

	rec := h.do(http.MethodGet, "/api/sessions/"+code)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
