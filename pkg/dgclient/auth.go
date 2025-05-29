package dgclient

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// AuthMethod defines the interface for SSH authentication methods
type AuthMethod interface {
	// GetSSHAuthMethod returns the underlying SSH auth method
	GetSSHAuthMethod() (ssh.AuthMethod, error)

	// Name returns a human-readable name for the auth method
	Name() string
}

// PasswordAuth implements password-based authentication
type PasswordAuth struct {
	password string
}

// NewPasswordAuth creates a new password authentication method
func NewPasswordAuth(password string) AuthMethod {
	return &PasswordAuth{password: password}
}

func (p *PasswordAuth) GetSSHAuthMethod() (ssh.AuthMethod, error) {
	return ssh.Password(p.password), nil
}

func (p *PasswordAuth) Name() string {
	return "password"
}

// KeyAuth implements key-based authentication
type KeyAuth struct {
	keyPath    string
	passphrase string
}

// NewKeyAuth creates a new key authentication method
func NewKeyAuth(keyPath string, passphrase string) AuthMethod {
	return &KeyAuth{
		keyPath:    keyPath,
		passphrase: passphrase,
	}
}

func (k *KeyAuth) GetSSHAuthMethod() (ssh.AuthMethod, error) {
	key, err := os.ReadFile(k.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	var signer ssh.Signer
	if k.passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(k.passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func (k *KeyAuth) Name() string {
	return "key"
}

// AgentAuth implements SSH agent-based authentication
type AgentAuth struct {
	socket string
}

// NewAgentAuth creates a new SSH agent authentication method
func NewAgentAuth() AuthMethod {
	return &AgentAuth{
		socket: os.Getenv("SSH_AUTH_SOCK"),
	}
}

func (a *AgentAuth) GetSSHAuthMethod() (ssh.AuthMethod, error) {
	if a.socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", a.socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

func (a *AgentAuth) Name() string {
	return "agent"
}

// InteractiveAuth implements keyboard-interactive authentication
type InteractiveAuth struct {
	callback func(name, instruction string, questions []string, echos []bool) ([]string, error)
}

// NewInteractiveAuth creates a new interactive authentication method
func NewInteractiveAuth(callback func(name, instruction string, questions []string, echos []bool) ([]string, error)) AuthMethod {
	return &InteractiveAuth{callback: callback}
}

func (i *InteractiveAuth) GetSSHAuthMethod() (ssh.AuthMethod, error) {
	return ssh.KeyboardInteractive(i.callback), nil
}

func (i *InteractiveAuth) Name() string {
	return "keyboard-interactive"
}

// HostKeyCallback defines host key verification behavior
type HostKeyCallback interface {
	Check(hostname string, remote net.Addr, key ssh.PublicKey) error
}

// KnownHostsCallback uses a known_hosts file for verification
type KnownHostsCallback struct {
	path     string
	callback ssh.HostKeyCallback // Store the parsed callback for reuse
}

// NewKnownHostsCallback creates a new known hosts callback
func NewKnownHostsCallback(path string) (HostKeyCallback, error) {
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load known hosts: %w", err)
	}

	// Store the parsed callback to avoid reloading the file
	return &KnownHostsCallback{
		path:     path,
		callback: callback,
	}, nil
}

func (k *KnownHostsCallback) Check(hostname string, remote net.Addr, key ssh.PublicKey) error {
	// Use the pre-parsed callback instead of reloading the file
	return k.callback(hostname, remote, key)
}

// InsecureHostKeyCallback accepts any host key (NOT FOR PRODUCTION)
type InsecureHostKeyCallback struct{}

func (i *InsecureHostKeyCallback) Check(hostname string, remote net.Addr, key ssh.PublicKey) error {
	return nil
}
