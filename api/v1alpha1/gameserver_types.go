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

// Package v1alpha1 contains API Schema definitions for the gameserver v1alpha1 API group.
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition type constants used in GameServer.Status.Conditions.
const (
	ConditionReady             = "Ready"
	ConditionReconciling       = "Reconciling"
	ConditionStalled           = "Stalled"
	ConditionStorageReady      = "StorageReady"
	ConditionNetworkReady      = "NetworkReady"
	ConditionPterodactylSynced = "PterodactylSynced"
)

// GameServerPhase describes the lifecycle state of a GameServer.
// +kubebuilder:validation:Enum=Pending;Provisioning;Starting;Running;Stopping;Failed;Suspended
type GameServerPhase string

const (
	GameServerPhasePending      GameServerPhase = "Pending"
	GameServerPhaseProvisioning GameServerPhase = "Provisioning"
	GameServerPhaseStarting     GameServerPhase = "Starting"
	GameServerPhaseRunning      GameServerPhase = "Running"
	GameServerPhaseStopping     GameServerPhase = "Stopping"
	GameServerPhaseFailed       GameServerPhase = "Failed"
	GameServerPhaseSuspended    GameServerPhase = "Suspended"
)

// DeletePolicy controls what happens to persistent resources when a GameServer is deleted.
// +kubebuilder:validation:Enum=Delete;Retain
type DeletePolicy string

const (
	// DeletePolicyDelete removes all owned resources including PVCs.
	DeletePolicyDelete DeletePolicy = "Delete"
	// DeletePolicyRetain removes the owner reference from PVCs so they survive deletion.
	DeletePolicyRetain DeletePolicy = "Retain"
)

// GameServerOwnerRef links the server to a Pterodactyl panel user/node.
// This is purely informational; the operator never calls the panel API.
type GameServerOwnerRef struct {
	// UserID is the Pterodactyl user identifier.
	// +optional
	UserID string `json:"userId,omitempty"`
	// NodeID is the Pterodactyl node identifier.
	// +optional
	NodeID string `json:"nodeId,omitempty"`
	// ServerUID is the Pterodactyl server UUID used by an external adapter.
	// +optional
	ServerUID string `json:"serverUid,omitempty"`
}

// GameSpec defines the game container image, entrypoint, and environment.
type GameSpec struct {
	// Type is a short game identifier, e.g. "minecraft", "csgo", "valheim".
	// Immutable once set.
	// +kubebuilder:validation:MinLength=1
	Type string `json:"type"`

	// Image is the OCI container image. May be inherited from a GameServerClass.
	// +optional
	Image string `json:"image,omitempty"`

	// Command overrides the container entrypoint.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args are passed as arguments to the entrypoint.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env injects environment variables into the game container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom populates environment variables from ConfigMaps or Secrets.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
}

// RuntimeSpec controls image pull behaviour.
type RuntimeSpec struct {
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// StorageSpec describes the PVC mounted at the game data directory.
type StorageSpec struct {
	// Size is the requested PVC capacity, e.g. "10Gi".
	// +optional
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName overrides the cluster default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessMode defaults to ReadWriteOnce.
	// +optional
	AccessMode corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty"`

	// MountPath is the path inside the container. Defaults to /data.
	// +optional
	MountPath string `json:"mountPath,omitempty"`

	// ExistingClaim adopts an existing PVC instead of creating a new one.
	// +optional
	ExistingClaim string `json:"existingClaim,omitempty"`
}

// PortSpec exposes a single game server port via the managed Service.
type PortSpec struct {
	// Name is a short label, e.g. "game" or "rcon".
	Name string `json:"name"`

	// +optional
	// +kubebuilder:validation:Enum=TCP;UDP
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// ContainerPort is the port the game server listens on inside the container.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ContainerPort int32 `json:"containerPort"`

	// NodePort is the host port. 0 lets Kubernetes assign one automatically.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32767
	NodePort int32 `json:"nodePort,omitempty"`
}

// NetworkSpec defines the Kubernetes Service configuration.
type NetworkSpec struct {
	// Ports are forwarded to the container via a Kubernetes Service.
	// +optional
	Ports []PortSpec `json:"ports,omitempty"`

	// ServiceType defaults to NodePort when ports are defined.
	// +optional
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

// SchedulingSpec constrains pod placement.
type SchedulingSpec struct {
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// LifecycleSpec controls suspension and resource cleanup on deletion.
type LifecycleSpec struct {
	// Suspended scales the StatefulSet to 0 without deleting the GameServer.
	// +optional
	Suspended bool `json:"suspended,omitempty"`

	// DeletePolicy determines whether PVCs are retained on deletion.
	// Defaults to Delete.
	// +optional
	DeletePolicy DeletePolicy `json:"deletePolicy,omitempty"`
}

// PterodactylSpec carries optional Pterodactyl panel metadata.
// External adapters may write to these fields; the operator treats them as read-only hints.
type PterodactylSpec struct {
	// ServerUUID is the Pterodactyl server UUID.
	// +optional
	ServerUUID string `json:"serverUuid,omitempty"`

	// NodeID is the Pterodactyl node ID.
	// +optional
	NodeID int32 `json:"nodeId,omitempty"`

	// Synchronized is set to true by an external adapter once it has synced
	// this server to the Pterodactyl panel.
	// +optional
	Synchronized bool `json:"synchronized,omitempty"`
}

// GameServerSpec is the desired state of a GameServer.
type GameServerSpec struct {
	// ClassRef optionally references a GameServerClass that supplies defaults.
	// +optional
	ClassRef *corev1.LocalObjectReference `json:"classRef,omitempty"`

	// OwnerRef links this server to a Pterodactyl panel user/node.
	// +optional
	OwnerRef *GameServerOwnerRef `json:"ownerRef,omitempty"`

	// Game defines the game type, image and entrypoint.
	Game GameSpec `json:"game"`

	// Runtime controls image pull behaviour.
	// +optional
	Runtime RuntimeSpec `json:"runtime,omitempty"`

	// Resources are CPU/memory requests and limits for the game container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Storage describes the PVC backing the game data directory.
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Network defines Service type and port configuration.
	// +optional
	Network NetworkSpec `json:"network,omitempty"`

	// Scheduling controls pod placement.
	// +optional
	Scheduling SchedulingSpec `json:"scheduling,omitempty"`

	// Lifecycle controls suspension and deletion behaviour.
	// +optional
	Lifecycle LifecycleSpec `json:"lifecycle,omitempty"`

	// Pterodactyl carries optional Pterodactyl panel metadata.
	// +optional
	Pterodactyl *PterodactylSpec `json:"pterodactyl,omitempty"`
}

// GameServerStatus is the observed state of a GameServer.
type GameServerStatus struct {
	// ObservedGeneration is the .metadata.generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase summarises the server lifecycle.
	// +optional
	Phase GameServerPhase `json:"phase,omitempty"`

	// Conditions holds detailed state transitions following the standard Kubernetes pattern.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Endpoint is the primary host:port the server is reachable on from outside the cluster.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// AllocatedNode is the name of the Kubernetes node running the server pod.
	// +optional
	AllocatedNode string `json:"allocatedNode,omitempty"`

	// PodName is the name of the current game server pod.
	// +optional
	PodName string `json:"podName,omitempty"`

	// ServiceName is the name of the managed Kubernetes Service.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// LastError holds the most recent reconcile error message; cleared on success.
	// +optional
	LastError string `json:"lastError,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.allocatedNode"
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.endpoint"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GameServer represents a single game server deployment managed by the cube-operator.
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec,omitempty"`
	Status GameServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GameServerList contains a list of GameServer.
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}
