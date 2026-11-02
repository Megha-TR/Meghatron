package agent

import (
	"context"
	"errors"
	"github.com/Megha-TR/Meghatron/internal/testutil"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"testing"
)

func fixture(t *testing.T) (context.Context, *testutil.Memory, *testutil.Runtime, *Agent, *types.Pod) {
	t.Helper()
	ctx := context.Background()
	store := testutil.NewStore()
	runtime := testutil.NewRuntime()
	a := &Agent{Store: store, Runtime: runtime, NodeName: "worker"}
	p := &types.Pod{Metadata: types.ObjectMeta{Name: "test", Namespace: "team", UID: "uid"}, Spec: types.PodSpec{NodeName: "worker", RestartPolicy: types.RestartPolicyAlways, Containers: []types.Container{{Name: "app", Image: "nginx"}, {Name: "sidecar", Image: "busybox"}}}}
	if err := store.SavePod(ctx, p); err != nil {
		t.Fatal(err)
	}
	return ctx, store, runtime, a, p
}
func TestRestartAdoptionAndContainerExit(t *testing.T) {
	ctx, store, runtime, a, _ := fixture(t)
	for i := 0; i < 2; i++ {
		if err := a.Reconcile(ctx); err != nil {
			t.Fatal(err)
		}
	}
	p, _ := store.GetPod(ctx, "team", "test")
	if !types.Ready(p) {
		t.Fatalf("pod not ready: %+v", p.Status)
	}
	restarted := &Agent{Store: store, Runtime: runtime, NodeName: "worker"}
	if err := restarted.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if runtime.Created != 2 {
		t.Fatal("restart created duplicate containers")
	}
	runtime.Exit("1", 1)
	if err := restarted.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if runtime.Started != 3 {
		t.Fatal("exited container was not restarted")
	}
}
func TestPartialFailureRetriesMissingContainer(t *testing.T) {
	ctx, store, runtime, a, _ := fixture(t)
	runtime.FailImage = "busybox"
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	runtime.FailImage = ""
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	p, _ := store.GetPod(ctx, "team", "test")
	if !types.Ready(p) || runtime.Created != 2 {
		t.Fatalf("partial pod did not recover: %+v, created=%d", p.Status, runtime.Created)
	}
}
func TestDeletionWaitsForDockerCleanup(t *testing.T) {
	ctx, store, runtime, a, _ := fixture(t)
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	p, _ := store.GetPod(ctx, "team", "test")
	types.MarkDeleting(p)
	if err := store.SavePod(ctx, p); err != nil {
		t.Fatal(err)
	}
	runtime.FailRemove = true
	if err := a.Reconcile(ctx); err == nil {
		t.Fatal("expected cleanup failure")
	}
	if _, err := store.GetPod(ctx, "team", "test"); err != nil {
		t.Fatal("record deleted before containers")
	}
	runtime.FailRemove = false
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if len(runtime.Containers) != 0 {
		t.Fatal("containers leaked")
	}
	if _, err := store.GetPod(ctx, "team", "test"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("pod record not removed")
	}
}
func TestNeverPolicyDoesNotRestart(t *testing.T) {
	ctx, store, runtime, a, p := fixture(t)
	p.Spec.RestartPolicy = types.RestartPolicyNever
	if err := store.SavePod(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	runtime.Exit("1", 1)
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if runtime.Started != 2 {
		t.Fatal("Never container restarted")
	}
	p, _ = store.GetPod(ctx, "team", "test")
	if p.Status.Phase != types.PodFailed {
		t.Fatalf("expected Failed, got %s", p.Status.Phase)
	}
}
