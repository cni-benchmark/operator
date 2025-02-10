package iperf3

import (
	config "cni-benchmark/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
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
	cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute iperf3: %w", err)
	}

	// Parse JSON output
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
	return
}

// Analyze JSON output and write metrics to InfluxDB
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
	for key, value := range cfg.InfluxDB.Tags {
		tags[key] = value
	}

	// Write summary metrics
	if err := writeSummaryMetrics(writeAPI, report, tags); err != nil {
		return fmt.Errorf("failed to write summary metrics: %w", err)
	}

	// Write interval metrics
	if err := writeIntervalMetrics(writeAPI, report, tags); err != nil {
		return fmt.Errorf("failed to write interval metrics: %w", err)
	}

	log.Println("Metrics successfully written to InfluxDB")
	return nil
}

func writeSummaryMetrics(writeAPI influxdb2api.WriteAPIBlocking, report *Report, tags map[string]string) error {
	// Calculate timestamp from report start time
	timestamp := time.Unix(int64(report.Start.Timestamp.Seconds), 0)

	// Transmit metrics
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

	if err := writeAPI.WritePoint(context.Background(), txPoint); err != nil {
		return fmt.Errorf("failed to write tx summary metrics: %w", err)
	}

	// Receive metrics
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

	if err := writeAPI.WritePoint(context.Background(), rxPoint); err != nil {
		return fmt.Errorf("failed to write rx summary metrics: %w", err)
	}

	return nil
}

func writeIntervalMetrics(writeAPI influxdb2api.WriteAPIBlocking, report *Report, tags map[string]string) error {
	baseTime := time.Unix(int64(report.Start.Timestamp.Seconds), 0)

	for _, interval := range report.Intervals {
		// Calculate timestamp for this interval
		intervalStart := baseTime.Add(time.Duration(interval.Sum.Start * float64(time.Second)))

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

		if err := writeAPI.WritePoint(context.Background(), point); err != nil {
			return fmt.Errorf("failed to write interval metrics: %w", err)
		}
	}

	return nil
}
