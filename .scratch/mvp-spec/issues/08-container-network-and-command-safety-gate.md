Type: grilling
Status: resolved

## Question

Does each Session's container get outbound network access, and if so, how is abuse risk (spam, scanning, destructive commands) managed?

## Answer

Outbound network access is allowed (not sandboxed off). Risk is managed by a command safety classifier instead of network isolation:

- The classifier evaluates one full command at a time, once the User presses Enter — not per keystroke.
- A command it flags as risky is blocked outright (not a confirm-to-proceed prompt); the User is shown why.
- The block message appears in the shared Scrollback, visible to every User in the Session, not only the one who typed it.
- What powers the classifier (rule/pattern-based vs local model vs external LLM call) is still open — see [13-command-safety-classifier-approach](13-command-safety-classifier-approach.md).
