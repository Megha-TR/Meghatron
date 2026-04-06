package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/orchestration-lite/core/pkg/storage"
	"github.com/orchestration-lite/core/pkg/types"
)

// Scheduler is responsible for assigning pods to nodes
type Scheduler struct {
	storage storage.Storage
}

// NewScheduler creates a new scheduler instance
func NewScheduler(store storage.Storage) *Scheduler {
	return &Scheduler{storage: store}
}

// Schedule assigns a pod to a suitable node
func (s *Scheduler) Schedule(ctx context.Context, pod *types.Pod) error {
	// Get all nodes
	nodes, err := s.storage.ListNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes available for scheduling")
	}

	// Find the best node using bin-packing algorithm
	selectedNode := s.findBestNode(ctx, nodes, pod)
	if selectedNode == nil {
		return fmt.Errorf("no suitable node found for pod %s/%s", pod.Metadata.Namespace, pod.Metadata.Name)
	}

	// Assign pod to node
	pod.Spec.NodeName = selectedNode.Metadata.Name
	pod.Status.Phase = types.PodPending

	// Save updated pod
	if err := s.storage.SavePod(ctx, pod); err != nil {
		return fmt.Errorf("failed to save pod: %w", err)
	}

	log.Printf("Pod %s/%s scheduled on node %s", pod.Metadata.Namespace, pod.Metadata.Name, selectedNode.Metadata.Name)
	return nil
}

// findBestNode implements a bin-packing algorithm to find the best node for a pod
func (s *Scheduler) findBestNode(ctx context.Context, nodes []*types.Node, pod *types.Pod) *types.Node {
	var bestNode *types.Node
	minWaste := float64(1<<63 - 1) // Max float

	podResources := s.getPodResources(pod)

	for _, node := range nodes {
		// Skip unhealthy nodes
		if !node.Status.Ready {
			continue
		}

		// Check if node has enough resources
		if !s.hasEnoughResources(node, podResources) {
			continue
		}

		// Calculate waste (bin-packing)
		waste := s.calculateWaste(node, podResources)
		if waste < minWaste {
			minWaste = waste
			bestNode = node
		}
	}

	return bestNode
}

// getPodResources extracts resource requests from a pod
func (s *Scheduler) getPodResources(pod *types.Pod) map[string]int64 {
	resources := make(map[string]int64)
	resources["cpu"] = 0
	resources["memory"] = 0

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if cpu, ok := container.Resources.Requests["cpu"]; ok {
				resources["cpu"] += parseResource(cpu)
			}
			if mem, ok := container.Resources.Requests["memory"]; ok {
				resources["memory"] += parseResource(mem)
			}
		}
	}

	return resources
}

// hasEnoughResources checks if a node has enough resources for a pod
func (s *Scheduler) hasEnoughResources(node *types.Node, podResources map[string]int64) bool {
	nodeAllocatable := node.Allocatable

	if cpu, ok := podResources["cpu"]; ok {
		if nodeCPU, ok := nodeAllocatable["cpu"]; ok {
			if parseResource(nodeCPU) < cpu {
				return false
			}
		}
	}

	if mem, ok := podResources["memory"]; ok {
		if nodeMem, ok := nodeAllocatable["memory"]; ok {
			if parseResource(nodeMem) < mem {
				return false
			}
		}
	}

	return true
}

// calculateWaste calculates resource waste when assigning a pod to a node
func (s *Scheduler) calculateWaste(node *types.Node, podResources map[string]int64) float64 {
	nodeAllocatable := node.Allocatable

	cpuWaste := float64(0)
	if cpu, ok := podResources["cpu"]; ok {
		if nodeCPU, ok := nodeAllocatable["cpu"]; ok {
			nodeCapacity := parseResource(nodeCPU)
			cpuWaste = float64(nodeCapacity-cpu) / float64(nodeCapacity)
		}
	}

	memWaste := float64(0)
	if mem, ok := podResources["memory"]; ok {
		if nodeMem, ok := nodeAllocatable["memory"]; ok {
			nodeCapacity := parseResource(nodeMem)
			memWaste = float64(nodeCapacity-mem) / float64(nodeCapacity)
		}
	}

	return (cpuWaste + memWaste) / 2
}

// parseResource converts a resource string to int64
func parseResource(resource string) int64 {
	// Simple parser for resource values
	// In production, this would handle units like "100m", "1Gi", etc.
	var value int64
	_, err := fmt.Sscanf(resource, "%d", &value)
	if err != nil {
		return 0
	}
	return value
}
