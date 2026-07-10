package ports

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

func TestClusterAllocatorAllocatesContiguousRange(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	client := ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects(&v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "existing", Namespace: "default"},
		Spec:       v1alpha1.GameServerSpec{Network: v1alpha1.NetworkSpec{NodePortAllocation: &v1alpha1.NodePortAllocationSpec{StartPort: 30005, Count: 2, Protocol: corev1.ProtocolTCP}}},
	}).Build()
	allocator := NewClusterAllocator(client, Config{MinPort: 30000, MaxPort: 30020})
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "next", Namespace: "default"},
		Spec:       v1alpha1.GameServerSpec{Network: v1alpha1.NetworkSpec{NodePortAllocation: &v1alpha1.NodePortAllocationSpec{Count: 2, Protocol: corev1.ProtocolTCP}}},
	}
	ports, err := allocator.Allocate(context.Background(), gs)
	if err != nil {
		t.Fatalf("Allocate returned error: %v", err)
	}
	if len(ports) != 3 {
		t.Fatalf("len(ports) = %d, want 3", len(ports))
	}
	if ports[0].NodePort != 30000 || ports[2].NodePort != 30002 {
		t.Fatalf("nodeports = %v, want 30000..30002", ports)
	}
}
