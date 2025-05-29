package tui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
)

// TerminalView implements dgclient.View using tcell for terminal rendering
type TerminalView struct {
	screen   tcell.Screen
	emulator *TerminalEmulator

	mu     sync.Mutex
	width  int
	height int

	inputCh chan []byte
	quitCh  chan struct{}

	// Options
	opts dgclient.ViewOptions
}

// NewTerminalView creates a new terminal-based view
func NewTerminalView(opts dgclient.ViewOptions) (dgclient.View, error) {
	return &TerminalView{
		opts:    opts,
		inputCh: make(chan []byte, 100),
		quitCh:  make(chan struct{}),
	}, nil
}

// Init initializes the terminal view
func (v *TerminalView) Init() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}

	v.screen = screen
	v.width, v.height = screen.Size()

	// Create terminal emulator
	v.emulator = NewTerminalEmulator(v.width, v.height)

	// Set up event handling
	go v.handleEvents()

	// Clear screen
	v.screen.Clear()
	v.screen.Show()

	return nil
}

// Render displays the provided data
func (v *TerminalView) Render(data []byte) error {
	// Process data without holding locks
	v.emulator.ProcessData(data)
	screenData := v.emulator.GetScreen()
	cursorX, cursorY := v.emulator.GetCursor()

	// Atomic state check and screen validation
	v.mu.Lock()
	screen := v.screen
	if screen == nil {
		v.mu.Unlock()
		return fmt.Errorf("screen not initialized")
	}
	v.mu.Unlock()

	// Perform all screen operations without holding mutex
	screen.Clear()

	for y, row := range screenData {
		for x, cell := range row {
			style := v.cellToTcellStyle(cell.Attr)
			screen.SetContent(x, y, cell.Char, nil, style)
		}
	}

	screen.ShowCursor(cursorX, cursorY)
	screen.Show()

	return nil
}

// cellToTcellStyle converts cell attributes to tcell style
func (v *TerminalView) cellToTcellStyle(attr CellAttributes) tcell.Style {
	style := tcell.StyleDefault

	// Convert colors
	fg := tcell.NewRGBColor(int32(attr.Foreground.R), int32(attr.Foreground.G), int32(attr.Foreground.B))
	bg := tcell.NewRGBColor(int32(attr.Background.R), int32(attr.Background.G), int32(attr.Background.B))

	style = style.Foreground(fg).Background(bg)

	// Apply attributes
	if attr.Bold {
		style = style.Bold(true)
	}
	if attr.Underline {
		style = style.Underline(true)
	}
	if attr.Reverse {
		style = style.Reverse(true)
	}

	return style
}

// Clear clears the display
func (v *TerminalView) Clear() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen == nil {
		return fmt.Errorf("screen not initialized")
	}

	v.screen.Clear()
	if v.emulator != nil {
		v.emulator.eraseScreen()
	}
	v.screen.Show()
	return nil
}

// SetSize updates the view dimensions
func (v *TerminalView) SetSize(width, height int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.width = width
	v.height = height

	if v.emulator != nil {
		v.emulator.Resize(width, height)
	}

	return nil
}

// GetSize returns current dimensions
func (v *TerminalView) GetSize() (width, height int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.width, v.height
}

// HandleInput reads and returns user input
func (v *TerminalView) HandleInput() ([]byte, error) {
	select {
	case input := <-v.inputCh:
		return input, nil
	case <-v.quitCh:
		return nil, io.EOF
	}
}

// Close cleans up resources
func (v *TerminalView) Close() error {
	close(v.quitCh)

	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen != nil {
		v.screen.Fini()
		v.screen = nil
	}

	return nil
}

// handleEvents processes terminal events with non-blocking polling
func (v *TerminalView) handleEvents() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60fps for responsive UI
	defer ticker.Stop()

	for {
		select {
		case <-v.quitCh:
			return
		case <-ticker.C:
			// Non-blocking event processing with proper synchronization
			for {
				// Check if screen is available without holding mutex during PollEvent
				v.mu.Lock()
				screen := v.screen
				if screen == nil {
					v.mu.Unlock()
					return // Screen has been closed, exit gracefully
				}
				v.mu.Unlock()

				// Call PollEvent without holding the mutex to prevent deadlock
				event := screen.PollEvent()

				if event == nil {
					break // No more events available
				}

				v.processEvent(event) // Extracted for testability
			}
		}
	}
}

// processEvent handles a single event - extracted for testing
func (v *TerminalView) processEvent(event tcell.Event) {
	switch ev := event.(type) {
	case *tcell.EventKey:
		v.handleKeyEvent(ev) // Now actually called
	case *tcell.EventResize:
		// Capture new dimensions
		newWidth, newHeight := ev.Size()

		// Atomic update of internal state
		v.mu.Lock()
		v.width, v.height = newWidth, newHeight
		if v.emulator != nil {
			v.emulator.Resize(newWidth, newHeight)
		}
		v.mu.Unlock()

		// Screen sync without holding mutex
		v.screen.Sync()
	}
}

// handleKeyEvent processes keyboard input
func (v *TerminalView) handleKeyEvent(ev *tcell.EventKey) {
	var data []byte

	// Handle special keys
	switch ev.Key() {
	case tcell.KeyRune:
		data = []byte(string(ev.Rune()))
	case tcell.KeyEnter:
		data = []byte("\r")
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		data = []byte{8} // ASCII backspace
	case tcell.KeyTab:
		data = []byte{9} // ASCII tab
	case tcell.KeyEscape:
		data = []byte{27} // ASCII escape
	case tcell.KeyUp:
		data = []byte{27, '[', 'A'}
	case tcell.KeyDown:
		data = []byte{27, '[', 'B'}
	case tcell.KeyRight:
		data = []byte{27, '[', 'C'}
	case tcell.KeyLeft:
		data = []byte{27, '[', 'D'}
	case tcell.KeyCtrlC:
		data = []byte{3} // ASCII ETX (Ctrl+C)
	case tcell.KeyCtrlD:
		data = []byte{4} // ASCII EOT (Ctrl+D)
	case tcell.KeyCtrlZ:
		data = []byte{26} // ASCII SUB (Ctrl+Z)
	default:
		// Handle other keys as needed
		return
	}

	select {
	case v.inputCh <- data:
	default:
		// Drop input if buffer is full
	}
}
