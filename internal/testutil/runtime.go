package testutil

import (
	"context"
	"fmt"

	"github.com/Megha-TR/Meghatron/pkg/runtime"
	"github.com/Megha-TR/Meghatron/pkg/types"
)

type Runtime struct {
	Containers                map[string]runtime.ManagedContainer
	Created, Started, Removed int
	FailImage                 string
	FailRemove                bool
}

func NewRuntime() *Runtime { return &Runtime{Containers: map[string]runtime.ManagedContainer{}} }
func (r *Runtime) ListContainers(_ context.Context, _ string) ([]runtime.ManagedContainer, error) {
	result := []runtime.ManagedContainer{}
	for _, c := range r.Containers {
		result = append(result, c)
	}
	return result, nil
}
func (r *Runtime) CreateContainer(_ context.Context, pod *types.Pod, c types.Container) (string, error) {
	if c.Image == r.FailImage {
		return "", fmt.Errorf("image unavailable")
	}
	r.Created++
	id := fmt.Sprint(r.Created)
	r.Containers[id] = runtime.ManagedContainer{PodUID: pod.Metadata.UID, ContainerName: c.Name, ContainerStatus: types.ContainerStatus{ContainerID: id, State: "created"}}
	return id, nil
}
func (r *Runtime) StartContainer(_ context.Context, id string) error {
	c, ok := r.Containers[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	r.Started++
	c.Ready = true
	c.Running = true
	c.State = "running"
	r.Containers[id] = c
	return nil
}
func (r *Runtime) RemoveContainer(_ context.Context, id string) error {
	if r.FailRemove {
		return fmt.Errorf("daemon unavailable")
	}
	delete(r.Containers, id)
	r.Removed++
	return nil
}
func (r *Runtime) Exit(id string, code int) {
	c := r.Containers[id]
	c.Running = false
	c.Ready = false
	c.State = "exited"
	c.ExitCode = code
	r.Containers[id] = c
}
