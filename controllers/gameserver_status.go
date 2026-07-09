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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

// syncStatus reads the current state of all child resources and updates
// GameServer.Status accordingly. It is always called at the end of a successful reconcile.
func (r *GameServerReconciler) syncStatus(ctx context.Context, gs *v1alpha1.GameServer, effective *v1alpha1.GameServer) error {
	logger := log.FromContext(ctx)
	patch := client.MergeFrom(gs.DeepCopy())

	gs.Status.ObservedGeneration = gs.Generation
	gs.Status.LastError = ""

	// Read StatefulSet (ignore NotFound — it may not exist yet on first run).
	ss := &appsv1.StatefulSet{}
	_ = r.Get(ctx, types.NamespacedName{Name: helpers.StatefulSetName(gs), Namespace: gs.Namespace}, ss)

	// Read pods to populate AllocatedNode and PodName before building the endpoint.
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(gs.Namespace),
		client.MatchingLabels(helpers.PodSelectorLabels(gs)),
	); err == nil && len(podList.Items) > 0 {
		gs.Status.PodName = podList.Items[0].Name
		gs.Status.AllocatedNode = podList.Items[0].Spec.NodeName
	}

	// Read Service and build the external endpoint string.
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: helpers.ServiceName(gs), Namespace: gs.Namespace}, svc); err == nil {
		gs.Status.ServiceName = helpers.ServiceName(gs)
		gs.Status.Endpoint = buildEndpoint(svc, gs.Status.AllocatedNode)
		helpers.SetNetworkReady(gs, true, "ServiceReady", "Service exists")
	} else if apierrors.IsNotFound(err) {
		gs.Status.ServiceName = ""
		gs.Status.Endpoint = ""
		helpers.SetNetworkReady(gs, !helpers.NeedsService(effective), "NoService", "No ports defined")
	} else {
		return fmt.Errorf("reading service: %w", err)
	}

	// PVC readiness.
	if helpers.NeedsStorage(effective) {
		pvc := &corev1.PersistentVolumeClaim{}
		if err := r.Get(ctx, types.NamespacedName{Name: helpers.PVCName(gs), Namespace: gs.Namespace}, pvc); err == nil {
			helpers.SetStorageReady(gs, pvc.Status.Phase == corev1.ClaimBound, "PVCPhase",
				fmt.Sprintf("PVC phase is %s", pvc.Status.Phase))
		} else if !apierrors.IsNotFound(err) {
			logger.Error(err, "failed to read PVC")
		}
	} else {
		helpers.SetStorageReady(gs, true, "NoStorage", "No persistent storage configured")
	}

	// Derive overall phase.
	gs.Status.Phase = derivePhase(gs, ss)

	// Ready condition mirrors the Running phase.
	isReady := gs.Status.Phase == v1alpha1.GameServerPhaseRunning
	helpers.SetReady(gs, isReady, string(gs.Status.Phase), fmt.Sprintf("Phase is %s", gs.Status.Phase))
	helpers.SetReconciling(gs, false, "ReconcileComplete", "Reconcile loop finished successfully")
	helpers.SetStalled(gs, false, "OK", "")

	// Optional Pterodactyl sync condition.
	if gs.Spec.Pterodactyl != nil {
		helpers.SetCondition(&gs.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.ConditionPterodactylSynced,
			Status:             boolCondStatus(gs.Spec.Pterodactyl.Synchronized),
			Reason:             "SyncStatus",
			Message:            "Set by external adapter service",
			ObservedGeneration: gs.Generation,
		})
	}

	return r.Status().Patch(ctx, gs, patch)
}

// derivePhase determines the GameServerPhase from the StatefulSet's current state.
func derivePhase(gs *v1alpha1.GameServer, ss *appsv1.StatefulSet) v1alpha1.GameServerPhase {
	if gs.Spec.Lifecycle.Suspended {
		return v1alpha1.GameServerPhaseSuspended
	}
	if ss.Name == "" {
		// StatefulSet does not exist yet.
		return v1alpha1.GameServerPhaseProvisioning
	}
	desired := int32(1)
	if ss.Spec.Replicas != nil {
		desired = *ss.Spec.Replicas
	}
	switch {
	case desired == 0:
		return v1alpha1.GameServerPhaseSuspended
	case ss.Status.ReadyReplicas >= desired:
		return v1alpha1.GameServerPhaseRunning
	case ss.Status.CurrentReplicas > 0:
		return v1alpha1.GameServerPhaseStarting
	default:
		return v1alpha1.GameServerPhasePending
	}
}

// buildEndpoint returns a "host:port" string for the first TCP port of svc.
// It uses the LoadBalancer ingress, NodePort, or falls back to empty.
func buildEndpoint(svc *corev1.Service, allocatedNode string) string {
	if len(svc.Spec.Ports) == 0 {
		return ""
	}

	host := ""
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) > 0 {
		ing := svc.Status.LoadBalancer.Ingress[0]
		if ing.Hostname != "" {
			host = ing.Hostname
		} else {
			host = ing.IP
		}
	}
	if host == "" && svc.Spec.Type == corev1.ServiceTypeNodePort && allocatedNode != "" {
		host = allocatedNode
	}
	if host == "" {
		return ""
	}

	for _, p := range svc.Spec.Ports {
		if p.Protocol == corev1.ProtocolTCP {
			port := p.NodePort
			if port == 0 {
				port = p.Port
			}
			return fmt.Sprintf("%s:%d", host, port)
		}
	}
	return ""
}

func boolCondStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
