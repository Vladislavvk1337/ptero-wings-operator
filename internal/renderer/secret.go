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

// Package renderer constructs Kubernetes resource objects from a GameServer effective spec.
// All functions are pure: they do not call the Kubernetes API.
package renderer

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

// Secret builds a Secret containing game server configuration data.
// The secret is always created so it can be optionally mounted/envFrom-ed;
// its content grows as the operator learns more about the server.
func Secret(gs *v1alpha1.GameServer) *corev1.Secret {
	labels := helpers.LabelsForGameServer(gs)
	data := map[string][]byte{}

	if gs.Spec.OwnerRef != nil {
		data["userId"] = []byte(gs.Spec.OwnerRef.UserID)
		data["nodeId"] = []byte(gs.Spec.OwnerRef.NodeID)
		data["serverUid"] = []byte(gs.Spec.OwnerRef.ServerUID)
	}
	if gs.Spec.Pterodactyl != nil {
		data["pterodactylUUID"] = []byte(gs.Spec.Pterodactyl.ServerUUID)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.SecretName(gs),
			Namespace: gs.Namespace,
			Labels:    labels,
		},
		Data: data,
	}
}
