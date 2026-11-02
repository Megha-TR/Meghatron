package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/resources"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
)

const HeartbeatTimeout = 30 * time.Second

// One scheduler is supported per cluster. The mutex serializes local reservations;
// etcd revisions prevent stale writes from resurrecting deleted or changed pods.
type Scheduler struct {
	storage storage.Storage
	mu      sync.Mutex
}

func NewScheduler(store storage.Storage) *Scheduler { return &Scheduler{storage: store} }
func Fresh(node *types.Node) bool {
	return node.Status.Ready && time.Since(node.LastHeartbeat) < HeartbeatTimeout && !node.LastHeartbeat.IsZero()
}
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := s.Reconcile(ctx); err != nil && ctx.Err() == nil {
			log.Printf("scheduler: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Scheduler) Reconcile(ctx context.Context) error {
	nodes, err := s.storage.ListNodes(ctx)
	if err != nil {
		return err
	}
	freshNodes := map[string]bool{}
	for _, node := range nodes {
		freshNodes[node.Metadata.Name] = Fresh(node)
		if node.Status.Ready && !Fresh(node) {
			node.Status.Ready = false
			node.Status.Phase = types.NodeNotReady
			node.Status.Message = "Heartbeat expired"
			if err := s.storage.SaveNode(ctx, node); err != nil {
				log.Printf("node health: %v", err)
			}
		}
	}
	pods, err := s.storage.ListPods(ctx, "")
	if err != nil {
		return err
	}
	for _, pod := range pods {
		if pod.Spec.NodeName != "" {
			if !freshNodes[pod.Spec.NodeName] && pod.Metadata.DeletionTimestamp == nil && !types.Terminal(pod) {
				pod.Status.Phase = types.PodUnknown
				pod.Status.Message = "Assigned worker heartbeat expired or worker missing"
				if err := s.storage.SavePod(ctx, pod); err != nil {
					return err
				}
			}
			continue
		}
		if pod.Metadata.DeletionTimestamp != nil {
			if err := s.storage.DeletePod(ctx, pod.Metadata.Namespace, pod.Metadata.Name); err != nil {
				return err
			}
			continue
		}
		if types.Terminal(pod) {
			continue
		}
		if err := s.Schedule(ctx, pod); err != nil {
			pod.Status.Message = err.Error()
			pod.UpdatedAt = time.Now()
			if err := s.storage.SavePod(ctx, pod); err != nil {
				log.Printf("pending status: %v", err)
			}
		}
	}
	return nil
}
func (s *Scheduler) Schedule(ctx context.Context, pod *types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pod.Spec.NodeName != "" || pod.Metadata.DeletionTimestamp != nil {
		return nil
	}
	nodes, err := s.storage.ListNodes(ctx)
	if err != nil {
		return err
	}
	pods, err := s.storage.ListPods(ctx, "")
	if err != nil {
		return err
	}
	need, err := resources.Requests(pod)
	if err != nil {
		return err
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Metadata.Name < nodes[j].Metadata.Name })
	var best *types.Node
	bestScore := float64(3)
	for _, node := range nodes {
		if !Fresh(node) {
			continue
		}
		capacity, err := resources.Parse(node.Allocatable)
		if err != nil || capacity.CPU == 0 || capacity.Memory == 0 {
			continue
		}
		available := capacity
		portConflict := false
		for _, assigned := range pods {
			if assigned.Spec.NodeName != node.Metadata.Name || assigned.Metadata.UID == pod.Metadata.UID {
				continue
			}
			// Reserve even terminal pods until the agent has cleaned their containers.
			used, err := resources.Requests(assigned)
			if err != nil {
				return err
			}
			available = available.Sub(used)
			if hostPortsConflict(pod, assigned) {
				portConflict = true
			}
		}
		if !available.Fits(need) || portConflict {
			continue
		}
		remaining := available.Sub(need)
		score := float64(remaining.CPU)/float64(capacity.CPU) + float64(remaining.Memory)/float64(capacity.Memory)
		if score < bestScore {
			best = node
			bestScore = score
		}
	}
	if best == nil {
		return fmt.Errorf("no healthy node has sufficient CPU, memory and host ports")
	}
	pod.Spec.NodeName = best.Metadata.Name
	pod.Status.Phase = types.PodPending
	pod.Status.Message = ""
	pod.UpdatedAt = time.Now()
	return s.storage.SavePod(ctx, pod)
}
func hostPortsConflict(a, b *types.Pod) bool {
	for _, ca := range a.Spec.Containers {
		for _, pa := range ca.Ports {
			if pa.HostPort == 0 {
				continue
			}
			for _, cb := range b.Spec.Containers {
				for _, pb := range cb.Ports {
					ap, bp := pa.Protocol, pb.Protocol
					if ap == "" {
						ap = "TCP"
					}
					if bp == "" {
						bp = "TCP"
					}
					if pa.HostPort == pb.HostPort && ap == bp {
						return true
					}
				}
			}
		}
	}
	return false
}
