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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameServerPhase describes the lifecycle phase of a GameServer.
// +kubebuilder:validation:Enum=Pending;Running;Stopping;Stopped;Error
type GameServerPhase string

const (
	// GameServerPhasePending means the GameServer has been accepted but resources are not yet ready.
	GameServerPhasePending GameServerPhase = "Pending"
	// GameServerPhaseRunning means all resources are running and the server is reachable.
	GameServerPhaseRunning GameServerPhase = "Running"
	// GameServerPhaseStopping means the server is being shut down.
	GameServerPhaseStopping GameServerPhase = "Stopping"
	// GameServerPhaseStopped means the server has been stopped (replicas scaled to 0).
	GameServerPhaseStopped GameServerPhase = "Stopped"
	// GameServerPhaseError means an unrecoverable error occurred.
	GameServerPhaseError GameServerPhase = "Error"
)

// PortSpec defines a single port that should be exposed by the game server.
type PortSpec struct {
	// Name is an optional label for this port (e.g. "game", "query", "rcon").
	// +optional
	Name string `json:"name,omitempty"`

	// Protocol specifies TCP or UDP. Defaults to TCP.
	// +optional
	// +kubebuilder:validation:Enum=TCP;UDP
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// ContainerPort is the port number the game server listens on inside the container.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ContainerPort int32 `json:"containerPort"`

	// NodePort is the host port that will be allocated on each cluster node.
	// Leave 0 to let Kubernetes assign one automatically (30000-32767 range).
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32767
	NodePort int32 `json:"nodePort,omitempty"`
}

// GameServerSpec defines the desired state of GameServer.
type GameServerSpec struct {
	// Image is the OCI image of the game server (e.g. "itzg/minecraft-server:latest").
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// ImagePullPolicy controls when the image is pulled.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling the image.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Replicas is the desired number of game server pods.
	// Set to 0 to stop the server without deleting the resource.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Ports lists the ports that should be forwarded to the game server container.
	// +optional
	Ports []PortSpec `json:"ports,omitempty"`

	// Env holds environment variables injected into the game server container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources defines CPU/memory requests and limits for the game server container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector constrains scheduling to nodes matching these labels.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations allow the pod to tolerate node taints.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PersistentVolumeClaim is the name of an existing PVC to mount at /data inside
	// the container. Leave empty for ephemeral storage.
	// +optional
	PersistentVolumeClaim string `json:"persistentVolumeClaim,omitempty"`
}

// GameServerStatus defines the observed state of GameServer.
type GameServerStatus struct {
	// Phase summarises the overall lifecycle state.
	// +optional
	Phase GameServerPhase `json:"phase,omitempty"`

	// ReadyReplicas is the number of pods that are currently ready.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// DeploymentName is the name of the managed Deployment.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// ServiceName is the name of the managed Service.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// Conditions holds detailed state transitions following the standard k8s pattern.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
//+kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GameServer is the Schema for the gameservers API.
// It represents a single Pterodactyl game server deployment inside the Kubernetes cluster.
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec,omitempty"`
	Status GameServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GameServerList contains a list of GameServer
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}
