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

package helpers_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
	"github.com/Vladislavvk1337/ptero-wings-operator/controllers/helpers"
)

func newGS(name string) *v1alpha1.GameServer {
	return &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}
}

// ---- naming ----------------------------------------------------------------

func TestNaming(t *testing.T) {
	gs := newGS("srv")
	if got := helpers.StatefulSetName(gs); got != "srv" {
		t.Errorf("StatefulSetName = %q, want %q", got, "srv")
	}
	if got := helpers.ServiceName(gs); got != "srv" {
		t.Errorf("ServiceName = %q, want %q", got, "srv")
	}
	if got := helpers.PVCName(gs); got != "srv-data" {
		t.Errorf("PVCName = %q, want %q", got, "srv-data")
	}
	if got := helpers.SecretName(gs); got != "srv-config" {
		t.Errorf("SecretName = %q, want %q", got, "srv-config")
	}
}

// ---- selectors -------------------------------------------------------------

func TestLabelsForGameServer_ContainsInstance(t *testing.T) {
	gs := newGS("my-server")
	gs.Spec.Game.Type = "csgo"
	labels := helpers.LabelsForGameServer(gs)
	if labels["app.kubernetes.io/instance"] != "my-server" {
		t.Error("missing instance label")
	}
	if labels["gameserver.pterodactyl.io/game"] != "csgo" {
		t.Error("missing game label")
	}
}

// ---- defaults (MergeWithClass) ---------------------------------------------

func TestMergeWithClass_NilClass(t *testing.T) {
	gs := newGS("gs")
	gs.Spec.Game.Image = "myimage:1"
	merged := helpers.MergeWithClass(gs, nil)
	if merged.Spec.Game.Image != "myimage:1" {
		t.Error("image should be unchanged when class is nil")
	}
}

func TestMergeWithClass_FillsImageFromClass(t *testing.T) {
	gs := newGS("gs")
	gs.Spec.Game.Type = "mc"

	class := &v1alpha1.GameServerClass{
		Spec: v1alpha1.GameServerClassSpec{
			GameType:     "mc",
			DefaultImage: "itzg/minecraft-server:latest",
		},
	}
	merged := helpers.MergeWithClass(gs, class)
	if merged.Spec.Game.Image != "itzg/minecraft-server:latest" {
		t.Errorf("expected image from class, got %q", merged.Spec.Game.Image)
	}
}

func TestMergeWithClass_DoesNotOverrideExistingImage(t *testing.T) {
	gs := newGS("gs")
	gs.Spec.Game.Image = "custom:v2"

	class := &v1alpha1.GameServerClass{
		Spec: v1alpha1.GameServerClassSpec{DefaultImage: "default:v1"},
	}
	merged := helpers.MergeWithClass(gs, class)
	if merged.Spec.Game.Image != "custom:v2" {
		t.Error("class default must not override explicit image")
	}
}

func TestMergeWithClass_FillsStorageFromClass(t *testing.T) {
	gs := newGS("gs")
	gs.Spec.Game.Image = "img:1"

	class := &v1alpha1.GameServerClass{
		Spec: v1alpha1.GameServerClassSpec{
			DefaultStorage: v1alpha1.StorageSpec{
				Size:       resource.MustParse("10Gi"),
				AccessMode: corev1.ReadWriteOnce,
				MountPath:  "/data",
			},
		},
	}
	merged := helpers.MergeWithClass(gs, class)
	if merged.Spec.Storage.Size.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Error("storage size not filled from class")
	}
	if merged.Spec.Storage.MountPath != "/data" {
		t.Error("mount path not filled from class")
	}
}

// ---- conditions ------------------------------------------------------------

func TestSetCondition_SetsAndUpdates(t *testing.T) {
	gs := newGS("gs")
	helpers.SetReady(gs, false, "Pending", "not ready")
	if len(gs.Status.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(gs.Status.Conditions))
	}
	cond := gs.Status.Conditions[0]
	if cond.Type != v1alpha1.ConditionReady {
		t.Errorf("wrong type: %s", cond.Type)
	}
	if cond.Status != metav1.ConditionFalse {
		t.Error("expected ConditionFalse")
	}

	// Flip to ready
	helpers.SetReady(gs, true, "Running", "all good")
	if len(gs.Status.Conditions) != 1 {
		t.Error("should still have exactly 1 Ready condition")
	}
	if gs.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Error("expected ConditionTrue after flip")
	}
}

func TestSetReconciling(t *testing.T) {
	gs := newGS("gs")
	helpers.SetReconciling(gs, true, "Busy", "")
	helpers.SetReconciling(gs, false, "Done", "")
	found := false
	for _, c := range gs.Status.Conditions {
		if c.Type == v1alpha1.ConditionReconciling {
			found = true
			if c.Status != metav1.ConditionFalse {
				t.Error("expected ConditionFalse after done")
			}
		}
	}
	if !found {
		t.Error("Reconciling condition not found")
	}
}

// ---- ports helpers ---------------------------------------------------------

func TestDefaultPortProtocol(t *testing.T) {
	if helpers.DefaultPortProtocol("") != corev1.ProtocolTCP {
		t.Error("empty protocol should default to TCP")
	}
	if helpers.DefaultPortProtocol(corev1.ProtocolUDP) != corev1.ProtocolUDP {
		t.Error("UDP should stay UDP")
	}
}

func TestNeedsStorage(t *testing.T) {
	gs := newGS("gs")
	if !helpers.NeedsStorage(gs) {
		t.Error("every GameServer should require a dedicated PVC")
	}
}

func TestNeedsService(t *testing.T) {
	gs := newGS("gs")
	if helpers.NeedsService(gs) {
		t.Error("no ports should not need a Service")
	}
	gs.Spec.Network.Ports = []v1alpha1.PortSpec{{Name: "game", ContainerPort: 25565}}
	if !helpers.NeedsService(gs) {
		t.Error("with ports should need a Service")
	}
}
