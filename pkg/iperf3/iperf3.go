package iperf3

import (
	"bytes"
	config "cni-benchmark/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/docker/docker/pkg/parsers/kernel"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"
)

// WaitForServer attempts to establish a TCP connection to the server with a timeout
func WaitForServer(ctx context.Context, cfg *config.Config) error {
	if cfg.Mode != config.ModeClient {
		return nil
	}
	if len(cfg.Server) == 0 {
		return fmt.Errorf("server must be set in client mode")
	}
	address := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	log.Printf("Waiting for server at %s", address)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server at %s", address)
		default:
			conn, err := net.DialTimeout("tcp", address, 5*time.Second)
			if err == nil {
				conn.Close()
				log.Printf("Server is reachable at %s", address)
				return nil
			}
			log.Printf("Server not yet reachable: %v", err)
			time.Sleep(time.Second)
		}
	}
}

// Run iperf3 and get JSON output
func Run(cfg *config.Config) (report *Report, err error) {
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

		// Get extra info
		kv, err := kernel.GetKernelVersion()
		if err != nil {
			return nil, fmt.Errorf("failed to get kernel info: %w", err)
		}

		report.System.KernelVersion = kv.String()
		report.System.Architecture = runtime.GOARCH
	}
	return
}

func Analyze(cfg *config.Config, report *Report) error {
	// Create an InfluxDB client
	client := influxdb2.NewClient(cfg.InfluxDB.Url.String(), cfg.InfluxDB.Token)
	defer client.Close()

	// Get non-blocking write client
	writeAPI := client.WriteAPIBlocking(cfg.InfluxDB.Org, cfg.InfluxDB.Bucket)

	// Common tags for all measurements
	tags := map[string]string{
		"iperf3_version": report.Start.Version,
		"kernel_arch":    report.System.Architecture,
		"kernel_version": report.System.KernelVersion,
		"protocol":       report.Start.Test.Protocol,
	}

	// Add custom tags from config
	if cfg.TestCase != "" {
		tags["test_case"] = cfg.TestCase
	}
	for key, value := range cfg.InfluxDB.Tags {
		tags[key] = value
	}

	// Write summary metrics with retry
	if err := writeSummaryMetricsWithRetry(writeAPI, report, tags); err != nil {
		return fmt.Errorf("failed to write summary metrics after retries: %w", err)
	}

	// Write interval metrics with retry
	if err := writeIntervalMetricsWithRetry(writeAPI, report, tags); err != nil {
		return fmt.Errorf("failed to write interval metrics after retries: %w", err)
	}

	log.Println("Metrics successfully written to InfluxDB")
	return nil
}

// Retry configuration
const (
	maxRetries  = 3
	baseBackoff = 1 * time.Second
	maxBackoff  = 10 * time.Second
)

// exponentialBackoff calculates the wait time between retries
func exponentialBackoff(retry int) time.Duration {
	// Calculate backoff with exponential increase and some jitter
	backoff := baseBackoff * time.Duration(math.Pow(2, float64(retry)))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	// Add some jitter to prevent synchronized retries
	jitter := time.Duration(rand.Float64() * float64(backoff) * 0.1)
	return backoff + jitter
}

func writeSummaryMetricsWithRetry(writeAPI influxdb2api.WriteAPIBlocking, report *Report, tags map[string]string) error {
	// Calculate timestamp from report start time
	timestamp := time.Unix(int64(report.Start.Timestamp.Seconds), 0)

	// Retry logic for transmit metrics
	txRetryFunc := func() error {
		txPoint := influxdb2.NewPoint(
			"iperf3_summary",
			tags,
			map[string]interface{}{
				"tx_bandwidth_bps":    report.End.Sent.BitsPerSecond,
				"tx_bytes":            report.End.Sent.Bytes,
				"tx_duration_seconds": report.End.Sent.DurationSeconds,
				"tx_retransmits":      report.End.Sent.Retransmits,
			},
			timestamp,
		)

		return writeAPI.WritePoint(context.Background(), txPoint)
	}

	if err := retryOperation(txRetryFunc); err != nil {
		return fmt.Errorf("failed to write tx summary metrics: %w", err)
	}

	// Retry logic for receive metrics
	rxRetryFunc := func() error {
		rxPoint := influxdb2.NewPoint(
			"iperf3_summary",
			tags,
			map[string]interface{}{
				"rx_bandwidth_bps":    report.End.Received.BitsPerSecond,
				"rx_bytes":            report.End.Received.Bytes,
				"rx_duration_seconds": report.End.Received.DurationSeconds,
			},
			timestamp,
		)

		return writeAPI.WritePoint(context.Background(), rxPoint)
	}

	if err := retryOperation(rxRetryFunc); err != nil {
		return fmt.Errorf("failed to write rx summary metrics: %w", err)
	}

	return nil
}

func writeIntervalMetricsWithRetry(writeAPI influxdb2api.WriteAPIBlocking, report *Report, tags map[string]string) error {
	baseTime := time.Unix(int64(report.Start.Timestamp.Seconds), 0)

	for _, interval := range report.Intervals {
		// Calculate timestamp for this interval
		intervalStart := baseTime.Add(time.Duration(interval.Sum.Start * float64(time.Second)))

		// Retry logic for each interval point
		retryFunc := func() error {
			point := influxdb2.NewPoint(
				"iperf3_interval",
				tags,
				map[string]interface{}{
					"bandwidth_bps":    interval.Sum.BitsPerSecond,
					"bytes":            interval.Sum.Bytes,
					"duration_seconds": interval.Sum.DurationSeconds,
					"retransmits":      interval.Sum.Retransmits,
				},
				intervalStart,
			)

			return writeAPI.WritePoint(context.Background(), point)
		}

		if err := retryOperation(retryFunc); err != nil {
			return fmt.Errorf("failed to write interval metrics: %w", err)
		}
	}

	return nil
}

// Generic retry function with exponential backoff
func retryOperation(operation func() error) error {
	var err error
	for retry := 0; retry < maxRetries; retry++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Log the error
		log.Printf("Attempt %d failed: %v", retry+1, err)

		// If it's the last retry, return the error
		if retry == maxRetries-1 {
			break
		}

		// Wait before next retry
		backoffDuration := exponentialBackoff(retry)
		log.Printf("Retrying in %v", backoffDuration)
		time.Sleep(backoffDuration)
	}

	return err
}
