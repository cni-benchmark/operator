package iperf3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	config "cni-benchmark/pkg/config"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// WaitForServer attempts to establish a TCP connection to the server with a timeout
func WaitForServer(ctx context.Context, cfg *config.Config) error {
	log := logf.FromContext(ctx)
	if cfg.Mode != config.ModeClient {
		return nil
	}
	if len(cfg.Server) == 0 {
		return errors.New("server must be set in client mode")
	}
	address := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	log.Info("waiting for server", "address", address)

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for server")
		default:
			conn, err := net.DialTimeout("tcp", address, 5*time.Second)
			if err == nil {
				conn.Close()
				log.Info("server is reachable")
				return nil
			}
			log.Info("still waiting for the server", "error", err.Error())
			time.Sleep(time.Second)
		}
	}
}

// Run iperf3 and get JSON output
func Run(_ context.Context, cfg *config.Config) (report *Report, err error) {
	if err = WaitForServer(context.Background(), cfg); err != nil {
		return nil, fmt.Errorf("failed waiting for server: %w", err)
	}

	// Execute iperf3
	var stdoutBuf bytes.Buffer
	cmd := exec.CommandContext(context.Background(), cfg.Command[0], cfg.Command[1:]...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute iperf3: %w", err)
	}
	if cmd.ProcessState.ExitCode() != 0 {
		return nil, fmt.Errorf("iperf3 exited with %d", cmd.ProcessState.ExitCode())
	}

	if cfg.Mode == config.ModeClient {
		// Parse JSON output
		output := stdoutBuf.Bytes()
		report = &Report{}
		if err := json.Unmarshal(output, report); err != nil {
			return nil, fmt.Errorf("failed to parse JSON output: %w", err)
		}
	}
	return
}
