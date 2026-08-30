Type: grilling

## Question

[15-deployment-hosting-target](15-deployment-hosting-target.md) surfaced five hosting options for spawning a Docker container per Session (plain VPS + Docker, Fly.io Machines API, Railway Sandboxes, Render, k3s on a VPS) with cost/complexity tradeoffs, but picked none. Given that research, choose the hosting target. Carries forward these open questions from the research:

- Does Railway Sandboxes' lack of public port exposure rule it out entirely?
- How should orphaned-container cleanup on server crash/restart work for the chosen option?
- Does the command-safety-classifier work ([08](08-container-network-and-command-safety-gate.md)) need raw Docker API access, further disadvantaging options without one?
- Is a hybrid topology (always-on server on one platform, Session containers spawned elsewhere) worth considering, or does [04-deployment-topology](04-deployment-topology.md)'s single-instance decision rule that out?
