package iperf3

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
