Type: grilling
Status: resolved

## Question

Does Session/Scrollback state survive a server restart?

## Answer

No. In-memory only. A Session and its Scrollback live in server process memory; a restart ends every Session. This is distinct from account data — see [12-account-persistence](12-account-persistence.md), which does persist.
