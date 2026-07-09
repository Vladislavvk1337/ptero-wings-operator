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

package controllers_test

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

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
	portsalloc "github.com/Vladislavvk1337/ptero-wings-operator/internal/ports"
)

var _ = Describe("GameServer Controller", func() {
	const (
		name      = "test-gs"
		namespace = "default"
		timeout   = 10 * time.Second
		interval  = 250 * time.Millisecond
	)

	ctx := context.Background()
	nsn := types.NamespacedName{Name: name, Namespace: namespace}

	newReconciler := func() *controllers.GameServerReconciler {
		return &controllers.GameServerReconciler{
			Client:    k8sClient,
			Scheme:    k8sClient.Scheme(),
			Allocator: &portsalloc.NoopAllocator{},
		}
	}

	reconcileOnce := func() {
		_, err := newReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: nsn})
		Expect(err).NotTo(HaveOccurred())
	}

	AfterEach(func() {
		gs := &v1alpha1.GameServer{}
		if err := k8sClient.Get(ctx, nsn, gs); err == nil {
			gs.Finalizers = nil
			_ = k8sClient.Update(ctx, gs)
			_ = k8sClient.Delete(ctx, gs)
		}

		_ = k8sClient.Delete(ctx, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.SecretName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			},
		})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.PVCName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			},
		})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "test-gs-stale", Namespace: namespace},
		})

		Eventually(func() bool {
			return errors.IsNotFound(k8sClient.Get(ctx, nsn, &v1alpha1.GameServer{}))
		}, timeout, interval).Should(BeTrue())
	})

	Context("Minimal GameServer (no ports, no storage)", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
				},
			})).To(Succeed())
		})

		It("adds the finalizer on first reconcile", func() {
			reconcileOnce()
			gs := &v1alpha1.GameServer{}
			Expect(k8sClient.Get(ctx, nsn, gs)).To(Succeed())
			Expect(gs.Finalizers).To(ContainElement(controllers.Finalizer))
		})

		It("creates a StatefulSet after two reconcile iterations", func() {
			reconcileOnce() // adds finalizer + requeues
			reconcileOnce() // creates resources

			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, nsn, ss)).To(Succeed())
			Expect(ss.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal("itzg/minecraft-server:latest"))
		})

		It("creates a config Secret", func() {
			reconcileOnce()
			reconcileOnce()

			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      helpers.SecretName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			}, secret)).To(Succeed())
		})

		It("does not create a Service when no ports are defined", func() {
			reconcileOnce()
			reconcileOnce()

			Expect(errors.IsNotFound(
				k8sClient.Get(ctx, nsn, &corev1.Service{}),
			)).To(BeTrue())
		})

		It("creates a dedicated PVC with default size", func() {
			reconcileOnce()
			reconcileOnce()

			pvc := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      helpers.PVCName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			}, pvc)).To(Succeed())
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
		})
	})

	Context("GameServer with TCP/UDP ports", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
					Network: v1alpha1.NetworkSpec{
						Ports: []v1alpha1.PortSpec{
							{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565},
							{Name: "query", Protocol: corev1.ProtocolUDP, ContainerPort: 25565},
						},
					},
				},
			})).To(Succeed())
		})

		It("creates a NodePort Service with the correct ports", func() {
			reconcileOnce()
			reconcileOnce()

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, nsn, svc)).To(Succeed())
			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			Expect(svc.Spec.Ports).To(HaveLen(2))
			Expect(svc.Spec.Ports[0].Name).To(Equal("game"))
			Expect(svc.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(svc.Spec.Ports[1].Protocol).To(Equal(corev1.ProtocolUDP))
		})

		It("exposes container ports in the StatefulSet", func() {
			reconcileOnce()
			reconcileOnce()

			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, nsn, ss)).To(Succeed())
			Expect(ss.Spec.Template.Spec.Containers[0].Ports).To(HaveLen(2))
		})
	})

	Context("GameServer with storage", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
					Storage: v1alpha1.StorageSpec{
						Size:      resource.MustParse("10Gi"),
						MountPath: "/data",
					},
				},
			})).To(Succeed())
		})

		It("creates a PVC", func() {
			reconcileOnce()
			reconcileOnce()

			pvc := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      helpers.PVCName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			}, pvc)).To(Succeed())
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).NotTo(Equal(resource.MustParse("0")))
		})

		It("mounts the PVC at /data in the StatefulSet container", func() {
			reconcileOnce()
			reconcileOnce()

			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, nsn, ss)).To(Succeed())
			Expect(ss.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(ss.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/data"))
		})
	})

	Context("GameServer with suspended lifecycle", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game:      v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
					Lifecycle: v1alpha1.LifecycleSpec{Suspended: true},
				},
			})).To(Succeed())
		})

		It("sets replicas to 0 in the StatefulSet", func() {
			reconcileOnce()
			reconcileOnce()

			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, nsn, ss)).To(Succeed())
			Expect(*ss.Spec.Replicas).To(Equal(int32(0)))
		})

		It("reflects the Suspended phase in status", func() {
			reconcileOnce()
			reconcileOnce()
			reconcileOnce() // sync status

			gs := &v1alpha1.GameServer{}
			Expect(k8sClient.Get(ctx, nsn, gs)).To(Succeed())
			Expect(gs.Status.Phase).To(Equal(v1alpha1.GameServerPhaseSuspended))
		})
	})

	Context("GameServer with resource limits", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
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
			})).To(Succeed())
		})

		It("propagates resource requirements to the StatefulSet container", func() {
			reconcileOnce()
			reconcileOnce()

			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, nsn, ss)).To(Succeed())
			c := ss.Spec.Template.Spec.Containers[0]
			Expect(c.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(c.Resources.Limits.Memory().String()).To(Equal("2Gi"))
			Expect(c.Resources.Requests.Memory().String()).To(Equal("512Mi"))
		})
	})

	Context("GameServer with ownerRef", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: v1alpha1.GameServerSpec{
					Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/minecraft-server:latest"},
					OwnerRef: &v1alpha1.GameServerOwnerRef{
						UserID:    "user-42",
						NodeID:    "node-1",
						ServerUID: "abc-123",
					},
				},
			})).To(Succeed())
		})

		It("stores ownerRef fields in the config Secret", func() {
			reconcileOnce()
			reconcileOnce()

			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      helpers.SecretName(&v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name}}),
				Namespace: namespace,
			}, secret)).To(Succeed())
			Expect(string(secret.Data["userId"])).To(Equal("user-42"))
			Expect(string(secret.Data["nodeId"])).To(Equal("node-1"))
		})
	})
})
