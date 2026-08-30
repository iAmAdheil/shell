Type: research
Status: resolved

## Question

What should power the command safety classifier described in [08-container-network-and-command-safety-gate](08-container-network-and-command-safety-gate.md)? The user's own reference point: "something similar to auto mode in Claude Code." Research how comparable systems (Claude Code's auto/permission mode, other AI-agent command-approval layers, existing shell-safety linters/allowlist tools) decide a command is risky, and what tradeoffs (latency, false positives, maintainability) each approach carries for a real-time, per-line classifier. Surface options; do not pick one.

## Answer

Full findings: [research/13-command-safety-classifier-approach.md](../research/13-command-safety-classifier-approach.md).

Six approaches researched against primary docs:
- **Claude Code auto mode**: external LLM call (a second model) reviews each pending action against versioned default-allow/block lists; adds a network round-trip; falls back to human prompting after repeated blocks.
- **Cursor Auto-review**: layers an allowlist, an OS sandbox, and a small hosted LLM classifier (Haiku/Mini-class, not frontier) only for what the sandbox can't contain.
- **OpenAI Codex CLI**: OS-level sandbox plus an approval policy; an optional separate reviewer agent only for actions that already cross the sandbox boundary; fails closed on reviewer error.
- **Aider**: no classifier at all — every shell command gets the same fixed human confirm-to-run prompt.
- **Non-AI mechanisms**: sudoers-style pattern/allowlist matching (sub-millisecond, deterministic, but brittle to bypass), `rbash` restricted shell (structural, not per-command), seccomp-bpf syscall filtering (kernel-level, continuous rather than once-per-command).
- **Local model vs. external API call**: named as its own axis — every AI-agent example found calls a hosted, vendor-managed model; a self-hosted local classifier has no citable precedent and shifts the full maintenance/training burden onto the project.

No option picked. Open questions carried to [18-choose-command-safety-classifier-approach](18-choose-command-safety-classifier-approach.md): whether to layer a cheap pattern/allowlist filter with an LLM classifier, the acceptable latency budget for a real-time shared terminal, per-Session vs. global rule config, local-model vs. external-API tradeoffs, how blocked-command rationale text should read given it's shown to every User (per issue 08), and fail-open vs. fail-closed behavior on classifier error/timeout.
