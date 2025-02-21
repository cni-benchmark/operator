package iperf3

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"gorm.io/gorm"

	config "cni-benchmark/pkg/config"
)

func Store(cfg *config.Config, report *Report, info *Info) (err error) {
	if cfg == nil {
		return errors.New("configuration is required")
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 5 * time.Minute
	b.InitialInterval = 100 * time.Millisecond
	b.MaxInterval = 2 * time.Second

	operation := func() error {
		db, err := gorm.Open(cfg.DatabaseDialector, &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		if err = db.AutoMigrate(&Metric{}); err != nil {
			return fmt.Errorf("failed to migrate database: %w", err)
		}

		return storeWithTransaction(cfg, db, report, info)
	}

	if err = backoff.Retry(operation, b); err != nil {
		return fmt.Errorf("failed to store metrics after retries: %w", err)
	}

	log.Println("Metrics successfully written to the database")
	return
}

func storeWithTransaction(cfg *config.Config, db *gorm.DB, report *Report, info *Info) error {
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("recovered from panic in storeWithTransaction: %v", r)
		}
	}()

	// If AlignTime is true, set baseTime to 12:00 of the current day
	var baseTime time.Time
	if cfg.AlignTime {
		now := time.Now()
		baseTime = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	} else {
		// Otherwise, use the timestamp from report.Start
		baseTime = time.Unix(int64(report.Start.Timestamp.Seconds), 0)
	}
	var metrics []*Metric

	for _, interval := range report.Intervals {
		intervalStart := baseTime.Add(time.Duration(interval.Sum.Start * float64(time.Second)))
		info.Iperf3Version = report.Start.Version
		info.Iperf3Protocol = report.Start.Test.Protocol
		metrics = append(metrics, &Metric{
			Timestamp:       intervalStart,
			BandwidthBps:    interval.Sum.BitsPerSecond,
			Bytes:           interval.Sum.Bytes,
			DurationSeconds: interval.Sum.DurationSeconds,
			Retransmits:     interval.Sum.Retransmits,
			IntervalStart:   interval.Sum.Start,
			IntervalEnd:     interval.Sum.End,
			Info:            *info,
		})
	}

	if err := tx.Create(&metrics).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create interval metrics: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
