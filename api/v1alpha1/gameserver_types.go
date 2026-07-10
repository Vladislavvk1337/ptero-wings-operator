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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionReady             = "Ready"
	ConditionReconciling       = "Reconciling"
	ConditionStalled           = "Stalled"
	ConditionStorageReady      = "StorageReady"
	ConditionNetworkReady      = "NetworkReady"
	ConditionPterodactylSynced = "PterodactylSynced"
)

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

// +kubebuilder:validation:Enum=Delete;Retain
type DeletePolicy string

const (
	DeletePolicyDelete DeletePolicy = "Delete"
	DeletePolicyRetain DeletePolicy = "Retain"
)

// +kubebuilder:validation:Enum=None;Snapshot;Backup
type BackupPolicy string

const (
	BackupPolicyNone     BackupPolicy = "None"
	BackupPolicySnapshot BackupPolicy = "Snapshot"
	BackupPolicyBackup   BackupPolicy = "Backup"
)

type GameServerOwnerRef struct {
	UserID    string `json:"userId,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	ServerUID string `json:"serverUid,omitempty"`
}

type GameSpec struct {
	Type    string                  `json:"type"`
	Image   string                  `json:"image,omitempty"`
	Command []string                `json:"command,omitempty"`
	Args    []string                `json:"args,omitempty"`
	Env     []corev1.EnvVar         `json:"env,omitempty"`
	EnvFrom []corev1.EnvFromSource  `json:"envFrom,omitempty"`
}

type StartupSpec struct {
	Raw       string            `json:"raw,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

type RuntimeSpec struct {
	ImagePullPolicy  corev1.PullPolicy               `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference   `json:"imagePullSecrets,omitempty"`
	Startup          *StartupSpec                    `json:"startup,omitempty"`
}

type StorageSpec struct {
	Size             resource.Quantity                   `json:"size,omitempty"`
	StorageClassName *string                             `json:"storageClassName,omitempty"`
	AccessMode       corev1.PersistentVolumeAccessMode  `json:"accessMode,omitempty"`
	MountPath        string                              `json:"mountPath,omitempty"`
	ExistingClaim    string                              `json:"existingClaim,omitempty"`
	DeletePolicy     DeletePolicy                        `json:"deletePolicy,omitempty"`
	BackupPolicy     BackupPolicy                        `json:"backupPolicy,omitempty"`
	RestoreFrom      string                              `json:"restoreFrom,omitempty"`
}

type PortSpec struct {
	Name          string          `json:"name"`
	Protocol      corev1.Protocol `json:"protocol,omitempty"`
	ContainerPort int32           `json:"containerPort"`
	NodePort      int32           `json:"nodePort,omitempty"`
}

type NodePortAllocationSpec struct {
	StartPort          int32           `json:"startPort,omitempty"`
	Count              int32           `json:"count,omitempty"`
	ContainerStartPort int32           `json:"containerStartPort,omitempty"`
	Protocol           corev1.Protocol `json:"protocol,omitempty"`
}

type NetworkSpec struct {
	Ports              []PortSpec               `json:"ports,omitempty"`
	ServiceType        corev1.ServiceType       `json:"serviceType,omitempty"`
	NodePortAllocation *NodePortAllocationSpec  `json:"nodePortAllocation,omitempty"`
}

type SchedulingSpec struct {
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Affinity     *corev1.Affinity `json:"affinity,omitempty"`
}

type LifecycleSpec struct {
	Suspended              bool                `json:"suspended,omitempty"`
	DeletePolicy           DeletePolicy        `json:"deletePolicy,omitempty"`
	RestartPolicy          corev1.RestartPolicy `json:"restartPolicy,omitempty"`
	TTLSecondsAfterFinished *int64             `json:"ttlSecondsAfterFinished,omitempty"`
}

type ProtectionSpec struct {
	BackupBeforeDelete   bool `json:"backupBeforeDelete,omitempty"`
	SnapshotBeforeDelete bool `json:"snapshotBeforeDelete,omitempty"`
}

type ExternalSpec struct {
	ExternalServerID string `json:"externalServerId,omitempty"`
}

type PterodactylSpec struct {
	ServerUUID   string `json:"serverUuid,omitempty"`
	NodeID       int32  `json:"nodeId,omitempty"`
	Synchronized bool   `json:"synchronized,omitempty"`
}

type GameServerSpec struct {
	ClassRef     *corev1.LocalObjectReference `json:"classRef,omitempty"`
	OwnerRef     *GameServerOwnerRef          `json:"ownerRef,omitempty"`
	Game         GameSpec                     `json:"game"`
	Runtime      RuntimeSpec                  `json:"runtime,omitempty"`
	Resources    corev1.ResourceRequirements  `json:"resources,omitempty"`
	Storage      StorageSpec                  `json:"storage,omitempty"`
	Network      NetworkSpec                  `json:"network,omitempty"`
	Scheduling   SchedulingSpec               `json:"scheduling,omitempty"`
	Lifecycle    LifecycleSpec                `json:"lifecycle,omitempty"`
	Protection   ProtectionSpec               `json:"protection,omitempty"`
	External     ExternalSpec                 `json:"external,omitempty"`
	Pterodactyl  *PterodactylSpec             `json:"pterodactyl,omitempty"`
}

type GameServerStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              GameServerPhase    `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	Endpoint           string             `json:"endpoint,omitempty"`
	AllocatedNode      string             `json:"allocatedNode,omitempty"`
	PodName            string             `json:"podName,omitempty"`
	ContainerName      string             `json:"containerName,omitempty"`
	ServiceName        string             `json:"serviceName,omitempty"`
	PVCName            string             `json:"pvcName,omitempty"`
	LastError          string             `json:"lastError,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.allocatedNode"
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.endpoint"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec,omitempty"`
	Status GameServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}
