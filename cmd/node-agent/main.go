package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/orchestration-lite/core/pkg/runtime"
	"github.com/orchestration-lite/core/pkg/storage"
	"github.com/orchestration-lite/core/pkg/types"
	"github.com/google/uuid"
)

func main() {
	nodeName := flag.String("node-name", "worker-1", "Name of this worker node")
	etcdEndpoint := flag.String("etcd", "localhost:2379", "etcd endpoint")
	flag.Parse()

	log.Printf("Starting node agent on %s", *nodeName)

	// Connect to etcd
	store, err := storage.NewEtcdStorage([]string{*etcdEndpoint})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer store.Close()

	// Initialize container runtime
	cr, err := runtime.NewContainerRuntime()
	if err != nil {
		log.Fatalf("Failed to initialize container runtime: %v", err)
	}
	defer cr.Close()

	// Register this node
	ctx := context.Background()
	node := &types.Node{
		Metadata: types.ObjectMeta{
			Name: *nodeName,
			UID:  uuid.New().String(),
		},
		Status: types.NodeStatus{
			Phase: types.NodeReady,
			Ready: true,
		},
		Capacity: types.ResourceList{
			"cpu":    "8",
			"memory": "16000",
		},
		Allocatable: types.ResourceList{
			"cpu":    "8",
			"memory": "16000",
		},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	if err := store.SaveNode(ctx, node); err != nil {
		log.Fatalf("Failed to register node: %v", err)
	}

	log.Printf("Node registered: %s", *nodeName)

	// Start heartbeat loop
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			node.LastHeartbeat = time.Now()
			store.SaveNode(ctx, node)
		}
	}()

	// Watch for pods scheduled on this node and run them
	watchTicker := time.NewTicker(5 * time.Second)
	defer watchTicker.Stop()

	runningContainers := make(map[string]string) // pod -> container ID

	for range watchTicker.C {
		pods, err := store.ListPods(ctx, "default")
		if err != nil {
			log.Printf("Error listing pods: %v", err)
			continue
		}

		for _, pod := range pods {
			// Check if pod is scheduled on this node
			if pod.Spec.NodeName != *nodeName {
				continue
			}

			podID := fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name)

			// Skip if already running
			if _, exists := runningContainers[podID]; exists {
				continue
			}

			// Create and start containers
			log.Printf("Starting pod %s on node %s", podID, *nodeName)

			pod.Status.Phase = types.PodRunning
			pod.Status.HostIP = "127.0.0.1"

			for _, container := range pod.Spec.Containers {
				containerID, err := cr.CreateContainer(ctx, pod, container)
				if err != nil {
					log.Printf("Error creating container: %v", err)
					pod.Status.Phase = types.PodFailed
					pod.Status.Message = fmt.Sprintf("Failed to create container: %v", err)
					break
				}

				if err := cr.StartContainer(ctx, containerID); err != nil {
					log.Printf("Error starting container: %v", err)
					pod.Status.Phase = types.PodFailed
					pod.Status.Message = fmt.Sprintf("Failed to start container: %v", err)
					break
				}

				log.Printf("Container %s started", containerID)
				runningContainers[podID] = containerID
			}

			// Update pod status
			if err := store.SavePod(ctx, pod); err != nil {
				log.Printf("Error updating pod status: %v", err)
			}
		}
	}
}
