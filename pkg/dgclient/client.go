package dgclient

import (
	"context"
	"fmt"
	"io"
	"net"
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

// Connect establishes a connection to the dgamelaunch server
func (c *Client) Connect(host string, port int, auth AuthMethod) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Build SSH client config
	sshAuth, err := auth.GetSSHAuthMethod()
	if err != nil {
		return &AuthError{Method: auth.Name(), Err: err}
	}

	config := &ssh.ClientConfig{
		User:            c.config.SSHConfig.User,
		Auth:            []ssh.AuthMethod{sshAuth},
		HostKeyCallback: c.config.SSHConfig.HostKeyCallback,
		Timeout:         c.config.ConnectTimeout,
	}

	// Connect with timeout
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, c.config.ConnectTimeout)
	if err != nil {
		return &ConnectionError{Host: host, Port: port, Err: err}
	}

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return &ConnectionError{Host: host, Port: port, Err: err}
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)
	c.host = host
	c.port = port
	c.connected = true

	// Start keepalive routine
	go c.keepAlive()

	return nil
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

// Run starts the main game loop
func (c *Client) Run(ctx context.Context) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return fmt.Errorf("not connected")
	}

	if c.view == nil {
		c.mu.Unlock()
		return ErrViewNotSet
	}

	// Create new session
	sshSession, err := c.sshClient.NewSession()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.session = NewSSHSession(sshSession)
	c.mu.Unlock()

	// Set up PTY
	width, height := c.view.GetSize()
	if err := c.session.RequestPTY(c.config.DefaultTerminal, height, width); err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes
	stdin, err := c.session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := c.session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start shell
	if err := c.session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Create error group for concurrent operations
	errCh := make(chan error, 3)

	// Handle output
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("stdout read error: %w", err)
				}
				return
			}

			if err := c.view.Render(buf[:n]); err != nil {
				errCh <- fmt.Errorf("render error: %w", err)
				return
			}
		}
	}()

	// Handle input
	go func() {
		for {
			input, err := c.view.HandleInput()
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("input error: %w", err)
				}
				return
			}

			if _, err := stdin.Write(input); err != nil {
				errCh <- fmt.Errorf("stdin write error: %w", err)
				return
			}
		}
	}()

	// Handle window resize
	go func() {
		// This would typically involve watching for resize events
		// Implementation depends on the view
	}()

	// Wait for completion or error
	select {
	case <-ctx.Done():
		c.session.Close()
		return ctx.Err()
	case err := <-errCh:
		c.session.Close()
		return err
	case <-c.done:
		return nil
	}
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
