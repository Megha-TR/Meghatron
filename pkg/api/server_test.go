package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Megha-TR/Meghatron/internal/testutil"
	"github.com/Megha-TR/Meghatron/pkg/agent"
	"github.com/Megha-TR/Meghatron/pkg/controller"
	"github.com/Megha-TR/Meghatron/pkg/scheduler"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"time"
)

func call(s *Server, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	s.Handler().ServeHTTP(w, r)
	return w
}
func TestAPIValidationAndCreateConflict(t *testing.T) {
	store := testutil.NewStore()
	s := NewServer(store, nil)
	valid := `{"metadata":{"name":"web"},"spec":{"containers":[{"name":"app","image":"nginx"}]}}`
	if w := call(s, "POST", "/api/v1/pods", valid); w.Code != 201 {
		t.Fatalf("%d: %s", w.Code, w.Body)
	}
	if w := call(s, "POST", "/api/v1/pods", valid); w.Code != 409 {
		t.Fatalf("duplicate status=%d", w.Code)
	}
	for _, body := range []string{`{}`, `{"metadata":{"name":"../bad"}}`, strings.Replace(valid, `"nginx"`, `"-oops"`, 1), valid + ` {}`, strings.Replace(valid, `"image":"nginx"`, `"image":"nginx","resources":{"requests":{"cpu":"1000"}}`, 1)} {
		if w := call(s, "POST", "/api/v1/pods", body); w.Code != 400 {
			t.Errorf("accepted invalid body %s: %d", body, w.Code)
		}
	}
	raw := httptest.NewRecorder()
	s.Handler().ServeHTTP(raw, httptest.NewRequest("POST", "/api/v1/pods", strings.NewReader(valid)))
	if raw.Code != 415 {
		t.Fatalf("non-JSON content type accepted: %d", raw.Code)
	}
	w := call(s, "GET", "/api/v1/pods/default/missing", "")
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
	w = call(s, "POST", "/api/v1/services", `{}`)
	if w.Code != 501 {
		t.Fatal("service API pretended networking exists")
	}
	w = call(s, "GET", "/api/v1/pods/empty", "")
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal("empty collection should be []")
	}
}
func TestDeploymentLifecycleThroughAPI(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	s := NewServer(store, nil)
	sched := scheduler.NewScheduler(store)
	dc := controller.NewDeploymentController(store)
	runtime := testutil.NewRuntime()
	worker := &agent.Agent{Store: store, Runtime: runtime, NodeName: "worker"}
	node := &types.Node{Metadata: types.ObjectMeta{Name: "worker"}, Status: types.NodeStatus{Ready: true}, LastHeartbeat: time.Now(), Allocatable: types.ResourceList{"cpu": "2", "memory": "2Gi"}}
	if err := store.SaveNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	body := `{"metadata":{"name":"web","namespace":"team"},"spec":{"replicas":3,"selector":{"app":"web"},"template":{"metadata":{"labels":{"app":"web"}},"spec":{"containers":[{"name":"app","image":"nginx"}]}}}}`
	w := call(s, "POST", "/api/v1/deployments", body)
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	step := func() {
		t.Helper()
		for _, run := range []func(context.Context) error{dc.Reconcile, sched.Reconcile, worker.Reconcile, worker.Reconcile, dc.Reconcile} {
			if err := run(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	step()
	d, _ := store.GetDeployment(ctx, "team", "web")
	if d.Status.AvailableReplicas != 3 || runtime.Created != 3 {
		t.Fatalf("deployment failed: %+v", d.Status)
	}
	w = call(s, "PATCH", "/api/v1/deployments/team/web/scale", `{"replicas":1}`)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	step()
	if len(runtime.Containers) != 1 {
		t.Fatalf("scale down leaked containers: %d", len(runtime.Containers))
	}
	w = call(s, "DELETE", "/api/v1/deployments/team/web", "")
	if w.Code != 202 {
		t.Fatal(w.Code)
	}
	step()
	if len(runtime.Containers) != 0 {
		t.Fatal("deployment delete leaked containers")
	}
	if _, err := store.GetDeployment(ctx, "team", "web"); err != storage.ErrNotFound {
		t.Fatalf("deployment not removed: %v", err)
	}
}
func TestOptimisticWritesCannotResurrectDeletion(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	s := NewServer(store, nil)
	w := call(s, "POST", "/api/v1/pods", `{"metadata":{"name":"web"},"spec":{"containers":[{"name":"app","image":"nginx"}]}}`)
	var stale types.Pod
	if err := json.Unmarshal(w.Body.Bytes(), &stale); err != nil {
		t.Fatal(err)
	}
	w = call(s, "DELETE", "/api/v1/pods/default/web", "")
	if w.Code != 202 {
		t.Fatal(w.Code)
	}
	if err := store.SavePod(ctx, &stale); err != storage.ErrConflict {
		t.Fatal("stale status overwrote deletion")
	}
}
