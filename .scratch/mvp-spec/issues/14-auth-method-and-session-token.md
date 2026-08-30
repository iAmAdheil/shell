Type: grilling
Status: resolved

## Question

[11-auth-requirement-and-model](11-auth-requirement-and-model.md) settled on full accounts. Which auth method(s) — email+password, OAuth (which providers), or both — and what session/token mechanism (JWT vs server-side session cookie) backs it?

## Answer

OAuth only — GitHub and Google — no email+password. No password storage, reset flow, or verification step to build; fits the portfolio-demo bar and the developer-leaning audience.

Session mechanism: a server-side session, stored in the same in-memory store already used for Sessions, referenced by an HTTP-only cookie. Not a JWT — the server is single-instance (ticket 04), so there's no multi-instance state problem a stateless token would solve, and server-side sessions make logout/revocation instant.

Cookie lifetime: long-lived (e.g. 30 days), refreshed on activity. A User stays logged in across visits; no frequent re-auth, matching the low-friction portfolio-demo bar.
