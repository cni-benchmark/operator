package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func decodeArgs(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[Args]() {
		return data, nil
	}
	switch f {
	case reflect.TypeFor[string]():
		var rawArgs map[string]any
		if err := yaml.Unmarshal([]byte(data.(string)), &rawArgs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args YAML: %w", err)
		}
		args := Args{}
		for key, value := range rawArgs {
			str, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("all values must be of type string, but key %s has non string value: %v", key, value)
			}
			args[key] = str
		}
		return args, nil
	default:
		return nil, fmt.Errorf("unsupported args type: %T", data)
	}
}

func decodeMode(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[Mode]() {
		return data, nil
	}
	switch f {
	case reflect.TypeFor[string]():
		switch strings.ToLower(data.(string)) {
		case "client":
			return ModeClient, nil
		case "server":
			return ModeServer, nil
		default:
			return nil, fmt.Errorf("unsupported mode: %s", data.(string))
		}
	default:
		return nil, fmt.Errorf("unsupported mode type: %T", data)
	}
}

func decodeServer(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[Address]() {
		return data, nil
	}
	switch f {
	case reflect.TypeFor[string]():
		domainRegex := `^([a-zA-Z0-9-]+\.)*[a-zA-Z]{2,}$`
		str := strings.TrimSpace(data.(string))
		if regexp.MustCompile(domainRegex).MatchString(str) {
			return Address(str), nil
		}
		if net.ParseIP(str) != nil {
			return Address(str), nil
		}
		return nil, fmt.Errorf("server is neither domain nor IP: %s", str)
	default:
		return nil, fmt.Errorf("unsupported server type: %T", data)
	}
}

func decodeURL(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[*url.URL]() {
		return data, nil
	}
	if f != reflect.TypeFor[string]() {
		return nil, fmt.Errorf("invalid URL: expects a string, got %T", data)
	}
	value := data.(string)
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Host == "" || parsedURL.Scheme == "" {
		return nil, errors.New("invalid URL: no scheme or host")
	}
	return parsedURL, nil
}

func decodeDatabaseDialector(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[gorm.Dialector]() {
		return data, nil
	}
	if f != reflect.TypeFor[string]() {
		return nil, fmt.Errorf("unsupported database connection string type: %T", data)
	}
	dsn := data.(string)
	parsedURL, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}
	switch parsedURL.Scheme {
	case "postgres", "postgresql":
		return postgres.Open(dsn), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(strings.Replace(dsn, "sqlite://", "file:", 1)), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", parsedURL.Scheme)
	}
}
