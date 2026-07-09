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

// SetCondition upserts c into the conditions slice by Type,
// updating LastTransitionTime only when the Status changes.
func SetCondition(conditions *[]metav1.Condition, c metav1.Condition) {
	now := metav1.Now()
	for i, existing := range *conditions {
		if existing.Type != c.Type {
			continue
		}
		if existing.Status != c.Status {
			c.LastTransitionTime = now
		} else {
			c.LastTransitionTime = existing.LastTransitionTime
		}
		(*conditions)[i] = c
		return
	}
	c.LastTransitionTime = now
	*conditions = append(*conditions, c)
}

// SetReady upserts the Ready condition.
func SetReady(gs *v1alpha1.GameServer, ready bool, reason, msg string) {
	SetCondition(&gs.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionReady,
		Status:             boolStatus(ready),
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gs.Generation,
	})
}

// SetReconciling upserts the Reconciling condition.
func SetReconciling(gs *v1alpha1.GameServer, active bool, reason, msg string) {
	SetCondition(&gs.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionReconciling,
		Status:             boolStatus(active),
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gs.Generation,
	})
}

// SetStalled upserts the Stalled condition.
func SetStalled(gs *v1alpha1.GameServer, stalled bool, reason, msg string) {
	SetCondition(&gs.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionStalled,
		Status:             boolStatus(stalled),
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gs.Generation,
	})
}

// SetStorageReady upserts the StorageReady condition.
func SetStorageReady(gs *v1alpha1.GameServer, ready bool, reason, msg string) {
	SetCondition(&gs.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionStorageReady,
		Status:             boolStatus(ready),
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gs.Generation,
	})
}

// SetNetworkReady upserts the NetworkReady condition.
func SetNetworkReady(gs *v1alpha1.GameServer, ready bool, reason, msg string) {
	SetCondition(&gs.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionNetworkReady,
		Status:             boolStatus(ready),
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gs.Generation,
	})
}

func boolStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
