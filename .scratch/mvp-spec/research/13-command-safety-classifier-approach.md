Type: research

## Question

[13-command-safety-classifier-approach](../issues/13-command-safety-classifier-approach.md) asks what should power the Session's command safety classifier: a rule/pattern engine, a local model, or an external LLM call. The project owner's reference point is "something similar to auto mode in Claude Code." This document surveys how Claude Code and comparable systems decide a command is risky, and what non-AI shell-safety mechanisms exist, so a later planning session can weigh the options. It does not recommend one option.

Each section below states how the approach classifies a command, then its latency profile for a real-time, once-per-Enter-press classifier, its false-positive/false-negative tradeoff, and who must maintain it.

---

## 1. Claude Code auto mode (LLM classifier, external API call)

Auto mode is one of several permission modes in Claude Code. In auto mode, "a second model, the classifier, reviews actions instead of you." The classifier runs on Claude Sonnet 5 by default, not on the session's own model selection, unless the account or session configuration excludes Sonnet 5, in which case it falls back to the session model or an Opus model.

Sources: [Permission modes](https://code.claude.com/docs/en/permission-modes) and [Configure permissions](https://code.claude.com/docs/en/permissions), both on code.claude.com (Anthropic's official Claude Code docs).

**How it classifies**: An external LLM call. Each pending action (a shell command, a file write outside the working directory, a network request, and so on) is sent to the classifier model along with a portion of the conversation transcript. The classifier returns allow or block. Anthropic documents a fixed, versioned list of default-blocked categories (for example `curl | bash`, `git reset --hard`, force push, granting IAM permissions, writing to a secret manager) and default-allowed categories (local file operations, dependency installs from a lock file, read-only HTTP requests). Users and organizations can add `permissions.ask` / `permissions.deny` rules that are checked before the classifier ever runs, and can list "trusted infrastructure" the classifier should treat as safe.

**Latency**: The docs state plainly: "Each check sends a portion of the transcript plus the pending action, adding a round-trip before execution." Reads and in-workspace edits skip the classifier entirely; the overhead applies mainly to shell commands and network operations. The classifier also caches its verdict per network host/port so repeat connections to the same host are not re-checked. No exact millisecond figure is published.

**False positives / false negatives**:
- Anthropic states directly: "Auto mode reduces permission prompts but does not guarantee safety... The classifier can make mistakes." (This phrasing about mistakes is closer to Cursor's docs; Claude Code's own docs say auto mode is "not a replacement for review on sensitive operations.")
- Repeated false positives are handled procedurally: if the classifier blocks an action 3 times in a row, or 20 times total, auto mode pauses and falls back to prompting the human.
- The classifier only sees user messages, tool calls, and CLAUDE.md content; tool *results* are stripped from what it sees, specifically so that hostile content returned by a prior command cannot manipulate the classifier's next verdict (a prompt-injection defense).

**Maintenance burden**: Anthropic maintains and versions the default rule categories (the docs show version numbers like "Claude Code v2.1.198 and later also block these by default," meaning the vendor updates the ruleset as new attack patterns are found). The team using the tool does not maintain the classifier's judgment itself, but can layer `permissions.ask`/`deny` rules and an `environment` config of "trusted infrastructure" on top, and that configuration is the team's own maintenance burden.

---

## 2. Comparable AI coding agents

### 2a. Cursor — Auto-review mode (small external LLM classifier)

Cursor's "Run Modes" include **Auto-review**, which the docs call "the safest useful setup for most people." It layers three mechanisms: an allowlist, an OS-level sandbox for shell commands, and an LLM classifier for anything the sandbox cannot contain.

Source: [Run Modes](https://cursor.com/docs/agent/security/run-modes) and [permissions.json reference](https://cursor.com/docs/reference/permissions), both on cursor.com (Cursor's official docs).

**How it classifies**: Allowlisted calls run immediately with no check. Shell commands that fit inside the sandbox's file/network limits run sandboxed with no LLM check. Everything else (commands needing full system access, MCP calls, fetch calls) goes to an LLM classifier. Cursor explicitly names the models: "Auto-review's classifier runs on a small Cursor-managed model. Today that is Claude 4.5 Haiku or GPT-5.4 Mini" — a deliberate choice of small, fast models rather than a frontier model. Teams can steer (not hard-enforce) the classifier with plain-English `allow_instructions` / `block_instructions` in a `permissions.json` file; the docs are explicit that these are "steering, not enforcement" — a matching block instruction can still be overridden if "Cursor insists."

**Latency**: Not quantified in the docs, but the choice of a "small" model (Haiku-class / Mini-class rather than a frontier model) is presented as a deliberate speed/cost tradeoff versus using a larger model.

**False positives / false negatives**: The docs state outright: "Auto-review is not a security boundary. The classifier can make mistakes. It can allow a call you would have blocked, or block a call you would have allowed." When the classifier blocks something the agent still believes is correct, Cursor shows a manual approval prompt rather than hard-failing — i.e., a false positive degrades to a human-in-the-loop prompt, not a silent block.

**Maintenance burden**: The underlying classifier model and its base policy are Cursor's to maintain (a hosted, "Cursor-managed model"). Teams additionally maintain their own `permissions.json` allow/block instructions and can be overridden entirely by team-admin dashboard policy, which takes top priority over both the local file and IDE settings.

### 2b. OpenAI Codex CLI — sandbox + approval policy, with an optional LLM "auto-review" layer

Codex CLI's security model is two independent layers: a **sandbox mode** (OS-level, what the process can technically reach) and an **approval policy** (when to stop and ask a human). A third, optional layer, **Auto-review**, replaces the human at the approval step with a separate reviewer agent.

Source: [Agent approvals & security](https://learn.chatgpt.com/docs/agent-approvals-security) and [Auto-review](https://learn.chatgpt.com/docs/sandboxing/auto-review), both on learn.chatgpt.com (OpenAI's official Codex docs).

**How it classifies**:
- The sandbox itself is not a classifier — it is OS-enforced containment: macOS uses Seatbelt (`sandbox-exec`); Linux uses `bwrap` plus `seccomp`; native Windows uses a Windows sandbox implementation. A command that fits inside the sandbox's write/network limits simply runs.
- The approval policy (`untrusted` / `on-request` / `never`, or a granular per-category policy) decides when a sandbox-crossing action must pause. By default it pauses for a human ("approvals_reviewer = user").
- Auto-review is opt-in (`approvals_reviewer = "auto_review"`). It routes only the actions that would already need human approval — sandbox escalations, blocked network requests, destructive-looking actions — to "a separate reviewer agent" (itself a Codex agent) which reads a compact transcript plus the pending request and returns approve/deny with a rationale. The default reviewer policy is open-source and versioned: [policy.md](https://github.com/openai/codex/blob/main/codex-rs/core/src/guardian/policy.md) in the `openai/codex` repository.

**Latency**: Not quantified in the docs. Structurally, Auto-review only fires for actions that already leave the sandbox boundary, so routine in-workspace commands never pay an LLM round-trip; only boundary-crossing actions do. The docs note the reviewer "can also perform read-only checks to gather missing context, but it does so rarely" — implying an occasional multi-step review, not a single fixed-cost call.

**False positives / false negatives**: The docs describe fail-closed behavior for infrastructure failure: "Prompt-build, review-session, and parse failures fail closed" (i.e., an error in the reviewer itself becomes a block, not a silent allow). A denial is treated as stronger than an ordinary sandbox error — the main agent is told to find a materially safer path or stop and ask the user, rather than silently retrying.

**Maintenance burden**: The default reviewer policy is OpenAI's to maintain, versioned in the public `openai/codex` repo. Enterprises can override its "tenant-specific section" with their own `guardian_policy_config`; individual users can add a local `[auto_review].policy` text. Codex without Auto-review (`approval_policy` + `sandbox_mode` alone) has effectively zero classifier maintenance burden, because there is no model in the loop — the OS sandbox rules and the fixed policy strings (`untrusted`/`on-request`/`never`) are Codex's own code, not a tunable ruleset the user must feed new attack patterns into over time.

### 2c. Aider — manual per-command confirmation (no classifier)

Aider is the simplest point of comparison: it has no risk classifier at all. When Aider's LLM output includes a shell command, Aider prints the command and asks the human "run this command?" before executing it.

Source: [Options reference](https://aider.chat/docs/config/options.html) (aider.chat, Aider's official docs) and [GitHub issue #3903](https://github.com/Aider-AI/aider/issues/3903) on `Aider-AI/aider` (official repository).

**How it classifies**: It does not classify. Every suggested shell command gets the same fixed confirm-to-run prompt, controlled by `--suggest-shell-commands` (default: on). The `--yes-always` / `--yes` flag ("Always say yes to every confirmation") answers *every* confirmation the same way, with no per-command risk judgment; the linked GitHub issue documents that in practice `--yes-always` caused shell commands to be skipped rather than silently auto-run, which the reporter called "not fine for unsupervised use." There is no allow/deny list and no model-based review step in Aider's own confirmation path.

**Latency**: Effectively zero added latency from Aider's own logic — the delay is purely however long the human takes to answer the Y/N prompt. There is no classifier round-trip.

**False positives / false negatives**: N/A in the classifier sense — Aider asks about every command uniformly, so it neither over-blocks nor under-blocks by content; the human bears 100% of the judgment. In "always yes" configurations, from the maintainer's own docs and the linked issue, this is explicitly *not* considered safe for unattended use.

**Maintenance burden**: Effectively none — there is no ruleset or model to keep current, because there is no differentiated safety judgment to maintain.

---

## 3. Non-AI shell-safety approaches

### 3a. Pattern / allowlist command filters (e.g., sudoers `Cmnd_Alias`)

The most direct real-world example of a pure pattern-matching command gate is `sudo`'s own configuration file, `/etc/sudoers`. An administrator writes `Cmnd_Alias` groups and per-user/per-host rules naming exactly which commands (optionally with fixed arguments, or an argument regex) a user may run with elevated privilege.

Source: [Sudoers Manual](https://www.sudo.ws/docs/man/sudoers.man/) (sudo.ws, the project's official man page).

**How it classifies**: Static string/regex matching against a fixed, hand-authored list, evaluated locally with no network call and no model inference. The grammar supports exact command paths, argument patterns, and negation (`!`) to explicitly exclude a subcommand from an otherwise-allowed alias.

- **Latency**: Sub-millisecond. A local string/regex match against a small config file, no I/O beyond reading the file once.
- **False positives / false negatives**: Deterministic and auditable — a given command either matches a rule or it does not, with no ambiguity, but that same rigidity is the source of most real-world sudoers escapes: many programs allowed for one narrow purpose can be abused for a shell escape (Claude Code's own docs make the identical point about Bash prefix-matching rules: `Bash(curl http://github.com/ *)` looks safe but is trivially bypassed by option reordering, protocol swap, redirects, or variable indirection). A pattern list only stops the exact shapes its author anticipated.
- **Maintenance burden**: Entirely human. Someone must know about a new dangerous command shape (a new CLI tool, a new flag, a new bypass technique) and hand-edit the rule file before the filter can catch it. No self-updating mechanism exists.

### 3b. Restricted shells (rbash)

`rbash` is bash's own restricted-shell mode (`bash -r`, or a symlink/copy named `rbash`). Once active for a shell session, it disallows: changing directories with `cd`; setting or unsetting `SHELL`, `PATH`, `ENV`, or `BASH_ENV`; running any command containing a `/` in its name (which blocks running arbitrary binaries by path); redirecting output (`>`, `>|`, `<>`, `>&`, `&>`, `>>`); using `exec` to replace the shell; adding or deleting builtins with `enable`; and a few other command-substitution and `hash -p` restrictions.

Source: GNU Bash Reference Manual §6.10, "The Restricted Shell" — [www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Restricted-Shell](https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Restricted-Shell) (the bash project's own manual; this is a standard, long-documented POSIX/bash mechanism, not a third-party product).

**How it classifies**: Structural restriction, not per-command judgment. It does not inspect command *intent*; it removes whole categories of shell syntax and builtins outright, so an entire class of escape techniques (path-based execution, redirection, environment tampering) is unavailable regardless of what the user types.

- **Latency**: None at runtime beyond the shell's own parsing — restrictions are enforced by the shell binary itself as it parses each line, no external check.
- **False positives / false negatives**: The manual is candid about this mechanism's own limits, in its own words: "The restricted shell mode is only one component of a useful restricted environment," and it must be paired with a locked-down `PATH` of "only a few verified commands," a non-writable working directory, and disallowing script execution (rbash turns off its own restrictions inside any script it runs, so a permitted script can do anything). Used alone it is easy to escape (e.g., through an allowed interpreter with its own shell-escape feature). The same manual section says outright: "Modern systems provide more secure ways to implement a restricted environment, such as jails, zones, or containers."
- **Maintenance burden**: Low ongoing burden once configured — it is a fixed set of shell-level restrictions, not a ruleset that needs new "dangerous pattern" entries over time. The burden is mostly upfront (correctly scoping `PATH` and the allowed command set) rather than continuous.

### 3c. seccomp / seccomp-bpf (Linux syscall filtering)

seccomp-bpf lets a process attach a Berkeley Packet Filter (BPF) program that inspects every system call (number plus arguments) the process attempts, and returns a fixed action per call: kill the process, kill the thread, return an error to the caller, trap to a signal handler, notify a supervising process, or allow the call. It operates below the shell entirely, at the kernel/syscall boundary, so it constrains what a process can *do* rather than what a user can *type*.

Source: Linux kernel documentation, "Seccomp BPF (SECure COMPuting with filters)" — [docs.kernel.org/userspace-api/seccomp_filter.html](https://docs.kernel.org/userspace-api/seccomp_filter.html) (the kernel project's own documentation).

Both Cursor's sandbox (Linux backend) and OpenAI Codex CLI's sandbox (`bwrap` plus `seccomp` on Linux) build on this same primitive, per the sources cited in sections 2a/2b above.

**How it classifies**: A compiled filter program, evaluated by the kernel on every syscall — a rule/pattern mechanism, but expressed at the syscall level (e.g., "deny `execve`," "deny `connect` to non-loopback") rather than as text pattern-matching on a shell command string.

- **Latency**: Per-syscall overhead is small (a BPF program evaluation per call) and happens inside the kernel, not as an added external round-trip. It applies continuously to every syscall the process and its children make for the life of the process, not once per Enter-press — a different granularity than the Session's proposed "one check per full command" design.
- **False positives / false negatives**: Precise but brittle at the syscall boundary — it can reliably deny a whole class of operations (e.g., no outbound `connect()` at all), but authoring a filter that permits everything a legitimate shell workload needs while blocking real abuse requires deep knowledge of which syscalls each allowed tool actually uses; overly narrow filters break legitimate programs (false positives against normal use), and overly broad filters leave dangerous syscalls reachable (false negatives). It says nothing about the *semantic intent* of a command — it cannot distinguish "delete my own temp file" from "delete someone else's file" if both use the same `unlink` syscall pattern.
- **Maintenance burden**: Requires a systems engineer to write and update the BPF filter/profile as the set of tools running inside the container changes over time (new tool = new set of syscalls it needs = filter must be revisited), separate from and lower-level than any command-text ruleset.

---

## 4. Local ML/LLM classifier vs. external LLM API call, as a distinct axis

The ticket asks about this axis directly, so it is worth naming separately from the specific products above, since every AI-agent example in Section 2 that uses a model (Claude Code, Cursor, Codex Auto-review) calls out to a **hosted, vendor-managed model over the network** — none of them ship a classifier that runs locally inside the user's own infrastructure. That leaves "run a small classifier model locally, in the Session's own infrastructure" as a genuinely distinct, unproven-by-precedent option rather than something with a citable primary source. Its properties, by construction rather than by vendor documentation:

- **How it would classify**: Either a small local LLM (a distilled/quantized open-weights model) or a non-LLM ML classifier (e.g., a gradient-boosted or embedding-based classifier trained on labeled safe/risky commands), hosted on infrastructure the Shell project itself controls, evaluated once per submitted command.
- **Latency**: Potentially lower and more predictable than an external API call, because there is no network hop to a third-party provider — but only if the local model is small enough to run inference within the real-time budget; a local model large enough to match Sonnet-5-class judgment quality may not be faster than a fast external API call to a small hosted model (as Cursor's choice of Haiku/Mini-class models over a frontier model suggests the vendors themselves are already trading accuracy for latency).
- **False positives / false negatives**: Directly tied to model size and training data quality. A small local model is more likely to both over-block (unfamiliar-but-legitimate commands) and under-block (novel attack phrasing it was never trained/tuned on) than a large frontier model such as the ones Claude Code and Codex Auto-review use by default.
- **Maintenance burden**: Falls entirely on the Shell project's own team — no vendor pushes updated default-blocked categories the way Anthropic and OpenAI version theirs. The team would need to source or build labeled training/eval data, retrain or re-tune the model as new attack patterns surface, and host the inference infrastructure, none of which is required by an external-API-call design (where the vendor absorbs that cost) or a rule-based design (where the cost is editing a text file instead of running a training pipeline).

---

## Open questions for the decision ticket

- Can a pattern/allowlist layer (cheap, deterministic) and an LLM classifier (slower, more semantic) be combined in layers, the way Cursor and Codex both do — sandbox/allowlist first, classifier only for what falls through?
- What latency budget is acceptable before an external LLM call disrupts the "real-time" feel of a shared multi-user terminal, given every AI-agent example here adds an uncosted round-trip for at least some commands?
- Does the classifier need per-Session configuration (e.g., an allowlist a Session's owner can edit) or a single global ruleset for all Sessions, given Cursor's and Claude Code's designs both support project-level and user/org-level rule layering?
- If an LLM classifier is chosen, does the project want to depend on a third-party model API's availability and pricing (external call) or take on hosting and retraining a local model, given neither path has an existing off-the-shelf product to adopt wholesale?
- Since a blocked command is shown to every connected User in the shared Scrollback (per issue 08), should the classifier's rationale text be free-form (like Codex Auto-review's stated rationale) or a fixed message (like Claude Code's `Blocked by classifier` string), given multiple Users will read it?
- How should the classifier behave if it fails or times out (a third-party outage, a local model crash) — fail-open (allow the command) or fail-closed (block it), given Codex's Auto-review explicitly fails closed on reviewer errors?
