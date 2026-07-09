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

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gameserverv1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

const (
	// finalizerName is added to every GameServer so we can clean up on deletion.
	finalizerName = "gameserver.pterodactyl.io/finalizer"

	// conditionTypeAvailable is the k8s condition type reported in status.
	conditionTypeAvailable = "Available"
)

// GameServerReconciler reconciles a GameServer object.
type GameServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gameserver.pterodactyl.io,resources=gameservers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile moves the cluster state towards the desired state described by the GameServer.
func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the GameServer resource.
	gs := &gameserverv1alpha1.GameServer{}
	if err := r.Get(ctx, req.NamespacedName, gs); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion via finalizer.
	if gs.DeletionTimestamp != nil {
		return r.reconcileDelete(ctx, gs)
	}

	// Ensure finalizer is present.
	if !controllerutil.ContainsFinalizer(gs, finalizerName) {
		controllerutil.AddFinalizer(gs, finalizerName)
		if err := r.Update(ctx, gs); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile Deployment and Service.
	if err := r.reconcileDeployment(ctx, gs); err != nil {
		logger.Error(err, "failed to reconcile Deployment")
		return ctrl.Result{}, r.setPhase(ctx, gs, gameserverv1alpha1.GameServerPhaseError)
	}

	if err := r.reconcileService(ctx, gs); err != nil {
		logger.Error(err, "failed to reconcile Service")
		return ctrl.Result{}, r.setPhase(ctx, gs, gameserverv1alpha1.GameServerPhaseError)
	}

	// Update status from the current Deployment state.
	if err := r.syncStatus(ctx, gs); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileDelete cleans up when a GameServer is deleted.
func (r *GameServerReconciler) reconcileDelete(ctx context.Context, gs *gameserverv1alpha1.GameServer) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(gs, finalizerName) {
		return ctrl.Result{}, nil
	}

	if err := r.setPhase(ctx, gs, gameserverv1alpha1.GameServerPhaseStopping); err != nil {
		return ctrl.Result{}, err
	}

	// The owned Deployment and Service will be garbage collected by the owner reference.
	controllerutil.RemoveFinalizer(gs, finalizerName)
	if err := r.Update(ctx, gs); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// reconcileDeployment ensures a Deployment exists that matches the GameServer spec.
func (r *GameServerReconciler) reconcileDeployment(ctx context.Context, gs *gameserverv1alpha1.GameServer) error {
	desired := r.buildDeployment(gs)

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Patch only when relevant fields changed.
	updated := existing.DeepCopy()
	updated.Spec = desired.Spec
	updated.Labels = desired.Labels
	if !equality.Semantic.DeepEqual(existing.Spec, updated.Spec) ||
		!equality.Semantic.DeepEqual(existing.Labels, updated.Labels) {
		return r.Update(ctx, updated)
	}
	return nil
}

// reconcileService ensures a NodePort Service exists that exposes all ports defined in the spec.
// If no ports are defined, no Service is created (or an existing one is left as-is).
func (r *GameServerReconciler) reconcileService(ctx context.Context, gs *gameserverv1alpha1.GameServer) error {
	if len(gs.Spec.Ports) == 0 {
		return nil
	}

	desired := r.buildService(gs)

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Preserve ClusterIP assigned by Kubernetes; only update ports.
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

// syncStatus reads the current Deployment and mirrors it into GameServer.Status.
func (r *GameServerReconciler) syncStatus(ctx context.Context, gs *gameserverv1alpha1.GameServer) error {
	deploy := &appsv1.Deployment{}
	deployName := deploymentName(gs)
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: gs.Namespace}, deploy); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	patch := client.MergeFrom(gs.DeepCopy())

	gs.Status.DeploymentName = deployName
	if len(gs.Spec.Ports) > 0 {
		gs.Status.ServiceName = serviceName(gs)
	}
	gs.Status.ReadyReplicas = deploy.Status.ReadyReplicas

	replicas := int32(1)
	if gs.Spec.Replicas != nil {
		replicas = *gs.Spec.Replicas
	}

	switch {
	case replicas == 0:
		gs.Status.Phase = gameserverv1alpha1.GameServerPhaseStopped
	case deploy.Status.ReadyReplicas >= replicas:
		gs.Status.Phase = gameserverv1alpha1.GameServerPhaseRunning
	default:
		gs.Status.Phase = gameserverv1alpha1.GameServerPhasePending
	}

	availableCondition := metav1.Condition{
		Type:               conditionTypeAvailable,
		ObservedGeneration: gs.Generation,
	}
	if gs.Status.Phase == gameserverv1alpha1.GameServerPhaseRunning {
		availableCondition.Status = metav1.ConditionTrue
		availableCondition.Reason = "DeploymentAvailable"
		availableCondition.Message = fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, replicas)
	} else {
		availableCondition.Status = metav1.ConditionFalse
		availableCondition.Reason = "DeploymentNotAvailable"
		availableCondition.Message = fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, replicas)
	}
	setCondition(&gs.Status.Conditions, availableCondition)

	return r.Status().Patch(ctx, gs, patch)
}

// setPhase is a helper to update just the phase field in status.
func (r *GameServerReconciler) setPhase(ctx context.Context, gs *gameserverv1alpha1.GameServer, phase gameserverv1alpha1.GameServerPhase) error {
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Status.Phase = phase
	return r.Status().Patch(ctx, gs, patch)
}

// buildDeployment constructs the desired Deployment for a GameServer.
func (r *GameServerReconciler) buildDeployment(gs *gameserverv1alpha1.GameServer) *appsv1.Deployment {
	labels := podLabels(gs)
	replicas := int32(1)
	if gs.Spec.Replicas != nil {
		replicas = *gs.Spec.Replicas
	}

	container := corev1.Container{
		Name:            "gameserver",
		Image:           gs.Spec.Image,
		ImagePullPolicy: gs.Spec.ImagePullPolicy,
		Env:             gs.Spec.Env,
		Resources:       gs.Spec.Resources,
	}
	for _, p := range gs.Spec.Ports {
		container.Ports = append(container.Ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
		})
	}

	podSpec := corev1.PodSpec{
		Containers:       []corev1.Container{container},
		NodeSelector:     gs.Spec.NodeSelector,
		Tolerations:      gs.Spec.Tolerations,
		ImagePullSecrets: gs.Spec.ImagePullSecrets,
	}

	if gs.Spec.PersistentVolumeClaim != "" {
		podSpec.Volumes = []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: gs.Spec.PersistentVolumeClaim,
					},
				},
			},
		}
		podSpec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "data", MountPath: "/data"},
		}
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName(gs),
			Namespace: gs.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       podSpec,
			},
		},
	}
	_ = controllerutil.SetControllerReference(gs, deploy, r.Scheme)
	return deploy
}

// buildService constructs the desired NodePort Service for a GameServer.
func (r *GameServerReconciler) buildService(gs *gameserverv1alpha1.GameServer) *corev1.Service {
	labels := podLabels(gs)

	var svcPorts []corev1.ServicePort
	for i, p := range gs.Spec.Ports {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("port-%d", i)
		}
		sp := corev1.ServicePort{
			Name:       name,
			Protocol:   p.Protocol,
			Port:       p.ContainerPort,
			TargetPort: intstr.FromInt32(p.ContainerPort),
		}
		if p.NodePort != 0 {
			sp.NodePort = p.NodePort
		}
		svcPorts = append(svcPorts, sp)
	}

	// Use ClusterIP when no ports are defined (no external exposure needed).
	svcType := corev1.ServiceTypeNodePort
	if len(svcPorts) == 0 {
		svcType = corev1.ServiceTypeClusterIP
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName(gs),
			Namespace: gs.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: labels,
			Ports:    svcPorts,
		},
	}
	_ = controllerutil.SetControllerReference(gs, svc, r.Scheme)
	return svc
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// ---- helpers ----------------------------------------------------------------

func deploymentName(gs *gameserverv1alpha1.GameServer) string {
	return gs.Name
}

func serviceName(gs *gameserverv1alpha1.GameServer) string {
	return gs.Name
}

func podLabels(gs *gameserverv1alpha1.GameServer) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "gameserver",
		"app.kubernetes.io/instance":   gs.Name,
		"app.kubernetes.io/managed-by": "ptero-wings-operator",
	}
}

// setCondition upserts a condition into the slice (by Type).
func setCondition(conditions *[]metav1.Condition, c metav1.Condition) {
	now := metav1.Now()
	for i, existing := range *conditions {
		if existing.Type == c.Type {
			if existing.Status != c.Status {
				c.LastTransitionTime = now
			} else {
				c.LastTransitionTime = existing.LastTransitionTime
			}
			(*conditions)[i] = c
			return
		}
	}
	c.LastTransitionTime = now
	*conditions = append(*conditions, c)
}
