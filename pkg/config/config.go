package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Build initializes the Config by loading from environment variables.
func Build() (*Config, error) {
	cfg := &Config{
		viper:   viper.NewWithOptions(viper.EnvKeyReplacer(&envReplacer{})),
		Port:    5201,
		Lease:   Lease{Namespace: "default", Name: "cni-benchmark"},
		Args:    Args{},
		Command: []string{"iperf3"},
	}

	// Automatically read environment variables
	cfg.viper.AutomaticEnv()

	// Unmarshal the configuration into the struct
	if err := cfg.viper.Unmarshal(cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			DecodeArgs,
			DecodeMode,
			DecodeServer,
			DecodeURL,
			DecodeInfluxDBTags,
		),
	)); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config into struct: %w", err)
	}

	// Set some arguments and check mandatory configuration fields are set
	cfg.Args["--port"] = strconv.Itoa(int(cfg.Port))
	switch cfg.Mode {
	case ModeClient:
		for name, test := range map[string]struct {
			Value   any
			Compare any
		}{
			"InfluxDB URL":          {cfg.InfluxDB.URL, nil},
			"InfluxDB Token":        {cfg.InfluxDB.Token, ""},
			"InfluxDB Organization": {cfg.InfluxDB.Org, ""},
			"InfluxDB Bucket":       {cfg.InfluxDB.Bucket, ""},
		} {
			if test.Value == test.Compare {
				return nil, fmt.Errorf("%s is not set", name)
			}
		}
		cfg.Args["--client"] = string(cfg.Server)
		cfg.Args["--json"] = ""
	case ModeServer:
		cfg.Args["--server"] = ""
	}

	// Prepare full command to run
	for key, value := range cfg.Args {
		cfg.Command = append(cfg.Command, strings.Trim(fmt.Sprintf("%s=%s", key, value), "="))
	}

	// Return nil if everything went fine
	return cfg, nil
}

type envReplacer struct{}

func (r *envReplacer) Replace(s string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_").Replace(s))
}
