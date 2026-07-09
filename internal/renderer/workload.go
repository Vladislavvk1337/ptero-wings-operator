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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

// StatefulSet builds a StatefulSet for the game server workload.
// StatefulSet is preferred over Deployment for persistent game servers because it
// provides stable pod identities and ordered rolling updates.
func StatefulSet(gs *v1alpha1.GameServer) *appsv1.StatefulSet {
	labels := helpers.LabelsForGameServer(gs)
	selector := helpers.SelectorForGameServer(gs)

	replicas := int32(1)
	if gs.Spec.Lifecycle.Suspended {
		replicas = 0
	}

	container := buildContainer(gs)
	podSpec := buildPodSpec(gs, container)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.StatefulSetName(gs),
			Namespace: gs.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: helpers.ServiceName(gs),
			Selector:    selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       podSpec,
			},
		},
	}
}

func buildContainer(gs *v1alpha1.GameServer) corev1.Container {
	c := corev1.Container{
		Name:            "gameserver",
		Image:           gs.Spec.Game.Image,
		Command:         gs.Spec.Game.Command,
		Args:            gs.Spec.Game.Args,
		Env:             gs.Spec.Game.Env,
		EnvFrom:         append([]corev1.EnvFromSource(nil), gs.Spec.Game.EnvFrom...),
		Resources:       gs.Spec.Resources,
		ImagePullPolicy: gs.Spec.Runtime.ImagePullPolicy,
	}

	// Always inject the config secret as an optional env source.
	// The secret is always created; Optional=true avoids pod failure if it is empty.
	c.EnvFrom = append(c.EnvFrom, corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: helpers.SecretName(gs),
			},
			Optional: boolPtr(true),
		},
	})

	// Container ports
	for _, p := range gs.Spec.Network.Ports {
		c.Ports = append(c.Ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      helpers.DefaultPortProtocol(p.Protocol),
		})
	}

	// Volume mount for persistent data
	if helpers.NeedsStorage(gs) {
		c.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: helpers.DefaultMountPath(gs.Spec.Storage),
			},
		}
	}

	return c
}

func buildPodSpec(gs *v1alpha1.GameServer, container corev1.Container) corev1.PodSpec {
	spec := corev1.PodSpec{
		Containers:       []corev1.Container{container},
		NodeSelector:     gs.Spec.Scheduling.NodeSelector,
		Tolerations:      gs.Spec.Scheduling.Tolerations,
		Affinity:         gs.Spec.Scheduling.Affinity,
		ImagePullSecrets: gs.Spec.Runtime.ImagePullSecrets,
	}

	if helpers.NeedsStorage(gs) {
		spec.Volumes = []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: helpers.PVCName(gs),
					},
				},
			},
		}
	}

	return spec
}

func boolPtr(b bool) *bool { return &b }
