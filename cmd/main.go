package main

import (
	"cni-benchmark/pkg/config"
	"cni-benchmark/pkg/iperf3"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	cfg, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to build a config: %v", err)
	}

	yb, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("Failed to YAML marshal a config: %v", err)
	}
	log.Printf("Configuration:\n  %s", strings.ReplaceAll(string(yb), "\n", "\n  "))

	switch cfg.Mode {
	case config.ModeClient:
		runClient(cfg)
	case config.ModeServer:
		runServer(cfg)
	}
}

func runServer(cfg *config.Config) {
	log.Println("Starting server")
	if _, err := iperf3.Run(cfg); err != nil {
		log.Fatalf("Server mode error: %v", err)
	}
}

func runClient(cfg *config.Config) {
	client, err := config.BuildKubernetesClient()
	if err != nil {
		log.Fatalf("Failed to build kubernetes client: %v", err)
	}

	// Create a unique identifier for this instance
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get the hostname: %v", err)
	}
	id := fmt.Sprintf("%s_%d", hostname, time.Now().Unix())

	// Configure the leader election
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.Lease.Name,
			Namespace: cfg.Lease.Namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	log.Printf("Gathering information about the system")
	info := &iperf3.Info{}
	if err = info.Build(cfg); err != nil {
		log.Fatalf("Failed to gather information: %v", err)
	}

	// Create leader election config
	leaderConfig := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   20 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				log.Printf("Got leadership, starting benchmark")
				var report *iperf3.Report
				if report, err = iperf3.Run(cfg); err != nil {
					log.Fatalf("Error running iperf: %v", err)
				}
				log.Printf("Saving data")
				if err = iperf3.Store(cfg, report, info); err != nil {
					log.Fatalf("Error analyzing iperf output: %v", err)
				}
				os.Exit(0)
			},
			OnStoppedLeading: func() {
				log.Fatalln("Leadership lost")
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					return
				}
				log.Printf("Leadership is held by: %s", identity)
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the leader election
	log.Printf("Starting leader election")
	leaderelection.RunOrDie(ctx, leaderConfig)
}
