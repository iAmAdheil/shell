Type: research
Status: resolved

## Question

[01-terminal-execution-model](01-terminal-execution-model.md) needs a host that can spawn a Docker container per Session on demand. Research hosting options that support this (e.g. a VPS running Docker directly, Fly.io Machines, Railway, a Kubernetes-based host) and their fit for a single-instance ([04-deployment-topology](04-deployment-topology.md)), portfolio-scale deployment. Surface options with rough cost/complexity; do not pick one.

## Answer

Full findings: [research/15-deployment-hosting-target.md](../research/15-deployment-hosting-target.md).

Five options researched against primary docs:
- **VPS + Docker** (DO/Hetzner/Linode): direct, unrestricted Docker Engine API access. ~$5–25/mo. Low–moderate complexity, but cleanup/patching is self-managed.
- **Fly.io Machines API**: purpose-built REST API for exactly this — create/destroy Firecracker microVMs per Session, per-second billing. Strong fit.
- **Railway**: standard Services can't spawn containers at runtime (blocker). Newer "Sandboxes" primitive is a closer fit but beta-gated, TypeScript-only SDK (server is Go), and has no public port exposure — no documented way to route public WebSocket/PTY traffic in.
- **Render**: no documented mechanism for app code to spawn sibling containers. Blocker as specified.
- **k3s on a single VPS**: works via the Kubernetes API, but is the heaviest option (control plane, manifests, CNI/firewall rules) for a single-instance, portfolio-scale deployment.

No option picked. Open questions carried to [17-choose-deployment-hosting-target](17-choose-deployment-hosting-target.md): whether Railway Sandboxes' lack of public exposure is a hard blocker, how orphaned-container cleanup differs by option, whether the command-safety-classifier work assumes raw Docker API access, and whether a hybrid topology (always-on server on one platform, Session containers on another) is worth considering.
