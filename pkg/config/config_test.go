package config_test

import (
	"cni-benchmark/pkg/config"
	"os"
	"testing"

	. "cni-benchmark/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config")
}

var _ = Describe("Configuration", func() {
	var cfg *Config
	var err error
	env := map[string]string{
		"MODE":            "client",
		"SERVER":          "example.com",
		"PORT":            "80",
		"LEASE_NAME":      "test",
		"LEASE_NAMESPACE": "test",
		"INFLUXDB_URL":    "http://example.com/path",
		"INFLUXDB_TOKEN":  "test-token",
		"INFLUXDB_ORG":    "test-org",
		"INFLUXDB_BUCKET": "test-bucket",
		"INFLUXDB_TAGS":   "some: tag\nanother: one",
		"ARGS":            "--help: ''\nkey: value",
		"TEST_CASE":       "01-p2sh-tcp",
	}

	BeforeEach(func() {
		for name, value := range env {
			Expect(os.Setenv(name, value)).To(Succeed())
		}
		cfg, err = Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())
	})

	AfterEach(func() {
		for name := range env {
			Expect(os.Unsetenv(name)).To(Succeed())
		}
	})

	It("should parse all configuration fields from environment", func() {
		Expect(cfg.Mode).To(Equal(ModeClient))
		Expect(cfg.Server).To(Equal(Address("example.com")))
		Expect(cfg.Port).To(Equal(uint16(80)))
		Expect(cfg.Lease.Namespace).To(Equal("test"))
		Expect(cfg.Lease.Name).To(Equal("test"))
		Expect(cfg.InfluxDB.URL.Scheme).To(Equal("http"))
		Expect(cfg.InfluxDB.URL.Host).To(Equal("example.com"))
		Expect(cfg.InfluxDB.URL.Path).To(Equal("/path"))
		Expect(cfg.InfluxDB.Token).To(Equal("test-token"))
		Expect(cfg.InfluxDB.Org).To(Equal("test-org"))
		Expect(cfg.InfluxDB.Bucket).To(Equal("test-bucket"))
		Expect(cfg.InfluxDB.Tags).To(Equal(config.InfluxDBTags{"some": "tag", "another": "one"}))
		Expect(cfg.Args).To(Equal(Args{
			"--json":   "",
			"--help":   "",
			"key":      "value",
			"--client": "example.com",
			"--port":   "80",
		}))
		Expect(cfg.Command).To(ConsistOf(
			"iperf3", "--json", "--help", "key=value", "--port=80", "--client=example.com",
		))
		Expect(cfg.TestCase).To(Equal("01-p2sh-tcp"))
	})
})
