package config

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
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
		"DURATION":        "1234",
		"LEASE_ID":        "test",
		"LEASE_NAME":      "test",
		"LEASE_NAMESPACE": "test",
		"DATABASE_URL":    "sqlite://:memory:?cache=shared",
		"ARGS":            "--help: ''\nkey: value",
		"TEST_CASE":       "01-p2sh-tcp",
		"ALIGN_TIME":      "false",
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
		Expect(cfg.Duration).To(Equal(uint16(1234)))
		Expect(cfg.AlignTime).To(BeFalse())
		Expect(cfg.Lease.Namespace).To(Equal("test"))
		Expect(cfg.Lease.Name).To(Equal("test"))
		Expect(cfg.Lease.ID).To(Equal("test"))
		Expect(cfg.DatabaseDialector).ToNot(BeNil())
		Expect(cfg.DatabaseDialector).To(Equal(sqlite.Open("file::memory:?cache=shared")))
		Expect(cfg.Args).To(Equal(Args{
			"--json":   "",
			"--help":   "",
			"key":      "value",
			"--client": "example.com",
			"--port":   "80",
			"--time":   "1234",
		}))
		Expect(cfg.Command).To(ConsistOf(
			"iperf3", "--json", "--help", "key=value", "--port=80", "--client=example.com", "--time=1234",
		))
		Expect(cfg.TestCase).To(Equal("01-p2sh-tcp"))
	})
})
