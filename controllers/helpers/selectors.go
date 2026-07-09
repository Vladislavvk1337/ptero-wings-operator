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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// LabelsForGameServer returns the standard labels applied to every resource owned by gs.
func LabelsForGameServer(gs *v1alpha1.GameServer) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":         "gameserver",
		"app.kubernetes.io/instance":     gs.Name,
		"app.kubernetes.io/managed-by":   "cube-operator",
		"gameserver.pterodactyl.io/name": gs.Name,
	}
	if gs.Spec.Game.Type != "" {
		labels["gameserver.pterodactyl.io/game"] = gs.Spec.Game.Type
	}
	return labels
}

// SelectorForGameServer returns the minimal LabelSelector used by Services and StatefulSets
// to identify pods belonging to gs.
func SelectorForGameServer(gs *v1alpha1.GameServer) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/instance":     gs.Name,
			"gameserver.pterodactyl.io/name": gs.Name,
		},
	}
}

// PodSelectorLabels returns the flat map used by corev1.Service.Spec.Selector
// and client.MatchingLabels for pod lookups.
func PodSelectorLabels(gs *v1alpha1.GameServer) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":     gs.Name,
		"gameserver.pterodactyl.io/name": gs.Name,
	}
}
