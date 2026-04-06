package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/orchestration-lite/core/pkg/storage"
	"github.com/orchestration-lite/core/pkg/types"
)

// DeploymentController reconciles deployment state
type DeploymentController struct {
	storage storage.Storage
	ticker  *time.Ticker
}

// NewDeploymentController creates a new deployment controller
func NewDeploymentController(store storage.Storage) *DeploymentController {
	return &DeploymentController{
		storage: store,
		ticker:  time.NewTicker(5 * time.Second),
	}
}

// Run starts the deployment controller reconciliation loop
func (dc *DeploymentController) Run(ctx context.Context) {
	log.Println("Starting deployment controller")

	go func() {
		for {
			select {
			case <-ctx.Done():
				dc.ticker.Stop()
				return
			case <-dc.ticker.C:
				dc.reconcile(ctx)
			}
		}
	}()
}

// reconcile reconciles deployment state
func (dc *DeploymentController) reconcile(ctx context.Context) {
	deployments, err := dc.storage.ListDeployments(ctx, "default")
	if err != nil {
		log.Printf("Error listing deployments: %v", err)
		return
	}

	for _, deployment := range deployments {
		dc.reconcileDeployment(ctx, deployment)
	}
}

// reconcileDeployment reconciles a single deployment
func (dc *DeploymentController) reconcileDeployment(ctx context.Context, deployment *types.Deployment) {
	// Get pods for this deployment
	pods, err := dc.storage.ListPods(ctx, deployment.Metadata.Namespace)
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		return
	}

	// Filter pods by selector
	var matchingPods []*types.Pod
	for _, pod := range pods {
		if dc.selectorMatches(deployment.Spec.Selector, pod.Metadata.Labels) {
			matchingPods = append(matchingPods, pod)
		}
	}

	currentReplicas := int32(len(matchingPods))
	desiredReplicas := deployment.Spec.Replicas

	log.Printf("Deployment %s/%s: current replicas=%d, desired=%d",
		deployment.Metadata.Namespace, deployment.Metadata.Name,
		currentReplicas, desiredReplicas)

	// Scale up: create new pods
	if currentReplicas < desiredReplicas {
		needed := desiredReplicas - currentReplicas
		for i := int32(0); i < needed; i++ {
			pod := dc.createPodFromTemplate(deployment)
			if err := dc.storage.SavePod(ctx, pod); err != nil {
				log.Printf("Error creating pod: %v", err)
			}
		}
	}

	// Update deployment status
	deployment.Status.Replicas = currentReplicas
	deployment.Status.UpdatedReplicas = currentReplicas
	deployment.Status.AvailableReplicas = currentReplicas

	if err := dc.storage.SaveDeployment(ctx, deployment); err != nil {
		log.Printf("Error updating deployment status: %v", err)
	}
}

// selectorMatches checks if labels match the selector
func (dc *DeploymentController) selectorMatches(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// createPodFromTemplate creates a pod from the deployment template
func (dc *DeploymentController) createPodFromTemplate(deployment *types.Deployment) *types.Pod {
	pod := &types.Pod{
		Metadata: types.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", deployment.Metadata.Name, time.Now().UnixNano()),
			Namespace: deployment.Metadata.Namespace,
			Labels:    deployment.Spec.Template.Metadata.Labels,
		},
		Spec:      deployment.Spec.Template.Spec,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	pod.Status.Phase = types.PodPending
	return pod
}

// Stop stops the deployment controller
func (dc *DeploymentController) Stop() {
	dc.ticker.Stop()
}
