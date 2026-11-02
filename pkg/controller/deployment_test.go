package controller

import (
	"context"
	"github.com/Megha-TR/Meghatron/internal/testutil"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"testing"
	"time"
)

func TestReplicaOwnershipAvailabilityAndScaleDown(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	dc := NewDeploymentController(store)
	d := &types.Deployment{Metadata: types.ObjectMeta{Name: "web", Namespace: "team", UID: "dep"}, Spec: types.DeploymentSpec{Replicas: 3, Selector: map[string]string{"app": "web"}, Template: types.PodTemplateSpec{Metadata: types.ObjectMeta{Labels: map[string]string{"app": "web"}}, Spec: types.PodSpec{Containers: []types.Container{{Name: "app"}}}}}}
	if err := store.SaveDeployment(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveNode(ctx, &types.Node{Metadata: types.ObjectMeta{Name: "worker"}, Status: types.NodeStatus{Ready: true}, LastHeartbeat: time.Now()}); err != nil {
		t.Fatal(err)
	}
	unrelated := &types.Pod{Metadata: types.ObjectMeta{Name: "unrelated", Namespace: "team", UID: "other", Labels: map[string]string{"app": "web"}}}
	if err := store.SavePod(ctx, unrelated); err != nil {
		t.Fatal(err)
	}
	if err := dc.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	current, _ := store.GetDeployment(ctx, "team", "web")
	if current.Status.Replicas != 3 || current.Status.AvailableReplicas != 0 {
		t.Fatalf("pending replicas reported available: %+v", current.Status)
	}
	pods, _ := store.ListPods(ctx, "team")
	if len(pods) != 4 {
		t.Fatal("deployment adopted unrelated matching labels")
	}
	for _, p := range pods {
		if p.Metadata.OwnerUID == "dep" {
			p.Spec.NodeName = "worker"
			p.Status = types.PodStatus{Phase: types.PodRunning, ContainerStatuses: []types.ContainerStatus{{Ready: true}}}
			if err := store.SavePod(ctx, p); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := dc.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	current, _ = store.GetDeployment(ctx, "team", "web")
	if current.Status.AvailableReplicas != 3 {
		t.Fatal("ready replicas not counted")
	}
	current.Spec.Replicas = 1
	if err := store.SaveDeployment(ctx, current); err != nil {
		t.Fatal(err)
	}
	if err := dc.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	pods, _ = store.ListPods(ctx, "team")
	terminating := 0
	for _, p := range pods {
		if p.Metadata.DeletionTimestamp != nil {
			terminating++
		}
	}
	if terminating != 2 {
		t.Fatalf("expected 2 terminating pods, got %d", terminating)
	}
	current, _ = store.GetDeployment(ctx, "team", "web")
	if current.Status.AvailableReplicas != 1 {
		t.Fatal("terminating replicas counted available")
	}
}
func TestFailedReplicaIsReplaced(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	dc := NewDeploymentController(store)
	d := &types.Deployment{Metadata: types.ObjectMeta{Name: "web", Namespace: "default", UID: "dep"}, Spec: types.DeploymentSpec{Replicas: 1}}
	if err := store.SaveDeployment(ctx, d); err != nil {
		t.Fatal(err)
	}
	failed := &types.Pod{Metadata: types.ObjectMeta{Name: "failed", Namespace: "default", UID: "failed", OwnerUID: "dep"}, Status: types.PodStatus{Phase: types.PodFailed}}
	if err := store.SavePod(ctx, failed); err != nil {
		t.Fatal(err)
	}
	if err := dc.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	pods, _ := store.ListPods(ctx, "default")
	if len(pods) != 2 {
		t.Fatal("failed replica blocked replacement")
	}
	p, _ := store.GetPod(ctx, "default", "failed")
	if p.Metadata.DeletionTimestamp == nil {
		t.Fatal("failed replica not marked for cleanup")
	}
}
