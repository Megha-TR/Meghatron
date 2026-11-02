package scheduler

import (
	"context"
	"github.com/Megha-TR/Meghatron/internal/testutil"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"testing"
	"time"
)

func TestCapacityReservationAndRetry(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	s := NewScheduler(store)
	n := &types.Node{Metadata: types.ObjectMeta{Name: "worker"}, Status: types.NodeStatus{Ready: true}, LastHeartbeat: time.Now(), Allocatable: types.ResourceList{"cpu": "1", "memory": "1Gi"}}
	if err := store.SaveNode(ctx, n); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		p := &types.Pod{Metadata: types.ObjectMeta{Name: name, UID: name, Namespace: "team"}, Spec: types.PodSpec{Containers: []types.Container{{Name: "app", Resources: types.ResourceRequirements{Requests: types.ResourceList{"cpu": "750m", "memory": "64Mi"}}}}}}
		if err := store.SavePod(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	pods, _ := store.ListPods(ctx, "team")
	var assigned, waiting *types.Pod
	for _, p := range pods {
		if p.Spec.NodeName != "" {
			assigned = p
		} else {
			waiting = p
		}
	}
	if assigned == nil || waiting == nil {
		t.Fatalf("expected one assignment and one pending pod: %+v", pods)
	}
	if err := store.DeletePod(ctx, "team", assigned.Metadata.Name); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	p, _ := store.GetPod(ctx, "team", waiting.Metadata.Name)
	if p.Spec.NodeName != "worker" {
		t.Fatal("pending pod did not retry after capacity was freed")
	}
}
func TestExpiredHeartbeatBlocksScheduling(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	s := NewScheduler(store)
	n := &types.Node{Metadata: types.ObjectMeta{Name: "dead"}, Status: types.NodeStatus{Ready: true}, LastHeartbeat: time.Now().Add(-time.Minute), Allocatable: types.ResourceList{"cpu": "8", "memory": "16Gi"}}
	if err := store.SaveNode(ctx, n); err != nil {
		t.Fatal(err)
	}
	p := &types.Pod{Metadata: types.ObjectMeta{Name: "test", Namespace: "default", UID: "test"}}
	if err := store.SavePod(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	node, _ := store.GetNode(ctx, "dead")
	if node.Status.Ready {
		t.Fatal("stale node remained ready")
	}
	pod, _ := store.GetPod(ctx, "default", "test")
	if pod.Spec.NodeName != "" {
		t.Fatal("scheduled onto dead node")
	}
}
func TestHostPortConflicts(t *testing.T) {
	pod := func(host int32) *types.Pod {
		return &types.Pod{Spec: types.PodSpec{Containers: []types.Container{{Ports: []types.ContainerPort{{ContainerPort: 80, HostPort: host}}}}}}
	}
	if hostPortsConflict(pod(0), pod(0)) {
		t.Fatal("container ports should not reserve host ports")
	}
	if !hostPortsConflict(pod(8080), pod(8080)) {
		t.Fatal("duplicate host port accepted")
	}
}
