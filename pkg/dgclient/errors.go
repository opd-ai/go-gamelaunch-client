package dgclient

import (
	"errors"
	"fmt"
)

var (
	// Connection errors
	ErrConnectionFailed     = errors.New("connection failed")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrHostKeyMismatch      = errors.New("host key mismatch")
	ErrConnectionTimeout    = errors.New("connection timeout")

	// Session errors
	ErrPTYAllocationFailed = errors.New("PTY allocation failed")
	ErrSessionNotStarted   = errors.New("session not started")
	ErrInvalidTerminalSize = errors.New("invalid terminal size")

	// View errors
	ErrViewNotSet     = errors.New("view not set")
	ErrViewInitFailed = errors.New("view initialization failed")

	// Game errors
	ErrGameNotFound        = errors.New("game not found")
	ErrGameSelectionFailed = errors.New("game selection failed")
)

// ConnectionError wraps connection-specific errors with additional context
type ConnectionError struct {
	Host string
	Port int
	Err  error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection to %s:%d failed: %v", e.Host, e.Port, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// AuthError wraps authentication-specific errors
type AuthError struct {
	Method string
	Err    error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed (method: %s): %v", e.Method, e.Err)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}
