# 22: Join an existing Session by code, with full Scrollback replay

**What to build:** A second authenticated User enters a running Session's Session Code and joins it. On joining, the User immediately sees the Session's complete Scrollback, not just output from the join point onward. From then on, every joined User's input merges into one input stream sent to the shared shell.

**Blocked by:** 20

**Status:** ready-for-agent

- [ ] An authenticated User can enter a valid, active Session Code and join that Session.
- [ ] On join, the User's terminal renders the full Scrollback recorded so far, in order.
- [ ] Keystrokes from every joined User merge into a single input stream sent to the one shared PTY.
- [ ] Output from the shared shell renders identically, in real time, in every joined User's terminal.
- [ ] The Session stays open while at least one User is connected; it ends only when the last User disconnects.
