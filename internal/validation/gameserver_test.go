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

package validation_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/internal/validation"
)

func newValid() *v1alpha1.GameServer {
	return &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs", Namespace: "default"},
		Spec: v1alpha1.GameServerSpec{
			Game: v1alpha1.GameSpec{Type: "mc", Image: "img:1"},
		},
	}
}

func TestValidateEffective_Valid(t *testing.T) {
	if err := validation.ValidateEffective(newValid()); err != nil {
		t.Errorf("unexpected error for valid spec: %v", err)
	}
}

func TestValidateEffective_MissingImage(t *testing.T) {
	gs := newValid()
	gs.Spec.Game.Image = ""
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for missing image")
	}
}

func TestValidateEffective_EmptyPortName(t *testing.T) {
	gs := newValid()
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{{ContainerPort: 25565}}
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for empty port name")
	}
}

func TestValidateEffective_DuplicatePortName(t *testing.T) {
	gs := newValid()
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{
		{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565},
		{Name: "game", Protocol: corev1.ProtocolUDP, ContainerPort: 25566},
	}
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for duplicate port name")
	}
}

func TestValidateEffective_InvalidContainerPort(t *testing.T) {
	gs := newValid()
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{{Name: "p", ContainerPort: 0}}
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for containerPort=0")
	}
}

func TestValidateEffective_InvalidNodePort(t *testing.T) {
	gs := newValid()
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{{Name: "p", ContainerPort: 25565, NodePort: 100}}
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for nodePort=100 (below valid range)")
	}
}

func TestValidateEffective_InvalidDeletePolicy(t *testing.T) {
	gs := newValid()
	gs.Spec.Lifecycle.DeletePolicy = "Bogus"
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for invalid deletePolicy")
	}
}

func TestValidateEffective_ValidDeletePolicies(t *testing.T) {
	for _, policy := range []v1alpha1.DeletePolicy{
		v1alpha1.DeletePolicyDelete,
		v1alpha1.DeletePolicyRetain,
		"",
	} {
		gs := newValid()
		gs.Spec.Lifecycle.DeletePolicy = policy
		if err := validation.ValidateEffective(gs); err != nil {
			t.Errorf("unexpected error for policy %q: %v", policy, err)
		}
	}
}

func TestValidateEffective_InvalidStorageDeletePolicy(t *testing.T) {
	gs := newValid()
	gs.Spec.Storage.DeletePolicy = "Bogus"
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for invalid spec.storage.deletePolicy")
	}
}

func TestValidateEffective_ValidStorageBackupPolicy(t *testing.T) {
	for _, policy := range []v1alpha1.BackupPolicy{
		"",
		v1alpha1.BackupPolicyNone,
		v1alpha1.BackupPolicySnapshot,
		v1alpha1.BackupPolicyBackup,
	} {
		gs := newValid()
		gs.Spec.Storage.BackupPolicy = policy
		if err := validation.ValidateEffective(gs); err != nil {
			t.Errorf("unexpected error for backup policy %q: %v", policy, err)
		}
	}
}

func TestValidateEffective_InvalidStorageBackupPolicy(t *testing.T) {
	gs := newValid()
	gs.Spec.Storage.BackupPolicy = "Bogus"
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for invalid spec.storage.backupPolicy")
	}
}

func TestValidateEffective_OnlyAlwaysRestartPolicyIsSupported(t *testing.T) {
	gs := newValid()
	gs.Spec.Lifecycle.RestartPolicy = corev1.RestartPolicyOnFailure
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for unsupported restart policy")
	}

	gs = newValid()
	gs.Spec.Lifecycle.RestartPolicy = corev1.RestartPolicyAlways
	if err := validation.ValidateEffective(gs); err != nil {
		t.Errorf("unexpected error for restartPolicy=Always: %v", err)
	}
}

func TestValidateEffective_ExistingClaimIsRejected(t *testing.T) {
	gs := newValid()
	gs.Spec.Storage.ExistingClaim = "legacy-pvc"
	if err := validation.ValidateEffective(gs); err == nil {
		t.Error("expected error for spec.storage.existingClaim")
	}
}
