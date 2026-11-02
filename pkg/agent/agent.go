// Package agent reconciles persisted pod intent with Docker state, including after restart.
package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/runtime"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
)

type Runtime interface {
	ListContainers(context.Context, string) ([]runtime.ManagedContainer, error)
	CreateContainer(context.Context, *types.Pod, types.Container) (string, error)
	StartContainer(context.Context, string) error
	RemoveContainer(context.Context, string) error
}
type Agent struct {
	Store    storage.Storage
	Runtime  Runtime
	NodeName string
}

func (a *Agent) Reconcile(ctx context.Context) error {
	pods, err := a.Store.ListPods(ctx, "")
	if err != nil {
		return err
	}
	containers, err := a.Runtime.ListContainers(ctx, a.NodeName)
	if err != nil {
		return err
	}
	desired := map[string]*types.Pod{}
	for _, pod := range pods {
		if pod.Spec.NodeName == a.NodeName && pod.Metadata.DeletionTimestamp == nil {
			desired[pod.Metadata.UID] = pod
		}
	}
	// Only containers carrying this node's management labels are ever removed.
	for _, c := range containers {
		if _, ok := desired[c.PodUID]; !ok {
			if err := a.Runtime.RemoveContainer(ctx, c.ContainerID); err != nil {
				return err
			}
		}
	}
	var failures []error
	for _, pod := range pods {
		if pod.Spec.NodeName != a.NodeName {
			continue
		}
		if pod.Metadata.DeletionTimestamp != nil {
			if err := a.Store.DeletePod(ctx, pod.Metadata.Namespace, pod.Metadata.Name); err != nil && !errors.Is(err, storage.ErrNotFound) {
				failures = append(failures, err)
			}
			continue
		}
		if err := a.reconcilePod(ctx, pod, containers); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
func (a *Agent) reconcilePod(ctx context.Context, pod *types.Pod, containers []runtime.ManagedContainer) error {
	pod.Status.ContainerStatuses = nil
	pod.Status.Phase = types.PodRunning
	pod.Status.Message = ""
	failed, completed, pending := false, 0, false
	for _, spec := range pod.Spec.Containers {
		var existing *runtime.ManagedContainer
		for i := range containers {
			if containers[i].PodUID == pod.Metadata.UID && containers[i].ContainerName == spec.Name {
				existing = &containers[i]
				break
			}
		}
		status := types.ContainerStatus{Name: spec.Name, State: "created"}
		if existing != nil {
			status = existing.ContainerStatus
			status.Name = spec.Name
		}
		if !status.Running {
			policy := spec.RestartPolicy
			if policy == "" {
				policy = pod.Spec.RestartPolicy
			}
			if existing != nil && existing.State == "exited" && (policy == types.RestartPolicyNever || (policy == types.RestartPolicyOnFailure && existing.ExitCode == 0)) {
				if existing.ExitCode == 0 {
					completed++
				} else {
					failed = true
				}
			} else {
				var err error
				if existing == nil {
					status.ContainerID, err = a.Runtime.CreateContainer(ctx, pod, spec)
				}
				if err == nil {
					err = a.Runtime.StartContainer(ctx, status.ContainerID)
				}
				if err != nil {
					pending = true
					pod.Status.Message = fmt.Sprintf("container %s: %v", spec.Name, err)
					status.State = "error"
				} else {
					// Readiness means observed Docker Running; the next reconciliation confirms it.
					pending = true
					status.State = "starting"
				}
			}
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, status)
	}
	switch {
	case failed:
		pod.Status.Phase = types.PodFailed
	case completed == len(pod.Spec.Containers):
		pod.Status.Phase = types.PodSucceeded
	case pending:
		pod.Status.Phase = types.PodPending
	default:
		pod.Status.Phase = types.PodRunning
	}
	pod.UpdatedAt = time.Now()
	return a.Store.SavePod(ctx, pod)
}
