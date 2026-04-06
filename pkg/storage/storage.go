package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/orchestration-lite/core/pkg/types"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Storage interface defines methods for storing and retrieving cluster resources
type Storage interface {
	SavePod(ctx context.Context, pod *types.Pod) error
	GetPod(ctx context.Context, namespace, name string) (*types.Pod, error)
	ListPods(ctx context.Context, namespace string) ([]*types.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error

	SaveNode(ctx context.Context, node *types.Node) error
	GetNode(ctx context.Context, name string) (*types.Node, error)
	ListNodes(ctx context.Context) ([]*types.Node, error)
	DeleteNode(ctx context.Context, name string) error

	SaveService(ctx context.Context, service *types.Service) error
	GetService(ctx context.Context, namespace, name string) (*types.Service, error)
	ListServices(ctx context.Context, namespace string) ([]*types.Service, error)
	DeleteService(ctx context.Context, namespace, name string) error

	SaveDeployment(ctx context.Context, deployment *types.Deployment) error
	GetDeployment(ctx context.Context, namespace, name string) (*types.Deployment, error)
	ListDeployments(ctx context.Context, namespace string) ([]*types.Deployment, error)
	DeleteDeployment(ctx context.Context, namespace, name string) error

	Close() error
}

// EtcdStorage implements Storage using etcd as the backend
type EtcdStorage struct {
	client *clientv3.Client
}

// NewEtcdStorage creates a new EtcdStorage instance
func NewEtcdStorage(endpoints []string) (*EtcdStorage, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdStorage{client: client}, nil
}

// Helper functions
func (es *EtcdStorage) podKey(namespace, name string) string {
	return fmt.Sprintf("/orchestration/pods/%s/%s", namespace, name)
}

func (es *EtcdStorage) nodeKey(name string) string {
	return fmt.Sprintf("/orchestration/nodes/%s", name)
}

func (es *EtcdStorage) serviceKey(namespace, name string) string {
	return fmt.Sprintf("/orchestration/services/%s/%s", namespace, name)
}

func (es *EtcdStorage) deploymentKey(namespace, name string) string {
	return fmt.Sprintf("/orchestration/deployments/%s/%s", namespace, name)
}

// Pod operations
func (es *EtcdStorage) SavePod(ctx context.Context, pod *types.Pod) error {
	data, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod: %w", err)
	}

	key := es.podKey(pod.Metadata.Namespace, pod.Metadata.Name)
	_, err = es.client.Put(ctx, key, string(data))
	return err
}

func (es *EtcdStorage) GetPod(ctx context.Context, namespace, name string) (*types.Pod, error) {
	key := es.podKey(namespace, name)
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("pod not found: %s/%s", namespace, name)
	}

	var pod types.Pod
	if err := json.Unmarshal(resp.Kvs[0].Value, &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	return &pod, nil
}

func (es *EtcdStorage) ListPods(ctx context.Context, namespace string) ([]*types.Pod, error) {
	prefix := fmt.Sprintf("/orchestration/pods/%s/", namespace)
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var pods []*types.Pod
	for _, kv := range resp.Kvs {
		var pod types.Pod
		if err := json.Unmarshal(kv.Value, &pod); err != nil {
			continue
		}
		pods = append(pods, &pod)
	}

	return pods, nil
}

func (es *EtcdStorage) DeletePod(ctx context.Context, namespace, name string) error {
	key := es.podKey(namespace, name)
	_, err := es.client.Delete(ctx, key)
	return err
}

// Node operations
func (es *EtcdStorage) SaveNode(ctx context.Context, node *types.Node) error {
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	key := es.nodeKey(node.Metadata.Name)
	_, err = es.client.Put(ctx, key, string(data))
	return err
}

func (es *EtcdStorage) GetNode(ctx context.Context, name string) (*types.Node, error) {
	key := es.nodeKey(name)
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("node not found: %s", name)
	}

	var node types.Node
	if err := json.Unmarshal(resp.Kvs[0].Value, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	return &node, nil
}

func (es *EtcdStorage) ListNodes(ctx context.Context) ([]*types.Node, error) {
	prefix := "/orchestration/nodes/"
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var nodes []*types.Node
	for _, kv := range resp.Kvs {
		var node types.Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			continue
		}
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func (es *EtcdStorage) DeleteNode(ctx context.Context, name string) error {
	key := es.nodeKey(name)
	_, err := es.client.Delete(ctx, key)
	return err
}

// Service operations
func (es *EtcdStorage) SaveService(ctx context.Context, service *types.Service) error {
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}

	key := es.serviceKey(service.Metadata.Namespace, service.Metadata.Name)
	_, err = es.client.Put(ctx, key, string(data))
	return err
}

func (es *EtcdStorage) GetService(ctx context.Context, namespace, name string) (*types.Service, error) {
	key := es.serviceKey(namespace, name)
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	var service types.Service
	if err := json.Unmarshal(resp.Kvs[0].Value, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service: %w", err)
	}

	return &service, nil
}

func (es *EtcdStorage) ListServices(ctx context.Context, namespace string) ([]*types.Service, error) {
	prefix := fmt.Sprintf("/orchestration/services/%s/", namespace)
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var services []*types.Service
	for _, kv := range resp.Kvs {
		var service types.Service
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}
		services = append(services, &service)
	}

	return services, nil
}

func (es *EtcdStorage) DeleteService(ctx context.Context, namespace, name string) error {
	key := es.serviceKey(namespace, name)
	_, err := es.client.Delete(ctx, key)
	return err
}

// Deployment operations
func (es *EtcdStorage) SaveDeployment(ctx context.Context, deployment *types.Deployment) error {
	data, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	key := es.deploymentKey(deployment.Metadata.Namespace, deployment.Metadata.Name)
	_, err = es.client.Put(ctx, key, string(data))
	return err
}

func (es *EtcdStorage) GetDeployment(ctx context.Context, namespace, name string) (*types.Deployment, error) {
	key := es.deploymentKey(namespace, name)
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("deployment not found: %s/%s", namespace, name)
	}

	var deployment types.Deployment
	if err := json.Unmarshal(resp.Kvs[0].Value, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}

func (es *EtcdStorage) ListDeployments(ctx context.Context, namespace string) ([]*types.Deployment, error) {
	prefix := fmt.Sprintf("/orchestration/deployments/%s/", namespace)
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var deployments []*types.Deployment
	for _, kv := range resp.Kvs {
		var deployment types.Deployment
		if err := json.Unmarshal(kv.Value, &deployment); err != nil {
			continue
		}
		deployments = append(deployments, &deployment)
	}

	return deployments, nil
}

func (es *EtcdStorage) DeleteDeployment(ctx context.Context, namespace, name string) error {
	key := es.deploymentKey(namespace, name)
	_, err := es.client.Delete(ctx, key)
	return err
}

// Close closes the etcd connection
func (es *EtcdStorage) Close() error {
	return es.client.Close()
}
