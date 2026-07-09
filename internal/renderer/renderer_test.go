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

package renderer_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/internal/renderer"
)

func baseGS(name string) *v1alpha1.GameServer {
	return &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: v1alpha1.GameServerSpec{
			Game: v1alpha1.GameSpec{Type: "minecraft", Image: "itzg/mc:latest"},
		},
	}
}

// ---- StatefulSet -----------------------------------------------------------

func TestStatefulSet_BasicImage(t *testing.T) {
	gs := baseGS("srv")
	ss := renderer.StatefulSet(gs)
	if ss.Name != "srv" {
		t.Errorf("StatefulSet name = %q", ss.Name)
	}
	c := ss.Spec.Template.Spec.Containers[0]
	if c.Image != "itzg/mc:latest" {
		t.Errorf("image = %q", c.Image)
	}
}

func TestStatefulSet_SuspendedSetsZeroReplicas(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Lifecycle.Suspended = true
	ss := renderer.StatefulSet(gs)
	if *ss.Spec.Replicas != 0 {
		t.Errorf("replicas = %d, want 0", *ss.Spec.Replicas)
	}
}

func TestStatefulSet_WithStorage(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Storage.Size = resource.MustParse("5Gi")
	gs.Spec.Storage.MountPath = "/data"
	ss := renderer.StatefulSet(gs)
	if len(ss.Spec.Template.Spec.Volumes) != 1 {
		t.Error("expected 1 volume")
	}
	if ss.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != "/data" {
		t.Error("expected /data mount")
	}
}

func TestStatefulSet_WithPorts(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{
		{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565},
	}
	ss := renderer.StatefulSet(gs)
	if len(ss.Spec.Template.Spec.Containers[0].Ports) != 1 {
		t.Error("expected 1 container port")
	}
}

// ---- PVC -------------------------------------------------------------------

func TestPVC_NilWhenNoStorage(t *testing.T) {
	gs := baseGS("srv")
	if renderer.PVC(gs) != nil {
		t.Error("expected nil PVC when no storage requested")
	}
}

func TestPVC_NilWhenExistingClaim(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Storage.ExistingClaim = "my-pvc"
	if renderer.PVC(gs) != nil {
		t.Error("expected nil PVC when existingClaim is set")
	}
}

func TestPVC_CreatesWhenSizeSet(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Storage.Size = resource.MustParse("10Gi")
	pvc := renderer.PVC(gs)
	if pvc == nil {
		t.Fatal("expected non-nil PVC")
	}
	got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if got.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Errorf("storage request = %s", got.String())
	}
}

// ---- Service ---------------------------------------------------------------

func TestService_NilWhenNoPorts(t *testing.T) {
	gs := baseGS("srv")
	if renderer.Service(gs) != nil {
		t.Error("expected nil Service when no ports")
	}
}

func TestService_NodePortType(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{
		{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565},
	}
	svc := renderer.Service(gs)
	if svc == nil {
		t.Fatal("expected non-nil Service")
	}
	if svc.Spec.Type != corev1.ServiceTypeNodePort {
		t.Errorf("service type = %s, want NodePort", svc.Spec.Type)
	}
	if svc.Spec.Ports[0].Port != 25565 {
		t.Errorf("port = %d, want 25565", svc.Spec.Ports[0].Port)
	}
}

func TestService_RespectsFixedNodePort(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{
		{Name: "game", Protocol: corev1.ProtocolTCP, ContainerPort: 25565, NodePort: 30025},
	}
	svc := renderer.Service(gs)
	if svc.Spec.Ports[0].NodePort != 30025 {
		t.Errorf("nodePort = %d, want 30025", svc.Spec.Ports[0].NodePort)
	}
}

// ---- Secret ----------------------------------------------------------------

func TestSecret_EmptyWhenNoOwnerOrPterodactyl(t *testing.T) {
	gs := baseGS("srv")
	sec := renderer.Secret(gs)
	if sec == nil {
		t.Fatal("expected non-nil Secret")
	}
	if len(sec.Data) != 0 {
		t.Errorf("expected empty data, got %v", sec.Data)
	}
}

func TestSecret_ContainsOwnerRefFields(t *testing.T) {
	gs := baseGS("srv")
	gs.Spec.OwnerRef = &v1alpha1.GameServerOwnerRef{
		UserID: "u1", NodeID: "n1", ServerUID: "s1",
	}
	sec := renderer.Secret(gs)
	if string(sec.Data["userId"]) != "u1" {
		t.Error("userId not in secret")
	}
}
