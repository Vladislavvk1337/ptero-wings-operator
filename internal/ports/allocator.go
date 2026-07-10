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

package ports

import (
	"context"
	"fmt"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

type Allocator interface {
	Allocate(ctx context.Context, gs *v1alpha1.GameServer) ([]v1alpha1.PortSpec, error)
	Release(ctx context.Context, gs *v1alpha1.GameServer) error
}

type Config struct {
	MinPort int32
	MaxPort int32
}

type ClusterAllocator struct {
	client  ctrlclient.Client
	minPort int32
	maxPort int32
}

func NewClusterAllocator(client ctrlclient.Client, cfg Config) *ClusterAllocator {
	minPort := cfg.MinPort
	maxPort := cfg.MaxPort
	if minPort == 0 {
		minPort = 30000
	}
	if maxPort == 0 {
		maxPort = 31000
	}
	return &ClusterAllocator{client: client, minPort: minPort, maxPort: maxPort}
}

func (a *ClusterAllocator) Allocate(ctx context.Context, gs *v1alpha1.GameServer) ([]v1alpha1.PortSpec, error) {
	ports := append([]v1alpha1.PortSpec(nil), gs.Spec.Network.Ports...)
	allocation := gs.Spec.Network.NodePortAllocation
	if allocation == nil {
		return ports, nil
	}

	// Count is modeled as the number of ports after StartPort, so the reserved
	// range size always includes the first port itself. Example: start=30005,count=5
	// reserves 30005-30010.
	rangeSize := int(allocation.Count) + 1
	if rangeSize <= 0 {
		rangeSize = len(ports)
	}
	if rangeSize == 0 {
		rangeSize = 1
	}

	if len(ports) < rangeSize {
		base := allocation.ContainerStartPort
		if base == 0 {
			base = allocation.StartPort
		}
		if base == 0 && len(ports) > 0 {
			base = ports[0].ContainerPort
		}
		if base == 0 {
			base = 25565
		}
		for idx := len(ports); idx < rangeSize; idx++ {
			ports = append(ports, v1alpha1.PortSpec{
				Name:          fmt.Sprintf("range-%d", idx),
				Protocol:      allocation.Protocol,
				ContainerPort: base + int32(idx),
			})
		}
	}

	used, err := a.usedPorts(ctx, gs)
	if err != nil {
		return nil, err
	}
	start := allocation.StartPort
	if start > 0 {
		if err := a.validateRange(start, rangeSize); err != nil {
			return nil, err
		}
		if a.rangeUsed(used, start, rangeSize) {
			return nil, fmt.Errorf("requested nodeport range %d-%d is already allocated", start, start+int32(rangeSize)-1)
		}
	} else {
		candidate, err := a.findFreeRange(used, rangeSize)
		if err != nil {
			return nil, err
		}
		start = candidate
	}

	for idx := range ports {
		ports[idx].NodePort = start + int32(idx)
		if ports[idx].Protocol == "" {
			ports[idx].Protocol = allocation.Protocol
		}
	}
	gs.Spec.Network.NodePortAllocation.StartPort = start
	return ports, nil
}

func (a *ClusterAllocator) Release(context.Context, *v1alpha1.GameServer) error {
	return nil
}

func (a *ClusterAllocator) usedPorts(ctx context.Context, current *v1alpha1.GameServer) (map[int32]struct{}, error) {
	list := &v1alpha1.GameServerList{}
	if err := a.client.List(ctx, list); err != nil {
		return nil, fmt.Errorf("list gameservers for port allocation: %w", err)
	}
	used := make(map[int32]struct{})
	for idx := range list.Items {
		item := &list.Items[idx]
		if item.Name == current.Name && item.Namespace == current.Namespace {
			continue
		}
		for _, port := range item.Spec.Network.Ports {
			if port.NodePort > 0 {
				used[port.NodePort] = struct{}{}
			}
		}
		if item.Spec.Network.NodePortAllocation != nil && item.Spec.Network.NodePortAllocation.StartPort > 0 {
			size := int(item.Spec.Network.NodePortAllocation.Count) + 1
			for offset := 0; offset < size; offset++ {
				used[item.Spec.Network.NodePortAllocation.StartPort+int32(offset)] = struct{}{}
			}
		}
	}
	return used, nil
}

func (a *ClusterAllocator) findFreeRange(used map[int32]struct{}, size int) (int32, error) {
	for start := a.minPort; start <= a.maxPort-int32(size)+1; start++ {
		if !a.rangeUsed(used, start, size) {
			return start, nil
		}
	}
	return 0, fmt.Errorf("no free nodeport range of size %d in %d-%d", size, a.minPort, a.maxPort)
}

func (a *ClusterAllocator) rangeUsed(used map[int32]struct{}, start int32, size int) bool {
	for offset := 0; offset < size; offset++ {
		if _, exists := used[start+int32(offset)]; exists {
			return true
		}
	}
	return false
}

func (a *ClusterAllocator) validateRange(start int32, size int) error {
	end := start + int32(size) - 1
	if start < a.minPort || end > a.maxPort {
		return fmt.Errorf("requested nodeport range %d-%d outside configured range %d-%d", start, end, a.minPort, a.maxPort)
	}
	return nil
}

type NoopAllocator struct{}

func (n *NoopAllocator) Allocate(_ context.Context, gs *v1alpha1.GameServer) ([]v1alpha1.PortSpec, error) {
	return append([]v1alpha1.PortSpec(nil), gs.Spec.Network.Ports...), nil
}

func (n *NoopAllocator) Release(_ context.Context, _ *v1alpha1.GameServer) error {
	return nil
}
