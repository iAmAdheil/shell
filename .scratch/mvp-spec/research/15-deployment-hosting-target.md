# Research: Deployment / hosting target (issue 15)

Question: where can Shell's Go server run such that its own code can spawn a
fresh Docker container per Session on demand and destroy it when the Session
ends, at single-instance, portfolio-demo scale? Options are surfaced with
rough cost/complexity tradeoffs. No option is picked here — that decision is
deferred to a later planning session.

The load-bearing question for each platform: **can application code create
and destroy arbitrary containers/VMs at runtime**, or does the platform only
run containers/services that were defined ahead of time (via dashboard, CLI,
or repo config)? Many PaaS products only do the latter, which is a hard
blocker for Shell's execution model (per
[01-terminal-execution-model](../issues/01-terminal-execution-model.md)).

---

## 1. Plain VPS + Docker (DigitalOcean, Hetzner, Linode/Akamai)

**How it fits:** Install Docker Engine on the VPS. The Go server talks to the
local Docker daemon over its Unix socket (`/var/run/docker.sock`, the
default bind per `dockerd` docs) using the official Docker Go SDK
(`github.com/docker/docker/client`), or the raw Docker Engine REST API
directly. This is the Docker Engine API's documented, first-class use case —
a RESTful API "for interacting with the Docker daemon," with the Go SDK
listed as an official client.
[Source: docs.docker.com/reference/api/engine/](https://docs.docker.com/reference/api/engine/),
[docs.docker.com/reference/cli/dockerd/](https://docs.docker.com/reference/cli/dockerd/)

**Can the app spawn containers at runtime?** Yes, unrestricted. The app has
root/daemon-level control: create, start, stop, remove, inspect, exec into,
and attach to any container, plus set resource limits (CPU/memory/pids) per
container. This is the most direct, least-abstracted fit for the "spawn a
container per Session" requirement — no intermediary platform API to work
around.

**Cost (small/portfolio scale, single droplet running the Go server + all
Session containers):**
- DigitalOcean: Droplets start at $4/month; a more realistic size for
  running several concurrent Session containers (e.g. 2 vCPU / 4 GiB) is
  ~$24/month ($0.03571/hr), billed monthly.
  [Source: digitalocean.com/pricing/droplets](https://www.digitalocean.com/pricing/droplets)
- Hetzner: Cost-optimized (shared vCPU) line starts at €5.99/month; the
  CPX21-class (3 vCPU / 4 GB, non-cost-optimized) region-dependent pricing
  runs roughly €12–32/month depending on region (US/Singapore priced higher
  than EU).
  [Source: hetzner.com/cloud/cost-optimized/](https://www.hetzner.com/cloud/cost-optimized/),
  [docs.hetzner.com price-adjustment page](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/)
- Linode (Akamai): Shared CPU instances start at $5/month (1 vCPU/1 GB);
  dedicated CPU starts at $36/month ($0.05/hr).
  [Source: akamai.com/products/cpu](https://www.akamai.com/products/cpu),
  [techdocs.akamai.com shared-cpu-compute-instances](https://techdocs.akamai.com/cloud-computing/docs/shared-cpu-compute-instances)

**Operational complexity:** Low-to-moderate. Install Docker (one-line apt
setup), no orchestration layer, one box to patch and monitor. The app owner
is fully responsible for: OS/Docker security updates, disk space growth as
container images accumulate, orphaned-container cleanup on server
crash/restart (flagged as "not yet specified" in the map), and any
horizontal scaling if the single-instance decision is later revisited (it
would require a rewrite, not a config change). No managed load balancer,
no managed TLS unless self-configured (e.g. Caddy/nginx + Let's Encrypt).

---

## 2. Fly.io Machines API

**How it fits:** Fly Machines are "fast-launching VMs with a simple REST
API" — the same primitive Fly's own platform (Fly Launch) is built on, and
Fly explicitly documents using it directly: "We use the Machines API to
build the orchestration for Fly Launch. You can use it however you'd like."
A Machine is created from a container image via a Create Machine REST call,
boots in well under a second once created, and is torn down via a Delete
Machine REST call. This is a purpose-built fit for "create one sandboxed
compute unit per Session, then destroy it."
[Source: fly.io/docs/machines/overview/](https://fly.io/docs/machines/overview/)

**Can the app spawn containers at runtime?** Yes — this is the API's
explicit purpose. The Go server would call the Machines REST API (or use
`flyctl`/an unofficial Go client) with a Fly API token to create a Machine
from a container image per Session, then delete it when the Session ends.
Machines run as Firecracker microVMs, giving VM-level isolation (stronger
than a bare Docker container) rather than the app talking to a shared local
Docker daemon.
[Source: fly.io/docs/reference/architecture/](https://fly.io/docs/reference/architecture/)

**Cost:** Machines are billed per second while in the `started` state (no
charge while stopped/destroyed) — a good fit for ephemeral, on-demand
Session containers. The smallest preset, `shared-cpu-1x` / 256MB, is
~$0.0028–0.0035/hr depending on region (~$2.02–2.54/month if run
continuously; far less for short-lived Sessions billed per second). The
always-on web server itself would also run as a Machine at this same rate,
plus outbound bandwidth.
[Source: fly.io/docs/about/pricing/](https://fly.io/docs/about/pricing/),
[fly.io/docs/about/billing/](https://fly.io/docs/about/billing/)

**Operational complexity:** Low for the container-spawning problem itself
(no daemon to install/secure — Fly runs the microVM host fleet), but adds a
new external dependency: the app now depends on Fly's control plane
(`flaps`) being reachable and responsive for every Session create/destroy.
Fly's own docs note placement can fail under regional capacity pressure and
say retry logic is the caller's responsibility. Regional placement,
volumes, and networking (WireGuard/private networking) are Fly-specific
concepts to learn.

---

## 3. Railway and Render

### Railway

**Services (the normal deployment primitive):** A Railway "service" is
described in its own docs as "a deployment target... containers deployed
from an image," sourced from a GitHub repo, a Docker image, or a local
directory — created via the dashboard, CLI, or API ahead of time, not
spawned ad hoc by code running inside another service.
[Source: docs.railway.com/services](https://docs.railway.com/services)

**Sandboxes (a newer, closer-fitting primitive):** Railway also documents
"Sandboxes" — "short-lived Linux environments you can provision on demand,
run commands in, and destroy," controllable programmatically via a
TypeScript SDK (`Sandbox.create()` / `.exec()` / `.destroy()`) or the CLI.
This is architecturally close to what Shell needs (an app-driven, ephemeral,
isolated compute unit per Session). Important caveats found in the docs:
- **Beta / gated access**: "Sandboxes are available through Priority
  Boarding. Breaking changes may occur" — not generally available.
- **Official SDK is TypeScript-only**; Shell's server is Go, so it would
  need to speak the underlying API directly (undocumented for Go) rather
  than use a supported client.
- **No public network exposure**: "A sandbox has no public endpoint...
  isn't reachable from the internet." Reaching a port requires SSH port
  forwarding from a local machine — there's no documented pattern for a
  production server to proxy a public user's WebSocket/PTY traffic into a
  sandbox the way it can into a container it controls directly via Docker.
- **Per-environment caps**: 50 concurrent sandboxes on Hobby, 100 on Pro —
  a hard ceiling on concurrent Sessions.
- **Idle timeout auto-destroy**: 30 min default (Hobby/Pro, up to 120 min),
  5 min fixed on Free/Trial.
[Source: docs.railway.com/sandboxes](https://docs.railway.com/sandboxes)

**Cost:** Hobby plan is $5/month base (includes $5 of usage credit) or Pro
at $20/month base. Sandbox VM resources are billed separately and more
expensively than regular services: $50/vCPU/month and $50/GB RAM/month
(vs. $10/GB and $20/vCPU for normal service usage), i.e. ~5x normal Railway
compute pricing, roughly on par with or pricier than a Fly Machine.
[Source: docs.railway.com/reference/pricing/plans](https://docs.railway.com/reference/pricing/plans)

**Fit assessment:** The plain "services" model is a blocker (can't spawn
containers from app code). Sandboxes are a genuine match for the *spawn/
destroy* requirement but are beta-gated, lack a Go SDK, and — critically —
have no documented way to expose a public-facing port for real-time PTY
traffic, which is central to Shell's design. Worth revisiting if it exits
beta with clearer networking support.

### Render

**Documented model:** Render "fully supports Docker-based deploys" — a
service either pulls a prebuilt image from a registry or builds one from a
Dockerfile in the repo, and that becomes the running service. All Docker
docs, quickstarts, and platform-capability lists (private networking,
persistent disks, zero-downtime deploys, etc.) describe this per-service,
pre-defined-at-deploy-time model. No page in Render's Docker documentation
mentions Docker socket access, privileged containers, or any way for a
running service to create sibling containers.
[Source: render.com/docs/docker](https://render.com/docs/docker)

**Can the app spawn containers at runtime?** No documented mechanism found.
Render's model is strictly "one image → one long-running (or scheduled)
service," configured via the dashboard/Blueprint, not something app code
creates and destroys dynamically. This is a blocker for Shell's execution
model as specified, unless Render has an undocumented/enterprise-only
capability not surfaced in public docs.

**General pattern context:** For contrast, some other platforms (e.g.
Clever Cloud) explicitly document an opt-in `CC_MOUNT_DOCKER_SOCKET=true`
environment variable specifically "to spawn sibling containers," with an
explicit security warning that it "breaks all isolation provided by
Docker." This confirms Docker-socket-based sibling-container spawning is a
known, occasionally-supported PaaS pattern — but it is the exception, not
the norm, and neither Railway's services nor Render's Docker docs offer an
equivalent for their standard service primitive.

---

## 4. Minimal Kubernetes (k3s) on a single VPS

**How it fits:** Run k3s as a single-node "cluster" (server + workload on
one box). The Go server would use the Kubernetes API (via `client-go`) to
create/delete a Pod per Session, instead of talking to the Docker daemon
directly. k3s uses containerd (or optionally cri-dockerd) under the hood,
so this is still "spawn a container per Session," just through the
Kubernetes API surface rather than the raw Docker API.

**Can the app spawn containers at runtime?** Yes — this is exactly what the
Kubernetes API is for (create/delete Pods, Jobs, etc. programmatically).
Fully supports the requirement, and additionally gives standard primitives
that could help later with the safety/network gating requirement
(NetworkPolicies, resource quotas/limits, PodSecurity admission) — though
none of that is required to just spawn/destroy a container.

**Cost:** No extra licensing cost — k3s itself is free/open-source; cost is
just the underlying VPS (same providers/pricing as Option 1). k3s's own
minimum hardware requirements for a server node are 2 CPU cores / 2 GB RAM
(before any workload), 1 core / 512 MB for an agent node — modest, but that
is baseline k3s + control-plane overhead on top of whatever the Session
containers themselves need.
[Source: docs.k3s.io/installation/requirements](https://docs.k3s.io/installation/requirements)

**Operational complexity:** Highest of the four options. Even a single-node
k3s "cluster" adds: a control plane to keep patched (etcd/SQLite datastore,
API server, kubelet, kube-proxy), Kubernetes YAML/manifest concepts (Pods,
resource limits expressed as k8s objects instead of Docker run flags), a
CNI (Flannel by default) with its own firewall port requirements (6443,
8472/UDP for VXLAN, 10250, etc. — each documented and each a thing to lock
down), and a steeper learning curve than either "SSH into a VPS and run
`docker run`" or "call a REST API." For a single-instance, portfolio-scale
deployment with no stated need for multi-node scheduling or high
availability, this is more machinery than the stated requirements call
for — flagged here as the "heavier alternative" the ticket asked to
surface, not as a natural fit.

---

## Summary table

| Option | App can spawn containers at runtime? | Rough monthly cost (small scale) | Operational complexity |
|---|---|---|---|
| VPS + Docker (DO/Hetzner/Linode) | Yes — direct Docker Engine API/socket access | ~$5–25 | Low–moderate (one box, Docker install, self-managed cleanup) |
| Fly.io Machines API | Yes — purpose-built REST API for this | ~$2+/machine, per-second billing (usage-dependent) | Low–moderate (no daemon to manage, but new platform-specific concepts) |
| Railway | Services: No. Sandboxes: Yes, but beta-gated, no Go SDK, no public port exposure | Sandboxes: $50/vCPU-GB-month; base plan $5–20 | Services: N/A (blocked). Sandboxes: moderate, plus beta risk |
| Render | No documented mechanism found | N/A (blocked) | N/A (blocked) |
| k3s on single VPS | Yes — via Kubernetes API | Same VPS cost as Option 1 | High (control plane, manifests, CNI, firewall rules) |

---

## Open questions for the decision ticket

- Does Railway Sandboxes' lack of public port exposure rule it out
  entirely, or is there an undocumented pattern (e.g. the Go server
  reaching the sandbox over Railway's private network instead of the
  public internet) worth asking Railway support about before ruling it out?
- If a VPS + Docker or k3s option is chosen, how should orphaned-container
  cleanup on server crash/restart be handled (already flagged as
  unspecified in the map) — does the answer differ by option (e.g. k3s's
  Pod garbage collection vs. hand-rolled Docker cleanup on boot)?
- Does the command-safety-classifier work (issue 08) assume raw Docker API
  access (e.g. inspecting container logs/exec streams directly), and if so,
  does that further disadvantage Railway Sandboxes or any option that
  doesn't expose an equivalent low-level API?
- Is there an appetite for a Fly.io-specific rewrite of the deployment
  scripts/health checks, versus the more provider-agnostic "just SSH and
  run Docker" model of a plain VPS?
- Should Render/Railway's standard "services" model be reconsidered later
  only for the always-on web server / signaling tier, with Session
  containers spawned on a separate VPS or Fly.io regardless of where the
  main app lives (a hybrid topology) — or does the "single server instance"
  decision (04-deployment-topology) rule that out?
