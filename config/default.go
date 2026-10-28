package config

func defaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		LogLevel:   "info",
		SocketPath: defaultSocket,
		Timeout:    "5s",
		Agents:     []string{},
		RulesMap:   make(map[string]string),
	}
}
