package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"backend/internal/login"
	"backend/internal/router"
	"backend/internal/session"
	"backend/internal/shell"
)

// This is the only test that runs the whole stack at once: a real WebSocket
// carrying real keystrokes to a real shell in a real container. Everything
// else above the shell seam uses a fake.

func TestARealContainerRunsARealCommandOverTheSocket(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") != "" {
		t.Skip("SKIP_DOCKER_TESTS is set")
	}

	runner, err := shell.NewDockerRunner()
	if err != nil {
		t.Skipf("no Docker to talk to: %v", err)
	}

	accounts := newFakeAccounts()
	google := &fakeProvider{identity: ada}
	sessions := session.NewStore(runner)
	t.Cleanup(sessions.CloseAll)

	engine := router.New(router.Deps{
		Accounts:   accounts,
		Logins:     login.NewStore(),
		Google:     google,
		Sessions:   sessions,
		WebBaseURL: "http://web.test",
	})
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	h := &harness{engine: engine, server: server, accounts: accounts, google: google}
	authCookie := h.logIn(t)
	code := h.createSession(t, authCookie)

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/sessions/" + code + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{
		"Cookie": {authCookie.Name + "=" + authCookie.Value},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := conn.WriteMessage(websocket.BinaryMessage, []byte("echo end-to-end-works\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var seen strings.Builder
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		seen.Write(data)
		if strings.Contains(seen.String(), "end-to-end-works") {
			return
		}
	}
	t.Fatalf("never saw the command output, saw: %q", seen.String())
}
