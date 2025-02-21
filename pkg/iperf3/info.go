package iperf3

import (
	"context"
	"fmt"
	"runtime"

	"github.com/docker/docker/pkg/parsers/kernel"

	config "cni-benchmark/pkg/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (info *Info) Build(cfg *config.Config) (err error) {
	// Get extra info
	kv, err := kernel.GetKernelVersion()
	if err != nil {
		return fmt.Errorf("failed to get kernel info: %w", err)
	}

	client, err := config.BuildKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to make a kubernetes client: %w", err)
	}

	// Fetch ConfigMaps
	cm := map[string]*corev1.ConfigMap{
		"os-info":  nil,
		"k8s-info": nil,
		"cni-info": nil,
	}
	for name := range cm {
		cm[name], err = client.CoreV1().ConfigMaps("default").Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	// Fill fields
	var ok bool
	info.TestCase = cfg.TestCase
	info.OsKernelVersion = kv.String()
	info.OsKernelArch = runtime.GOARCH
	type ref struct {
		ConfigMap *corev1.ConfigMap
		Variable  *string
	}
	m := map[string]ref{
		"OS_NAME":              {cm["os-info"], &info.OsName},
		"OS_VERSION":           {cm["os-info"], &info.OsVersion},
		"K8S_PROVIDER":         {cm["k8s-info"], &info.K8sProvider},
		"K8S_PROVIDER_VERSION": {cm["k8s-info"], &info.K8sProviderVersion},
		"K8S_VERSION":          {cm["k8s-info"], &info.K8sVersion},
		"CNI_NAME":             {cm["cni-info"], &info.CNIName},
		"CNI_VERSION":          {cm["cni-info"], &info.CNIVersion},
		"CNI_DESCRIPTION":      {cm["cni-info"], &info.CNIDescription},
	}
	for field, r := range m {
		if *r.Variable, ok = r.ConfigMap.Data[field]; !ok {
			return fmt.Errorf("could not find %s in %s", field, r.ConfigMap.Name)
		}
	}
	return
}
