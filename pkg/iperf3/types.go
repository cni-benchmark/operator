package iperf3

import (
	"time"
)

type Report struct {
	System struct {
		KernelVersion string `json:"kernel_version"`
		Architecture  string `json:"architecture"`
	} `json:"system"`
	Start struct {
		Version   string `json:"version"`
		Timestamp struct {
			Seconds uint `json:"timesecs"`
		} `json:"timestamp"`
		Test struct {
			Protocol string `json:"protocol"`
		} `json:"test_start"`
	} `json:"start"`
	Intervals []Interval `json:"intervals"`
	End       struct {
		Sent struct {
			ReportSum
			Retransmits uint64 `json:"retransmits"`
		} `json:"sum_sent"`
		Received ReportSum `json:"sum_received"`
	} `json:"end"`
}

type ReportSum struct {
	DurationSeconds float64 `json:"seconds"`
	Bytes           uint64  `json:"bytes"`
	BitsPerSecond   float64 `json:"bits_per_second"`
}

type Interval struct {
	Sum struct {
		Start       float64 `json:"start"`
		End         float64 `json:"end"`
		Retransmits uint64  `json:"retransmits"`
		ReportSum
	} `json:"sum"`
}

// Metric represents interval metrics from iperf3
type Metric struct {
	ID        uint      `gorm:"primaryKey"`
	Timestamp time.Time `gorm:"index;not null"`
	Info

	// Metrics
	BandwidthBps    float64 `gorm:"not null;check:bandwidth_bps >= 0"`
	Bytes           uint64  `gorm:"not null"`
	DurationSeconds float64 `gorm:"not null;check:duration_seconds >= 0"`
	Retransmits     uint64  `gorm:"not null"`
	IntervalStart   float64 `gorm:"not null;check:interval_start >= 0"`
	IntervalEnd     float64 `gorm:"not null;check:interval_end >= interval_start"`
}

// Extra information about the test environment
type Info struct {
	TestCase           string `gorm:"type:varchar(100);index"`
	OsName             string `gorm:"type:varchar(50);index;not null"`
	OsVersion          string `gorm:"type:varchar(50);index;not null"`
	OsKernelArch       string `gorm:"type:varchar(50);index;not null"`
	OsKernelVersion    string `gorm:"type:varchar(100);index;not null"`
	K8sProvider        string `gorm:"type:varchar(50);index;not null"`
	K8sProviderVersion string `gorm:"type:varchar(50);index;not null"`
	K8sVersion         string `gorm:"type:varchar(50);index;not null"`
	CNIName            string `gorm:"type:varchar(50);index;not null"`
	CNIVersion         string `gorm:"type:varchar(50);index;not null"`
	CNIDescription     string `gorm:"type:varchar(200);index;not null"`
	Iperf3Version      string `gorm:"type:varchar(50);index;not null"`
	Iperf3Protocol     string `gorm:"type:varchar(20);index;not null"`
}
