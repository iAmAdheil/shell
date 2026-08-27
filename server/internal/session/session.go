// Package session holds the running Sessions. A Session is one shell, its
// Scrollback, and everyone currently watching it (CONTEXT.md).
//
// Everything here lives in server memory, so a restart ends every Session.
// That is ticket 03's decision, not an oversight.
package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"backend/internal/shell"
)

// codeAlphabet leaves out characters that are easy to misread aloud or in a
// shared link: 0/o, 1/l/i. A Session Code is meant to be passed to a friend.
const codeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// codeLength gives about 40 bits of choice, which is far more than enough for
// the handful of Sessions one server instance holds at a time.
const codeLength = 8

// DefaultRows and DefaultCols size a Session's terminal until the browser
// reports the real viewport, which it does as soon as it connects.
const (
	DefaultRows = 24
	DefaultCols = 80
)

// readChunk is the biggest single read taken from a shell.
const readChunk = 32 * 1024

// defaultJoinGrace is how long a freshly created Session waits for its first
// watcher. Without it, a browser that never connects would leave a container
// running with nobody to end it.
const defaultJoinGrace = 60 * time.Second

// Watcher is one connected User, as the roster shows them. Identity comes
// from the account they logged in with (ticket 10), never a nickname.
type Watcher struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
}

// watch is one open browser tab. A User with two tabs open has two of these
// but appears on the roster once.
type watch struct {
	who      Watcher
	onOutput func([]byte)
	onRoster func([]Watcher)
}

// Session is one shared shell.
type Session struct {
	Code string

	shell shell.Shell

	// store is where this Session removes itself once its last watcher goes.
	store *Store

	mu         sync.Mutex
	scrollback []byte
	watches    map[int]watch
	nextWatch  int
	joined     bool
	ended      bool
}

// Store holds every running Session, keyed by Session Code.
type Store struct {
	runner shell.Runner

	// JoinGrace is how long a new Session waits for its first watcher before
	// it is reaped. Tests shorten it.
	JoinGrace time.Duration

	mu     sync.Mutex
	byCode map[string]*Session
}

func NewStore(runner shell.Runner) *Store {
	return &Store{
		runner:    runner,
		JoinGrace: defaultJoinGrace,
		byCode:    map[string]*Session{},
	}
}

// Create starts a Session: a new Code, a new container, and a goroutine
// pumping the shell's output into Scrollback and out to subscribers.
func (st *Store) Create(ctx context.Context, rows, cols uint16) (*Session, error) {
	sh, err := st.runner.Start(ctx, rows, cols)
	if err != nil {
		return nil, fmt.Errorf("start shell: %w", err)
	}

	s := &Session{
		Code:    st.freshCode(),
		shell:   sh,
		store:   st,
		watches: map[int]watch{},
	}

	st.mu.Lock()
	st.byCode[s.Code] = s
	st.mu.Unlock()

	go st.pump(s)
	time.AfterFunc(st.JoinGrace, func() { s.reapIfNobodyJoined() })
	return s, nil
}

// freshCode returns a Code no running Session is using.
func (st *Store) freshCode() string {
	st.mu.Lock()
	defer st.mu.Unlock()

	for {
		code := randomCode()
		if _, taken := st.byCode[code]; !taken {
			return code
		}
	}
}

// pump moves output from the shell to the Scrollback and to every watcher. It
// returns when the shell stops, which also ends the Session.
func (st *Store) pump(s *Session) {
	buf := make([]byte, readChunk)
	for {
		n, err := s.shell.Read(buf)
		if n > 0 {
			s.broadcast(buf[:n])
		}
		if err != nil {
			break
		}
	}
	st.End(s.Code)
}

// ByCode returns a running Session. An ended Code finds nothing, which is what
// ticket 06 asks for: a Code dies with its Session.
func (st *Store) ByCode(code string) (*Session, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s, ok := st.byCode[code]
	return s, ok
}

// End destroys a Session's container and forgets the Session and its
// Scrollback. Calling it twice is safe.
func (st *Store) End(code string) {
	st.mu.Lock()
	s, ok := st.byCode[code]
	delete(st.byCode, code)
	st.mu.Unlock()

	if ok {
		s.end()
	}
}

// CloseAll ends every Session. The server calls it on the way down so no
// container outlives the process that started it.
func (st *Store) CloseAll() {
	st.mu.Lock()
	all := make([]*Session, 0, len(st.byCode))
	for _, s := range st.byCode {
		all = append(all, s)
	}
	st.byCode = map[string]*Session{}
	st.mu.Unlock()

	for _, s := range all {
		s.end()
	}
}

// Type sends keystrokes to the shell.
func (s *Session) Type(p []byte) error {
	_, err := s.shell.Write(p)
	return err
}

// Resize sets the shell's terminal size.
func (s *Session) Resize(rows, cols uint16) error {
	return s.shell.Resize(rows, cols)
}

// Scrollback returns everything the shell has printed so far. A User who joins
// late is shown all of it (CONTEXT.md).
func (s *Session) Scrollback() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.scrollback...)
}

// Join connects one User to the Session. onOutput receives live terminal
// output. onRoster receives the list of connected Users, immediately and again
// every time it changes.
//
// The returned function is how the User leaves. A Session ends the moment its
// last User goes (CONTEXT.md), so the last call to it destroys the container.
func (s *Session) Join(who Watcher, onOutput func([]byte), onRoster func([]Watcher)) func() {
	s.mu.Lock()
	id := s.nextWatch
	s.nextWatch++
	s.watches[id] = watch{who: who, onOutput: onOutput, onRoster: onRoster}
	s.joined = true
	s.mu.Unlock()

	s.announceRoster()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.watches, id)
			last := len(s.watches) == 0 && !s.ended
			s.mu.Unlock()

			if last {
				s.store.End(s.Code)
				return
			}
			s.announceRoster()
		})
	}
}

// Roster is who is connected right now, in one stable order so that every
// User sees the same list.
func (s *Session) Roster() []Watcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rosterLocked()
}

// rosterLocked collects the distinct Users behind the open tabs. One User with
// two tabs open is one entry.
func (s *Session) rosterLocked() []Watcher {
	seen := make(map[string]Watcher, len(s.watches))
	for _, w := range s.watches {
		seen[w.who.ID] = w.who
	}

	roster := make([]Watcher, 0, len(seen))
	for _, w := range seen {
		roster = append(roster, w)
	}
	sort.Slice(roster, func(i, j int) bool {
		if roster[i].Name != roster[j].Name {
			return roster[i].Name < roster[j].Name
		}
		return roster[i].ID < roster[j].ID
	})
	return roster
}

// announceRoster tells every connected User who else is here.
func (s *Session) announceRoster() {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	roster := s.rosterLocked()
	tell := make([]func([]Watcher), 0, len(s.watches))
	for _, w := range s.watches {
		tell = append(tell, w.onRoster)
	}
	s.mu.Unlock()

	for _, notify := range tell {
		notify(roster)
	}
}

// reapIfNobodyJoined ends a Session that was created but never connected to.
func (s *Session) reapIfNobodyJoined() {
	s.mu.Lock()
	abandoned := !s.joined && !s.ended
	s.mu.Unlock()

	if abandoned {
		s.store.End(s.Code)
	}
}

// broadcast records output and hands it to every watcher.
func (s *Session) broadcast(b []byte) {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.scrollback = append(s.scrollback, b...)
	watchers := make([]func([]byte), 0, len(s.watches))
	for _, w := range s.watches {
		watchers = append(watchers, w.onOutput)
	}
	s.mu.Unlock()

	for _, w := range watchers {
		w(b)
	}
}

// end stops the shell and drops the Scrollback.
func (s *Session) end() {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.scrollback = nil
	s.watches = map[int]watch{}
	s.mu.Unlock()

	_ = s.shell.Close()
}

func randomCode() string {
	b := make([]byte, codeLength)
	rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}
