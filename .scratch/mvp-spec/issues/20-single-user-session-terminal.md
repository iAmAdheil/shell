# 20: Create a Session and run commands in an isolated single-user terminal

**What to build:** An authenticated User starts a new Session with one explicit action. The server spawns a Docker container running a real shell behind a PTY, one container per Session. Keystrokes travel from the browser to the container's PTY over a WebSocket; output renders in a terminal-emulator UI (e.g. xterm.js), not plain text. The Session and its Scrollback live only in server memory. The container is destroyed when the User disconnects.

**Blocked by:** 19

**Status:** ready-for-agent

- [ ] An authenticated User can trigger "New Session" and land in a running terminal within a few seconds.
- [ ] Visiting the app with no Session Code does not create a Session.
- [ ] The server spawns one Docker container per Session, running a shell through a PTY.
- [ ] Keystrokes typed in the browser reach the container's shell; its output renders in the browser through a terminal-emulator library.
- [ ] The PTY is sized to the User's own viewport at connect time.
- [ ] Output is kept as Scrollback in server memory for the life of the Session.
- [ ] When the User disconnects, the server destroys the container and drops the Session and its Scrollback.
- [ ] A server restart ends every Session; nothing about a Session survives it.
