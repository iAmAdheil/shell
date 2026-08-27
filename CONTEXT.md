# Shell

An online terminal. A User runs commands in a shared, real-time shell. Multiple Users can join the same running shell at once.

## Language

**User**:
A person who uses the app.
_Avoid_: Player, participant, client, customer.

**Session**:
A running shell process that one or more Users interact with together. Input from every joined User merges into one input stream sent to the shell. A Session ends the moment its last User disconnects.
_Avoid_: Room, terminal, shell (as a noun for the running process).

**Session Code**:
The unique identifier for a Session. A shareable link carries this code so a second User can join.
_Avoid_: Invite code, room code, join link.

**Scrollback**:
The full output history of a Session from its start. A User who joins a Session sees the complete Scrollback, not just output from the point they joined.

**Account**:
A User's persistent record, created on their first OAuth login and keyed by the identity their provider reports. Accounts live in Postgres and survive a restart. Sessions do not.
_Avoid_: Profile, user record.

**Login**:
The server-side record that one browser is currently authenticated. An HTTP-only cookie carries only an opaque token; the server maps that token to an Account. Deliberately not called a "Session", which this glossary reserves for the running shell.
_Avoid_: Session (for auth), auth session, token.
