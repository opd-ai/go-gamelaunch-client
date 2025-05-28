package dgclient

// ViewOptions contains configuration for view creation
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

// DefaultViewOptions returns sensible defaults for view creation
func DefaultViewOptions() ViewOptions {
	return ViewOptions{
		TerminalType:   "xterm-256color",
		InitialWidth:   80,
		InitialHeight:  24,
		ColorEnabled:   true,
		UnicodeEnabled: true,
		Config:         make(map[string]interface{}),
	}
}

// View defines the interface for rendering game output and handling input
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

// ViewFactory creates View instances
type ViewFactory interface {
	CreateView(opts ViewOptions) (View, error)
}

// ViewFactoryFunc is an adapter to allow functions to be used as ViewFactory
type ViewFactoryFunc func(opts ViewOptions) (View, error)

func (f ViewFactoryFunc) CreateView(opts ViewOptions) (View, error) {
	return f(opts)
}

// InputEvent represents a user input event
type InputEvent struct {
	Type InputEventType
	Data []byte
	Key  string // For special keys
}

// InputEventType defines types of input events
type InputEventType int

const (
	InputEventTypeKey InputEventType = iota
	InputEventTypeResize
	InputEventTypePaste
)

// ResizeEvent contains terminal resize information
type ResizeEvent struct {
	Width  int
	Height int
}
