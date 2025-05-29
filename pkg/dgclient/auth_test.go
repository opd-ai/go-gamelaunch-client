package dgclient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordAuth(t *testing.T) {
	password := "testpassword"
	auth := NewPasswordAuth(password)

	if auth.Name() != "password" {
		t.Errorf("Expected name 'password', got '%s'", auth.Name())
	}

	sshAuth, err := auth.GetSSHAuthMethod()
	if err != nil {
		t.Fatalf("GetSSHAuthMethod() failed: %v", err)
	}

	if sshAuth == nil {
		t.Error("GetSSHAuthMethod() returned nil")
	}
}

func TestAgentAuth(t *testing.T) {
	auth := NewAgentAuth()

	if auth.Name() != "agent" {
		t.Errorf("Expected name 'agent', got '%s'", auth.Name())
	}

	// This will fail without SSH_AUTH_SOCK, which is expected
	_, err := auth.GetSSHAuthMethod()
	if err == nil && os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Error("Expected error when SSH_AUTH_SOCK not set")
	}
}

func TestKeyAuth(t *testing.T) {
	// Create a temporary key file for testing
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key")

	// Create a dummy private key (this won't be valid for actual SSH)
	keyContent := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEAwJbykjmz1Q7G8aK1K5f3hG4OlJj5EKy1V8sZ9xbJQZbZoFpgW7
-----END OPENSSH PRIVATE KEY-----`

	err := os.WriteFile(keyPath, []byte(keyContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	auth := NewKeyAuth(keyPath, "")

	if auth.Name() != "key" {
		t.Errorf("Expected name 'key', got '%s'", auth.Name())
	}

	// This will fail with invalid key format, which is expected for our dummy key
	_, err = auth.GetSSHAuthMethod()
	if err == nil {
		t.Error("Expected error with invalid key format")
	}
}

func TestKeyAuthNonexistentFile(t *testing.T) {
	auth := NewKeyAuth("/nonexistent/path", "")

	_, err := auth.GetSSHAuthMethod()
	if err == nil {
		t.Error("Expected error with nonexistent key file")
	}
}
