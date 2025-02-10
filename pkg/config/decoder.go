package config

import (
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func DecodeArgs(f reflect.Type, t reflect.Type, data any) (any, error) {
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

func DecodeMode(f reflect.Type, t reflect.Type, data any) (any, error) {
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

func DecodeServer(f reflect.Type, t reflect.Type, data any) (any, error) {
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

func DecodeURL(f reflect.Type, t reflect.Type, data any) (any, error) {
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
		return nil, fmt.Errorf("invalid URL: no scheme or host")
	}
	return parsedURL, nil
}

func DecodeInfluxDBTags(f reflect.Type, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeFor[InfluxDBTags]() {
		return data, nil
	}
	switch f {
	case reflect.TypeFor[string]():
		var rawTags map[string]any
		if err := yaml.Unmarshal([]byte(data.(string)), &rawTags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args YAML: %w", err)
		}
		tags := InfluxDBTags{}
		for key, value := range rawTags {
			str, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("all values must be of type string, but key %s has non string value: %v", key, value)
			}
			tags[key] = str
		}
		return tags, nil
	default:
		return nil, fmt.Errorf("unsupported tags type: %T", data)
	}
}
