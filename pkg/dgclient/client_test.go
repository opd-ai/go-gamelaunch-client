package dgclient

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)
	defer client.Close()

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.config != config {
		t.Error("Client config not set correctly")
	}

	if client.IsConnected() {
		t.Error("New client should not be connected")
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected ConnectTimeout 30s, got %v", config.ConnectTimeout)
	}

	if config.KeepAliveInterval != 30*time.Second {
		t.Errorf("Expected KeepAliveInterval 30s, got %v", config.KeepAliveInterval)
	}

	if config.MaxReconnectAttempts != 3 {
		t.Errorf("Expected MaxReconnectAttempts 3, got %d", config.MaxReconnectAttempts)
	}

	if config.ReconnectDelay != 5*time.Second {
		t.Errorf("Expected ReconnectDelay 5s, got %v", config.ReconnectDelay)
	}

	if config.DefaultTerminal != "xterm-256color" {
		t.Errorf("Expected DefaultTerminal xterm-256color, got %s", config.DefaultTerminal)
	}
}

func TestClientSetView(t *testing.T) {
	client := NewClient(nil)
	defer client.Close()

	// Mock view implementation
	mockView := &MockView{}

	err := client.SetView(mockView)
	if err != nil {
		t.Fatalf("SetView() failed: %v", err)
	}

	if !mockView.InitCalled {
		t.Error("View.Init() was not called")
	}
}

func TestClientDisconnectWhenNotConnected(t *testing.T) {
	client := NewClient(nil)
	defer client.Close()

	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() on unconnected client should not error, got: %v", err)
	}
}

// MockView implements the View interface for testing
type MockView struct {
	InitCalled   bool
	RenderCalled bool
	InputData    []byte
}

func (m *MockView) Init() error {
	m.InitCalled = true
	return nil
}

func (m *MockView) Render(data []byte) error {
	m.RenderCalled = true
	return nil
}

func (m *MockView) Clear() error {
	return nil
}

func (m *MockView) SetSize(width, height int) error {
	return nil
}

func (m *MockView) GetSize() (width, height int) {
	return 80, 24
}

func (m *MockView) HandleInput() ([]byte, error) {
	return m.InputData, nil
}

func (m *MockView) Close() error {
	return nil
}
