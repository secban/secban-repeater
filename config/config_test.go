package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/secban/secban-repeater/config"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configContent := `
log_level: debug
socket_path: /tmp/test-repeater.sock
timeout: 3s
agents:
  - http://10.0.0.1:8080
  - unix:///var/run/secban/agent.sock
`

	writeErr := os.WriteFile(filepath.Join(tmpDir, "repeater.yaml"), []byte(configContent), 0o600)
	if writeErr != nil {
		t.Fatalf("failed to write test config: %v", writeErr)
	}

	cfg, loadErr := config.LoadConfig(tmpDir)
	if loadErr != nil {
		t.Fatalf("unexpected error loading config: %v", loadErr)
	}

	if cfg.LogLevel != zerolog.DebugLevel {
		t.Errorf("expected debug log level, got %v", cfg.LogLevel)
	}

	if cfg.SocketPath != "/tmp/test-repeater.sock" {
		t.Errorf("expected socket path /tmp/test-repeater.sock, got %s", cfg.SocketPath)
	}

	if cfg.Timeout != 3*time.Second {
		t.Errorf("expected timeout 3s, got %v", cfg.Timeout)
	}

	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}
}

func TestLoadConfigNoAgents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configContent := `
log_level: info
socket_path: /tmp/test.sock
agents: []
`

	writeErr := os.WriteFile(filepath.Join(tmpDir, "repeater.yaml"), []byte(configContent), 0o600)
	if writeErr != nil {
		t.Fatalf("failed to write test config: %v", writeErr)
	}

	_, loadErr := config.LoadConfig(tmpDir)
	if loadErr == nil {
		t.Fatal("expected error when no agents configured, got nil")
	}
}
