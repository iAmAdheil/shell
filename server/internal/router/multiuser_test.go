package router_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// These tests put two Users in one Session, which is the whole point of the
// app: one shell, one input stream, one Scrollback, several people.

func TestASecondUserSeesTheScrollbackFromBeforeTheyJoined(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)

	adaConn := h.connect(t, code, adaCookie)
	h.shells.Last().Say("output from before Grace arrived")
	readOutput(t, adaConn)

	graceConn := h.connect(t, code, h.logInAs(t, grace))

	if got := readOutput(t, graceConn); !strings.Contains(got, "output from before Grace arrived") {
		t.Errorf("the joining User saw %q, want the earlier output replayed", got)
	}
}

func TestScrollbackIsReplayedInOrder(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)

	adaConn := h.connect(t, code, adaCookie)
	for _, line := range []string{"first\n", "second\n", "third\n"} {
		h.shells.Last().Say(line)
		readOutput(t, adaConn)
	}

	graceConn := h.connect(t, code, h.logInAs(t, grace))

	got := readOutput(t, graceConn)
	if want := "first\nsecond\nthird\n"; got != want {
		t.Errorf("replayed %q, want %q", got, want)
	}
}

func TestBothUsersSeeTheSameLiveOutput(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	graceConn := h.connect(t, code, h.logInAs(t, grace))

	h.shells.Last().Say("everyone sees this")

	if got := readOutput(t, adaConn); !strings.Contains(got, "everyone sees this") {
		t.Errorf("Ada saw %q", got)
	}
	if got := readOutput(t, graceConn); !strings.Contains(got, "everyone sees this") {
		t.Errorf("Grace saw %q", got)
	}
}

func TestInputFromBothUsersMergesIntoOneStream(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	graceConn := h.connect(t, code, h.logInAs(t, grace))

	if err := adaConn.WriteMessage(websocket.BinaryMessage, []byte("ada-typed ")); err != nil {
		t.Fatalf("Ada write: %v", err)
	}
	waitFor(t, "Ada's keystrokes to reach the shell", func() bool {
		return h.shells.Last().Input() == "ada-typed "
	})

	if err := graceConn.WriteMessage(websocket.BinaryMessage, []byte("grace-typed")); err != nil {
		t.Fatalf("Grace write: %v", err)
	}

	waitFor(t, "both Users' keystrokes to reach the one shell", func() bool {
		return h.shells.Last().Input() == "ada-typed grace-typed"
	})
}

func TestTheSessionSurvivesOneUserLeaving(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	graceConn := h.connect(t, code, h.logInAs(t, grace))
	h.shells.Last().Say("ready")
	readOutput(t, adaConn)
	readOutput(t, graceConn)

	adaConn.Close()

	// Grace is still connected, so the Session must stay open for her.
	h.shells.Last().Say("still running")
	if got := readOutput(t, graceConn); !strings.Contains(got, "still running") {
		t.Errorf("Grace saw %q after Ada left, want the Session still running", got)
	}
	if h.shells.Last().IsClosed() {
		t.Error("the container was destroyed while a User was still connected")
	}
}

func TestTheSessionEndsWhenTheLastUserLeaves(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	graceConn := h.connect(t, code, h.logInAs(t, grace))
	h.shells.Last().Say("ready")
	readOutput(t, adaConn)
	readOutput(t, graceConn)

	adaConn.Close()
	graceConn.Close()

	waitFor(t, "the container to be destroyed", h.shells.Last().IsClosed)

	rec := h.do(http.MethodGet, "/api/sessions/"+code, adaCookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("the Code still works after everyone left: status %d", rec.Code)
	}
}

func TestAJoiningUserIsSentTheRoster(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)

	conn := h.connect(t, code, adaCookie)

	got := readRoster(t, conn)
	if want := []string{"Ada Lovelace"}; !reflect.DeepEqual(rosterNames(got), want) {
		t.Errorf("roster = %v, want %v", rosterNames(got), want)
	}
	if got.Users[0].AvatarURL != ada.AvatarURL {
		t.Errorf("avatar = %q, want the one from the OAuth provider %q", got.Users[0].AvatarURL, ada.AvatarURL)
	}
}

func TestTheRosterUpdatesForEveryoneWhenSomeoneJoins(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	readRoster(t, adaConn)

	graceConn := h.connect(t, code, h.logInAs(t, grace))

	both := []string{"Ada Lovelace", "Grace Hopper"}
	if got := rosterNames(readRoster(t, adaConn)); !reflect.DeepEqual(got, both) {
		t.Errorf("Ada's roster = %v, want %v", got, both)
	}
	if got := rosterNames(readRoster(t, graceConn)); !reflect.DeepEqual(got, both) {
		t.Errorf("Grace's roster = %v, want %v", got, both)
	}
}

func TestTheRosterUpdatesForEveryoneWhenSomeoneLeaves(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)
	adaConn := h.connect(t, code, adaCookie)
	graceConn := h.connect(t, code, h.logInAs(t, grace))
	readRoster(t, adaConn)

	graceConn.Close()

	want := []string{"Ada Lovelace"}
	deadline := time.Now().Add(settle)
	for time.Now().Before(deadline) {
		if reflect.DeepEqual(rosterNames(readRoster(t, adaConn)), want) {
			return
		}
	}
	t.Errorf("Ada was never told Grace left")
}

func TestTheRosterNamesUsersByTheirAccountNotANickname(t *testing.T) {
	h := newHarness(t)
	adaCookie := h.logIn(t)
	code := h.createSession(t, adaCookie)

	got := readRoster(t, h.connect(t, code, adaCookie))

	if len(got.Users) != 1 {
		t.Fatalf("roster has %d Users, want 1", len(got.Users))
	}
	if got.Users[0].Name != ada.Name {
		t.Errorf("name = %q, want the account name %q", got.Users[0].Name, ada.Name)
	}
	if got.Users[0].ID == "" {
		t.Error("roster entry has no account ID")
	}
}
