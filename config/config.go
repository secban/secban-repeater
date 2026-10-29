package config

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/secban/secban-agent/configloader"
)

const (
	fileName = "repeater.yaml"
)

var (
	ErrNoAgents        = errors.New("no agents configured in repeater config")
	ErrUnknownLogLevel = errors.New("unknown log level")
)

type ParsedConfig struct {
	Profiler   string            `koanf:"profiler"`
	RulesMap   map[string]string `koanf:"rules_map"`
	LogLevel   string            `koanf:"log_level"`
	SocketPath string            `koanf:"socket_path"`
	Timeout    string            `koanf:"timeout"`
	Agents     []string          `koanf:"agents"`
}

type Config struct {
	RulesMap   map[string]string
	SocketPath string
	Profiler   string
	Agents     []string
	Timeout    time.Duration
	LogLevel   zerolog.Level
}

func LoadConfig(configDir string) (Config, error) {
	parsed, parseErr := parseConfig(configDir)
	if parseErr != nil {
		return Config{}, parseErr
	}

	logLevel, logLevelErr := configloader.TransformLogLevel(parsed.LogLevel)
	if logLevelErr != nil {
		return Config{}, logLevelErr
	}

	parsedTimeout, timeoutErr := time.ParseDuration(parsed.Timeout)
	if timeoutErr != nil {
		return Config{}, fmt.Errorf("invalid timeout value: %w", timeoutErr)
	}

	if len(parsed.Agents) == 0 {
		return Config{}, ErrNoAgents
	}

	rulesMap := make(map[string]string)
	maps.Copy(rulesMap, parsed.RulesMap)

	return Config{
		Agents:     parsed.Agents,
		LogLevel:   logLevel,
		RulesMap:   rulesMap,
		SocketPath: parsed.SocketPath,
		Profiler:   parsed.Profiler,
		Timeout:    parsedTimeout,
	}, nil
}

func parseConfig(configDir string) (*ParsedConfig, error) {
	filePath := path.Join(configDir, fileName)

	parsed, parseErr := configloader.ParseConfig(DefaultParsedConfig(), filePath, nil)
	if parseErr != nil {
		return nil, parseErr
	}

	return &parsed, nil
}
