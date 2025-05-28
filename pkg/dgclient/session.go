package dgclient

import (
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Session wraps an SSH session with PTY support
type Session interface {
	// RequestPTY requests a pseudo-terminal
	RequestPTY(term string, h, w int) error

	// WindowChange notifies the server of terminal size changes
	WindowChange(h, w int) error

	// StdinPipe returns a pipe for writing to the session
	StdinPipe() (io.WriteCloser, error)

	// StdoutPipe returns a pipe for reading from the session
	StdoutPipe() (io.Reader, error)

	// StderrPipe returns a pipe for reading stderr from the session
	StderrPipe() (io.Reader, error)

	// Start starts the session with the given command
	Start(cmd string) error

	// Shell starts an interactive shell session
	Shell() error

	// Wait waits for the session to finish
	Wait() error

	// Signal sends a signal to the remote process
	Signal(sig ssh.Signal) error

	// Close closes the session
	Close() error
}

// sshSession implements Session using golang.org/x/crypto/ssh
type sshSession struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader

	mu         sync.Mutex
	started    bool
	ptyRequest *ptyRequestInfo
}

type ptyRequestInfo struct {
	term   string
	height int
	width  int
}

// NewSSHSession creates a new Session from an ssh.Session
func NewSSHSession(session *ssh.Session) Session {
	return &sshSession{
		session: session,
	}
}

func (s *sshSession) RequestPTY(term string, h, w int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("cannot request PTY after session started")
	}

	// SSH PTY request includes terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := s.session.RequestPty(term, h, w, modes); err != nil {
		return fmt.Errorf("PTY request failed: %w", err)
	}

	s.ptyRequest = &ptyRequestInfo{
		term:   term,
		height: h,
		width:  w,
	}

	return nil
}

func (s *sshSession) WindowChange(h, w int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ptyRequest == nil {
		return fmt.Errorf("no PTY requested")
	}

	if err := s.session.WindowChange(h, w); err != nil {
		return fmt.Errorf("window change failed: %w", err)
	}

	s.ptyRequest.height = h
	s.ptyRequest.width = w

	return nil
}

func (s *sshSession) StdinPipe() (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdin != nil {
		return s.stdin, nil
	}

	stdin, err := s.session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.stdin = stdin
	return stdin, nil
}

func (s *sshSession) StdoutPipe() (io.Reader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdout != nil {
		return s.stdout, nil
	}

	stdout, err := s.session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	s.stdout = stdout
	return stdout, nil
}

func (s *sshSession) StderrPipe() (io.Reader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stderr != nil {
		return s.stderr, nil
	}

	stderr, err := s.session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	s.stderr = stderr
	return stderr, nil
}

func (s *sshSession) Start(cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("session already started")
	}

	if err := s.session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	s.started = true
	return nil
}

func (s *sshSession) Shell() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("session already started")
	}

	if err := s.session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	s.started = true
	return nil
}

func (s *sshSession) Wait() error {
	if err := s.session.Wait(); err != nil {
		return fmt.Errorf("session wait failed: %w", err)
	}
	return nil
}

func (s *sshSession) Signal(sig ssh.Signal) error {
	if err := s.session.Signal(sig); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}
	return nil
}

func (s *sshSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdin != nil {
		s.stdin.Close()
	}

	if err := s.session.Close(); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}
