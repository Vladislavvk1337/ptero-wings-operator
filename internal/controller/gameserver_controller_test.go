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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

var _ = Describe("GameServer Controller", func() {
	const (
		resourceName = "test-gameserver"
		namespace    = "default"
		timeout      = 10 * time.Second
		interval     = 250 * time.Millisecond
	)

	ctx := context.Background()
	nsn := types.NamespacedName{Name: resourceName, Namespace: namespace}

	newReconciler := func() *GameServerReconciler {
		return &GameServerReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	}

	reconcileOnce := func() {
		_, err := newReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: nsn})
		Expect(err).NotTo(HaveOccurred())
	}

	AfterEach(func() {
		gs := &gameserverv1alpha1.GameServer{}
		if err := k8sClient.Get(ctx, nsn, gs); err == nil {
			// Remove finalizer so deletion succeeds in envtest.
			gs.Finalizers = nil
			_ = k8sClient.Update(ctx, gs)
			_ = k8sClient.Delete(ctx, gs)
		}

		// Wait for deletion.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nsn, &gameserverv1alpha1.GameServer{})
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	Context("When creating a minimal GameServer", func() {
		BeforeEach(func() {
			gs := &gameserverv1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
				Spec: gameserverv1alpha1.GameServerSpec{
					Image: "itzg/minecraft-server:latest",
				},
			}
			Expect(k8sClient.Create(ctx, gs)).To(Succeed())
		})

		It("adds the finalizer on the first reconcile", func() {
			reconcileOnce()

			gs := &gameserverv1alpha1.GameServer{}
			Expect(k8sClient.Get(ctx, nsn, gs)).To(Succeed())
			Expect(gs.Finalizers).To(ContainElement(finalizerName))
		})

		It("creates a Deployment owned by the GameServer", func() {
			// Run reconcile twice: first adds finalizer (requeues), second creates resources.
			reconcileOnce()
			reconcileOnce()

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, nsn, deploy)).To(Succeed())
			Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal("itzg/minecraft-server:latest"))
		})

		It("does not create a Service when no ports are defined", func() {
			reconcileOnce()
			reconcileOnce()

			svc := &corev1.Service{}
			err := k8sClient.Get(ctx, nsn, svc)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When creating a GameServer with ports", func() {
		BeforeEach(func() {
			replicas := int32(1)
			gs := &gameserverv1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
				Spec: gameserverv1alpha1.GameServerSpec{
					Image:    "itzg/minecraft-server:latest",
					Replicas: &replicas,
					Ports: []gameserverv1alpha1.PortSpec{
						{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565},
						{Name: "query", Protocol: corev1.ProtocolUDP, ContainerPort: 25565},
					},
				},
			}
			Expect(k8sClient.Create(ctx, gs)).To(Succeed())
		})

		It("creates a NodePort Service with the correct ports", func() {
			reconcileOnce()
			reconcileOnce()

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, nsn, svc)).To(Succeed())
			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			Expect(svc.Spec.Ports).To(HaveLen(2))
			Expect(svc.Spec.Ports[0].Name).To(Equal("game"))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(25565)))
			Expect(svc.Spec.Ports[1].Protocol).To(Equal(corev1.ProtocolUDP))
		})

		It("reflects ports in the Deployment container", func() {
			reconcileOnce()
			reconcileOnce()

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, nsn, deploy)).To(Succeed())
			Expect(deploy.Spec.Template.Spec.Containers[0].Ports).To(HaveLen(2))
		})
	})

	Context("When creating a GameServer with replicas=0 (stopped)", func() {
		BeforeEach(func() {
			replicas := int32(0)
			gs := &gameserverv1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
				Spec: gameserverv1alpha1.GameServerSpec{
					Image:    "itzg/minecraft-server:latest",
					Replicas: &replicas,
				},
			}
			Expect(k8sClient.Create(ctx, gs)).To(Succeed())
		})

		It("sets phase to Stopped", func() {
			reconcileOnce()
			reconcileOnce()
			reconcileOnce()

			gs := &gameserverv1alpha1.GameServer{}
			Expect(k8sClient.Get(ctx, nsn, gs)).To(Succeed())
			Expect(gs.Status.Phase).To(Equal(gameserverv1alpha1.GameServerPhaseStopped))
		})
	})

	Context("When creating a GameServer with resource limits", func() {
		BeforeEach(func() {
			gs := &gameserverv1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
				Spec: gameserverv1alpha1.GameServerSpec{
					Image: "itzg/minecraft-server:latest",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, gs)).To(Succeed())
		})

		It("propagates resource requirements to the Deployment container", func() {
			reconcileOnce()
			reconcileOnce()

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, nsn, deploy)).To(Succeed())
			c := deploy.Spec.Template.Spec.Containers[0]
			Expect(c.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(c.Resources.Limits.Memory().String()).To(Equal("2Gi"))
		})
	})

	Context("When creating a GameServer with a PVC", func() {
		BeforeEach(func() {
			gs := &gameserverv1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
				Spec: gameserverv1alpha1.GameServerSpec{
					Image:                 "itzg/minecraft-server:latest",
					PersistentVolumeClaim: "my-data-pvc",
				},
			}
			Expect(k8sClient.Create(ctx, gs)).To(Succeed())
		})

		It("mounts the PVC at /data in the container", func() {
			reconcileOnce()
			reconcileOnce()

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, nsn, deploy)).To(Succeed())
			Expect(deploy.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(deploy.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal("my-data-pvc"))
			Expect(deploy.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/data"))
		})
	})
})
