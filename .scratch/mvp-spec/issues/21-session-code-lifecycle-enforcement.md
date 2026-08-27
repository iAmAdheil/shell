# 21: Session Code dies with its Session

**What to build:** Once a Session's last User disconnects, its Session Code no longer opens any Session. Visiting an ended Session's code shows a clear error instead of starting a new Session.

**Blocked by:** 20

**Status:** ready-for-agent

- [ ] After a Session ends, opening its Session Code shows a "this Session has ended" message.
- [ ] Opening an ended Session's code never creates a new Session under that code.
- [ ] Opening a Session Code that never existed shows a clear error, not a crash.
