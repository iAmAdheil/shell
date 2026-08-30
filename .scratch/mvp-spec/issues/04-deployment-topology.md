Type: grilling
Status: resolved

## Question

Does the MVP need to run on more than one server instance?

## Answer

No. Single instance. All Sessions live in one server process; no shared-state/multi-instance requirement (e.g. no Redis-backed Session store needed). See [15-deployment-hosting-target](15-deployment-hosting-target.md) for where that single instance actually runs.
