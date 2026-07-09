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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/internal/renderer"
)

// reconcileResources reconciles all child resources in dependency order.
// gs is the original object used for OwnerReference; effective is the merged spec.
func (r *GameServerReconciler) reconcileResources(ctx context.Context, gs, effective *v1alpha1.GameServer) error {
	if err := r.reconcileSecret(ctx, gs, effective); err != nil {
		return fmt.Errorf("secret: %w", err)
	}
	if err := r.reconcilePVC(ctx, gs, effective); err != nil {
		return fmt.Errorf("pvc: %w", err)
	}
	if err := r.reconcileService(ctx, gs, effective); err != nil {
		return fmt.Errorf("service: %w", err)
	}
	if err := r.reconcileStatefulSet(ctx, gs, effective); err != nil {
		return fmt.Errorf("statefulset: %w", err)
	}
	return nil
}

func (r *GameServerReconciler) reconcileSecret(ctx context.Context, gs, effective *v1alpha1.GameServer) error {
	desired := renderer.Secret(effective)
	if err := controllerutil.SetControllerReference(gs, desired, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Only update when data has actually changed.
	if !equality.Semantic.DeepEqual(existing.Data, desired.Data) {
		updated := existing.DeepCopy()
		updated.Data = desired.Data
		updated.Labels = desired.Labels
		return r.Update(ctx, updated)
	}
	return nil
}

func (r *GameServerReconciler) reconcilePVC(ctx context.Context, gs, effective *v1alpha1.GameServer) error {
	desired := renderer.PVC(effective)
	if desired == nil {
		return nil
	}
	if err := controllerutil.SetControllerReference(gs, desired, r.Scheme); err != nil {
		return err
	}

	if err := r.deleteExtraPVCs(ctx, gs, desired.Name); err != nil {
		return err
	}

	existing := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// PVC spec fields are largely immutable; only sync labels.
	if !equality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		updated := existing.DeepCopy()
		updated.Labels = desired.Labels
		return r.Update(ctx, updated)
	}
	return nil
}

func (r *GameServerReconciler) deleteExtraPVCs(ctx context.Context, gs *v1alpha1.GameServer, desiredName string) error {
	var pvcList corev1.PersistentVolumeClaimList
	if err := r.List(ctx, &pvcList,
		client.InNamespace(gs.Namespace),
		client.MatchingLabels{"gameserver.pterodactyl.io/name": gs.Name},
	); err != nil {
		return err
	}

	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		if pvc.Name == desiredName {
			continue
		}
		if err := r.Delete(ctx, pvc); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

func (r *GameServerReconciler) reconcileService(ctx context.Context, gs, effective *v1alpha1.GameServer) error {
	desired := renderer.Service(effective)
	if desired == nil {
		return nil
	}
	if err := controllerutil.SetControllerReference(gs, desired, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Preserve ClusterIP; update ports and selector only.
	updated := existing.DeepCopy()
	updated.Spec.Ports = desired.Spec.Ports
	updated.Spec.Selector = desired.Spec.Selector
	updated.Labels = desired.Labels
	if !equality.Semantic.DeepEqual(existing.Spec.Ports, updated.Spec.Ports) ||
		!equality.Semantic.DeepEqual(existing.Spec.Selector, updated.Spec.Selector) {
		return r.Update(ctx, updated)
	}
	return nil
}

func (r *GameServerReconciler) reconcileStatefulSet(ctx context.Context, gs, effective *v1alpha1.GameServer) error {
	desired := renderer.StatefulSet(effective)
	if err := controllerutil.SetControllerReference(gs, desired, r.Scheme); err != nil {
		return err
	}

	existing := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update replicas and pod template; leave everything else unchanged.
	updated := existing.DeepCopy()
	updated.Spec.Replicas = desired.Spec.Replicas
	updated.Spec.Template = desired.Spec.Template
	updated.Labels = desired.Labels
	if !equality.Semantic.DeepEqual(existing.Spec.Replicas, updated.Spec.Replicas) ||
		!equality.Semantic.DeepEqual(existing.Spec.Template, updated.Spec.Template) {
		return r.Update(ctx, updated)
	}
	return nil
}
