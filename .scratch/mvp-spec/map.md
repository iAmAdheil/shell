## Destination

An MVP feature spec for Shell: a multiplayer, real-time terminal. Authenticated Users create or join a Session by Session Code, type into one shared input stream that drives a per-Session sandboxed shell, and see the full Scrollback plus a roster of connected Users. Built to a portfolio-demo polish bar on the existing Go/Gin + React/Vite monorepo.

## Notes

- Domain glossary: CONTEXT.md (User, Session, Session Code, Scrollback).
- Per ticket, default to calling the Skill tool for grilling + domain-modeling; use the Type: line to pick the right skill (research/prototype/grilling/task).
- Standing bar: portfolio-demo polish, not hardened for large-scale public abuse — but real, authenticated Users are expected, so basic safety (container isolation, command safety gate) is required, not optional.
- Docker is confirmed available in the local dev environment (checked 2026-08-26).

## Decisions so far

- [Terminal execution model](issues/01-terminal-execution-model.md): real shell via PTY, one Docker container per Session, full isolation.
- [Transport protocol](issues/02-transport-protocol.md): WebSocket, one connection per User.
- [Session/Scrollback persistence](issues/03-session-scrollback-persistence.md): in-memory only, lost on server restart.
- [Deployment topology](issues/04-deployment-topology.md): single server instance, no shared-state requirement.
- [Terminal rendering](issues/05-terminal-rendering.md): a terminal-emulator library (e.g. xterm.js), not plain text.
- [Session Code lifecycle](issues/06-session-code-lifecycle.md): the code dies with the Session; no reuse.
- [Terminal resize policy](issues/07-terminal-resize-policy.md): PTY sized to the smallest connected viewport, live.
- [Container network and command safety gate](issues/08-container-network-and-command-safety-gate.md): outbound network allowed; a per-command classifier blocks risky commands outright, visible to all Users in Scrollback.
- [Session creation flow](issues/09-session-creation-flow.md): explicit "New Session" action, no auto-create.
- [Connected-User roster](issues/10-connected-user-roster.md): in scope, identity from the authenticated account.
- [Auth requirement and model](issues/11-auth-requirement-and-model.md): full accounts required to create or join a Session (revises the original core-mechanic-only scope).
- [Account persistence](issues/12-account-persistence.md): accounts in a database, separate from the in-memory Session store.
- [Deployment / hosting target — research](issues/15-deployment-hosting-target.md): five options surveyed (VPS+Docker, Fly.io Machines, Railway, Render, k3s); Render and plain Railway Services ruled out as blockers, no option picked yet — see [17-choose-deployment-hosting-target](issues/17-choose-deployment-hosting-target.md).
- [Command safety classifier — research](issues/13-command-safety-classifier-approach.md): six approaches surveyed (Claude Code auto mode, Cursor Auto-review, Codex CLI sandbox+reviewer, Aider manual confirm, pattern/rbash/seccomp mechanisms, local-vs-external-model); no option picked yet — see [18-choose-command-safety-classifier-approach](issues/18-choose-command-safety-classifier-approach.md).
- [Auth method & session/token mechanism](issues/14-auth-method-and-session-token.md): OAuth only (GitHub + Google), server-side session via HTTP-only cookie (not JWT), long-lived and refreshed on activity.

## Not yet specified

- Per-Session container image contents (base image, pre-installed tools) and concrete resource limits (CPU/memory/pids/timeout).
- Session Code format (length, character set) and whether it is embedded in a shareable URL.
- Roster and blocked-command UI treatment (exact placement, wording, visual style).
- Orphaned-container cleanup if the server crashes or restarts mid-Session.

## Out of scope

(none yet)
