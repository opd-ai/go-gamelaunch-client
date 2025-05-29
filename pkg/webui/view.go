package webui

import (
	"io"
	"sync"
	"time"

	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
)

// Cell represents a single character cell with rendering attributes
type Cell struct {
	Char    rune   `json:"char"`
	FgColor string `json:"fg_color"`
	BgColor string `json:"bg_color"`
	Bold    bool   `json:"bold"`
	Inverse bool   `json:"inverse"`
	Blink   bool   `json:"blink"`
	TileX   int    `json:"tile_x,omitempty"`
	TileY   int    `json:"tile_y,omitempty"`
	Changed bool   `json:"-"`
}

// GameState represents the current state of the game screen
type GameState struct {
	Buffer    [][]Cell `json:"buffer"`
	Width     int      `json:"width"`
	Height    int      `json:"height"`
	CursorX   int      `json:"cursor_x"`
	CursorY   int      `json:"cursor_y"`
	Version   uint64   `json:"version"`
	Timestamp int64    `json:"timestamp"`
}

// StateDiff represents changes between game states
type StateDiff struct {
	Version   uint64     `json:"version"`
	Changes   []CellDiff `json:"changes"`
	CursorX   int        `json:"cursor_x"`
	CursorY   int        `json:"cursor_y"`
	Timestamp int64      `json:"timestamp"`
}

// CellDiff represents a change to a specific cell
type CellDiff struct {
	X    int  `json:"x"`
	Y    int  `json:"y"`
	Cell Cell `json:"cell"`
}

// WebView implements dgclient.View for web browser rendering
type WebView struct {
	mu           sync.RWMutex
	buffer       [][]Cell
	width        int
	height       int
	cursorX      int
	cursorY      int
	inputChan    chan []byte
	updateNotify chan struct{}
	stateManager *StateManager
	tileset      *TilesetConfig
}

// NewWebView creates a new web-based view
func NewWebView(opts dgclient.ViewOptions) (*WebView, error) {
	width := opts.InitialWidth
	height := opts.InitialHeight

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	view := &WebView{
		width:        width,
		height:       height,
		inputChan:    make(chan []byte, 100),
		updateNotify: make(chan struct{}, 10),
		stateManager: NewStateManager(),
	}

	view.initBuffer()
	return view, nil
}

// Init initializes the web view
func (v *WebView) Init() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.initBuffer()
	return nil
}

// initBuffer initializes the screen buffer
func (v *WebView) initBuffer() {
	v.buffer = make([][]Cell, v.height)
	for y := range v.buffer {
		v.buffer[y] = make([]Cell, v.width)
		for x := range v.buffer[y] {
			v.buffer[y][x] = Cell{
				Char:    ' ',
				FgColor: "#FFFFFF",
				BgColor: "#000000",
			}
		}
	}
}

// Render processes terminal data and updates the screen buffer
func (v *WebView) Render(data []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Process the terminal data to update buffer
	v.processTerminalData(data)

	// Update state manager with new version
	state := v.getCurrentState()
	v.stateManager.UpdateState(state)

	// Notify polling clients of updates
	select {
	case v.updateNotify <- struct{}{}:
	default:
	}

	return nil
}

// processTerminalData parses terminal escape sequences and updates buffer
func (v *WebView) processTerminalData(data []byte) {
	// Simple implementation - in practice would need full ANSI parser
	for _, b := range data {
		switch b {
		case '\n':
			v.cursorY++
			v.cursorX = 0
			if v.cursorY >= v.height {
				v.scrollUp()
				v.cursorY = v.height - 1
			}
		case '\r':
			v.cursorX = 0
		case '\b':
			if v.cursorX > 0 {
				v.cursorX--
			}
		default:
			if b >= 32 && b < 127 { // Printable ASCII
				if v.cursorX < v.width && v.cursorY < v.height {
					cell := &v.buffer[v.cursorY][v.cursorX]
					cell.Char = rune(b)
					cell.Changed = true

					// Apply tileset mapping if available
					if v.tileset != nil {
						if mapping := v.tileset.GetMapping(rune(b)); mapping != nil {
							cell.TileX = mapping.X
							cell.TileY = mapping.Y
							if mapping.FgColor != "" {
								cell.FgColor = mapping.FgColor
							}
							if mapping.BgColor != "" {
								cell.BgColor = mapping.BgColor
							}
						}
					}
				}
				v.cursorX++
				if v.cursorX >= v.width {
					v.cursorX = 0
					v.cursorY++
					if v.cursorY >= v.height {
						v.scrollUp()
						v.cursorY = v.height - 1
					}
				}
			}
		}
	}
}

// scrollUp scrolls the buffer up by one line
func (v *WebView) scrollUp() {
	for y := 0; y < v.height-1; y++ {
		copy(v.buffer[y], v.buffer[y+1])
	}
	// Clear last line
	for x := 0; x < v.width; x++ {
		v.buffer[v.height-1][x] = Cell{
			Char:    ' ',
			FgColor: "#FFFFFF",
			BgColor: "#000000",
		}
	}
}

// Clear clears the display
func (v *WebView) Clear() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.initBuffer()
	v.cursorX = 0
	v.cursorY = 0

	return nil
}

// SetSize updates the view dimensions
func (v *WebView) SetSize(width, height int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if width <= 0 || height <= 0 {
		return dgclient.ErrInvalidTerminalSize
	}

	oldBuffer := v.buffer
	oldWidth := v.width
	oldHeight := v.height

	v.width = width
	v.height = height
	v.initBuffer()

	// Copy old content
	copyHeight := oldHeight
	if height < oldHeight {
		copyHeight = height
	}

	for y := 0; y < copyHeight; y++ {
		copyWidth := oldWidth
		if width < oldWidth {
			copyWidth = width
		}

		for x := 0; x < copyWidth; x++ {
			v.buffer[y][x] = oldBuffer[y][x]
		}
	}

	// Adjust cursor position
	if v.cursorX >= width {
		v.cursorX = width - 1
	}
	if v.cursorY >= height {
		v.cursorY = height - 1
	}

	return nil
}

// GetSize returns current dimensions
func (v *WebView) GetSize() (width, height int) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.width, v.height
}

// HandleInput reads and returns user input
func (v *WebView) HandleInput() ([]byte, error) {
	select {
	case input := <-v.inputChan:
		return input, nil
	default:
		return nil, io.EOF
	}
}

// Close cleans up resources
func (v *WebView) Close() error {
	close(v.inputChan)
	close(v.updateNotify)
	return nil
}

// SendInput queues input from web client
func (v *WebView) SendInput(data []byte) {
	select {
	case v.inputChan <- data:
	default:
		// Channel full, drop input
	}
}

// GetCurrentState returns the current game state
func (v *WebView) GetCurrentState() *GameState {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.getCurrentState()
}

// getCurrentState returns current state without locking (internal use)
func (v *WebView) getCurrentState() *GameState {
	// Deep copy buffer
	buffer := make([][]Cell, v.height)
	for y := range buffer {
		buffer[y] = make([]Cell, v.width)
		copy(buffer[y], v.buffer[y])
	}

	return &GameState{
		Buffer:    buffer,
		Width:     v.width,
		Height:    v.height,
		CursorX:   v.cursorX,
		CursorY:   v.cursorY,
		Version:   v.stateManager.GetCurrentVersion(),
		Timestamp: time.Now().UnixNano(),
	}
}

// WaitForUpdate waits for the next screen update
func (v *WebView) WaitForUpdate(timeout time.Duration) bool {
	select {
	case <-v.updateNotify:
		return true
	case <-time.After(timeout):
		return false
	}
}

// SetTileset updates the tileset configuration
func (v *WebView) SetTileset(tileset *TilesetConfig) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.tileset = tileset

	// Reapply tileset to current buffer
	for y := 0; y < v.height; y++ {
		for x := 0; x < v.width; x++ {
			cell := &v.buffer[y][x]
			if mapping := tileset.GetMapping(cell.Char); mapping != nil {
				cell.TileX = mapping.X
				cell.TileY = mapping.Y
				if mapping.FgColor != "" {
					cell.FgColor = mapping.FgColor
				}
				if mapping.BgColor != "" {
					cell.BgColor = mapping.BgColor
				}
			}
		}
	}
}
