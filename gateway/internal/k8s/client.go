/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package k8s provides a typed client abstraction over the Kubernetes API
// and the GameServer CRD for use by the ptero-wings-gateway.
package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// Client wraps both a typed controller-runtime client (for CRDs) and a
// client-go clientset (for Pod logs and exec which need the core REST client).
type Client struct {
	crClient  client.Client
	clientset kubernetes.Interface
	restCfg   *rest.Config
}

// Clientset returns the underlying kubernetes.Interface for use by log/exec helpers.
func (c *Client) Clientset() kubernetes.Interface { return c.clientset }

// RESTConfig returns the REST config for use by the exec executor.
func (c *Client) RESTConfig() *rest.Config { return c.restCfg }

// NewClient builds a Client from the in-cluster or kubeconfig REST config.
func NewClient(cfg *rest.Config) (*Client, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding core scheme: %w", err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding v1alpha1 scheme: %w", err)
	}

	crClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("building cr client: %w", err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building clientset: %w", err)
	}

	return &Client{
		crClient:  crClient,
		clientset: cs,
		restCfg:   cfg,
	}, nil
}

// ─── GameServer CRUD ──────────────────────────────────────────────────────────

// CreateGameServer creates a new GameServer CR in the given namespace.
func (c *Client) CreateGameServer(ctx context.Context, gs *v1alpha1.GameServer) error {
	return c.crClient.Create(ctx, gs)
}

// GetGameServer retrieves a GameServer by namespace and name.
func (c *Client) GetGameServer(ctx context.Context, namespace, name string) (*v1alpha1.GameServer, error) {
	gs := &v1alpha1.GameServer{}
	if err := c.crClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, gs); err != nil {
		return nil, err
	}
	return gs, nil
}

// ListGameServers returns all GameServers in the namespace that match the given labels.
func (c *Client) ListGameServers(ctx context.Context, namespace string, matchLabels map[string]string) ([]v1alpha1.GameServer, error) {
	list := &v1alpha1.GameServerList{}
	opts := []client.ListOption{client.InNamespace(namespace)}
	if len(matchLabels) > 0 {
		opts = append(opts, client.MatchingLabels(matchLabels))
	}
	if err := c.crClient.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// DeleteGameServer deletes a GameServer CR.
func (c *Client) DeleteGameServer(ctx context.Context, namespace, name string) error {
	gs := &v1alpha1.GameServer{}
	gs.Name = name
	gs.Namespace = namespace
	err := c.crClient.Delete(ctx, gs)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// PatchGameServerSpec applies a merge-patch to the GameServer spec.
// patch is the serialised JSON merge-patch document.
func (c *Client) PatchGameServerSpec(ctx context.Context, namespace, name string, patch []byte) error {
	gs := &v1alpha1.GameServer{}
	gs.Name = name
	gs.Namespace = namespace
	return c.crClient.Patch(ctx, gs, client.RawPatch(types.MergePatchType, patch))
}

// SetSuspended scales the StatefulSet to 0 (suspend) or 1 (resume) by patching lifecycle.suspended.
func (c *Client) SetSuspended(ctx context.Context, namespace, name string, suspended bool) error {
	type lifecycleSpec struct {
		Suspended bool `json:"suspended"`
	}
	type spec struct {
		Lifecycle lifecycleSpec `json:"lifecycle"`
	}
	type patchDoc struct {
		Spec spec `json:"spec"`
	}
	doc := patchDoc{Spec: spec{Lifecycle: lifecycleSpec{Suspended: suspended}}}
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshalling suspended patch: %w", err)
	}
	return c.PatchGameServerSpec(ctx, namespace, name, raw)
}

// ─── Pod helpers ─────────────────────────────────────────────────────────────

// FindPodForGameServer returns the first Running pod that belongs to the game server.
func (c *Client) FindPodForGameServer(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	gs, err := c.GetGameServer(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("get game server: %w", err)
	}
	if gs.Status.PodName == "" {
		return nil, fmt.Errorf("no pod for game server %s/%s yet", namespace, name)
	}

	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, gs.Status.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod: %w", err)
	}
	return pod, nil
}

// ─── Resource stats ───────────────────────────────────────────────────────────

// PodResourceStats is a lightweight stats snapshot for a single pod.
type PodResourceStats struct {
	// CPUCores is the sum of CPU requests across all containers.
	CPUCores float64 `json:"cpu_absolute"`
	// MemoryBytes is the sum of memory requests across all containers.
	MemoryBytes int64 `json:"memory_bytes"`
	// Phase is the pod phase (Running, Pending, …).
	Phase string `json:"state"`
}

// GetResourceStats returns a lightweight resource snapshot from the pod spec.
// Production deployments should wire in the Metrics API for live usage data.
func (c *Client) GetResourceStats(ctx context.Context, namespace, name string) (*PodResourceStats, error) {
	pod, err := c.FindPodForGameServer(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	var cpuMillis int64
	var memBytes int64
	for _, ctr := range pod.Spec.Containers {
		if req := ctr.Resources.Requests; req != nil {
			if cpu, ok := req[corev1.ResourceCPU]; ok {
				cpuMillis += cpu.MilliValue()
			}
			if mem, ok := req[corev1.ResourceMemory]; ok {
				memBytes += mem.Value()
			}
		}
	}

	// If no explicit requests are set, fall back to limits.
	if cpuMillis == 0 && memBytes == 0 {
		for _, ctr := range pod.Spec.Containers {
			if lim := ctr.Resources.Limits; lim != nil {
				if cpu, ok := lim[corev1.ResourceCPU]; ok {
					cpuMillis += cpu.MilliValue()
				}
				if mem, ok := lim[corev1.ResourceMemory]; ok {
					memBytes += mem.Value()
				}
			}
		}
	}

	return &PodResourceStats{
		CPUCores:    float64(cpuMillis) / 1000.0,
		MemoryBytes: memBytes,
		Phase:       string(pod.Status.Phase),
	}, nil
}

// DefaultGameServerFrom creates a minimal GameServer spec from a Wings-compatible
// CreateServerRequest, applying sensible defaults.
func DefaultGameServerFrom(req *CreateServerRequest, namespace string) *v1alpha1.GameServer {
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.UUID,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":               "ptero-wings-gateway",
				"app.kubernetes.io/part-of":                  "ptero-wings-operator",
				"gameserver.pterodactyl.io/pterodactyl-uuid": req.UUID,
			},
		},
		Spec: v1alpha1.GameServerSpec{
			Game: v1alpha1.GameSpec{
				Type:  req.GameType,
				Image: req.Image,
			},
			Runtime: v1alpha1.RuntimeSpec{
				ImagePullPolicy: corev1.PullIfNotPresent,
			},
			Storage: v1alpha1.StorageSpec{
				DeletePolicy: v1alpha1.DeletePolicyRetain,
				BackupPolicy: v1alpha1.BackupPolicyNone,
			},
			Lifecycle: v1alpha1.LifecycleSpec{
				DeletePolicy: v1alpha1.DeletePolicyRetain,
			},
		},
	}

	// Env vars
	for k, v := range req.Environment {
		gs.Spec.Game.Env = append(gs.Spec.Game.Env, corev1.EnvVar{Name: k, Value: v})
	}

	// Resources
	if req.Limits.CPU > 0 || req.Limits.Memory > 0 {
		rl := corev1.ResourceList{}
		if req.Limits.CPU > 0 {
			rl[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(req.Limits.CPU*1000), resource.DecimalSI)
		}
		if req.Limits.Memory > 0 {
			rl[corev1.ResourceMemory] = *resource.NewQuantity(int64(req.Limits.Memory)*1024*1024, resource.BinarySI)
		}
		gs.Spec.Resources = corev1.ResourceRequirements{Limits: rl, Requests: rl}
	}

	// Storage
	if req.DiskMB > 0 {
		gs.Spec.Storage.Size = *resource.NewQuantity(int64(req.DiskMB)*1024*1024, resource.BinarySI)
		gs.Spec.Storage.MountPath = "/data"
	}

	// Ports
	for _, p := range req.Allocations {
		gs.Spec.Network.Ports = append(gs.Spec.Network.Ports, v1alpha1.PortSpec{
			Name:          p.Name,
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: int32(p.Port),
			NodePort:      int32(p.BindPort),
		})
	}
	if len(gs.Spec.Network.Ports) > 0 {
		gs.Spec.Network.ServiceType = corev1.ServiceTypeNodePort
	}

	// Pterodactyl metadata
	gs.Spec.Pterodactyl = &v1alpha1.PterodactylSpec{
		ServerUUID: req.UUID,
	}

	return gs
}

// ─── Request/Response DTOs ────────────────────────────────────────────────────

// AllocationRequest is a single port mapping inside a CreateServerRequest.
type AllocationRequest struct {
	// Name is an optional label for this port, e.g. "game" or "rcon".
	Name string `json:"name"`
	// Port is the container-internal port.
	Port int `json:"port"`
	// BindPort is the desired NodePort (0 = auto-assign).
	BindPort int `json:"bind_port"`
}

// ResourceLimits holds the Wings-compatible resource limit fields.
type ResourceLimits struct {
	// CPU is the number of CPU cores (fractional allowed).
	CPU float64 `json:"cpu"`
	// Memory is the amount of RAM in MiB.
	Memory int64 `json:"memory"`
}

// CreateServerRequest is the Wings-compatible payload for POST /api/servers.
type CreateServerRequest struct {
	// UUID is the Pterodactyl server UUID; used as the GameServer name.
	UUID string `json:"uuid"`
	// GameType is a short game identifier, e.g. "minecraft".
	GameType string `json:"game_type"`
	// Image is the OCI image to run.
	Image string `json:"image"`
	// Environment holds environment variables injected into the container.
	Environment map[string]string `json:"environment"`
	// Limits specifies CPU and memory constraints.
	Limits ResourceLimits `json:"limits"`
	// DiskMB is the requested persistent disk size in MiB.
	DiskMB int64 `json:"disk"`
	// Allocations lists the ports to expose.
	Allocations []AllocationRequest `json:"allocations"`
}
