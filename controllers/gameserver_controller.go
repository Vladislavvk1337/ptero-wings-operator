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

// Package controllers implements the cube-operator reconcilers.
package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
	portsalloc "github.com/Vladislavvk1337/ptero-wings-operator/internal/ports"
	"github.com/Vladislavvk1337/ptero-wings-operator/internal/validation"
	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

// Finalizer is added to every GameServer to ensure cleanup runs before deletion.
const Finalizer = "gameserver.pterodactyl.io/finalizer"

// GameServerReconciler reconciles GameServer objects.
type GameServerReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Allocator portsalloc.Allocator
}

//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers/finalizers,verbs=update
//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameserverclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop.
//
// Flow:
//  1. Load GameServer (return if not found)
//  2. Ensure finalizer
//  3. Handle deletion
//  4. Mark Reconciling
//  5. Load optional GameServerClass
//  6. Merge class defaults
//  7. Validate effective spec
//  8. Resolve ports via Allocator
//  9. Reconcile Secret, PVC, Service, StatefulSet
//  10. Sync status (phase, conditions, endpoint)
func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Load GameServer
	gs := &v1alpha1.GameServer{}
	if err := r.Get(ctx, req.NamespacedName, gs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Ensure finalizer
	if !controllerutil.ContainsFinalizer(gs, Finalizer) {
		patch := client.MergeFrom(gs.DeepCopy())
		controllerutil.AddFinalizer(gs, Finalizer)
		if err := r.Patch(ctx, gs, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 3. Handle deletion
	if !gs.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, gs)
	}

	// 4. Mark as reconciling (best-effort; ignore patch errors here)
	{
		statusPatch := client.MergeFrom(gs.DeepCopy())
		helpers.SetReconciling(gs, true, "Reconciling", "Controller is reconciling the resource")
		if err := r.Status().Patch(ctx, gs, statusPatch); err != nil {
			logger.V(1).Info("failed to set Reconciling condition", "err", err)
		}
	}

	// 5. Load optional GameServerClass
	var class *v1alpha1.GameServerClass
	if gs.Spec.ClassRef != nil {
		class = &v1alpha1.GameServerClass{}
		if err := r.Get(ctx, types.NamespacedName{Name: gs.Spec.ClassRef.Name}, class); err != nil {
			return ctrl.Result{}, r.failWith(ctx, gs,
				fmt.Errorf("loading GameServerClass %q: %w", gs.Spec.ClassRef.Name, err))
		}
	}

	// 6. Merge class defaults → effective spec
	effective := helpers.MergeWithClass(gs, class)

	if err := wingscontroller.ApplyStartup(effective, wingscontroller.StartupMapper{}); err != nil {
		return ctrl.Result{}, r.failWith(ctx, gs, fmt.Errorf("startup mapping: %w", err))
	}

	// 7. Validate effective spec
	if err := validation.ValidateEffective(effective); err != nil {
		return ctrl.Result{}, r.failWith(ctx, gs, fmt.Errorf("validation: %w", err))
	}

	// 8. Resolve ports
	allocatedPorts, err := r.Allocator.Allocate(ctx, effective)
	if err != nil {
		return ctrl.Result{}, r.failWith(ctx, gs, fmt.Errorf("port allocation: %w", err))
	}
	effective.Spec.Network.Ports = allocatedPorts

	// 9. Reconcile child resources
	if err := r.reconcileResources(ctx, gs, effective); err != nil {
		return ctrl.Result{}, r.failWith(ctx, gs, err)
	}

	// 10. Sync status
	if err := r.syncStatus(ctx, gs, effective); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// failWith records a failure in status and returns the original error.
func (r *GameServerReconciler) failWith(ctx context.Context, gs *v1alpha1.GameServer, err error) error {
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Status.Phase = v1alpha1.GameServerPhaseFailed
	gs.Status.LastError = err.Error()
	helpers.SetReady(gs, false, "ReconcileError", err.Error())
	helpers.SetStalled(gs, true, "ReconcileError", err.Error())
	helpers.SetReconciling(gs, false, "Failed", "Reconcile failed")
	_ = r.Status().Patch(ctx, gs, patch)
	return err
}

// SetupWithManager registers the controller with the manager.
func (r *GameServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GameServer{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
