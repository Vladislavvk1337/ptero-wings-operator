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

package helpers

import (
	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// DefaultPortProtocol returns TCP when protocol is unset.
func DefaultPortProtocol(p corev1.Protocol) corev1.Protocol {
	if p == "" {
		return corev1.ProtocolTCP
	}
	return p
}

// DefaultMountPath returns /data when the StorageSpec.MountPath is unset.
func DefaultMountPath(spec v1alpha1.StorageSpec) string {
	if spec.MountPath == "" {
		return "/data"
	}
	return spec.MountPath
}

// NeedsStorage returns true when the GameServer requires a PVC.
func NeedsStorage(gs *v1alpha1.GameServer) bool {
	return true
}

// NeedsService returns true when the GameServer needs a Kubernetes Service.
func NeedsService(gs *v1alpha1.GameServer) bool {
	return len(gs.Spec.Network.Ports) > 0
}
