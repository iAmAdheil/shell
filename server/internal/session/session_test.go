package session_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"backend/internal/session"
	"backend/internal/shell/shelltest"
)

const settle = 2 * time.Second

func newStore(t *testing.T) (*session.Store, *shelltest.Runner) {
	t.Helper()

	runner := &shelltest.Runner{}
	store := session.NewStore(runner)
	t.Cleanup(store.CloseAll)
	return store, runner
}

// waitFor polls until cond holds, or fails the test. Output travels through a
// goroutine, so a test cannot read it the instant it is written.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(settle)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestCreatingASessionStartsAShell(t *testing.T) {
	store, runner := newStore(t)

	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if s.Code == "" {
		t.Error("Session has no Code")
	}
	if runner.Count() != 1 {
		t.Errorf("started %d shells, want 1", runner.Count())
	}
	if rows, cols := runner.Last().Size(); rows != 24 || cols != 80 {
		t.Errorf("shell sized %dx%d, want 24x80", rows, cols)
	}
}

func TestEverySessionGetsItsOwnCodeAndContainer(t *testing.T) {
	store, runner := newStore(t)

	first, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	second, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if first.Code == second.Code {
		t.Errorf("two Sessions share the Code %q", first.Code)
	}
	if runner.Count() != 2 {
		t.Errorf("started %d shells for 2 Sessions, want 2", runner.Count())
	}
}

func TestASessionCanBeFoundByItsCode(t *testing.T) {
	store, _ := newStore(t)
	created, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, ok := store.ByCode(created.Code)

	if !ok {
		t.Fatalf("no Session for Code %q", created.Code)
	}
	if found != created {
		t.Error("ByCode returned a different Session")
	}
}

func TestAnUnknownCodeFindsNothing(t *testing.T) {
	store, _ := newStore(t)

	if _, ok := store.ByCode("never-existed"); ok {
		t.Error("an unknown Code found a Session")
	}
}

func TestShellOutputIsKeptAsScrollback(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	runner.Last().Say("hello ")
	runner.Last().Say("world")

	waitFor(t, "Scrollback to hold both writes", func() bool {
		return string(s.Scrollback()) == "hello world"
	})
}

func TestTypingReachesTheShell(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Type([]byte("echo hi\n")); err != nil {
		t.Fatalf("Type: %v", err)
	}

	waitFor(t, "the shell to receive the keystrokes", func() bool {
		return runner.Last().Input() == "echo hi\n"
	})
}

func TestAConnectedUserSeesLiveOutput(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	seen := make(chan []byte, 8)
	leave := s.Join(ada, func(b []byte) { seen <- append([]byte(nil), b...) }, func([]session.Watcher) {})
	defer leave()

	runner.Last().Say("live output")

	select {
	case got := <-seen:
		if string(got) != "live output" {
			t.Errorf("subscriber saw %q, want %q", got, "live output")
		}
	case <-time.After(settle):
		t.Fatal("subscriber saw nothing")
	}
}

func TestEndingASessionDestroysTheContainerAndForgetsTheSession(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sh := runner.Last()

	store.End(s.Code)

	if !sh.IsClosed() {
		t.Error("the shell is still running, so the container was not destroyed")
	}
	if _, ok := store.ByCode(s.Code); ok {
		t.Error("the store still knows the Code of an ended Session")
	}
}

func TestScrollbackDiesWithTheSession(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	runner.Last().Say("secret output")
	waitFor(t, "Scrollback to fill", func() bool { return len(s.Scrollback()) > 0 })

	store.End(s.Code)

	if _, ok := store.ByCode(s.Code); ok {
		t.Fatal("Session still in the store")
	}
	if got := s.Scrollback(); len(got) != 0 {
		t.Errorf("Scrollback survived the Session: %q", got)
	}
}

func TestASessionCodeIsUrlSafe(t *testing.T) {
	store, _ := newStore(t)

	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const allowed = "abcdefghjkmnpqrstuvwxyz23456789"
	for _, r := range s.Code {
		if !strings.ContainsRune(allowed, r) {
			t.Errorf("Code %q holds %q, which is easy to misread or needs escaping", s.Code, r)
		}
	}
}

func TestASessionEndsWhenItsLastWatcherLeaves(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sh := runner.Last()

	leave := s.Join(ada, func([]byte) {}, func([]session.Watcher) {})
	leave()

	waitFor(t, "the container to be destroyed", sh.IsClosed)
	if _, ok := store.ByCode(s.Code); ok {
		t.Error("the store still knows the Code of an ended Session")
	}
}

func TestASessionSurvivesWhileOneWatcherRemains(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sh := runner.Last()

	leaveFirst := s.Join(ada, func([]byte) {}, func([]session.Watcher) {})
	defer s.Join(grace, func([]byte) {}, func([]session.Watcher) {})()
	leaveFirst()

	if sh.IsClosed() {
		t.Error("the Session ended while a watcher was still connected")
	}
	if _, ok := store.ByCode(s.Code); !ok {
		t.Error("the store forgot a Session that still has a watcher")
	}
}

func TestASessionNobodyJoinsIsReaped(t *testing.T) {
	runner := &shelltest.Runner{}
	store := session.NewStore(runner)
	store.JoinGrace = 30 * time.Millisecond
	t.Cleanup(store.CloseAll)

	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	waitFor(t, "the abandoned Session to be reaped", runner.Last().IsClosed)
	if _, ok := store.ByCode(s.Code); ok {
		t.Error("the store still knows an abandoned Session")
	}
}
