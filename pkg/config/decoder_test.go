package config_test

import (
	"cni-benchmark/pkg/config"
	"net/url"
	"reflect"

	. "cni-benchmark/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Decoder", func() {
	Context("Args", func() {
		It("should decode valid YAML", func() {
			for input, expected := range map[string]Args{
				"a: b\nc: d":       {"a": "b", "c": "d"},
				"--json: \"true\"": {"--json": "true"},
			} {
				output, err := DecodeArgs(reflect.TypeOf(input), reflect.TypeOf(expected), input)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			}
		})

		It("should fail for non string values and wrong input", func() {
			for _, input := range []any{
				"key: true", "key: 1234", "key: 3.14", "key: null",
				[]string{"array"},
				[]map[string]string{{"a": "b"}},
				true, 3.14,
			} {
				_, err := DecodeArgs(reflect.TypeOf(input), reflect.TypeFor[Args](), input)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("Mode", func() {
		It("should parse a valid mode", func() {
			for input, expected := range map[string]Mode{
				"Client": ModeClient, "client": ModeClient, "CLIENT": ModeClient,
				"server": ModeServer, "Server": ModeServer, "SERVER": ModeServer,
			} {
				output, err := DecodeMode(reflect.TypeOf(input), reflect.TypeOf(expected), input)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			}
		})

		It("should return an error", func() {
			for _, input := range []any{
				"invalid", " client", "!@#$", "",
				[]string{"array"},
				[]map[string]string{{"a": "b"}},
				true, 3.14,
			} {
				_, err := DecodeMode(reflect.TypeOf(input), reflect.TypeFor[Mode](), input)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("Server", func() {
		It("should decode valid domain from single string value", func() {
			for input, expected := range map[string]Address{
				"localhost":   "localhost",
				"example.com": "example.com",
				"a.b.c.d.efg": "a.b.c.d.efg",
			} {
				output, err := DecodeServer(reflect.TypeOf(input), reflect.TypeOf(expected), input)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			}
		})

		It("should return an error for invalid domain", func() {
			for _, input := range []any{
				",", "", "*", "*.*", "invalid..com",
				"*.wildcard.com", "two.domains,go.here",
				[]string{"array"},
				[]map[string]string{{"a": "b"}},
				true, 3.14,
			} {
				_, err := DecodeServer(reflect.TypeOf(input), reflect.TypeFor[Address](), input)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("URL", func() {
		It("should decode valid URL", func() {
			for input, expected := range map[string]*url.URL{
				"http://localhost:1234": {Scheme: "http", Host: "localhost:1234"},
				"https://prom.com":      {Scheme: "https", Host: "prom.com"},
				"http://username:password@example.com:8080/metrics": {
					Scheme: "http", Host: "example.com:8080", Path: "/metrics",
					User: url.UserPassword("username", "password"),
				},
			} {
				output, err := DecodeURL(reflect.TypeOf(input), reflect.TypeOf(expected), input)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			}
		})

		It("should return an error for invalid url", func() {
			for _, input := range []any{
				"localhost", ",", "", "*", "http:///wrong.com", "invalid..com",
				[]string{"array"},
				[]map[string]string{{"a": "b"}},
				true, 3.14,
			} {
				_, err := DecodeURL(reflect.TypeOf(input), reflect.TypeFor[*url.URL](), input)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("InfluxDBTags", func() {
		It("should decode valid YAML", func() {
			for input, expected := range map[string]config.InfluxDBTags{
				"a: b\nc: d": {"a": "b", "c": "d"},
			} {
				output, err := DecodeInfluxDBTags(reflect.TypeOf(input), reflect.TypeOf(expected), input)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			}
		})

		It("should fail for non string values and wrong input", func() {
			for _, input := range []any{
				"key: true", "key: 1234", "key: 3.14", "key: null",
				[]string{"array"},
				[]map[string]string{{"a": "b"}},
				true, 3.14,
			} {
				_, err := DecodeInfluxDBTags(reflect.TypeOf(input), reflect.TypeFor[config.InfluxDBTags](), input)
				Expect(err).To(HaveOccurred())
			}
		})
	})
})
