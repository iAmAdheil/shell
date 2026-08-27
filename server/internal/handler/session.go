package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/session"
)

// Terminal traffic rides the socket as binary frames, in both directions.
// Control messages ride as JSON text frames. Splitting them by frame type
// keeps terminal bytes raw: no encoding, and no guess about what is text.
const (
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingEvery   = (pongWait * 9) / 10
	maxInputMsg = 8 * 1024
)

// outputQueue is how many frames may wait for one slow browser before they
// are dropped. A stuck reader must not stall the Session for everyone else.
const outputQueue = 256

// frame is one WebSocket message waiting to go out. Terminal bytes go as
// binary, roster updates as JSON text.
type frame struct {
	kind int
	data []byte
}

// upgrader rejects cross-origin sockets. The browser sends the auth cookie
// automatically, so without this check another site could open a Session on a
// logged-in User's behalf.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || sameHost(origin, r.Host)
	},
}

// CreateSession starts a Session for the logged-in User and returns its Code.
func CreateSession(sessions *session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := sessions.Create(c.Request.Context(), session.DefaultRows, session.DefaultCols)
		if err != nil {
			log.Printf("create session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start a Session"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"code": s.Code})
	}
}

// endedMessage covers both an ended Code and one that never existed. The
// server forgets a Session the moment it ends (ticket 06), so it genuinely
// cannot tell the two apart, and a message that claimed to would be a guess.
const endedMessage = "this Session has ended, or that Session Code is wrong"

// CheckSession reports whether a Session Code still opens a Session. The
// browser asks before it opens a terminal, so a dead Code shows a message
// instead of an empty black rectangle.
func CheckSession(sessions *session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, ok := sessions.ByCode(c.Param("code"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": endedMessage})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": s.Code})
	}
}

// control is a JSON message from the browser.
type control struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// JoinSession streams one User's terminal: Scrollback first, then live output,
// while their keystrokes go the other way. It also sends the roster of
// connected Users, and again every time that roster changes.
func JoinSession(sessions *session.Store, accounts account.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, ok := sessions.ByCode(c.Param("code"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": endedMessage})
			return
		}

		me, err := accounts.ByID(c.Request.Context(), auth.AccountID(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade has already answered the request.
		}
		defer conn.Close()

		// One goroutine owns every write: a WebSocket allows only one writer
		// at a time, and output, the roster, and pings all want to write.
		//
		// done stops that writer. The channel of frames is never closed,
		// because another User leaving can make this Session announce a new
		// roster through this callback at any moment, and a send on a closed
		// channel would panic.
		out := make(chan frame, outputQueue)
		done := make(chan struct{})
		defer close(done)
		go writeToBrowser(conn, out, done)

		sendTerminal(out, s.Scrollback())
		member := s.Join(
			session.Watcher{ID: me.ID, Name: me.Identity.Name, AvatarURL: me.Identity.AvatarURL},
			func(b []byte) { sendTerminal(out, append([]byte(nil), b...)) },
			func(roster []session.Watcher) { sendRoster(out, roster) },
		)
		defer member.Leave()

		readFromBrowser(conn, s, member)
	}
}

// sendTerminal queues terminal output, dropping it if the browser has fallen
// too far behind.
func sendTerminal(out chan frame, b []byte) {
	if len(b) == 0 {
		return
	}
	queue(out, frame{kind: websocket.BinaryMessage, data: b})
}

// sendRoster queues the list of connected Users as a JSON text frame.
func sendRoster(out chan frame, roster []session.Watcher) {
	body, err := json.Marshal(map[string]any{"type": "roster", "users": roster})
	if err != nil {
		log.Printf("encode roster: %v", err)
		return
	}
	queue(out, frame{kind: websocket.TextMessage, data: body})
}

func queue(out chan frame, f frame) {
	select {
	case out <- f:
	default: // The reader is too slow. Drop rather than stall the Session.
	}
}

// writeToBrowser is the socket's only writer.
func writeToBrowser(conn *websocket.Conn, out <-chan frame, done <-chan struct{}) {
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()

	for {
		select {
		case <-done:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			conn.WriteMessage(websocket.CloseMessage, nil)
			return
		case f := <-out:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(f.kind, f.data); err != nil {
				return
			}
		case <-ping.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readFromBrowser passes keystrokes to the shell and applies resize messages.
// It returns when the User disconnects.
func readFromBrowser(conn *websocket.Conn, s *session.Session, member *session.Membership) {
	conn.SetReadLimit(maxInputMsg)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		switch kind {
		case websocket.BinaryMessage:
			if err := s.Type(data); err != nil {
				return
			}
		case websocket.TextMessage:
			var msg control
			if err := json.Unmarshal(data, &msg); err != nil {
				continue // Ignore a message this server does not understand.
			}
			if msg.Type == "resize" {
				// This is one User's own viewport, not the shell's size. The
				// Session works out the size that fits everyone.
				if err := member.Resize(msg.Rows, msg.Cols); err != nil {
					log.Printf("resize session %s: %v", s.Code, err)
				}
			}
		}
	}
}

// sameHost reports whether an Origin header points at the host serving this
// request.
func sameHost(origin, host string) bool {
	for _, prefix := range []string{"http://", "https://"} {
		if len(origin) > len(prefix) && origin[:len(prefix)] == prefix {
			return origin[len(prefix):] == host
		}
	}
	return false
}
