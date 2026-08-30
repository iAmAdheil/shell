Type: grilling
Status: resolved

## Question

What runs inside a Session? Pick the execution model for the shell process a Session drives.

## Answer

A real OS shell (bash/zsh) runs through a PTY, one process per Session, inside its own Docker container — full per-Session isolation, not a shared or bare process. This also resolves the isolation-level branch: no shared host process, one container per Session.
