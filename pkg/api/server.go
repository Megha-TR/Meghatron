package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/orchestration-lite/core/pkg/scheduler"
	"github.com/orchestration-lite/core/pkg/storage"
	"github.com/orchestration-lite/core/pkg/types"
)

// Server handles API requests for the orchestration system
type Server struct {
	router    chi.Router
	storage   storage.Storage
	scheduler *scheduler.Scheduler
}

// NewServer creates a new API server
func NewServer(store storage.Storage, sched *scheduler.Scheduler) *Server {
	s := &Server{
		router:    chi.NewRouter(),
		storage:   store,
		scheduler: sched,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(jsonMiddleware)

	// Pod routes
	s.router.Post("/api/v1/pods", s.createPod)
	s.router.Get("/api/v1/pods/{namespace}/{name}", s.getPod)
	s.router.Get("/api/v1/pods/{namespace}", s.listPods)
	s.router.Delete("/api/v1/pods/{namespace}/{name}", s.deletePod)

	// Node routes
	s.router.Post("/api/v1/nodes", s.createNode)
	s.router.Get("/api/v1/nodes/{name}", s.getNode)
	s.router.Get("/api/v1/nodes", s.listNodes)
	s.router.Delete("/api/v1/nodes/{name}", s.deleteNode)

	// Service routes
	s.router.Post("/api/v1/services", s.createService)
	s.router.Get("/api/v1/services/{namespace}/{name}", s.getService)
	s.router.Get("/api/v1/services/{namespace}", s.listServices)
	s.router.Delete("/api/v1/services/{namespace}/{name}", s.deleteService)

	// Deployment routes
	s.router.Post("/api/v1/deployments", s.createDeployment)
	s.router.Get("/api/v1/deployments/{namespace}/{name}", s.getDeployment)
	s.router.Get("/api/v1/deployments/{namespace}", s.listDeployments)
	s.router.Delete("/api/v1/deployments/{namespace}/{name}", s.deleteDeployment)

	// Health check
	s.router.Get("/health", s.health)
}

// Handler functions
func (s *Server) createPod(w http.ResponseWriter, r *http.Request) {
	var pod types.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Populate metadata if not provided
	if pod.Metadata.UID == "" {
		pod.Metadata.UID = uuid.New().String()
	}
	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = "default"
	}
	pod.CreatedAt = time.Now()
	pod.UpdatedAt = time.Now()
	pod.Status.Phase = types.PodPending

	// Schedule the pod
	if err := s.scheduler.Schedule(r.Context(), &pod); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Scheduling failed: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, pod)
}

func (s *Server) getPod(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	pod, err := s.storage.GetPod(r.Context(), namespace, name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Pod not found")
		return
	}

	respondJSON(w, http.StatusOK, pod)
}

func (s *Server) listPods(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")

	pods, err := s.storage.ListPods(r.Context(), namespace)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list pods")
		return
	}

	respondJSON(w, http.StatusOK, pods)
}

func (s *Server) deletePod(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := s.storage.DeletePod(r.Context(), namespace, name); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete pod")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Pod deleted"})
}

// Node handlers
func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {
	var node types.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if node.Metadata.UID == "" {
		node.Metadata.UID = uuid.New().String()
	}
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	node.Status.Phase = types.NodeReady
	node.Status.Ready = true
	node.LastHeartbeat = time.Now()

	if err := s.storage.SaveNode(r.Context(), &node); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create node")
		return
	}

	respondJSON(w, http.StatusCreated, node)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	node, err := s.storage.GetNode(r.Context(), name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Node not found")
		return
	}

	respondJSON(w, http.StatusOK, node)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.storage.ListNodes(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list nodes")
		return
	}

	respondJSON(w, http.StatusOK, nodes)
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.storage.DeleteNode(r.Context(), name); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete node")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Node deleted"})
}

// Service handlers
func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var service types.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if service.Metadata.UID == "" {
		service.Metadata.UID = uuid.New().String()
	}
	if service.Metadata.Namespace == "" {
		service.Metadata.Namespace = "default"
	}
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()
	service.Status.ClusterIP = generateClusterIP()

	if err := s.storage.SaveService(r.Context(), &service); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create service")
		return
	}

	respondJSON(w, http.StatusCreated, service)
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	service, err := s.storage.GetService(r.Context(), namespace, name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Service not found")
		return
	}

	respondJSON(w, http.StatusOK, service)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")

	services, err := s.storage.ListServices(r.Context(), namespace)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list services")
		return
	}

	respondJSON(w, http.StatusOK, services)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := s.storage.DeleteService(r.Context(), namespace, name); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete service")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Service deleted"})
}

// Deployment handlers
func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	var deployment types.Deployment
	if err := json.NewDecoder(r.Body).Decode(&deployment); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if deployment.Metadata.UID == "" {
		deployment.Metadata.UID = uuid.New().String()
	}
	if deployment.Metadata.Namespace == "" {
		deployment.Metadata.Namespace = "default"
	}
	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()

	if err := s.storage.SaveDeployment(r.Context(), &deployment); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create deployment")
		return
	}

	respondJSON(w, http.StatusCreated, deployment)
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	deployment, err := s.storage.GetDeployment(r.Context(), namespace, name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	respondJSON(w, http.StatusOK, deployment)
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")

	deployments, err := s.storage.ListDeployments(r.Context(), namespace)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list deployments")
		return
	}

	respondJSON(w, http.StatusOK, deployments)
}

func (s *Server) deleteDeployment(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := s.storage.DeleteDeployment(r.Context(), namespace, name); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete deployment")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Deployment deleted"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// Helper functions
func (s *Server) Listen(port string) error {
	log.Printf("Starting API server on :%s", port)
	return http.ListenAndServe(":"+port, s.router)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func generateClusterIP() string {
	return fmt.Sprintf("10.0.%d.%d", 1+(uuid.New().ID()%255), (uuid.New().ID()%256))
}
