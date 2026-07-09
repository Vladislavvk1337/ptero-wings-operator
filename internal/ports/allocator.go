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

// Package ports provides an Allocator interface for managing game server port assignments.
// Implementations can use Kubernetes annotations, a dedicated CRD, or an external service.
// The NoopAllocator is provided as a pass-through placeholder.
package ports

import (
	"context"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// Allocator manages NodePort assignments for GameServer instances.
type Allocator interface {
	// Allocate returns the effective port list for gs, assigning NodePorts where needed.
	// The returned slice may be a modified copy; the caller must apply it to the spec.
	Allocate(ctx context.Context, gs *v1alpha1.GameServer) ([]v1alpha1.PortSpec, error)

	// Release frees any ports reserved for gs. Called during finalization.
	Release(ctx context.Context, gs *v1alpha1.GameServer) error
}

// NoopAllocator passes port specs through unchanged.
// Replace this with a real implementation that tracks used NodePorts cluster-wide.
type NoopAllocator struct{}

func (n *NoopAllocator) Allocate(_ context.Context, gs *v1alpha1.GameServer) ([]v1alpha1.PortSpec, error) {
	return append([]v1alpha1.PortSpec(nil), gs.Spec.Network.Ports...), nil
}

func (n *NoopAllocator) Release(_ context.Context, _ *v1alpha1.GameServer) error {
	return nil
}
