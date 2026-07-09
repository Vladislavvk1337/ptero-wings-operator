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

package webhooks

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

// GameServerClassWebhook handles validation for GameServerClass objects.
type GameServerClassWebhook struct{}

// SetupWebhookWithManager registers the webhook handlers with the manager.
func (w *GameServerClassWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.GameServerClass{}).
		WithValidator(w).
		Complete()
}

// ValidateCreate validates a new GameServerClass.
func (w *GameServerClassWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	class, ok := obj.(*v1alpha1.GameServerClass)
	if !ok {
		return nil, fmt.Errorf("expected *GameServerClass, got %T", obj)
	}
	if class.Spec.GameType == "" {
		return nil, fmt.Errorf("spec.gameType must not be empty")
	}
	return nil, nil
}

// ValidateUpdate rejects changes to the immutable gameType field.
func (w *GameServerClassWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldClass, ok := oldObj.(*v1alpha1.GameServerClass)
	if !ok {
		return nil, fmt.Errorf("expected *GameServerClass, got %T", oldObj)
	}
	newClass, ok := newObj.(*v1alpha1.GameServerClass)
	if !ok {
		return nil, fmt.Errorf("expected *GameServerClass, got %T", newObj)
	}
	if oldClass.Spec.GameType != newClass.Spec.GameType {
		return nil, fmt.Errorf("spec.gameType is immutable once set (current value: %q)", oldClass.Spec.GameType)
	}
	return nil, nil
}

// ValidateDelete is a no-op.
func (w *GameServerClassWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
