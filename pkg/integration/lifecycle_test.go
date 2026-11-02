package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/agent"
	"github.com/Megha-TR/Meghatron/pkg/api"
	"github.com/Megha-TR/Meghatron/pkg/controller"
	"github.com/Megha-TR/Meghatron/pkg/runtime"
	"github.com/Megha-TR/Meghatron/pkg/scheduler"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"github.com/google/uuid"
)

// Run against an isolated etcd with scripts/test-e2e.sh. Every Docker container
// is scoped by a unique node label; the test never touches unrelated containers.
func TestRealDockerDeploymentLifecycle(t *testing.T) {
	endpoint := os.Getenv("MEGHATRON_TEST_ETCD")
	if endpoint == "" {
		t.Skip("set MEGHATRON_TEST_ETCD or run scripts/test-e2e.sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	store, err := storage.NewEtcdStorage([]string{endpoint})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cr, err := runtime.NewContainerRuntime()
	if err != nil {
		t.Fatal(err)
	}
	name := "e2e-" + uuid.NewString()[:8]
	ns := name
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cs, err := cr.ListContainers(cleanup, name)
		if err != nil {
			t.Error(err)
		}
		for _, c := range cs {
			if err := cr.RemoveContainer(cleanup, c.ContainerID); err != nil {
				t.Error(err)
			}
		}
	}()
	node := &types.Node{Metadata: types.ObjectMeta{Name: name, UID: uuid.NewString()}, Status: types.NodeStatus{Ready: true}, Allocatable: types.ResourceList{"cpu": "2", "memory": "1Gi"}, LastHeartbeat: time.Now()}
	if err := store.SaveNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	sched := scheduler.NewScheduler(store)
	dc := controller.NewDeploymentController(store)
	a := &agent.Agent{Store: store, Runtime: cr, NodeName: name}
	server := api.NewServer(store, sched)
	call := func(method, path, body string, intended int) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		server.Handler().ServeHTTP(w, r)
		if w.Code != intended {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
	}
	wait := func(check func() bool) {
		t.Helper()
		for {
			current, err := store.GetNode(ctx, name)
			if err != nil {
				t.Fatal(err)
			}
			current.LastHeartbeat = time.Now()
			current.Status.Ready = true
			if err := store.SaveNode(ctx, current); err != nil {
				t.Fatal(err)
			}
			for _, run := range []func(context.Context) error{dc.Reconcile, sched.Reconcile, a.Reconcile} {
				if err := run(ctx); err != nil {
					t.Fatal(err)
				}
			}
			if check() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("lifecycle did not converge: ", ctx.Err())
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	body := fmt.Sprintf(`{"metadata":{"name":"web","namespace":%q},"spec":{"replicas":3,"selector":{"app":"web"},"template":{"metadata":{"labels":{"app":"web"}},"spec":{"containers":[{"name":"app","image":"nginx:1.28-alpine","ports":[{"containerPort":80}]}]}}}}`, ns)
	call("POST", "/api/v1/deployments", body, 201)
	call("POST", "/api/v1/deployments", body, 409)
	wait(func() bool {
		d, err := store.GetDeployment(ctx, ns, "web")
		return err == nil && d.Status.AvailableReplicas == 3
	})
	t.Log("three replicas are running and available with identical container ports")
	// A new Agent has no process-local container registry to restore.
	a = &agent.Agent{Store: store, Runtime: cr, NodeName: name}
	if err := a.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	cs, err := cr.ListContainers(ctx, name)
	if err != nil || len(cs) != 3 {
		t.Fatalf("restart adoption: containers=%d err=%v", len(cs), err)
	}
	if err := cr.StopContainer(ctx, cs[0].ContainerID); err != nil {
		t.Fatal(err)
	}
	wait(func() bool {
		cs, err := cr.ListContainers(ctx, name)
		if err != nil {
			return false
		}
		for _, c := range cs {
			if !c.Running {
				return false
			}
		}
		return len(cs) == 3
	})
	t.Log("agent restart adopted containers; stopped container recovered")
	stale, err := store.GetDeployment(ctx, ns, "web")
	if err != nil {
		t.Fatal(err)
	}
	call("PATCH", "/api/v1/deployments/"+ns+"/web/scale", `{"replicas":1}`, 200)
	if err := store.SaveDeployment(ctx, stale); err != storage.ErrConflict {
		t.Fatalf("etcd accepted stale update: %v", err)
	}
	wait(func() bool {
		cs, err := cr.ListContainers(ctx, name)
		d, e := store.GetDeployment(ctx, ns, "web")
		return err == nil && e == nil && len(cs) == 1 && d.Status.AvailableReplicas == 1
	})
	t.Log("scale down removed two Docker containers")
	call("DELETE", "/api/v1/deployments/"+ns+"/web", "", 202)
	wait(func() bool { _, err := store.GetDeployment(ctx, ns, "web"); return err == storage.ErrNotFound })
	cs, err = cr.ListContainers(ctx, name)
	if err != nil || len(cs) != 0 {
		t.Fatalf("cleanup leaked containers: %d %v", len(cs), err)
	}
	t.Log("deployment deletion removed all owned pods and Docker containers")
}
