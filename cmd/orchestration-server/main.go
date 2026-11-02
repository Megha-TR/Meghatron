package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Megha-TR/Meghatron/pkg/api"
	"github.com/Megha-TR/Meghatron/pkg/controller"
	"github.com/Megha-TR/Meghatron/pkg/scheduler"
	"github.com/Megha-TR/Meghatron/pkg/storage"
)

func main() {
	endpoint := flag.String("etcd", "localhost:2379", "etcd endpoint")
	listen := flag.String("listen", "127.0.0.1", "API bind address")
	port := flag.String("port", "8080", "API port")
	flag.Parse()
	if err := run(*endpoint, *listen+":"+*port); err != nil {
		log.Fatal(err)
	}
}
func run(endpoint, address string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := storage.NewEtcdStorage([]string{endpoint})
	if err != nil {
		return err
	}
	defer store.Close()
	sched := scheduler.NewScheduler(store)
	dc := controller.NewDeploymentController(store)
	var wg sync.WaitGroup
	for _, run := range []func(context.Context){sched.Run, dc.Run} {
		wg.Add(1)
		go func(run func(context.Context)) { defer wg.Done(); run(ctx) }(run)
	}
	defer func() { stop(); wg.Wait() }()
	server := &http.Server{Addr: address, Handler: api.NewServer(store, sched).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(c); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Printf("Meghatron API listening on %s", address)
	err = server.ListenAndServe()
	stop()
	<-done
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
