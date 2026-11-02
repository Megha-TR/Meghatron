// Package testutil supplies an isolated optimistic store for behavioral tests.
package testutil

import (
	"context"
	"encoding/json"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"strings"
	"sync"
)

type Memory struct {
	mu      sync.Mutex
	values  map[string][]byte
	version int64
}

func NewStore() *Memory        { return &Memory{values: map[string][]byte{}} }
func (m *Memory) Close() error { return nil }
func (m *Memory) save(key string, value any, meta *types.ObjectMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var previous struct{ Metadata types.ObjectMeta }
	if data, ok := m.values[key]; ok {
		if err := json.Unmarshal(data, &previous); err != nil {
			return err
		}
	}
	if previous.Metadata.ResourceVersion != meta.ResourceVersion {
		return storage.ErrConflict
	}
	m.version++
	meta.ResourceVersion = m.version
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.values[key] = data
	return nil
}
func (m *Memory) remove(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.values[key]; !ok {
		return storage.ErrNotFound
	}
	delete(m.values, key)
	return nil
}
func (m *Memory) SavePod(ctx context.Context, v *types.Pod) error {
	return m.save("pods/"+v.Metadata.Namespace+"/"+v.Metadata.Name, v, &v.Metadata)
}
func (m *Memory) GetPod(ctx context.Context, namespace, name string) (*types.Pod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.values["pods/"+namespace+"/"+name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	var v types.Pod
	err := json.Unmarshal(data, &v)
	return &v, err
}
func (m *Memory) DeletePod(ctx context.Context, namespace, name string) error {
	return m.remove("pods/" + namespace + "/" + name)
}
func (m *Memory) ListPods(ctx context.Context, namespace string) ([]*types.Pod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := "pods/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	result := make([]*types.Pod, 0)
	for key, data := range m.values {
		if strings.HasPrefix(key, prefix) {
			var v types.Pod
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, err
			}
			result = append(result, &v)
		}
	}
	return result, nil
}
func (m *Memory) SaveNode(ctx context.Context, v *types.Node) error {
	return m.save("nodes/"+v.Metadata.Name, v, &v.Metadata)
}
func (m *Memory) GetNode(ctx context.Context, name string) (*types.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.values["nodes/"+name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	var v types.Node
	err := json.Unmarshal(data, &v)
	return &v, err
}
func (m *Memory) DeleteNode(ctx context.Context, name string) error { return m.remove("nodes/" + name) }
func (m *Memory) ListNodes(ctx context.Context) ([]*types.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := "nodes/"
	result := make([]*types.Node, 0)
	for key, data := range m.values {
		if strings.HasPrefix(key, prefix) {
			var v types.Node
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, err
			}
			result = append(result, &v)
		}
	}
	return result, nil
}
func (m *Memory) SaveService(ctx context.Context, v *types.Service) error {
	return m.save("services/"+v.Metadata.Namespace+"/"+v.Metadata.Name, v, &v.Metadata)
}
func (m *Memory) GetService(ctx context.Context, namespace, name string) (*types.Service, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.values["services/"+namespace+"/"+name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	var v types.Service
	err := json.Unmarshal(data, &v)
	return &v, err
}
func (m *Memory) DeleteService(ctx context.Context, namespace, name string) error {
	return m.remove("services/" + namespace + "/" + name)
}
func (m *Memory) ListServices(ctx context.Context, namespace string) ([]*types.Service, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := "services/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	result := make([]*types.Service, 0)
	for key, data := range m.values {
		if strings.HasPrefix(key, prefix) {
			var v types.Service
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, err
			}
			result = append(result, &v)
		}
	}
	return result, nil
}
func (m *Memory) SaveDeployment(ctx context.Context, v *types.Deployment) error {
	return m.save("deployments/"+v.Metadata.Namespace+"/"+v.Metadata.Name, v, &v.Metadata)
}
func (m *Memory) GetDeployment(ctx context.Context, namespace, name string) (*types.Deployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.values["deployments/"+namespace+"/"+name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	var v types.Deployment
	err := json.Unmarshal(data, &v)
	return &v, err
}
func (m *Memory) DeleteDeployment(ctx context.Context, namespace, name string) error {
	return m.remove("deployments/" + namespace + "/" + name)
}
func (m *Memory) ListDeployments(ctx context.Context, namespace string) ([]*types.Deployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := "deployments/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	result := make([]*types.Deployment, 0)
	for key, data := range m.values {
		if strings.HasPrefix(key, prefix) {
			var v types.Deployment
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, err
			}
			result = append(result, &v)
		}
	}
	return result, nil
}
