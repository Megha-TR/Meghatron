package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/orchestration-lite/core/pkg/api"
	"github.com/orchestration-lite/core/pkg/controller"
	"github.com/orchestration-lite/core/pkg/scheduler"
	"github.com/orchestration-lite/core/pkg/storage"
)

func main() {
	etcdEndpoint := flag.String("etcd", "localhost:2379", "etcd endpoint")
	port := flag.String("port", "8080", "API server port")
	flag.Parse()

	// Connect to etcd
	log.Println("Connecting to etcd...")
	store, err := storage.NewEtcdStorage([]string{*etcdEndpoint})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer store.Close()

	// Create scheduler
	sched := scheduler.NewScheduler(store)

	// Create API server
	apiServer := api.NewServer(store, sched)

	// Create and start deployment controller
	deploymentController := controller.NewDeploymentController(store)
	ctx, cancel := context.WithCancel(context.Background())
	deploymentController.Run(ctx)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
		deploymentController.Stop()
		os.Exit(0)
	}()

	// Start API server
	if err := apiServer.Listen(*port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
