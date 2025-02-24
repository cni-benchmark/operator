package main

import (
	"cni-benchmark/pkg/config"
	"cni-benchmark/pkg/iperf3"
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var log logr.Logger

func init() {
	logf.SetLogger(zap.New(zap.ConsoleEncoder(), zap.UseDevMode(true)))
	log = logf.FromContext(context.Background())
}

func main() {
	cfg, err := config.Build()
	if err != nil {
		log.Error(err, "failed to build a config")
		os.Exit(1)
	}

	log.Info("configuration object is built", "configuration", cfg)

	switch cfg.Mode {
	case config.ModeClient:
		runClient(cfg)
	case config.ModeServer:
		runServer(cfg)
	}
}

func runServer(cfg *config.Config) {
	log.Info("starting in server mode")
	if _, err := iperf3.Run(context.Background(), cfg); err != nil {
		log.Error(err, "server fatal error")
		os.Exit(1)
	}
}

func runClient(cfg *config.Config) {
	log.Info("starting in client mode")
	client, err := config.BuildKubernetesClient()
	if err != nil {
		log.Error(err, "failed to build kubernetes client")
		os.Exit(1)
	}

	// Configure the leader election
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.Lease.Name,
			Namespace: cfg.Lease.Namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.Lease.ID,
		},
	}

	info := &iperf3.Info{}
	if err = info.Build(cfg); err != nil {
		log.Error(err, "failed to gather information")
		os.Exit(1)
	}
	log.Info("gathering system information", "info", info)

	// Create leader election config
	leaderConfig := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   20 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Info("got leadership, starting benchmark")
				var report *iperf3.Report
				if report, err = iperf3.Run(ctx, cfg); err != nil {
					log.Error(err, "iperf3 run failed")
					os.Exit(1)
				}
				log.Info("saving data")
				if err = iperf3.Store(ctx, cfg, report, info); err != nil {
					log.Error(err, "metrics upload failed")
					os.Exit(1)
				}
				os.Exit(0)
			},
			OnStoppedLeading: func() {
				log.Error(nil, "leadership lost")
				os.Exit(1)
			},
			OnNewLeader: func(identity string) {
				if identity == cfg.Lease.ID {
					return
				}
				log.Info("leadership is held by a new leader", "leader", identity)
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the leader election
	log.Info("starting leader election")
	leaderelection.RunOrDie(ctx, leaderConfig)
}
