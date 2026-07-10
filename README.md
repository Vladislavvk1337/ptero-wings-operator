# ptero-wings-operator

Kubernetes-native replacement for the classic Pterodactyl Wings daemon.

## Components

| Component | Responsibility |
| --- | --- |
| `ptero-wings-gateway/` | External Wings-like HTTP/WebSocket entrypoint with token/JWT auth |
| `ptero-wings-controller/` | Business logic, Kubernetes client access, startup mapping, metrics, optional panel REST integration |
| `ptero-wings-operator/` | CRD reconciliation for `GameServer` and `GameServerClass` |

The gateway accepts external panel traffic, forwards it to the controller service, and the controller is the only layer that creates or mutates `GameServer` resources. The operator remains purely declarative and reconciles cluster resources from CRDs.

## Documentation

- [ARCHITECTURE.md](./ARCHITECTURE.md) — English architecture reference with Mermaid diagrams
- [Doc_DE.md](./Doc_DE.md) — German operator/admin guide

## Request Flow

1. External client sends HTTP or WebSocket traffic to `ptero-wings-gateway`.
2. The gateway authenticates the request and delegates it to `ptero-wings-controller`.
3. The controller creates, updates, deletes, or reads `GameServer` resources through the Kubernetes API.
4. The operator reconciles `GameServer` objects into `StatefulSet`, `Service`, `PVC`, and `Secret` resources.
5. Status, logs, console output, and metrics flow back through controller → gateway → panel.

## Key Features

- `GameServer` / `GameServerClass` CRDs with finalizers, conditions, and status
- Startup command mapping from Panel-style startup strings to Kubernetes `command` + `args`
- Bundled NodePort allocation via `spec.network.nodePortAllocation`
- Optional panel resource push using `PANEL_URL` + `PANEL_TOKEN`
- Wings-like HTTP and WebSocket endpoints for create/power/resources/logs/console

## Install

```bash
make install
make run
```

Run the gateway locally:

```bash
export GATEWAY_TOKEN="my-secret-token"
export GAMESERVER_NS="game-servers"
export PANEL_URL="https://panel.example.com"
export PANEL_TOKEN="panel-token"
go run ./gateway/cmd/main.go --addr :8090
```

## Gateway Deployment

```bash
kubectl apply -f config/gateway/rbac.yaml
kubectl apply -f config/gateway/deployment.yaml
kubectl apply -f config/gateway/service.yaml
kubectl apply -f config/gateway/ingress.yaml
```

The default gateway deployment embeds the controller package in-process. Example manifests for a future standalone controller split are available in `examples/controller/`.

## GameServer Highlights

Example fields supported by the CRD:

- `spec.runtime.startup.raw` and `spec.runtime.startup.variables`
- `spec.game.command` and `spec.game.args` as the resolved container entrypoint
- `spec.network.nodePortAllocation.startPort`
- `spec.network.nodePortAllocation.count`
- `spec.network.nodePortAllocation.containerStartPort`

Example resource: [`examples/gameserver.yaml`](./examples/gameserver.yaml)

## Development

```bash
make test
make build
make lint
```

Current repository behavior:

- `make test` passes (excluding e2e, which requires a real cluster)
- `make build` builds the operator binary
- `make lint` may still fail on pre-existing `golangci-lint` typecheck/import-version issues in this environment
