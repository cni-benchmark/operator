package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Build initializes the Config by loading from environment variables.
func Build() (cfg *Config, err error) {
	cfg = &Config{
		viper:     viper.NewWithOptions(viper.EnvKeyReplacer(&envReplacer{})),
		Port:      5201,
		Lease:     Lease{Namespace: "default", Name: "cni-benchmark"},
		Args:      Args{},
		AlignTime: true,
		Duration:  10,
		Command:   []string{"iperf3"},
	}

	// Automatically read environment variables
	cfg.viper.AutomaticEnv()

	// Unmarshal the configuration into the struct
	if err = cfg.viper.Unmarshal(cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			decodeArgs,
			decodeMode,
			decodeServer,
			decodeURL,
			decodeDatabaseDialector,
		),
	)); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config into struct: %w", err)
	}

	// Set some arguments and check mandatory configuration fields are set
	cfg.Args["--port"] = strconv.Itoa(int(cfg.Port))
	switch cfg.Mode {
	case ModeClient:
		if cfg.DatabaseDialector == nil {
			return nil, errors.New("database connection string is not set")
		}
		cfg.Args["--client"] = string(cfg.Server)
		cfg.Args["--time"] = strconv.Itoa(int(cfg.Duration))
		cfg.Args["--json"] = ""
	case ModeServer:
		cfg.Args["--server"] = ""
	}

	// Prepare full command to run
	for key, value := range cfg.Args {
		cfg.Command = append(cfg.Command, strings.Trim(fmt.Sprintf("%s=%s", key, value), "="))
	}

	return
}

type envReplacer struct{}

func (r *envReplacer) Replace(s string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_").Replace(s))
}

func BuildKubernetesClient() (client *kubernetes.Clientset, err error) {
	// Create kubernetes client
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, fmt.Errorf("failed to get KUBECONFIG: %w", err)
	}

	client, err = kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	return
}
