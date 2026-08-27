Type: grilling

## Question

[13-command-safety-classifier-approach](13-command-safety-classifier-approach.md) surveyed six approaches (external LLM call like Claude Code auto mode, a small-hosted-model layered approach like Cursor, sandbox+optional-reviewer like Codex CLI, manual confirmation like Aider, pattern/allowlist filters, rbash, seccomp-bpf, and local-vs-external-model as its own axis) but picked none. Given that research, choose the classifier's design. Carries forward these open questions from the research:

- Should a cheap pattern/allowlist layer run first, with an LLM classifier only for what falls through (as Cursor and Codex both do)?
- What latency budget is acceptable before a classifier round-trip disrupts the "real-time" feel of a shared multi-user terminal?
- Does the classifier need per-Session configuration (an editable allowlist) or one global ruleset for all Sessions?
- Local model (self-hosted, full maintenance burden) vs. external LLM API call (depends on a third party's availability/pricing) — which tradeoff fits this project?
- Blocked-command rationale is shown to every connected User in the shared Scrollback ([08](08-container-network-and-command-safety-gate.md)) — free-form explanation or a fixed message string?
- Fail-open or fail-closed if the classifier itself errors or times out?
