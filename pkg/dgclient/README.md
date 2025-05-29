# dgclient
--
    import "github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"


## Usage

```go
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
```

#### type AgentAuth

```go
type AgentAuth struct {
}
```

AgentAuth implements SSH agent-based authentication

#### func (*AgentAuth) GetSSHAuthMethod

```go
func (a *AgentAuth) GetSSHAuthMethod() (ssh.AuthMethod, error)
```

#### func (*AgentAuth) Name

```go
func (a *AgentAuth) Name() string
```

#### type AuthError

```go
type AuthError struct {
	Method string
	Err    error
}
```

AuthError wraps authentication-specific errors

#### func (*AuthError) Error

```go
func (e *AuthError) Error() string
```

#### func (*AuthError) Unwrap

```go
func (e *AuthError) Unwrap() error
```

#### type AuthMethod

```go
type AuthMethod interface {
	// GetSSHAuthMethod returns the underlying SSH auth method
	GetSSHAuthMethod() (ssh.AuthMethod, error)

	// Name returns a human-readable name for the auth method
	Name() string
}
```

AuthMethod defines the interface for SSH authentication methods

#### func  NewAgentAuth

```go
func NewAgentAuth() AuthMethod
```
NewAgentAuth creates a new SSH agent authentication method

#### func  NewInteractiveAuth

```go
func NewInteractiveAuth(callback func(name, instruction string, questions []string, echos []bool) ([]string, error)) AuthMethod
```
NewInteractiveAuth creates a new interactive authentication method

#### func  NewKeyAuth

```go
func NewKeyAuth(keyPath, passphrase string) AuthMethod
```
NewKeyAuth creates a new key authentication method

#### func  NewPasswordAuth

```go
func NewPasswordAuth(password string) AuthMethod
```
NewPasswordAuth creates a new password authentication method

#### type Client

```go
type Client struct {
}
```

Client manages connections to dgamelaunch servers

#### func  NewClient

```go
func NewClient(config *ClientConfig) *Client
```
NewClient creates a new dgamelaunch client

#### func (*Client) Close

```go
func (c *Client) Close() error
```
Close closes the client and cleans up resources

#### func (*Client) Connect

```go
func (c *Client) Connect(host string, port int, auth AuthMethod) error
```
Connect establishes a connection to the dgamelaunch server

#### func (*Client) ConnectWithConn

```go
func (c *Client) ConnectWithConn(conn net.Conn, auth AuthMethod) error
```
ConnectWithConn establishes a connection to the dgamelaunch server using an
existing net.Conn

#### func (*Client) Disconnect

```go
func (c *Client) Disconnect() error
```
Disconnect closes the connection to the server

#### func (*Client) IsConnected

```go
func (c *Client) IsConnected() bool
```
IsConnected returns true if the client is connected

#### func (*Client) ListGames

```go
func (c *Client) ListGames() ([]GameInfo, error)
```
ListGames returns available games (this is a placeholder - actual implementation
would need to parse server output)

#### func (*Client) Reconnect

```go
func (c *Client) Reconnect(auth AuthMethod) error
```
Reconnect attempts to reconnect to the server

#### func (*Client) Run

```go
func (c *Client) Run(ctx context.Context) error
```
Run starts the main game loop with automatic reconnection support

#### func (*Client) SelectGame

```go
func (c *Client) SelectGame(gameName string) error
```
SelectGame sends commands to select a specific game

#### func (*Client) SetView

```go
func (c *Client) SetView(view View) error
```
SetView sets the view for rendering game output

#### type ClientConfig

```go
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
```

ClientConfig contains configuration for the client

#### func  DefaultClientConfig

```go
func DefaultClientConfig() *ClientConfig
```
DefaultClientConfig returns a client configuration with sensible defaults

#### type ConnectionError

```go
type ConnectionError struct {
	Host string
	Port int
	Err  error
}
```

ConnectionError wraps connection-specific errors with additional context

#### func (*ConnectionError) Error

```go
func (e *ConnectionError) Error() string
```

#### func (*ConnectionError) Unwrap

```go
func (e *ConnectionError) Unwrap() error
```

#### type GameInfo

```go
type GameInfo struct {
	Name        string
	Description string
	Command     string
	Available   bool
}
```

GameInfo contains information about an available game

#### type HostKeyCallback

```go
type HostKeyCallback interface {
	Check(hostname string, remote net.Addr, key ssh.PublicKey) error
}
```

HostKeyCallback defines host key verification behavior

#### func  NewKnownHostsCallback

```go
func NewKnownHostsCallback(path string) (HostKeyCallback, error)
```
NewKnownHostsCallback creates a new known hosts callback

#### type InputEvent

```go
type InputEvent struct {
	Type InputEventType
	Data []byte
	Key  string // For special keys
}
```

InputEvent represents a user input event

#### type InputEventType

```go
type InputEventType int
```

InputEventType defines types of input events

```go
const (
	InputEventTypeKey InputEventType = iota
	InputEventTypeResize
	InputEventTypePaste
)
```

#### type InsecureHostKeyCallback

```go
type InsecureHostKeyCallback struct{}
```

InsecureHostKeyCallback accepts any host key (NOT FOR PRODUCTION)

#### func (*InsecureHostKeyCallback) Check

```go
func (i *InsecureHostKeyCallback) Check(hostname string, remote net.Addr, key ssh.PublicKey) error
```

#### type InteractiveAuth

```go
type InteractiveAuth struct {
}
```

InteractiveAuth implements keyboard-interactive authentication

#### func (*InteractiveAuth) GetSSHAuthMethod

```go
func (i *InteractiveAuth) GetSSHAuthMethod() (ssh.AuthMethod, error)
```

#### func (*InteractiveAuth) Name

```go
func (i *InteractiveAuth) Name() string
```

#### type KeyAuth

```go
type KeyAuth struct {
}
```

KeyAuth implements key-based authentication

#### func (*KeyAuth) GetSSHAuthMethod

```go
func (k *KeyAuth) GetSSHAuthMethod() (ssh.AuthMethod, error)
```

#### func (*KeyAuth) Name

```go
func (k *KeyAuth) Name() string
```

#### type KnownHostsCallback

```go
type KnownHostsCallback struct {
}
```

KnownHostsCallback uses a known_hosts file for verification

#### func (*KnownHostsCallback) Check

```go
func (k *KnownHostsCallback) Check(hostname string, remote net.Addr, key ssh.PublicKey) error
```

#### type PasswordAuth

```go
type PasswordAuth struct {
}
```

PasswordAuth implements password-based authentication

#### func (*PasswordAuth) GetSSHAuthMethod

```go
func (p *PasswordAuth) GetSSHAuthMethod() (ssh.AuthMethod, error)
```

#### func (*PasswordAuth) Name

```go
func (p *PasswordAuth) Name() string
```

#### type ResizeEvent

```go
type ResizeEvent struct {
	Width  int
	Height int
}
```

ResizeEvent contains terminal resize information

#### type Session

```go
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
```

Session wraps an SSH session with PTY support

#### func  NewSSHSession

```go
func NewSSHSession(session *ssh.Session) Session
```
NewSSHSession creates a new Session from an ssh.Session

#### type View

```go
type View interface {
	// Init initializes the view
	Init() error

	// Render displays the provided data
	Render(data []byte) error

	// Clear clears the display
	Clear() error

	// SetSize updates the view dimensions
	SetSize(width, height int) error

	// GetSize returns current dimensions
	GetSize() (width, height int)

	// HandleInput reads and returns user input
	// Returns nil, io.EOF when input stream is closed
	HandleInput() ([]byte, error)

	// Close cleans up resources
	Close() error
}
```

View defines the interface for rendering game output and handling input

#### type ViewFactory

```go
type ViewFactory interface {
	CreateView(opts ViewOptions) (View, error)
}
```

ViewFactory creates View instances

#### type ViewFactoryFunc

```go
type ViewFactoryFunc func(opts ViewOptions) (View, error)
```

ViewFactoryFunc is an adapter to allow functions to be used as ViewFactory

#### func (ViewFactoryFunc) CreateView

```go
func (f ViewFactoryFunc) CreateView(opts ViewOptions) (View, error)
```

#### type ViewOptions

```go
type ViewOptions struct {
	// Terminal type (e.g., "xterm-256color", "vt100")
	TerminalType string

	// Initial dimensions
	InitialWidth  int
	InitialHeight int

	// Color support
	ColorEnabled bool

	// Unicode support
	UnicodeEnabled bool

	// Custom configuration
	Config map[string]interface{}
}
```

ViewOptions contains configuration for view creation

#### func  DefaultViewOptions

```go
func DefaultViewOptions() ViewOptions
```
DefaultViewOptions returns sensible defaults for view creation
