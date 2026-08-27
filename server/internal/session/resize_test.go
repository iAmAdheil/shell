package session_test

import (
	"context"
	"testing"

	"backend/internal/session"
	"backend/internal/shell/shelltest"
)

// The one shared shell can only be one size, so it is sized so that its output
// fits every connected viewport: the smallest number of rows anyone has, and
// the smallest number of columns anyone has (ticket 07).

func joined(t *testing.T, s *session.Session, who session.Watcher) *session.Membership {
	t.Helper()

	m := s.Join(who, func([]byte) {}, func([]session.Watcher) {})
	t.Cleanup(m.Leave)
	return m
}

func expectSize(t *testing.T, sh *shelltest.Shell, wantRows, wantCols uint16, why string) {
	t.Helper()

	waitFor(t, why, func() bool {
		rows, cols := sh.Size()
		return rows == wantRows && cols == wantCols
	})
}

func TestTheShellIsSizedToTheSmallerOfTwoViewports(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	joined(t, s, ada).Resize(50, 200)
	joined(t, s, grace).Resize(30, 100)

	expectSize(t, runner.Last(), 30, 100, "the shell to match Grace's smaller window")
}

func TestTheShellShrinksWhenASmallerUserJoins(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	joined(t, s, ada).Resize(50, 200)
	expectSize(t, runner.Last(), 50, 200, "the shell to match the only window")

	joined(t, s, grace).Resize(20, 60)

	expectSize(t, runner.Last(), 20, 60, "the shell to shrink to the new smaller window")
}

func TestTheShellGrowsWhenTheSmallestUserLeaves(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	joined(t, s, ada).Resize(50, 200)
	small := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})
	small.Resize(20, 60)
	expectSize(t, runner.Last(), 20, 60, "the shell to match the smallest window")

	small.Leave()

	expectSize(t, runner.Last(), 50, 200, "the shell to grow back to the remaining window")
}

func TestResizingAWindowResizesTheShell(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	big := joined(t, s, ada)
	big.Resize(50, 200)
	joined(t, s, grace).Resize(30, 100)
	expectSize(t, runner.Last(), 30, 100, "the shell to match the smaller window")

	// Ada drags her window smaller than Grace's, so she is now the limit.
	big.Resize(10, 40)

	expectSize(t, runner.Last(), 10, 40, "the shell to follow the newly smallest window")
}

func TestEachDimensionIsTakenFromWhoeverIsSmallestThere(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Neither window is smaller overall: Ada is shorter, Grace is narrower.
	// Output has to fit both, so the shell takes the smaller of each.
	joined(t, s, ada).Resize(20, 200)
	joined(t, s, grace).Resize(50, 60)

	expectSize(t, runner.Last(), 20, 60, "the shell to fit inside both windows")
}

func TestAUserWhoHasNotReportedASizeDoesNotShrinkTheShell(t *testing.T) {
	store, runner := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	joined(t, s, ada).Resize(40, 120)

	// Grace's browser has connected but has not said how big her window is.
	joined(t, s, grace)

	expectSize(t, runner.Last(), 40, 120, "the shell to ignore a window with no reported size")
}
