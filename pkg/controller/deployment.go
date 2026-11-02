package controller

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/scheduler"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"github.com/google/uuid"
)

type DeploymentController struct{ storage storage.Storage }

func NewDeploymentController(store storage.Storage) *DeploymentController {
	return &DeploymentController{storage: store}
}
func (dc *DeploymentController) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := dc.Reconcile(ctx); err != nil && ctx.Err() == nil {
			log.Printf("deployment controller: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (dc *DeploymentController) Reconcile(ctx context.Context) error {
	deployments, err := dc.storage.ListDeployments(ctx, "")
	if err != nil {
		return err
	}
	for _, deployment := range deployments {
		if err := dc.reconcileDeployment(ctx, deployment); err != nil {
			return err
		}
	}
	return nil
}
func (dc *DeploymentController) reconcileDeployment(ctx context.Context, d *types.Deployment) error {
	pods, err := dc.storage.ListPods(ctx, d.Metadata.Namespace)
	if err != nil {
		return err
	}
	nodes, err := dc.storage.ListNodes(ctx)
	if err != nil {
		return err
	}
	fresh := map[string]bool{}
	for _, n := range nodes {
		fresh[n.Metadata.Name] = scheduler.Fresh(n)
	}
	isAvailable := func(p *types.Pod) bool { return types.Ready(p) && fresh[p.Spec.NodeName] }
	owned := make([]*types.Pod, 0)
	for _, pod := range pods {
		if pod.Metadata.OwnerUID == d.Metadata.UID {
			owned = append(owned, pod)
		}
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].Metadata.Name < owned[j].Metadata.Name })
	active := make([]*types.Pod, 0)
	available := int32(0)
	for _, pod := range owned {
		if pod.Metadata.DeletionTimestamp != nil {
			continue
		}
		if d.Metadata.DeletionTimestamp != nil || types.Terminal(pod) {
			types.MarkDeleting(pod)
			if err := dc.storage.SavePod(ctx, pod); err != nil {
				return err
			}
			continue
		}
		active = append(active, pod)
		if isAvailable(pod) {
			available++
		}
	}
	if d.Metadata.DeletionTimestamp != nil {
		if len(owned) == 0 {
			return dc.storage.DeleteDeployment(ctx, d.Metadata.Namespace, d.Metadata.Name)
		}
		return nil
	}
	current := int32(len(active))
	if current > d.Spec.Replicas {
		for _, pod := range active[d.Spec.Replicas:] {
			if isAvailable(pod) {
				available--
			}
			types.MarkDeleting(pod)
			if err := dc.storage.SavePod(ctx, pod); err != nil {
				return err
			}
		}
		current = d.Spec.Replicas
	}
	for i := current; i < d.Spec.Replicas; i++ {
		pod := dc.createPodFromTemplate(d)
		if err := dc.storage.SavePod(ctx, pod); err != nil {
			return err
		}
		current++
	}
	d.Status.Replicas = current
	d.Status.UpdatedReplicas = current
	d.Status.AvailableReplicas = available
	d.UpdatedAt = time.Now()
	return dc.storage.SaveDeployment(ctx, d)
}
func (dc *DeploymentController) createPodFromTemplate(d *types.Deployment) *types.Pod {
	now := time.Now()
	id := uuid.New().String()
	// JSON templates contain maps/slices; storage serialization isolates each copy.
	name := d.Metadata.Name
	if len(name) > 50 {
		name = name[:50]
	}
	return &types.Pod{Metadata: types.ObjectMeta{Name: fmt.Sprintf("%s-%s", name, id[:8]), Namespace: d.Metadata.Namespace, UID: id, OwnerUID: d.Metadata.UID, Labels: d.Spec.Template.Metadata.Labels}, Spec: d.Spec.Template.Spec, Status: types.PodStatus{Phase: types.PodPending}, CreatedAt: now, UpdatedAt: now}
}
