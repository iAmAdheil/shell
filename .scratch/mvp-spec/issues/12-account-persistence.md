Type: grilling
Status: resolved

## Question

[03-session-scrollback-persistence](03-session-scrollback-persistence.md) makes Session state in-memory only, lost on restart. Accounts can't work that way. Where do accounts live?

## Answer

A database (e.g. Postgres), separate from the in-memory Session/Scrollback store. Accounts persist across restarts; Sessions still don't.
