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
	"fmt"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// StatefulSetName returns the name used for the StatefulSet owned by gs.
func StatefulSetName(gs *v1alpha1.GameServer) string {
	return gs.Name
}

// ServiceName returns the name used for the Service owned by gs.
func ServiceName(gs *v1alpha1.GameServer) string {
	return gs.Name
}

// PVCName returns the name used for the PVC owned by gs.
func PVCName(gs *v1alpha1.GameServer) string {
	return fmt.Sprintf("%s-data", gs.Name)
}

// SecretName returns the name used for the config Secret owned by gs.
func SecretName(gs *v1alpha1.GameServer) string {
	return fmt.Sprintf("%s-config", gs.Name)
}
