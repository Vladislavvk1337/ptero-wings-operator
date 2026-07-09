# ptero-wings-operator

A Kubernetes operator that acts as the bridge between a **Pterodactyl Panel** and a Kubernetes cluster.
It watches `GameServer` custom resources and reconciles the desired state into Kubernetes workloads:

* **Deployment** – manages the game server pod(s) with the configured container image, environment variables, resource limits, node selectors and optional persistent storage.
* **Service (NodePort)** – exposes the game server ports to the outside world, forwarding both TCP and UDP traffic.
* **Status monitoring** – reports `phase`, `readyReplicas`, and a standard `Available` condition back on the `GameServer` resource so the Panel can poll the operator for live state.

## Architecture

```
Pterodactyl Panel
       │  REST / webhook
       ▼
  GameServer CR  (gameserver.pterodactyl.io/v1alpha1)
       │
  ptero-wings-operator (controller-runtime)
       ├──► Deployment  (game server pod)
       └──► Service     (NodePort, TCP + UDP)
```

## Quick Start

### Prerequisites

* Kubernetes 1.25+
* `kubectl` configured to point at your cluster
* `make`, `go 1.21+`

### Install the CRD and run the operator locally

```bash
# Install CRDs
make install

# Run the operator (watches all namespaces)
make run
```

### Deploy a Minecraft server

```bash
kubectl apply -f config/samples/gameserver_v1alpha1_gameserver.yaml
kubectl get gameservers
# NAME                PHASE     READY   IMAGE                          AGE
# gameserver-sample   Running   1       itzg/minecraft-server:latest   30s
```

### Stop the server without deleting the resource

```bash
kubectl patch gameserver gameserver-sample --type=merge -p '{"spec":{"replicas":0}}'
```

## GameServer Spec Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `image` | string | **required** | Container image for the game server |
| `imagePullPolicy` | string | `IfNotPresent` | `Always`, `Never`, or `IfNotPresent` |
| `imagePullSecrets` | list | `[]` | Registry credential secrets |
| `replicas` | integer | `1` | Running pods. Set to `0` to stop the server |
| `ports` | list | `[]` | Ports to expose via NodePort Service |
| `ports[].name` | string | | Optional label (e.g. `game`, `query`) |
| `ports[].protocol` | string | `TCP` | `TCP` or `UDP` |
| `ports[].containerPort` | integer | **required** | Port inside the container |
| `ports[].nodePort` | integer | auto | Fixed host port (30000-32767), or 0 for auto |
| `env` | list | `[]` | Environment variables injected into the container |
| `resources` | object | | CPU / memory requests & limits |
| `nodeSelector` | map | | Constrain scheduling to specific nodes |
| `tolerations` | list | | Node taint tolerations |
| `persistentVolumeClaim` | string | | Existing PVC name to mount at `/data` |

## GameServer Status

| Field | Description |
|---|---|
| `phase` | `Pending` / `Running` / `Stopping` / `Stopped` / `Error` |
| `readyReplicas` | Number of pods currently ready |
| `deploymentName` | Name of the managed Deployment |
| `serviceName` | Name of the managed Service |
| `conditions` | Standard k8s conditions (e.g. `Available`) |

## Development

```bash
# Run tests (requires kubebuilder envtest binaries)
make test

# Regenerate deepcopy / CRD manifests after changing types
make generate manifests

# Build the operator binary
make build

# Build and push the container image
make docker-build docker-push IMG=<registry>/ptero-wings-operator:tag
```
