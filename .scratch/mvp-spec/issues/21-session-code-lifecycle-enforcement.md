# 21: Session Code dies with its Session

**What to build:** Once a Session's last User disconnects, its Session Code no longer opens any Session. Visiting an ended Session's code shows a clear error instead of starting a new Session.

**Blocked by:** 20

**Status:** done

- [x] After a Session ends, opening its Session Code shows a "this Session has ended" message. One message covers this and the unknown-Code case: "this Session has ended, or that Session Code is wrong". The server forgets a Session the moment it ends, so it cannot tell the two apart, and a message that claimed to would be a guess.
- [x] Opening an ended Session's code never creates a new Session under that code.
- [x] Opening a Session Code that never existed shows a clear error, not a crash.
