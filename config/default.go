package config

func defaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		Profiler:   "localhost:6063",
		LogLevel:   "info",
		SocketPath: "/var/run/secban/repeater.sock",
		Timeout:    "5s",
		Agents:     []string{},
		RulesMap:   make(map[string]string),
	}
}
