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

// MergeWithClass returns a deep copy of gs with fields filled in from class wherever
// they are unset. If class is nil, a plain deep copy is returned.
func MergeWithClass(gs *v1alpha1.GameServer, class *v1alpha1.GameServerClass) *v1alpha1.GameServer {
	merged := gs.DeepCopy()
	if class == nil {
		return merged
	}

	// Game image and command
	if merged.Spec.Game.Image == "" {
		merged.Spec.Game.Image = class.Spec.DefaultImage
	}
	if len(merged.Spec.Game.Command) == 0 && len(class.Spec.DefaultCommand) > 0 {
		merged.Spec.Game.Command = append([]string(nil), class.Spec.DefaultCommand...)
	}

	// Resources
	if merged.Spec.Resources.Requests == nil && class.Spec.DefaultResources.Requests != nil {
		merged.Spec.Resources.Requests = class.Spec.DefaultResources.Requests.DeepCopy()
	}
	if merged.Spec.Resources.Limits == nil && class.Spec.DefaultResources.Limits != nil {
		merged.Spec.Resources.Limits = class.Spec.DefaultResources.Limits.DeepCopy()
	}

	// Storage
	if merged.Spec.Storage.Size.IsZero() && !class.Spec.DefaultStorage.Size.IsZero() {
		merged.Spec.Storage.Size = class.Spec.DefaultStorage.Size.DeepCopy()
	}
	if merged.Spec.Storage.StorageClassName == nil && class.Spec.DefaultStorage.StorageClassName != nil {
		sc := *class.Spec.DefaultStorage.StorageClassName
		merged.Spec.Storage.StorageClassName = &sc
	}
	if merged.Spec.Storage.AccessMode == "" {
		merged.Spec.Storage.AccessMode = class.Spec.DefaultStorage.AccessMode
	}
	if merged.Spec.Storage.MountPath == "" {
		merged.Spec.Storage.MountPath = class.Spec.DefaultStorage.MountPath
	}

	// Network
	if len(merged.Spec.Network.Ports) == 0 && len(class.Spec.DefaultNetwork.Ports) > 0 {
		merged.Spec.Network.Ports = append([]v1alpha1.PortSpec(nil), class.Spec.DefaultNetwork.Ports...)
	}
	if merged.Spec.Network.ServiceType == "" {
		merged.Spec.Network.ServiceType = class.Spec.DefaultNetwork.ServiceType
	}

	// Scheduling
	if len(merged.Spec.Scheduling.NodeSelector) == 0 && len(class.Spec.PlacementDefaults.NodeSelector) > 0 {
		merged.Spec.Scheduling.NodeSelector = make(map[string]string)
		for k, v := range class.Spec.PlacementDefaults.NodeSelector {
			merged.Spec.Scheduling.NodeSelector[k] = v
		}
	}
	if len(merged.Spec.Scheduling.Tolerations) == 0 && len(class.Spec.PlacementDefaults.Tolerations) > 0 {
		merged.Spec.Scheduling.Tolerations = append([]corev1.Toleration(nil), class.Spec.PlacementDefaults.Tolerations...)
	}

	return merged
}
