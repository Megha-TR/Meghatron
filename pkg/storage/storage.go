package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/types"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource already exists or was changed; retry with the latest version")

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

type EtcdStorage struct{ client *clientv3.Client }

func NewEtcdStorage(endpoints []string) (*EtcdStorage, error) {
	c, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	return &EtcdStorage{client: c}, nil
}
func (es *EtcdStorage) save(ctx context.Context, key string, value any, meta *types.ObjectMeta) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	result, err := es.client.Txn(ctx).If(clientv3.Compare(clientv3.ModRevision(key), "=", meta.ResourceVersion)).Then(clientv3.OpPut(key, string(data))).Commit()
	if err != nil {
		return err
	}
	if !result.Succeeded {
		return ErrConflict
	}
	meta.ResourceVersion = result.Header.Revision
	return nil
}
func (es *EtcdStorage) Close() error { return es.client.Close() }
func (es *EtcdStorage) SavePod(ctx context.Context, value *types.Pod) error {
	return es.save(ctx, fmt.Sprintf("/orchestration/pods/%s/%s", value.Metadata.Namespace, value.Metadata.Name), value, &value.Metadata)
}
func (es *EtcdStorage) GetPod(ctx context.Context, namespace, name string) (*types.Pod, error) {
	resp, err := es.client.Get(ctx, fmt.Sprintf("/orchestration/pods/%s/%s", namespace, name))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	var value types.Pod
	if err := json.Unmarshal(resp.Kvs[0].Value, &value); err != nil {
		return nil, err
	}
	value.Metadata.ResourceVersion = resp.Kvs[0].ModRevision
	return &value, nil
}
func (es *EtcdStorage) ListPods(ctx context.Context, namespace string) ([]*types.Pod, error) {
	prefix := "/orchestration/pods/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	values := make([]*types.Pod, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var value types.Pod
		if err := json.Unmarshal(kv.Value, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", kv.Key, err)
		}
		value.Metadata.ResourceVersion = kv.ModRevision
		values = append(values, &value)
	}
	return values, nil
}
func (es *EtcdStorage) DeletePod(ctx context.Context, namespace, name string) error {
	result, err := es.client.Delete(ctx, fmt.Sprintf("/orchestration/pods/%s/%s", namespace, name))
	if err != nil {
		return err
	}
	if result.Deleted == 0 {
		return ErrNotFound
	}
	return nil
}
func (es *EtcdStorage) SaveNode(ctx context.Context, value *types.Node) error {
	return es.save(ctx, "/orchestration/nodes/"+value.Metadata.Name, value, &value.Metadata)
}
func (es *EtcdStorage) GetNode(ctx context.Context, name string) (*types.Node, error) {
	resp, err := es.client.Get(ctx, "/orchestration/nodes/"+name)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	var value types.Node
	if err := json.Unmarshal(resp.Kvs[0].Value, &value); err != nil {
		return nil, err
	}
	value.Metadata.ResourceVersion = resp.Kvs[0].ModRevision
	return &value, nil
}
func (es *EtcdStorage) ListNodes(ctx context.Context) ([]*types.Node, error) {
	prefix := "/orchestration/nodes/"
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	values := make([]*types.Node, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var value types.Node
		if err := json.Unmarshal(kv.Value, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", kv.Key, err)
		}
		value.Metadata.ResourceVersion = kv.ModRevision
		values = append(values, &value)
	}
	return values, nil
}
func (es *EtcdStorage) DeleteNode(ctx context.Context, name string) error {
	result, err := es.client.Delete(ctx, "/orchestration/nodes/"+name)
	if err != nil {
		return err
	}
	if result.Deleted == 0 {
		return ErrNotFound
	}
	return nil
}
func (es *EtcdStorage) SaveService(ctx context.Context, value *types.Service) error {
	return es.save(ctx, fmt.Sprintf("/orchestration/services/%s/%s", value.Metadata.Namespace, value.Metadata.Name), value, &value.Metadata)
}
func (es *EtcdStorage) GetService(ctx context.Context, namespace, name string) (*types.Service, error) {
	resp, err := es.client.Get(ctx, fmt.Sprintf("/orchestration/services/%s/%s", namespace, name))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	var value types.Service
	if err := json.Unmarshal(resp.Kvs[0].Value, &value); err != nil {
		return nil, err
	}
	value.Metadata.ResourceVersion = resp.Kvs[0].ModRevision
	return &value, nil
}
func (es *EtcdStorage) ListServices(ctx context.Context, namespace string) ([]*types.Service, error) {
	prefix := "/orchestration/services/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	values := make([]*types.Service, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var value types.Service
		if err := json.Unmarshal(kv.Value, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", kv.Key, err)
		}
		value.Metadata.ResourceVersion = kv.ModRevision
		values = append(values, &value)
	}
	return values, nil
}
func (es *EtcdStorage) DeleteService(ctx context.Context, namespace, name string) error {
	result, err := es.client.Delete(ctx, fmt.Sprintf("/orchestration/services/%s/%s", namespace, name))
	if err != nil {
		return err
	}
	if result.Deleted == 0 {
		return ErrNotFound
	}
	return nil
}
func (es *EtcdStorage) SaveDeployment(ctx context.Context, value *types.Deployment) error {
	return es.save(ctx, fmt.Sprintf("/orchestration/deployments/%s/%s", value.Metadata.Namespace, value.Metadata.Name), value, &value.Metadata)
}
func (es *EtcdStorage) GetDeployment(ctx context.Context, namespace, name string) (*types.Deployment, error) {
	resp, err := es.client.Get(ctx, fmt.Sprintf("/orchestration/deployments/%s/%s", namespace, name))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	var value types.Deployment
	if err := json.Unmarshal(resp.Kvs[0].Value, &value); err != nil {
		return nil, err
	}
	value.Metadata.ResourceVersion = resp.Kvs[0].ModRevision
	return &value, nil
}
func (es *EtcdStorage) ListDeployments(ctx context.Context, namespace string) ([]*types.Deployment, error) {
	prefix := "/orchestration/deployments/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	resp, err := es.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	values := make([]*types.Deployment, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var value types.Deployment
		if err := json.Unmarshal(kv.Value, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", kv.Key, err)
		}
		value.Metadata.ResourceVersion = kv.ModRevision
		values = append(values, &value)
	}
	return values, nil
}
func (es *EtcdStorage) DeleteDeployment(ctx context.Context, namespace, name string) error {
	result, err := es.client.Delete(ctx, fmt.Sprintf("/orchestration/deployments/%s/%s", namespace, name))
	if err != nil {
		return err
	}
	if result.Deleted == 0 {
		return ErrNotFound
	}
	return nil
}
