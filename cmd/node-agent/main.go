package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/agent"
	"github.com/Megha-TR/Meghatron/pkg/runtime"
	"github.com/Megha-TR/Meghatron/pkg/storage"
	"github.com/Megha-TR/Meghatron/pkg/types"
	"github.com/google/uuid"
)

func main() {
	nodeName := flag.String("node-name", "worker-1", "Unique worker name (one agent per Docker daemon)")
	endpoint := flag.String("etcd", "localhost:2379", "etcd endpoint")
	flag.Parse()
	if !types.ValidName(*nodeName) {
		log.Fatal("invalid node name")
	}
	if err := run(*nodeName, *endpoint); err != nil {
		log.Fatal(err)
	}
}
func run(name, endpoint string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := storage.NewEtcdStorage([]string{endpoint})
	if err != nil {
		return err
	}
	defer store.Close()
	cr, err := runtime.NewContainerRuntime()
	if err != nil {
		return err
	}
	defer cr.Close()
	a := agent.Agent{Store: store, Runtime: cr, NodeName: name}
	heartbeat := func() error {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		capacity, err := cr.Capacity(c)
		if err != nil {
			return err
		}
		node, err := store.GetNode(c, name)
		if errors.Is(err, storage.ErrNotFound) {
			node = &types.Node{Metadata: types.ObjectMeta{Name: name, UID: uuid.NewString()}, CreatedAt: time.Now()}
		} else if err != nil {
			return err
		}
		node.Capacity = capacity
		node.Allocatable = capacity
		node.LastHeartbeat = time.Now()
		node.UpdatedAt = time.Now()
		node.Status = types.NodeStatus{Ready: true, Phase: types.NodeReady}
		return store.SaveNode(c, node)
	}
	if err := heartbeat(); err != nil {
		return err
	}
	log.Printf("Node %s registered; reconciling managed containers", name)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	beats := time.NewTicker(10 * time.Second)
	defer beats.Stop()
	// Keep heartbeats independent of slow image pulls.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-beats.C:
				if err := heartbeat(); err != nil && ctx.Err() == nil {
					log.Printf("heartbeat: %v", err)
				}
			}
		}
	}()
	defer func() { stop(); <-done }()
	for {
		c, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := a.Reconcile(c)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Printf("agent: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
