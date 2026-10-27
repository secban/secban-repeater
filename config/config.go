package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.yaml.in/yaml/v3"
)

const (
	fileName       = "repeater.yaml"
	defaultTimeout = 5 * time.Second
	defaultSocket  = "/var/run/secban/repeater.sock"
	emptyString    = ""
)

var (
	ErrNoAgents        = errors.New("no agents configured in repeater config")
	ErrUnknownLogLevel = errors.New("unknown log level")
)

type ParsedConfig struct {
	RulesMap   map[string]string `yaml:"rules_map"`
	LogLevel   string            `yaml:"log_level"`
	SocketPath string            `yaml:"socket_path"`
	Timeout    string            `yaml:"timeout"`
	Agents     []string          `yaml:"agents"`
}

type Config struct {
	RulesMap   map[string]string
	SocketPath string
	Agents     []string
	Timeout    time.Duration
	LogLevel   zerolog.Level
}

func LoadConfig(configDir string) (*Config, error) {
	parsed, parseErr := parseConfig(configDir)
	if parseErr != nil {
		return nil, parseErr
	}

	logLevel, logErr := parseLogLevel(parsed.LogLevel)
	if logErr != nil {
		return nil, logErr
	}

	timeout := defaultTimeout

	if parsed.Timeout != emptyString {
		parsedTimeout, timeoutErr := time.ParseDuration(parsed.Timeout)
		if timeoutErr != nil {
			return nil, fmt.Errorf("invalid timeout value: %w", timeoutErr)
		}

		timeout = parsedTimeout
	}

	socketPath := defaultSocket
	if parsed.SocketPath != emptyString {
		socketPath = parsed.SocketPath
	}

	if len(parsed.Agents) == 0 {
		return nil, ErrNoAgents
	}

	rulesMap := make(map[string]string)
	maps.Copy(rulesMap, parsed.RulesMap)

	return &Config{
		Agents:     parsed.Agents,
		LogLevel:   logLevel,
		RulesMap:   rulesMap,
		SocketPath: socketPath,
		Timeout:    timeout,
	}, nil
}

func parseConfig(configDir string) (*ParsedConfig, error) {
	filePath := path.Join(configDir, fileName)

	file, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read repeater config file: %w", readErr)
	}

	var parsed ParsedConfig

	yamlErr := yaml.Unmarshal(file, &parsed)
	if yamlErr != nil {
		return nil, fmt.Errorf("failed to parse repeater config: %w", yamlErr)
	}

	return &parsed, nil
}

func parseLogLevel(raw string) (zerolog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info", emptyString:
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("%w: %s", ErrUnknownLogLevel, raw)
	}
}
