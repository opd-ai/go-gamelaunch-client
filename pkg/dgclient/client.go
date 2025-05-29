package dgclient

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// GameInfo contains information about an available game
type GameInfo struct {
	Name        string
	Description string
	Command     string
	Available   bool
}

// ClientConfig contains configuration for the client
type ClientConfig struct {
	// SSH client configuration
	SSHConfig *ssh.ClientConfig

	// Connection settings
	ConnectTimeout    time.Duration
	KeepAliveInterval time.Duration

	// Retry settings
	MaxReconnectAttempts int
	ReconnectDelay       time.Duration

	// Terminal settings
	DefaultTerminal string

	// Debug options
	Debug bool
}

// DefaultClientConfig returns a client configuration with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ConnectTimeout:       30 * time.Second,
		KeepAliveInterval:    30 * time.Second,
		MaxReconnectAttempts: 3,
		ReconnectDelay:       5 * time.Second,
		DefaultTerminal:      "xterm-256color",
		Debug:                false,
	}
}

// Client manages connections to dgamelaunch servers
type Client struct {
	config *ClientConfig

	// Connection state
	mu        sync.RWMutex
	sshClient *ssh.Client
	session   Session
	connected bool

	// View management
	view   View
	viewMu sync.RWMutex

	// Current connection info
	host string
	port int

	// Channels for communication
	done   chan struct{}
	errors chan error
}

// NewClient creates a new dgamelaunch client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	return &Client{
		config: config,
		done:   make(chan struct{}),
		errors: make(chan error, 10),
	}
}

// Disconnect closes the connection to the server
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// Close session if exists
	if c.session != nil {
		c.session.Close()
		c.session = nil
	}

	// Close SSH client
	if c.sshClient != nil {
		err := c.sshClient.Close()
		c.sshClient = nil
		c.connected = false
		return err
	}

	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SetView sets the view for rendering game output
func (c *Client) SetView(view View) error {
	c.viewMu.Lock()
	defer c.viewMu.Unlock()

	// Close existing view
	if c.view != nil {
		c.view.Close()
	}

	c.view = view

	// Initialize new view
	if err := view.Init(); err != nil {
		return fmt.Errorf("failed to initialize view: %w", err)
	}

	return nil
}

// SelectGame sends commands to select a specific game
func (c *Client) SelectGame(gameName string) error {
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	if session == nil {
		return ErrSessionNotStarted
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	// Send game selection command
	// This is server-specific and might need customization
	_, err = fmt.Fprintf(stdin, "%s\n", gameName)
	return err
}

// ListGames returns available games (this is a placeholder - actual implementation
// would need to parse server output)
func (c *Client) ListGames() ([]GameInfo, error) {
	// This would typically involve:
	// 1. Sending a list command to the server
	// 2. Parsing the response
	// 3. Returning structured game information

	return []GameInfo{
		{
			Name:        "nethack",
			Description: "NetHack - A classic roguelike",
			Available:   true,
		},
		{
			Name:        "dcss",
			Description: "Dungeon Crawl Stone Soup",
			Available:   true,
		},
	}, nil
}

// keepAlive sends periodic keepalive messages
func (c *Client) keepAlive() {
	ticker := time.NewTicker(c.config.KeepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			client := c.sshClient
			c.mu.RUnlock()

			if client != nil {
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					c.errors <- fmt.Errorf("keepalive failed: %w", err)
					return
				}
			}
		case <-c.done:
			return
		}
	}
}

// Reconnect attempts to reconnect to the server
func (c *Client) Reconnect(auth AuthMethod) error {
	c.mu.Lock()
	host := c.host
	port := c.port
	c.mu.Unlock()

	// Disconnect first
	c.Disconnect()

	// Attempt reconnection with exponential backoff
	var lastErr error
	delay := c.config.ReconnectDelay

	for i := 0; i < c.config.MaxReconnectAttempts; i++ {
		if i > 0 {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}

		err := c.Connect(host, port, auth)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed to reconnect after %d attempts: %w",
		c.config.MaxReconnectAttempts, lastErr)
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	close(c.done)

	c.viewMu.Lock()
	if c.view != nil {
		c.view.Close()
	}
	c.viewMu.Unlock()

	return c.Disconnect()
}
