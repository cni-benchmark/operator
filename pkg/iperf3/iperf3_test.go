package iperf3_test

import (
	"cni-benchmark/pkg/config"
	"cni-benchmark/pkg/iperf3"
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIperf3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iperf3")
}

var _ = Describe("iperf3", func() {
	var cfg *config.Config
	var err error
	env := map[string]string{
		"MODE":            "client",
		"SERVER":          "example.com",
		"PORT":            "80",
		"INFLUXDB_URL":    "http://stub.com",
		"INFLUXDB_TOKEN":  "stub",
		"INFLUXDB_ORG":    "stub",
		"INFLUXDB_BUCKET": "stub",
	}

	BeforeEach(func() {
		for name, value := range env {
			Expect(os.Setenv(name, value)).To(Succeed())
		}
		cfg, err = config.Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())
	})

	AfterEach(func() {
		for name := range env {
			Expect(os.Unsetenv(name)).To(Succeed())
		}
	})

	Context("WaitForServer", func() {
		It("should end waiting for the server without an error", func() {
			err = iperf3.WaitForServer(context.Background(), cfg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail on timeout", func() {
			cfg.Port = 1234
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := iperf3.WaitForServer(ctx, cfg)
			Expect(err).To(HaveOccurred())
		})
	})
})
