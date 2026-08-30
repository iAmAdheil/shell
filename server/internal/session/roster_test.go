package session_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"backend/internal/session"
)

var (
	ada   = session.Watcher{ID: "acct-1", Name: "Ada Lovelace", AvatarURL: "https://example.test/ada.png"}
	grace = session.Watcher{ID: "acct-2", Name: "Grace Hopper", AvatarURL: "https://example.test/grace.png"}
)

// rosterSpy records every roster a watcher is told about.
type rosterSpy struct {
	mu   sync.Mutex
	seen [][]session.Watcher
}

func (r *rosterSpy) record(w []session.Watcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, w)
}

func (r *rosterSpy) latest() []session.Watcher {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.seen) == 0 {
		return nil
	}
	return r.seen[len(r.seen)-1]
}

func names(ws []session.Watcher) []string {
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Name)
	}
	return out
}

func TestAJoiningUserIsOnTheRoster(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	spy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, spy.record).Leave()

	waitFor(t, "Ada to appear on the roster", func() bool {
		return reflect.DeepEqual(names(spy.latest()), []string{"Ada Lovelace"})
	})
}

func TestEveryWatcherIsToldWhenSomeoneJoins(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adaSpy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, adaSpy.record).Leave()
	graceSpy := &rosterSpy{}
	defer s.Join(grace, func([]byte) {}, graceSpy.record).Leave()

	both := []string{"Ada Lovelace", "Grace Hopper"}
	waitFor(t, "Ada to be told Grace arrived", func() bool {
		return reflect.DeepEqual(names(adaSpy.latest()), both)
	})
	waitFor(t, "Grace to see the same roster Ada sees", func() bool {
		return reflect.DeepEqual(names(graceSpy.latest()), both)
	})
}

func TestLeavingTakesAUserOffTheRoster(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adaSpy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, adaSpy.record).Leave()
	graceLeaves := s.Join(grace, func([]byte) {}, func([]session.Watcher) {})

	graceLeaves.Leave()

	waitFor(t, "Grace to drop off Ada's roster", func() bool {
		return reflect.DeepEqual(names(adaSpy.latest()), []string{"Ada Lovelace"})
	})
}

func TestOneUserWithTwoTabsIsListedOnce(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	spy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, spy.record).Leave()
	defer s.Join(ada, func([]byte) {}, func([]session.Watcher) {}).Leave()

	waitFor(t, "Ada to be listed once, not twice", func() bool {
		return reflect.DeepEqual(names(spy.latest()), []string{"Ada Lovelace"})
	})
}

func TestClosingOneOfTwoTabsKeepsTheUserOnTheRoster(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	spy := &rosterSpy{}
	defer s.Join(grace, func([]byte) {}, spy.record).Leave()
	closeOneTab := s.Join(ada, func([]byte) {}, func([]session.Watcher) {})
	defer s.Join(ada, func([]byte) {}, func([]session.Watcher) {}).Leave()

	closeOneTab.Leave()

	waitFor(t, "Ada to stay on the roster with one tab still open", func() bool {
		return reflect.DeepEqual(names(spy.latest()), []string{"Ada Lovelace", "Grace Hopper"})
	})
}

func TestTheRosterIsInTheSameOrderForEveryone(t *testing.T) {
	store, _ := newStore(t)
	s, err := store.Create(context.Background(), 24, 80)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Join out of alphabetical order. Everyone must still see one order.
	graceSpy := &rosterSpy{}
	defer s.Join(grace, func([]byte) {}, graceSpy.record).Leave()
	adaSpy := &rosterSpy{}
	defer s.Join(ada, func([]byte) {}, adaSpy.record).Leave()

	want := []string{"Ada Lovelace", "Grace Hopper"}
	waitFor(t, "both Users to see the same roster order", func() bool {
		return reflect.DeepEqual(names(adaSpy.latest()), want) &&
			reflect.DeepEqual(names(graceSpy.latest()), want)
	})
}
