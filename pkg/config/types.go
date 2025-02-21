package config

import (
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	viper     *viper.Viper `json:"-" yaml:"-"`
	K8sClient *kubernetes.Clientset
	Lease     Lease `mapstructure:"lease"`
	Command   []string
	// Name of the test case we run
	TestCase string `mapstructure:"test_case"`
	// iperf3 server address
	Server Address `mapstructure:"server"`
	// Database connection string URL is parsed to Dialector
	DatabaseDialector gorm.Dialector `mapstructure:"database_url"`
	// Total test duration
	Duration uint16 `mapstructure:"duration"`
	// Extra args to iperf3
	Args Args `mapstructure:"args"`
	// Port to connect/listen (depending on the mode)
	Port uint16 `mapstructure:"port"`
	// Mode to run in: client or server
	Mode Mode `mapstructure:"mode"`
	// Align all data points starting from midday
	AlignTime bool `mapstructure:"align_time"`
}

type Lease struct {
	Namespace string `mapstructure:"namespace"`
	Name      string `mapstructure:"name"`
}

type (
	Args    map[string]string
	Port    uint16
	Address string
	Mode    uint8
)

const (
	ModeClient Mode = iota
	ModeServer
)
