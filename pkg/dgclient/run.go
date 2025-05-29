package dgclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Run starts the main game loop with automatic reconnection support
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

	// Store auth method for reconnection
	var lastAuth AuthMethod
	c.mu.Unlock()

	// Main session loop with reconnection
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create new session with proper synchronization
		c.mu.Lock()
		if !c.connected || c.sshClient == nil {
			c.mu.Unlock()
			return fmt.Errorf("not connected")
		}

		// Capture sshClient reference while holding lock to prevent TOCTOU
		sshClient := c.sshClient
		c.mu.Unlock()

		// Now safely use the captured reference
		sshSession, err := sshClient.NewSession()
		if err != nil {
			// Try to reconnect if session creation fails
			if reconnectErr := c.handleReconnection(lastAuth, err); reconnectErr != nil {
				return fmt.Errorf("failed to create session and reconnect failed: %v (original: %v)", reconnectErr, err)
			}
			continue // Retry with new connection
		}

		c.mu.Lock()
		c.session = NewSSHSession(sshSession)
		c.mu.Unlock()

		// Run session
		sessionErr := c.runSession(ctx)

		// Close current session
		c.mu.Lock()
		if c.session != nil {
			c.session.Close()
			c.session = nil
		}
		c.mu.Unlock()

		// Handle session errors
		if sessionErr != nil {
			if sessionErr == ctx.Err() {
				return sessionErr // Context cancellation, don't reconnect
			}

			// Check if this is a connection error that warrants reconnection
			if c.shouldReconnect(sessionErr) {
				if c.config.Debug {
					fmt.Printf("Session error occurred, attempting reconnection: %v\n", sessionErr)
				}

				if reconnectErr := c.handleReconnection(lastAuth, sessionErr); reconnectErr != nil {
					return fmt.Errorf("session failed and reconnect failed: %v (original: %v)", reconnectErr, sessionErr)
				}

				if c.config.Debug {
					fmt.Println("Reconnection successful, resuming session...")
				}
				continue // Retry with new connection
			}

			return sessionErr // Non-recoverable error
		}

		// Session completed normally
		return nil
	}
}

// runSession handles a single session lifecycle
func (c *Client) runSession(ctx context.Context) error {
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

	// Create error channel for concurrent operations
	errCh := make(chan error, 3)
	sessionDone := make(chan struct{})

	// Handle output
	go func() {
		defer close(sessionDone)
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
			select {
			case <-sessionDone:
				return
			case <-ctx.Done():
				return
			default:
			}

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
		// Monitor for resize events - this is a simplified version
		// A full implementation would use platform-specific signal handling
		for {
			select {
			case <-sessionDone:
				return
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Check if view size changed
				newWidth, newHeight := c.view.GetSize()
				if newWidth != width || newHeight != height {
					width, height = newWidth, newHeight
					c.session.WindowChange(height, width)
				}
			}
		}
	}()

	// Wait for completion or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-sessionDone:
		return nil
	}
}

// shouldReconnect determines if an error warrants a reconnection attempt
func (c *Client) shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	// Check for network-related errors
	errorStr := err.Error()
	networkErrors := []string{
		"connection reset",
		"broken pipe",
		"connection refused",
		"no route to host",
		"network is unreachable",
		"connection timed out",
		"EOF",
		"ssh: disconnect",
		"ssh: connection lost",
	}

	for _, netErr := range networkErrors {
		if strings.Contains(strings.ToLower(errorStr), netErr) {
			return true
		}
	}

	return false
}

// handleReconnection manages the reconnection process
func (c *Client) handleReconnection(lastAuth AuthMethod, originalErr error) error {
	if c.config.MaxReconnectAttempts <= 0 {
		return fmt.Errorf("reconnection disabled")
	}

	c.mu.Lock()
	host := c.host
	port := c.port
	c.mu.Unlock()

	if c.config.Debug {
		fmt.Printf("Connection lost (%v), attempting to reconnect to %s:%d\n", originalErr, host, port)
	}

	// Disconnect current connection
	c.Disconnect()

	// If no auth method stored, try to detect from config
	if lastAuth == nil {
		// Try SSH agent first
		if os.Getenv("SSH_AUTH_SOCK") != "" {
			lastAuth = NewAgentAuth()
		} else {
			return fmt.Errorf("no authentication method available for reconnection")
		}
	}

	// Attempt reconnection with exponential backoff
	delay := c.config.ReconnectDelay
	for i := 0; i < c.config.MaxReconnectAttempts; i++ {
		if i > 0 {
			if c.config.Debug {
				fmt.Printf("Reconnection attempt %d/%d in %v...\n", i+1, c.config.MaxReconnectAttempts, delay)
			}
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * 1.5) // Exponential backoff
		}

		err := c.Connect(host, port, lastAuth)
		if err == nil {
			if c.config.Debug {
				fmt.Printf("Reconnection successful on attempt %d\n", i+1)
			}
			return nil
		}

		if c.config.Debug {
			fmt.Printf("Reconnection attempt %d failed: %v\n", i+1, err)
		}
	}

	return fmt.Errorf("failed to reconnect after %d attempts", c.config.MaxReconnectAttempts)
}

// ConnectWithConn establishes a connection to the dgamelaunch server using an existing net.Conn
func (c *Client) ConnectWithConn(conn net.Conn, auth AuthMethod) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		// Allow reconnection by first disconnecting
		if c.sshClient != nil {
			c.sshClient.Close()
			c.sshClient = nil
		}
		c.connected = false
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

	// Perform SSH handshake on existing connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
	if err != nil {
		conn.Close()
		return &ConnectionError{Host: conn.RemoteAddr().String(), Port: 0, Err: err}
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)
	// For net.Conn, we'll store the remote address info
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		c.host = tcpAddr.IP.String()
		c.port = tcpAddr.Port
	} else {
		// Fallback for non-TCP connections
		c.host = conn.RemoteAddr().String()
		c.port = 0
	}
	c.connected = true

	// Start keepalive routine
	go c.keepAlive()

	return nil
}

// Connect establishes a connection to the dgamelaunch server
func (c *Client) Connect(host string, port int, auth AuthMethod) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		// Allow reconnection by first disconnecting
		if c.sshClient != nil {
			c.sshClient.Close()
			c.sshClient = nil
		}
		c.connected = false
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
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
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
