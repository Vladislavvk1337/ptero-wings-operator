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

func MergeWithClass(gs *v1alpha1.GameServer, class *v1alpha1.GameServerClass) *v1alpha1.GameServer {
	merged := gs.DeepCopy()
	if class == nil {
		return merged
	}

	if merged.Spec.Game.Image == "" {
		merged.Spec.Game.Image = class.Spec.DefaultImage
	}
	if len(merged.Spec.Game.Command) == 0 && len(class.Spec.DefaultCommand) > 0 {
		merged.Spec.Game.Command = append([]string(nil), class.Spec.DefaultCommand...)
	}
	if len(merged.Spec.Game.Args) == 0 && len(class.Spec.DefaultArgs) > 0 {
		merged.Spec.Game.Args = append([]string(nil), class.Spec.DefaultArgs...)
	}
	if merged.Spec.Runtime.Startup == nil && class.Spec.DefaultStartup != nil {
		merged.Spec.Runtime.Startup = &v1alpha1.StartupSpec{Raw: class.Spec.DefaultStartup.Raw}
		if len(class.Spec.DefaultStartup.Variables) > 0 {
			merged.Spec.Runtime.Startup.Variables = make(map[string]string, len(class.Spec.DefaultStartup.Variables))
			for key, value := range class.Spec.DefaultStartup.Variables {
				merged.Spec.Runtime.Startup.Variables[key] = value
			}
		}
	}

	if merged.Spec.Resources.Requests == nil && class.Spec.DefaultResources.Requests != nil {
		merged.Spec.Resources.Requests = class.Spec.DefaultResources.Requests.DeepCopy()
	}
	if merged.Spec.Resources.Limits == nil && class.Spec.DefaultResources.Limits != nil {
		merged.Spec.Resources.Limits = class.Spec.DefaultResources.Limits.DeepCopy()
	}

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
	if merged.Spec.Storage.DeletePolicy == "" {
		merged.Spec.Storage.DeletePolicy = class.Spec.DefaultStorage.DeletePolicy
	}
	if merged.Spec.Storage.BackupPolicy == "" {
		merged.Spec.Storage.BackupPolicy = class.Spec.DefaultStorage.BackupPolicy
	}
	if merged.Spec.Storage.RestoreFrom == "" {
		merged.Spec.Storage.RestoreFrom = class.Spec.DefaultStorage.RestoreFrom
	}

	if len(merged.Spec.Network.Ports) == 0 && len(class.Spec.DefaultNetwork.Ports) > 0 {
		merged.Spec.Network.Ports = append([]v1alpha1.PortSpec(nil), class.Spec.DefaultNetwork.Ports...)
	}
	if merged.Spec.Network.ServiceType == "" {
		merged.Spec.Network.ServiceType = class.Spec.DefaultNetwork.ServiceType
	}
	if merged.Spec.Network.NodePortAllocation == nil && class.Spec.DefaultNetwork.NodePortAllocation != nil {
		copy := *class.Spec.DefaultNetwork.NodePortAllocation
		merged.Spec.Network.NodePortAllocation = &copy
	}

	if len(merged.Spec.Scheduling.NodeSelector) == 0 && len(class.Spec.PlacementDefaults.NodeSelector) > 0 {
		merged.Spec.Scheduling.NodeSelector = make(map[string]string, len(class.Spec.PlacementDefaults.NodeSelector))
		for key, value := range class.Spec.PlacementDefaults.NodeSelector {
			merged.Spec.Scheduling.NodeSelector[key] = value
		}
	}
	if len(merged.Spec.Scheduling.Tolerations) == 0 && len(class.Spec.PlacementDefaults.Tolerations) > 0 {
		merged.Spec.Scheduling.Tolerations = append([]corev1.Toleration(nil), class.Spec.PlacementDefaults.Tolerations...)
	}
	if merged.Spec.Scheduling.Affinity == nil && class.Spec.PlacementDefaults.Affinity != nil {
		merged.Spec.Scheduling.Affinity = class.Spec.PlacementDefaults.Affinity.DeepCopy()
	}

	return merged
}
