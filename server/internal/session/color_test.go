package session_test

import (
	"context"
	"testing"

	"backend/internal/session"
)

// One shared shell has one cursor, so a User needs a colour to be told apart
// from the others, and the cursor needs to say who is typing.

// find returns the roster entry for one Name, or the zero Watcher.
func find(ws []session.Watcher, name string) session.Watcher {
	for _, w := range ws {
		if w.Name == name {
			return w
		}
	}
	return session.Watcher{}
}

func newSession(t *testing.T) *session.Session {
	t.Helper()

	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return s
}

func TestTwoUsersInOneSessionGetDifferentColors(t *testing.T) {
	s := newSession(t)

	spy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, spy.record).Leave()
	defer s.Join(grace, func([]byte) {}, func([]session.Watcher) {}).Leave()

	waitFor(t, "both Users to have a colour", func() bool {
		r := spy.latest()
		a, g := find(r, ada.Name), find(r, grace.Name)
		return a.Color != "" && g.Color != "" && a.Color != g.Color
	})
}

func TestAUserKeepsTheirColorWhenTheyComeBack(t *testing.T) {
	s := newSession(t)

	spy := &rosterSpy{}
	stay := s.Join(ada, func([]byte) {}, spy.record)
	defer stay.Leave()

	first := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})
	var was string
	waitFor(t, "Grace to have a colour", func() bool {
		was = find(spy.latest(), grace.Name).Color
		return was != ""
	})
	first.Leave()

	defer s.Join(grace, func([]byte) {}, func([]session.Watcher) {}).Leave()
	waitFor(t, "Grace to come back with the same colour", func() bool {
		return find(spy.latest(), grace.Name).Color == was
	})
}

func TestTypingMakesAUserTheActiveOne(t *testing.T) {
	s := newSession(t)

	spy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, spy.record).Leave()
	graceTypes := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})
	defer graceTypes.Leave()

	// Nobody has typed, so nobody is active.
	waitFor(t, "both Users on the roster", func() bool {
		return len(spy.latest()) == 2
	})
	for _, w := range spy.latest() {
		if w.Active {
			t.Fatalf("%s is active before anyone typed", w.Name)
		}
	}

	if err := graceTypes.Type([]byte("x")); err != nil {
		t.Fatalf("Type: %v", err)
	}

	waitFor(t, "Grace to become the active User", func() bool {
		r := spy.latest()
		return find(r, grace.Name).Active && !find(r, ada.Name).Active
	})
}

func TestTheActiveUserChangesToWhoeverTypedLast(t *testing.T) {
	s := newSession(t)

	spy := &rosterSpy{}
	adaTypes := s.Join(ada, func([]byte) {}, spy.record)
	defer adaTypes.Leave()
	graceTypes := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})
	defer graceTypes.Leave()

	if err := adaTypes.Type([]byte("a")); err != nil {
		t.Fatalf("Type: %v", err)
	}
	waitFor(t, "Ada to become active", func() bool {
		return find(spy.latest(), ada.Name).Active
	})

	if err := graceTypes.Type([]byte("g")); err != nil {
		t.Fatalf("Type: %v", err)
	}
	waitFor(t, "the cursor to pass to Grace", func() bool {
		r := spy.latest()
		return find(r, grace.Name).Active && !find(r, ada.Name).Active
	})
}

// The cursor must not keep the colour of somebody who has left.
func TestNobodyIsActiveOnceTheActiveUserLeaves(t *testing.T) {
	s := newSession(t)

	spy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, spy.record).Leave()

	graceTypes := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})
	if err := graceTypes.Type([]byte("g")); err != nil {
		t.Fatalf("Type: %v", err)
	}
	waitFor(t, "Grace to become active", func() bool {
		return find(spy.latest(), grace.Name).Active
	})

	graceTypes.Leave()

	waitFor(t, "nobody to be active", func() bool {
		r := spy.latest()
		return len(r) == 1 && !r[0].Active
	})
}
