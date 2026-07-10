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

import v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"

// EffectiveDeletePolicy returns the resolved delete policy with storage-level
// policy taking precedence over deprecated lifecycle-level policy.
func EffectiveDeletePolicy(gs *v1alpha1.GameServer) v1alpha1.DeletePolicy {
	if gs.Spec.Storage.DeletePolicy != "" {
		return gs.Spec.Storage.DeletePolicy
	}
	if gs.Spec.Lifecycle.DeletePolicy != "" {
		return gs.Spec.Lifecycle.DeletePolicy
	}
	return v1alpha1.DeletePolicyDelete
}
