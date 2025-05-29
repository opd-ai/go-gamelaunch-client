package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
default_server: test-server
servers:
  test-server:
    host: example.com
    port: 22
    username: testuser
    auth:
      method: password
preferences:
  terminal: xterm-256color
  reconnect_attempts: 3
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.DefaultServer != "test-server" {
		t.Errorf("Expected default_server 'test-server', got '%s'", config.DefaultServer)
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.Servers))
	}

	server := config.Servers["test-server"]
	if server.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", server.Host)
	}

	if server.Port != 22 {
		t.Errorf("Expected port 22, got %d", server.Port)
	}

	if server.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", server.Username)
	}

	if server.Auth.Method != "password" {
		t.Errorf("Expected auth method 'password', got '%s'", server.Auth.Method)
	}
}

func TestLoadConfigNonexistent(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path")
	if err == nil {
		t.Error("Expected error when loading nonexistent config file")
	}
}

func TestValidateConfig(t *testing.T) {
	validConfig := &Config{
		DefaultServer: "test-server",
		Servers: map[string]ServerConfig{
			"test-server": {
				Host:     "example.com",
				Port:     22,
				Username: "testuser",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateConfig() failed for valid config: %v", err)
	}
}

func TestValidateConfigNilConfig(t *testing.T) {
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestValidateConfigNoServers(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for config with no servers")
	}
}

func TestValidateConfigMissingHost(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Username: "testuser",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for server with missing host")
	}
}

func TestValidateConfigMissingUsername(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Host: "example.com",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for server with missing username")
	}
}

func TestValidateConfigKeyAuthMissingPath(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Host:     "example.com",
				Username: "testuser",
				Auth: AuthConfig{
					Method: "key",
					// KeyPath missing
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for key auth with missing key_path")
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	config := GenerateExampleConfig()

	if config == nil {
		t.Fatal("GenerateExampleConfig() returned nil")
	}

	if len(config.Servers) == 0 {
		t.Error("Example config should have servers")
	}

	if config.DefaultServer == "" {
		t.Error("Example config should have default_server set")
	}

	// Validate that the generated config is valid
	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Generated example config is invalid: %v", err)
	}
}
