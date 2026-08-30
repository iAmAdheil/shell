// Package shell runs the real OS shell a Session drives.
//
// Shell is the seam Docker sits behind. Everything above it — Sessions,
// Scrollback, the WebSocket — works against the interface, so a test can
// drive the whole path with no container.
package shell

import (
	"context"
	"io"
)

// Shell is one running shell process. Read returns its output, Write sends it
// input, and Close ends it and destroys whatever it runs inside.
type Shell interface {
	io.ReadWriteCloser

	// Resize tells the shell how big the terminal is. Programs like vim and
	// top read this to lay themselves out.
	Resize(rows, cols uint16) error
}

// Runner starts a Shell. One Session holds one Shell.
type Runner interface {
	Start(ctx context.Context, rows, cols uint16) (Shell, error)
}
