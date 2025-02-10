package config

import (
	"net/url"

	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	viper    *viper.Viper `json:"-" yaml:"-"`
	Lease    Lease        `mapstructure:"lease"`
	InfluxDB InfluxDB     `mapstructure:"influxdb"`
	Command  []string
	TestCase string  `mapstructure:"test_case"`
	Server   Address `mapstructure:"server"`
	Args     Args    `mapstructure:"args"`
	Port     uint16  `mapstructure:"port"`
	Mode     Mode    `mapstructure:"mode"`
}

type InfluxDB struct {
	Url    *url.URL     `mapstructure:"url"`
	Token  string       `mapstructure:"token"`
	Org    string       `mapstructure:"org"`
	Bucket string       `mapstructure:"bucket"`
	Tags   InfluxDBTags `mapstructure:"tags"`
}

type InfluxDBTags map[string]string

type Lease struct {
	Namespace string `mapstructure:"namespace"`
	Name      string `mapstructure:"name"`
}

type Args map[string]string
type Port uint16
type Address string
type Mode uint8

const (
	ModeClient Mode = iota
	ModeServer
)
