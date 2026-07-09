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

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

// finalize runs cleanup when DeletionTimestamp is set and then removes the finalizer.
func (r *GameServerReconciler) finalize(ctx context.Context, gs *v1alpha1.GameServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "finalize")

	// Mark as Stopping (best-effort).
	{
		sp := client.MergeFrom(gs.DeepCopy())
		gs.Status.Phase = v1alpha1.GameServerPhaseStopping
		if err := r.Status().Patch(ctx, gs, sp); err != nil {
			logger.V(1).Info("failed to set Stopping phase", "err", err)
		}
	}

	// Release any allocated ports.
	if err := r.Allocator.Release(ctx, gs); err != nil {
		return ctrl.Result{}, fmt.Errorf("releasing ports: %w", err)
	}

	// When deletePolicy is Retain, detach the PVC owner reference so it survives
	// GameServer deletion (garbage collection would otherwise delete it).
	if gs.Spec.Lifecycle.DeletePolicy == v1alpha1.DeletePolicyRetain {
		if err := r.detachPVC(ctx, gs); err != nil {
			// Non-fatal; log and continue.
			logger.Error(err, "failed to detach PVC owner reference")
		}
	}

	// Re-fetch for the latest resource version before patching finalizers.
	fresh := &v1alpha1.GameServer{}
	if err := r.Get(ctx, types.NamespacedName{Name: gs.Name, Namespace: gs.Namespace}, fresh); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patch := client.MergeFrom(fresh.DeepCopy())
	controllerutil.RemoveFinalizer(fresh, Finalizer)
	if err := r.Patch(ctx, fresh, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	logger.Info("GameServer finalized")
	return ctrl.Result{}, nil
}

// detachPVC strips the GameServer's OwnerReference from its PVC so the PVC
// is not garbage-collected when the GameServer is deleted.
func (r *GameServerReconciler) detachPVC(ctx context.Context, gs *v1alpha1.GameServer) error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: helpers.PVCName(gs), Namespace: gs.Namespace}, pvc)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	patch := client.MergeFrom(pvc.DeepCopy())
	filtered := pvc.OwnerReferences[:0]
	for _, ref := range pvc.OwnerReferences {
		if ref.UID != gs.UID {
			filtered = append(filtered, ref)
		}
	}
	pvc.OwnerReferences = filtered
	return r.Patch(ctx, pvc, patch)
}
