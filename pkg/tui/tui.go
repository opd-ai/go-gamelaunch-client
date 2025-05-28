package tui

import (
	"fmt"
	"io"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
)

// TerminalView implements dgclient.View using tcell for terminal rendering
type TerminalView struct {
	screen tcell.Screen

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

	// Set up event handling
	go v.handleEvents()

	// Clear screen
	v.screen.Clear()
	v.screen.Show()

	return nil
}

// Render displays the provided data
func (v *TerminalView) Render(data []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen == nil {
		return fmt.Errorf("screen not initialized")
	}

	// Simple implementation - in reality, you'd need a proper terminal emulator
	// This is a placeholder that just prints the data
	// A real implementation would parse ANSI sequences, handle cursor movement, etc.

	// For now, just write to screen at current position
	// This is highly simplified and won't handle most terminal sequences correctly
	x, y := 0, 0
	for _, b := range data {
		switch b {
		case '\n':
			x = 0
			y++
			if y >= v.height {
				y = v.height - 1
				// Scroll would happen here
			}
		case '\r':
			x = 0
		case '\b':
			if x > 0 {
				x--
			}
		default:
			if x < v.width && y < v.height {
				v.screen.SetContent(x, y, rune(b), nil, tcell.StyleDefault)
				x++
				if x >= v.width {
					x = 0
					y++
				}
			}
		}
	}

	v.screen.Show()
	return nil
}

// Clear clears the display
func (v *TerminalView) Clear() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen == nil {
		return fmt.Errorf("screen not initialized")
	}

	v.screen.Clear()
	v.screen.Show()
	return nil
}

// SetSize updates the view dimensions
func (v *TerminalView) SetSize(width, height int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.width = width
	v.height = height

	// tcell handles resize automatically, but we store the dimensions
	return nil
}

// GetSize returns current dimensions
func (v *TerminalView) GetSize() (width, height int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen != nil {
		return v.screen.Size()
	}
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

// handleEvents processes terminal events
func (v *TerminalView) handleEvents() {
	for {
		select {
		case <-v.quitCh:
			return
		default:
			event := v.screen.PollEvent()
			if event == nil {
				continue
			}

			switch ev := event.(type) {
			case *tcell.EventKey:
				v.handleKeyEvent(ev)
			case *tcell.EventResize:
				v.mu.Lock()
				v.width, v.height = ev.Size()
				v.screen.Sync()
				v.mu.Unlock()
			}
		}
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
