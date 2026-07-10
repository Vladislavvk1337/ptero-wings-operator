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

type GameServerClassSpec struct {
	GameType          string                      `json:"gameType"`
	DefaultImage      string                      `json:"defaultImage,omitempty"`
	DefaultCommand    []string                    `json:"defaultCommand,omitempty"`
	DefaultArgs       []string                    `json:"defaultArgs,omitempty"`
	DefaultStartup    *StartupSpec                `json:"defaultStartup,omitempty"`
	DefaultResources  corev1.ResourceRequirements `json:"defaultResources,omitempty"`
	DefaultStorage    StorageSpec                 `json:"defaultStorage,omitempty"`
	DefaultNetwork    NetworkSpec                 `json:"defaultNetwork,omitempty"`
	PlacementDefaults SchedulingSpec              `json:"placementDefaults,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="GameType",type="string",JSONPath=".spec.gameType"
//+kubebuilder:printcolumn:name="DefaultImage",type="string",JSONPath=".spec.defaultImage"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type GameServerClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerClassSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

type GameServerClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServerClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameServerClass{}, &GameServerClassList{})
}
