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

// Package webhooks provides Defaulting and Validating webhooks for the cube-operator CRDs.
package webhooks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// GameServerWebhook handles defaulting and validation for GameServer objects.
type GameServerWebhook struct{}

// SetupWebhookWithManager registers the webhook handlers with the manager.
func (w *GameServerWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.GameServer{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

// Default applies server-side defaults to a GameServer before it is persisted.
func (w *GameServerWebhook) Default(_ context.Context, obj runtime.Object) error {
	gs, ok := obj.(*v1alpha1.GameServer)
	if !ok {
		return fmt.Errorf("expected *GameServer, got %T", obj)
	}

	if gs.Spec.Runtime.ImagePullPolicy == "" {
		gs.Spec.Runtime.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if gs.Spec.Storage.DeletePolicy == "" {
		gs.Spec.Storage.DeletePolicy = v1alpha1.DeletePolicyDelete
	}
	if gs.Spec.Lifecycle.DeletePolicy == "" {
		gs.Spec.Lifecycle.DeletePolicy = gs.Spec.Storage.DeletePolicy
	}
	if gs.Spec.Storage.BackupPolicy == "" {
		gs.Spec.Storage.BackupPolicy = v1alpha1.BackupPolicyNone
	}
	if gs.Spec.Storage.MountPath == "" && !gs.Spec.Storage.Size.IsZero() {
		gs.Spec.Storage.MountPath = "/data"
	}
	if gs.Spec.Lifecycle.RestartPolicy == "" {
		gs.Spec.Lifecycle.RestartPolicy = corev1.RestartPolicyAlways
	}
	for i := range gs.Spec.Network.Ports {
		if gs.Spec.Network.Ports[i].Protocol == "" {
			gs.Spec.Network.Ports[i].Protocol = corev1.ProtocolTCP
		}
	}

	return nil
}

// ValidateCreate validates a new GameServer.
func (w *GameServerWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	gs, ok := obj.(*v1alpha1.GameServer)
	if !ok {
		return nil, fmt.Errorf("expected *GameServer, got %T", obj)
	}
	if gs.Spec.Game.Type == "" {
		return nil, fmt.Errorf("spec.game.type must not be empty")
	}
	return nil, nil
}

// ValidateUpdate rejects changes to immutable fields.
func (w *GameServerWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldGS, ok := oldObj.(*v1alpha1.GameServer)
	if !ok {
		return nil, fmt.Errorf("expected *GameServer, got %T", oldObj)
	}
	newGS, ok := newObj.(*v1alpha1.GameServer)
	if !ok {
		return nil, fmt.Errorf("expected *GameServer, got %T", newObj)
	}

	// spec.game.type is immutable once set; changing it would require a new server.
	if oldGS.Spec.Game.Type != "" && oldGS.Spec.Game.Type != newGS.Spec.Game.Type {
		return nil, fmt.Errorf("spec.game.type is immutable once set (current value: %q)", oldGS.Spec.Game.Type)
	}

	return nil, nil
}

// ValidateDelete is a no-op; deletion is guarded by the finalizer in the controller.
func (w *GameServerWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
