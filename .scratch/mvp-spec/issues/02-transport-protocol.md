Type: grilling
Status: resolved

## Question

How does input/output travel between the browser and the server?

## Answer

WebSocket. One connection per User. See [08-container-network-and-command-safety-gate](08-container-network-and-command-safety-gate.md) and [10-connected-user-roster](10-connected-user-roster.md) for what rides over it besides raw terminal bytes.
