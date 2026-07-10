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

// Package validation provides pure-Go spec validation for GameServer objects.
package validation

import (
	"errors"
	"fmt"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// ValidateEffective validates the effective GameServer spec after class defaults are merged.
// This is pure Go – no Kubernetes API calls are made.
func ValidateEffective(gs *v1alpha1.GameServer) error {
	var errs []error

	if gs.Spec.Game.Image == "" {
		errs = append(errs, fmt.Errorf("spec.game.image must not be empty (set it directly or via a GameServerClass)"))
	}

	seen := map[string]struct{}{}
	for i, p := range gs.Spec.Network.Ports {
		if p.Name == "" {
			errs = append(errs, fmt.Errorf("spec.network.ports[%d].name must not be empty", i))
		}
		if _, dup := seen[p.Name]; dup && p.Name != "" {
			errs = append(errs, fmt.Errorf("spec.network.ports[%d].name %q is duplicated", i, p.Name))
		}
		seen[p.Name] = struct{}{}
		if p.ContainerPort <= 0 || p.ContainerPort > 65535 {
			errs = append(errs, fmt.Errorf("spec.network.ports[%d].containerPort %d is out of valid range [1, 65535]", i, p.ContainerPort))
		}
		if p.NodePort < 0 || (p.NodePort > 0 && p.NodePort < 30000) || p.NodePort > 32767 {
			errs = append(errs, fmt.Errorf("spec.network.ports[%d].nodePort %d must be 0 (auto) or in [30000, 32767]", i, p.NodePort))
		}
	}

	switch gs.Spec.Lifecycle.DeletePolicy {
	case "", v1alpha1.DeletePolicyDelete, v1alpha1.DeletePolicyRetain:
		// valid
	default:
		errs = append(errs, fmt.Errorf("spec.lifecycle.deletePolicy %q is invalid; use Delete or Retain", gs.Spec.Lifecycle.DeletePolicy))
	}
	switch gs.Spec.Storage.DeletePolicy {
	case "", v1alpha1.DeletePolicyDelete, v1alpha1.DeletePolicyRetain:
		// valid
	default:
		errs = append(errs, fmt.Errorf("spec.storage.deletePolicy %q is invalid; use Delete or Retain", gs.Spec.Storage.DeletePolicy))
	}
	switch gs.Spec.Storage.BackupPolicy {
	case "", v1alpha1.BackupPolicyNone, v1alpha1.BackupPolicySnapshot, v1alpha1.BackupPolicyBackup:
		// valid
	default:
		errs = append(errs, fmt.Errorf("spec.storage.backupPolicy %q is invalid; use None, Snapshot, or Backup", gs.Spec.Storage.BackupPolicy))
	}

	if gs.Spec.Storage.ExistingClaim != "" {
		errs = append(errs, fmt.Errorf("spec.storage.existingClaim is not supported; each GameServer gets a dedicated PVC"))
	}
	if gs.Spec.Lifecycle.RestartPolicy != "" && gs.Spec.Lifecycle.RestartPolicy != "Always" {
		errs = append(errs, fmt.Errorf("spec.lifecycle.restartPolicy %q is not supported for StatefulSet-backed servers; use Always", gs.Spec.Lifecycle.RestartPolicy))
	}

	return errors.Join(errs...)
}
