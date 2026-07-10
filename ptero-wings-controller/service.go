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

package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	sharedconsole "github.com/Vladislavvk1337/ptero-wings-operator/internal/console"
	sharedlogs "github.com/Vladislavvk1337/ptero-wings-operator/internal/logs"
	portsalloc "github.com/Vladislavvk1337/ptero-wings-operator/internal/ports"
)

type EmbeddedService struct {
	namespace string
	crClient  ctrlclient.Client
	clientset kubernetes.Interface
	logs      *sharedlogs.Reader
	exec      *sharedconsole.Executor
	panel     PanelClient
	allocator portsalloc.Allocator
	mapper    StartupMapper
}

func NewService(cfg Config) (*EmbeddedService, error) {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("add kubernetes scheme: %w", err)
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("add gameserver scheme: %w", err)
	}
	crClient, err := ctrlclient.New(cfg.RESTConfig, ctrlclient.Options{Scheme: s})
	if err != nil {
		return nil, fmt.Errorf("build controller client: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg.RESTConfig)
	if err != nil {
		return nil, fmt.Errorf("build kubernetes clientset: %w", err)
	}
	allocator := portsalloc.NewClusterAllocator(crClient, portsalloc.Config{MinPort: cfg.NodePortMin, MaxPort: cfg.NodePortMax})
	return &EmbeddedService{
		namespace: cfg.Namespace,
		crClient:  crClient,
		clientset: clientset,
		logs:      sharedlogs.NewReader(clientset),
		exec:      sharedconsole.NewExecutor(clientset, cfg.RESTConfig),
		panel:     NewPanelClient(cfg.PanelURL, cfg.PanelToken),
		allocator: allocator,
		mapper:    StartupMapper{},
	}, nil
}

func (s *EmbeddedService) CreateServer(ctx context.Context, req *CreateServerRequest) (*ServerView, error) {
	gs, err := s.gameServerFromRequest(req)
	if err != nil {
		return nil, err
	}
	allocatedPorts, err := s.allocator.Allocate(ctx, gs)
	if err != nil {
		return nil, fmt.Errorf("allocate bundled nodeports: %w", err)
	}
	gs.Spec.Network.Ports = allocatedPorts
	if err := s.crClient.Create(ctx, gs); err != nil {
		return nil, err
	}
	return &ServerView{ID: gs.Name, Phase: string(gs.Status.Phase), Endpoint: gs.Status.Endpoint}, nil
}

func (s *EmbeddedService) GetServer(ctx context.Context, id string) (*ServerView, error) {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ServerView{ID: gs.Name, Phase: string(gs.Status.Phase), Endpoint: gs.Status.Endpoint}, nil
}

func (s *EmbeddedService) DeleteServer(ctx context.Context, id string) error {
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: s.namespace}}
	if err := s.crClient.Delete(ctx, gs); apierrors.IsNotFound(err) {
		return nil
	} else {
		return err
	}
}

func (s *EmbeddedService) SetPowerState(ctx context.Context, id string, action PowerAction) error {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return err
	}
	original := gs.DeepCopy()
	switch action {
	case PowerActionStart:
		gs.Spec.Lifecycle.Suspended = false
	case PowerActionStop, PowerActionKill:
		gs.Spec.Lifecycle.Suspended = true
	case PowerActionRestart:
		gs.Annotations = ensureMap(gs.Annotations)
		gs.Annotations["gameserver.pterodactyl.io/restarted-at"] = time.Now().UTC().Format(time.RFC3339)
		gs.Spec.Lifecycle.Suspended = false
	default:
		return fmt.Errorf("unknown power action %q", action)
	}
	return s.crClient.Patch(ctx, gs, ctrlclient.MergeFrom(original))
}

func (s *EmbeddedService) GetResources(ctx context.Context, id string) (*ResourcesResponse, error) {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return nil, err
	}
	pod, err := s.findPodForGameServer(ctx, gs)
	if err != nil {
		return nil, err
	}
	stats, err := s.collectResources(ctx, gs, pod)
	if err != nil {
		return nil, err
	}
	serverID := gs.Spec.External.ExternalServerID
	if serverID == "" {
		serverID = id
	}
	_ = s.panel.PushResources(ctx, &PanelResourcePayload{ServerID: serverID, Stats: stats})
	return stats, nil
}

func (s *EmbeddedService) TailLogs(ctx context.Context, id string, lines int64) ([]byte, error) {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return nil, err
	}
	pod, err := s.findPodForGameServer(ctx, gs)
	if err != nil {
		return nil, err
	}
	return s.logs.Tail(ctx, s.namespace, pod.Name, lines)
}

func (s *EmbeddedService) StreamLogs(ctx context.Context, id string, w io.Writer) error {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return err
	}
	pod, err := s.findPodForGameServer(ctx, gs)
	if err != nil {
		return err
	}
	return s.logs.Stream(ctx, s.namespace, pod.Name, w)
}

func (s *EmbeddedService) ExecCommand(ctx context.Context, id, command string, stdout, stderr io.Writer) error {
	gs, err := s.getGameServer(ctx, id)
	if err != nil {
		return err
	}
	pod, err := s.findPodForGameServer(ctx, gs)
	if err != nil {
		return err
	}
	return s.exec.ExecStream(ctx, s.namespace, pod.Name, []string{"/bin/sh", "-c", command}, bytes.NewBufferString(command+"\n"), stdout, stderr)
}

func (s *EmbeddedService) gameServerFromRequest(req *CreateServerRequest) (*v1alpha1.GameServer, error) {
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.UUID,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":               "ptero-wings-controller",
				"app.kubernetes.io/part-of":                  "ptero-wings-operator",
				"gameserver.pterodactyl.io/pterodactyl-uuid": req.UUID,
			},
		},
		Spec: v1alpha1.GameServerSpec{
			Game: v1alpha1.GameSpec{Type: req.GameType, Image: req.Image},
			Runtime: v1alpha1.RuntimeSpec{ImagePullPolicy: corev1.PullIfNotPresent},
			Storage: v1alpha1.StorageSpec{DeletePolicy: v1alpha1.DeletePolicyRetain, BackupPolicy: v1alpha1.BackupPolicyNone},
			Lifecycle: v1alpha1.LifecycleSpec{DeletePolicy: v1alpha1.DeletePolicyRetain},
			External: v1alpha1.ExternalSpec{ExternalServerID: req.ExternalServerID},
			Pterodactyl: &v1alpha1.PterodactylSpec{ServerUUID: req.UUID},
		},
	}
	for key, value := range req.Environment {
		gs.Spec.Game.Env = append(gs.Spec.Game.Env, corev1.EnvVar{Name: key, Value: value})
	}
	sort.Slice(gs.Spec.Game.Env, func(i, j int) bool { return gs.Spec.Game.Env[i].Name < gs.Spec.Game.Env[j].Name })
	if req.Limits.CPU > 0 || req.Limits.Memory > 0 {
		resources := corev1.ResourceList{}
		if req.Limits.CPU > 0 {
			resources[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(req.Limits.CPU*1000), resource.DecimalSI)
		}
		if req.Limits.Memory > 0 {
			resources[corev1.ResourceMemory] = *resource.NewQuantity(req.Limits.Memory*1024*1024, resource.BinarySI)
		}
		gs.Spec.Resources = corev1.ResourceRequirements{Limits: resources, Requests: resources}
	}
	if req.DiskMB > 0 {
		gs.Spec.Storage.Size = *resource.NewQuantity(req.DiskMB*1024*1024, resource.BinarySI)
		gs.Spec.Storage.MountPath = "/data"
	}
	for _, allocation := range req.Allocations {
		gs.Spec.Network.Ports = append(gs.Spec.Network.Ports, v1alpha1.PortSpec{
			Name:          allocation.Name,
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: int32(allocation.Port),
			NodePort:      int32(allocation.BindPort),
		})
	}
	if req.PortRangeCount > 0 || req.PortRangeStart > 0 {
		gs.Spec.Network.NodePortAllocation = &v1alpha1.NodePortAllocationSpec{
			StartPort:          req.PortRangeStart,
			Count:              req.PortRangeCount,
			ContainerStartPort: req.PortRangeBase,
			Protocol:           corev1.ProtocolTCP,
		}
	}
	if len(gs.Spec.Network.Ports) > 0 || gs.Spec.Network.NodePortAllocation != nil {
		gs.Spec.Network.ServiceType = corev1.ServiceTypeNodePort
	}
	if req.Startup != "" {
		gs.Spec.Runtime.Startup = &v1alpha1.StartupSpec{Raw: req.Startup, Variables: req.StartupVariables}
	}
	if err := ApplyStartup(gs, s.mapper); err != nil {
		return nil, fmt.Errorf("resolve startup command: %w", err)
	}
	return gs, nil
}

func (s *EmbeddedService) getGameServer(ctx context.Context, id string) (*v1alpha1.GameServer, error) {
	gs := &v1alpha1.GameServer{}
	if err := s.crClient.Get(ctx, types.NamespacedName{Name: id, Namespace: s.namespace}, gs); err != nil {
		return nil, err
	}
	return gs, nil
}

func (s *EmbeddedService) findPodForGameServer(ctx context.Context, gs *v1alpha1.GameServer) (*corev1.Pod, error) {
	podName := gs.Status.PodName
	if podName != "" {
		return s.clientset.CoreV1().Pods(s.namespace).Get(ctx, podName, metav1.GetOptions{})
	}
	podList := &corev1.PodList{}
	if err := s.crClient.List(ctx, podList, ctrlclient.InNamespace(s.namespace), ctrlclient.MatchingLabels{"gameserver.pterodactyl.io/name": gs.Name}); err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pod found for gameserver %s", gs.Name)
	}
	return &podList.Items[0], nil
}

func (s *EmbeddedService) collectResources(ctx context.Context, gs *v1alpha1.GameServer, pod *corev1.Pod) (*ResourcesResponse, error) {
	response := &ResourcesResponse{State: string(pod.Status.Phase)}
	for _, container := range pod.Spec.Containers {
		if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
			response.CPUAbsolute += float64(cpu.MilliValue()) / 1000
		} else if cpu, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
			response.CPUAbsolute += float64(cpu.MilliValue()) / 1000
		}
		if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			response.MemoryBytes += memory.Value()
		} else if memory, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
			response.MemoryBytes += memory.Value()
		}
	}
	if !gs.Spec.Storage.Size.IsZero() {
		response.DiskBytes = gs.Spec.Storage.Size.Value()
	}
	if pod.Status.StartTime != nil {
		response.UptimeSeconds = int64(time.Since(pod.Status.StartTime.Time).Seconds())
	}
	cluster, err := s.collectClusterResources(ctx)
	if err == nil {
		response.Cluster = cluster
	}
	return response, nil
}

func (s *EmbeddedService) collectClusterResources(ctx context.Context) (*ClusterResources, error) {
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := s.clientset.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	cluster := &ClusterResources{NodeCount: len(nodes.Items), PodCount: len(pods.Items)}
	for _, node := range nodes.Items {
		cluster.CPUMillis += node.Status.Capacity.Cpu().MilliValue()
		cluster.MemoryBytes += node.Status.Capacity.Memory().Value()
	}
	return cluster, nil
}

func ensureMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	return in
}

var _ Service = (*EmbeddedService)(nil)
var _ = appsv1.StatefulSet{}
var _ = rest.Config{}
