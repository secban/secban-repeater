package config

func DefaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		Profiler:   "",
		LogLevel:   "info",
		SocketPath: "/var/run/secban/repeater.sock",
		Timeout:    "5s",
		Agents:     []string{},
		RulesMap:   make(map[string]string),
	}
}
