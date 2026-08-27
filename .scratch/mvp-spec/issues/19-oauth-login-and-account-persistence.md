# 19: OAuth login & account persistence

**What to build:** A visitor logs in with GitHub or Google. The server creates an HTTP-only session cookie and creates or finds one account record in a database. A logged-in User can log out.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A visitor can start a GitHub OAuth login and land back on the app, authenticated.
- [ ] A visitor can start a Google OAuth login and land back on the app, authenticated.
- [ ] The first login for a given OAuth identity creates one account record; a repeat login reuses that same record.
- [ ] The server issues an HTTP-only session cookie on login; protected routes reject requests without a valid cookie.
- [ ] A logged-in User can log out; the session cookie is cleared and the old cookie is rejected afterward.
- [ ] The session cookie is long-lived (e.g. 30 days) and refreshes on activity.
