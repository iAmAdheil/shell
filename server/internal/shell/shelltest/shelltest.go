// Package shelltest provides a Shell that does not need Docker, so tests above
// the shell seam run fast and offline.
package shelltest

import (
	"context"
	"errors"
	"io"
	"sync"

	"backend/internal/shell"
)

// Shell stands in for a real shell in a container. Output is whatever a
// test feeds it; input is recorded so a test can check what the shell was told.
type Shell struct {
	mu     sync.Mutex
	input  []byte
	rows   uint16
	cols   uint16
	closed bool

	output chan []byte
	rest   []byte
}

func New() *Shell {
	return &Shell{output: make(chan []byte, 32)}
}

// Say makes the shell produce output, as if a command had printed it.
func (f *Shell) Say(s string) { f.output <- []byte(s) }

func (f *Shell) Read(p []byte) (int, error) {
	if len(f.rest) == 0 {
		b, ok := <-f.output
		if !ok {
			return 0, io.EOF
		}
		f.rest = b
	}
	n := copy(p, f.rest)
	f.rest = f.rest[n:]
	return n, nil
}

func (f *Shell) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, errors.New("shell is closed")
	}
	f.input = append(f.input, p...)
	return len(p), nil
}

func (f *Shell) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.output)
	}
	return nil
}

func (f *Shell) Resize(rows, cols uint16) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows, f.cols = rows, cols
	return nil
}

func (f *Shell) Input() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.input)
}

func (f *Shell) Size() (uint16, uint16) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows, f.cols
}

func (f *Shell) IsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// Runner hands out fakeShells and remembers every one it started.
type Runner struct {
	mu      sync.Mutex
	started []*Shell
	// Err, when set, makes Start fail.
	Err error
}

func (r *Runner) Start(_ context.Context, rows, cols uint16) (shell.Shell, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Err != nil {
		return nil, r.Err
	}
	s := New()
	s.rows, s.cols = rows, cols
	r.started = append(r.started, s)
	return s, nil
}

// Last returns the most recently started Shell.
func (r *Runner) Last() *Shell {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started[len(r.started)-1]
}

func (r *Runner) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.started)
}
