package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/scheduler"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type Server struct {
	router  chi.Router
	storage storage.Storage
}

func NewServer(store storage.Storage, _ *scheduler.Scheduler) *Server {
	s := &Server{router: chi.NewRouter(), storage: store}
	s.router.Use(middleware.Recoverer, middleware.Timeout(10*time.Second), jsonMiddleware)
	s.router.Post("/api/v1/pods", s.createPod)
	s.router.Get("/api/v1/pods/{namespace}", s.listPods)
	s.router.Get("/api/v1/pods/{namespace}/{name}", s.getPod)
	s.router.Delete("/api/v1/pods/{namespace}/{name}", s.deletePod)
	s.router.Get("/api/v1/nodes", s.listNodes)
	s.router.Get("/api/v1/nodes/{name}", s.getNode)
	s.router.Post("/api/v1/deployments", s.createDeployment)
	s.router.Get("/api/v1/deployments/{namespace}", s.listDeployments)
	s.router.Get("/api/v1/deployments/{namespace}/{name}", s.getDeployment)
	s.router.Patch("/api/v1/deployments/{namespace}/{name}/scale", s.scaleDeployment)
	s.router.Delete("/api/v1/deployments/{namespace}/{name}", s.deleteDeployment)
	// Service routing has no data plane yet; never return a fabricated working IP.
	s.router.Post("/api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		respondError(w, 501, "service networking is not implemented")
	})
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, map[string]string{"status": "alive"})
	})
	s.router.Get("/ready", s.ready)
	return s
}
func (s *Server) Handler() http.Handler { return s.router }
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respondError(w, 415, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	if err := d.Decode(value); err != nil {
		respondError(w, 400, "invalid JSON request body (maximum 1 MiB)")
		return false
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		respondError(w, 400, "expected one JSON value")
		return false
	}
	return true
}
func newMeta(meta types.ObjectMeta) types.ObjectMeta {
	if meta.Namespace == "" {
		meta.Namespace = "default"
	}
	return types.ObjectMeta{Name: meta.Name, Namespace: meta.Namespace, Labels: meta.Labels, UID: uuid.NewString()}
}
func (s *Server) createPod(w http.ResponseWriter, r *http.Request) {
	var pod types.Pod
	if !decode(w, r, &pod) {
		return
	}
	pod.Metadata = newMeta(pod.Metadata)
	if err := types.ValidateMeta(pod.Metadata); err != nil {
		respondError(w, 400, err.Error())
		return
	}
	if err := defaultAndValidateSpec(&pod.Spec); err != nil {
		respondError(w, 400, err.Error())
		return
	}
	pod.Status = types.PodStatus{Phase: types.PodPending}
	pod.CreatedAt = time.Now()
	pod.UpdatedAt = pod.CreatedAt
	if err := s.storage.SavePod(r.Context(), &pod); err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 201, pod)
}
func (s *Server) getPod(w http.ResponseWriter, r *http.Request) {
	pod, err := s.storage.GetPod(r.Context(), chi.URLParam(r, "namespace"), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 200, pod)
}
func (s *Server) listPods(w http.ResponseWriter, r *http.Request) {
	pods, err := s.storage.ListPods(r.Context(), chi.URLParam(r, "namespace"))
	if err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 200, pods)
}
func (s *Server) deletePod(w http.ResponseWriter, r *http.Request) {
	pod, err := s.storage.GetPod(r.Context(), chi.URLParam(r, "namespace"), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	types.MarkDeleting(pod)
	if err := s.storage.SavePod(r.Context(), pod); err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 202, map[string]string{"message": "Pod terminating; containers will be removed before its record"})
}
func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.storage.ListNodes(r.Context())
	if err != nil {
		storageError(w, err)
		return
	}
	for _, node := range nodes {
		if !scheduler.Fresh(node) {
			node.Status.Ready = false
			node.Status.Phase = types.NodeNotReady
		}
	}
	respondJSON(w, 200, nodes)
}
func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	node, err := s.storage.GetNode(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	if !scheduler.Fresh(node) {
		node.Status.Ready = false
		node.Status.Phase = types.NodeNotReady
	}
	respondJSON(w, 200, node)
}
func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	var d types.Deployment
	if !decode(w, r, &d) {
		return
	}
	d.Metadata = newMeta(d.Metadata)
	if err := validateDeployment(&d); err != nil {
		respondError(w, 400, err.Error())
		return
	}
	d.Status = types.DeploymentStatus{}
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	if err := s.storage.SaveDeployment(r.Context(), &d); err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 201, d)
}
func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	d, err := s.storage.GetDeployment(r.Context(), chi.URLParam(r, "namespace"), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 200, d)
}
func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	d, err := s.storage.ListDeployments(r.Context(), chi.URLParam(r, "namespace"))
	if err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 200, d)
}
func (s *Server) scaleDeployment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Replicas *int32 `json:"replicas"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Replicas == nil || *body.Replicas < 0 || *body.Replicas > 100 {
		respondError(w, 400, "replicas must be between 0 and 100")
		return
	}
	d, err := s.storage.GetDeployment(r.Context(), chi.URLParam(r, "namespace"), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	if d.Metadata.DeletionTimestamp != nil {
		respondError(w, 409, "deployment is terminating")
		return
	}
	d.Spec.Replicas = *body.Replicas
	d.UpdatedAt = time.Now()
	if err := s.storage.SaveDeployment(r.Context(), d); err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 200, d)
}
func (s *Server) deleteDeployment(w http.ResponseWriter, r *http.Request) {
	d, err := s.storage.GetDeployment(r.Context(), chi.URLParam(r, "namespace"), chi.URLParam(r, "name"))
	if err != nil {
		storageError(w, err)
		return
	}
	now := time.Now()
	d.Metadata.DeletionTimestamp = &now
	if err := s.storage.SaveDeployment(r.Context(), d); err != nil {
		storageError(w, err)
		return
	}
	respondJSON(w, 202, map[string]string{"message": "Deployment terminating; owned pods will be cleaned up"})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if _, err := s.storage.ListNodes(ctx); err != nil {
		respondError(w, 503, "etcd unavailable")
		return
	}
	respondJSON(w, 200, map[string]string{"status": "ready"})
}
func storageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		respondError(w, 404, "resource not found")
	case errors.Is(err, storage.ErrConflict):
		respondError(w, 409, err.Error())
	default:
		log.Printf("storage: %v", err)
		respondError(w, 503, "storage unavailable")
	}
}
func respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("response: %v", err)
	}
}
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
