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

package renderer

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

// Service builds a Kubernetes Service for the game server's network ports.
// Returns nil when no ports are defined.
func Service(gs *v1alpha1.GameServer) *corev1.Service {
	if !helpers.NeedsService(gs) {
		return nil
	}

	labels := helpers.LabelsForGameServer(gs)

	svcPorts := make([]corev1.ServicePort, 0, len(gs.Spec.Network.Ports))
	for i, p := range gs.Spec.Network.Ports {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("port-%d", i)
		}
		proto := helpers.DefaultPortProtocol(p.Protocol)
		sp := corev1.ServicePort{
			Name:       name,
			Protocol:   proto,
			Port:       p.ContainerPort,
			TargetPort: intstr.FromInt32(p.ContainerPort),
		}
		if p.NodePort > 0 {
			sp.NodePort = p.NodePort
		}
		svcPorts = append(svcPorts, sp)
	}

	svcType := gs.Spec.Network.ServiceType
	if svcType == "" {
		svcType = corev1.ServiceTypeNodePort
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ServiceName(gs),
			Namespace: gs.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: helpers.PodSelectorLabels(gs),
			Ports:    svcPorts,
		},
	}
}
