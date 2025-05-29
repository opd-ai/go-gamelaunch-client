Project Path: go-gamelaunch-client

Source Tree:

```
go-gamelaunch-client
├── pkg
│   ├── tui
│   │   ├── tui.go
│   │   ├── input.go
│   │   ├── emulator.go
│   │   └── emulator_test.go
│   └── dgclient
│       ├── client_test.go
│       ├── auth_test.go
│       ├── session.go
│       ├── client.go
│       ├── run.go
│       ├── view.go
│       ├── errors.go
│       └── auth.go
├── LICENSE
├── go.mod
├── README.md
├── cmd
│   └── dgconnect
│       ├── main.go
│       ├── config_test.go
│       ├── commands.go
│       └── config.go
├── Makefile
└── go.sum

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/tui/tui.go`:

```go
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
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.screen == nil {
		return fmt.Errorf("screen not initialized")
	}

	// Process data through terminal emulator
	v.emulator.ProcessData(data)

	// Get the processed screen and render it
	screenData := v.emulator.GetScreen()
	cursorX, cursorY := v.emulator.GetCursor()

	// Clear the display
	v.screen.Clear()

	// Render each cell
	for y, row := range screenData {
		for x, cell := range row {
			if y < v.height && x < v.width {
				style := v.cellToTcellStyle(cell.Attr)
				v.screen.SetContent(x, y, cell.Char, nil, style)
			}
		}
	}

	// Set cursor position
	if cursorY < v.height && cursorX < v.width {
		v.screen.ShowCursor(cursorX, cursorY)
	}

	v.screen.Show()
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
				if v.emulator != nil {
					v.emulator.Resize(v.width, v.height)
				}
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/tui/input.go`:

```go
package tui

import (
	"io"

	"github.com/gdamore/tcell/v2"
)

// InputMode defines how input is processed
type InputMode int

const (
	// InputModeNormal processes input normally
	InputModeNormal InputMode = iota

	// InputModeRaw sends all input without processing
	InputModeRaw

	// InputModePassword hides input
	InputModePassword
)

// InputHandler processes user input with different modes
type InputHandler struct {
	mode    InputMode
	buffer  []byte
	echoOff bool
}

// NewInputHandler creates a new input handler
func NewInputHandler() *InputHandler {
	return &InputHandler{
		mode:   InputModeNormal,
		buffer: make([]byte, 0, 1024),
	}
}

// SetMode changes the input processing mode
func (h *InputHandler) SetMode(mode InputMode) {
	h.mode = mode
}

// ProcessKey processes a key event based on current mode
func (h *InputHandler) ProcessKey(ev *tcell.EventKey) ([]byte, bool) {
	switch h.mode {
	case InputModeRaw:
		return h.processRawKey(ev)
	case InputModePassword:
		return h.processPasswordKey(ev)
	default:
		return h.processNormalKey(ev)
	}
}

// processNormalKey handles normal input processing
func (h *InputHandler) processNormalKey(ev *tcell.EventKey) ([]byte, bool) {
	// Convert key event to bytes
	switch ev.Key() {
	case tcell.KeyRune:
		return []byte(string(ev.Rune())), true
	case tcell.KeyEnter:
		return []byte("\r"), true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return []byte{8}, true
	case tcell.KeyTab:
		return []byte{9}, true
	case tcell.KeyEscape:
		return []byte{27}, true
	case tcell.KeyUp:
		return []byte("\033[A"), true
	case tcell.KeyDown:
		return []byte("\033[B"), true
	case tcell.KeyRight:
		return []byte("\033[C"), true
	case tcell.KeyLeft:
		return []byte("\033[D"), true
	case tcell.KeyHome:
		return []byte("\033[H"), true
	case tcell.KeyEnd:
		return []byte("\033[F"), true
	case tcell.KeyPgUp:
		return []byte("\033[5~"), true
	case tcell.KeyPgDn:
		return []byte("\033[6~"), true
	case tcell.KeyDelete:
		return []byte("\033[3~"), true
	case tcell.KeyInsert:
		return []byte("\033[2~"), true
	case tcell.KeyF1:
		return []byte("\033OP"), true
	case tcell.KeyF2:
		return []byte("\033OQ"), true
	case tcell.KeyF3:
		return []byte("\033OR"), true
	case tcell.KeyF4:
		return []byte("\033OS"), true
	case tcell.KeyF5:
		return []byte("\033[15~"), true
	case tcell.KeyF6:
		return []byte("\033[17~"), true
	case tcell.KeyF7:
		return []byte("\033[18~"), true
	case tcell.KeyF8:
		return []byte("\033[19~"), true
	case tcell.KeyF9:
		return []byte("\033[20~"), true
	case tcell.KeyF10:
		return []byte("\033[21~"), true
	case tcell.KeyF11:
		return []byte("\033[23~"), true
	case tcell.KeyF12:
		return []byte("\033[24~"), true
	}

	// Handle Ctrl combinations
	if ev.Modifiers()&tcell.ModCtrl != 0 {
		if ev.Key() == tcell.KeyRune {
			r := ev.Rune()
			if r >= 'a' && r <= 'z' {
				return []byte{byte(r - 'a' + 1)}, true
			}
			if r >= 'A' && r <= 'Z' {
				return []byte{byte(r - 'A' + 1)}, true
			}
		}
	}

	return nil, false
}

// processRawKey handles raw input mode
func (h *InputHandler) processRawKey(ev *tcell.EventKey) ([]byte, bool) {
	// In raw mode, pass everything through
	return h.processNormalKey(ev)
}

// processPasswordKey handles password input mode
func (h *InputHandler) processPasswordKey(ev *tcell.EventKey) ([]byte, bool) {
	// Similar to normal but might not echo
	return h.processNormalKey(ev)
}

// BufferedReader provides a buffered input reader
type BufferedReader struct {
	inputCh <-chan []byte
	buffer  []byte
	closed  bool
}

// NewBufferedReader creates a new buffered reader
func NewBufferedReader(inputCh <-chan []byte) *BufferedReader {
	return &BufferedReader{
		inputCh: inputCh,
		buffer:  make([]byte, 0),
	}
}

// Read implements io.Reader
func (r *BufferedReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.EOF
	}

	// If we have buffered data, return it first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Wait for new input
	select {
	case data, ok := <-r.inputCh:
		if !ok {
			r.closed = true
			return 0, io.EOF
		}

		n = copy(p, data)
		if n < len(data) {
			// Buffer remaining data
			r.buffer = append(r.buffer, data[n:]...)
		}
		return n, nil
	}
}

// InputBuffer provides input buffering with line editing capabilities
type InputBuffer struct {
	data     []rune
	position int
	history  []string
	histPos  int
}

// NewInputBuffer creates a new input buffer
func NewInputBuffer() *InputBuffer {
	return &InputBuffer{
		data:    make([]rune, 0),
		history: make([]string, 0),
	}
}

// AddChar adds a character at the current position
func (b *InputBuffer) AddChar(r rune) {
	if b.position == len(b.data) {
		b.data = append(b.data, r)
	} else {
		b.data = append(b.data[:b.position+1], b.data[b.position:]...)
		b.data[b.position] = r
	}
	b.position++
}

// DeleteChar deletes the character before the cursor
func (b *InputBuffer) DeleteChar() bool {
	if b.position > 0 {
		b.data = append(b.data[:b.position-1], b.data[b.position:]...)
		b.position--
		return true
	}
	return false
}

// MoveLeft moves the cursor left
func (b *InputBuffer) MoveLeft() bool {
	if b.position > 0 {
		b.position--
		return true
	}
	return false
}

// MoveRight moves the cursor right
func (b *InputBuffer) MoveRight() bool {
	if b.position < len(b.data) {
		b.position++
		return true
	}
	return false
}

// GetLine returns the current line and resets the buffer
func (b *InputBuffer) GetLine() string {
	line := string(b.data)
	b.data = b.data[:0]
	b.position = 0

	// Add to history
	if line != "" {
		b.history = append(b.history, line)
		b.histPos = len(b.history)
	}

	return line
}

// HistoryUp moves up in history
func (b *InputBuffer) HistoryUp() bool {
	if b.histPos > 0 {
		b.histPos--
		if b.histPos < len(b.history) {
			b.data = []rune(b.history[b.histPos])
			b.position = len(b.data)
			return true
		}
	}
	return false
}

// HistoryDown moves down in history
func (b *InputBuffer) HistoryDown() bool {
	if b.histPos < len(b.history)-1 {
		b.histPos++
		b.data = []rune(b.history[b.histPos])
		b.position = len(b.data)
		return true
	} else if b.histPos == len(b.history)-1 {
		b.histPos++
		b.data = []rune{}
		b.position = 0
		return true
	}
	return false
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/tui/emulator.go`:

```go
package tui

import (
	"sync"
)

// TerminalEmulator provides a proper terminal emulation layer
type TerminalEmulator struct {
	mu     sync.RWMutex
	width  int
	height int

	// Screen buffer - stores characters and attributes
	screen [][]Cell

	// Cursor position
	cursorX, cursorY int

	// Terminal state
	savedCursorX, savedCursorY int

	// Parser state for ANSI sequences
	parser *AnsiParser

	// Scrolling region
	scrollTop, scrollBottom int

	// Character attributes
	currentAttr CellAttributes
}

// Cell represents a single character cell with attributes
type Cell struct {
	Char rune
	Attr CellAttributes
}

// CellAttributes stores text formatting information
type CellAttributes struct {
	Foreground Color
	Background Color
	Bold       bool
	Underline  bool
	Reverse    bool
}

// Color represents a terminal color
type Color struct {
	R, G, B uint8
	IsIndex bool
	Index   uint8
}

// AnsiParser handles ANSI escape sequence parsing
type AnsiParser struct {
	state      ParserState
	buffer     []byte
	params     []int
	paramIndex int
}

type ParserState int

const (
	StateNormal ParserState = iota
	StateEscape
	StateCSI
	StateOSC
)

// NewTerminalEmulator creates a new terminal emulator
func NewTerminalEmulator(width, height int) *TerminalEmulator {
	te := &TerminalEmulator{
		width:        width,
		height:       height,
		screen:       make([][]Cell, height),
		parser:       &AnsiParser{state: StateNormal},
		scrollBottom: height - 1,
		currentAttr:  CellAttributes{Foreground: Color{R: 255, G: 255, B: 255}},
	}

	// Initialize screen buffer
	for i := range te.screen {
		te.screen[i] = make([]Cell, width)
		for j := range te.screen[i] {
			te.screen[i][j] = Cell{Char: ' ', Attr: te.currentAttr}
		}
	}

	return te
}

// ProcessData processes incoming terminal data and updates the screen
func (te *TerminalEmulator) ProcessData(data []byte) {
	te.mu.Lock()
	defer te.mu.Unlock()

	for _, b := range data {
		te.processByte(b)
	}
}

// processByte processes a single byte through the ANSI parser
func (te *TerminalEmulator) processByte(b byte) {
	switch te.parser.state {
	case StateNormal:
		te.processNormalByte(b)
	case StateEscape:
		te.processEscapeByte(b)
	case StateCSI:
		te.processCSIByte(b)
	case StateOSC:
		te.processOSCByte(b)
	}
}

// processNormalByte handles normal characters and escape sequences
func (te *TerminalEmulator) processNormalByte(b byte) {
	switch b {
	case 0x1B: // ESC
		te.parser.state = StateEscape
		te.parser.buffer = te.parser.buffer[:0]
	case '\r': // Carriage Return
		te.cursorX = 0
	case '\n': // Line Feed
		te.newline()
	case '\b': // Backspace
		if te.cursorX > 0 {
			te.cursorX--
		}
	case '\t': // Tab
		te.cursorX = ((te.cursorX / 8) + 1) * 8
		if te.cursorX >= te.width {
			te.cursorX = te.width - 1
		}
	case 7: // Bell
		// Ignore bell for now
	default:
		if b >= 32 { // Printable character
			te.putChar(rune(b))
		}
	}
}

// processEscapeByte handles escape sequence detection
func (te *TerminalEmulator) processEscapeByte(b byte) {
	switch b {
	case '[':
		te.parser.state = StateCSI
		te.parser.params = te.parser.params[:0]
		te.parser.paramIndex = 0
	case ']':
		te.parser.state = StateOSC
	case 'c': // Reset
		te.reset()
		te.parser.state = StateNormal
	case 'D': // Index (move down)
		te.newline()
		te.parser.state = StateNormal
	case 'M': // Reverse Index (move up)
		te.reverseNewline()
		te.parser.state = StateNormal
	case '7': // Save cursor
		te.savedCursorX = te.cursorX
		te.savedCursorY = te.cursorY
		te.parser.state = StateNormal
	case '8': // Restore cursor
		te.cursorX = te.savedCursorX
		te.cursorY = te.savedCursorY
		te.parser.state = StateNormal
	default:
		te.parser.state = StateNormal
	}
}

// processCSIByte handles CSI (Control Sequence Introducer) sequences
func (te *TerminalEmulator) processCSIByte(b byte) {
	if b >= '0' && b <= '9' {
		// Build parameter
		if len(te.parser.params) <= te.parser.paramIndex {
			te.parser.params = append(te.parser.params, 0)
		}
		te.parser.params[te.parser.paramIndex] = te.parser.params[te.parser.paramIndex]*10 + int(b-'0')
	} else if b == ';' {
		// Parameter separator
		te.parser.paramIndex++
	} else {
		// Command character
		te.executeCSICommand(b)
		te.parser.state = StateNormal
	}
}

// processOSCByte handles OSC (Operating System Command) sequences
func (te *TerminalEmulator) processOSCByte(b byte) {
	if b == 7 || b == 0x1B { // BEL or ESC terminates OSC
		te.parser.state = StateNormal
	}
	// For now, ignore OSC sequences
}

// executeCSICommand executes CSI commands
func (te *TerminalEmulator) executeCSICommand(cmd byte) {
	params := te.parser.params

	switch cmd {
	case 'A': // Cursor Up
		n := 1
		if len(params) > 0 && params[0] > 0 {
			n = params[0]
		}
		te.cursorY = max(0, te.cursorY-n)

	case 'B': // Cursor Down
		n := 1
		if len(params) > 0 && params[0] > 0 {
			n = params[0]
		}
		te.cursorY = min(te.height-1, te.cursorY+n)

	case 'C': // Cursor Forward
		n := 1
		if len(params) > 0 && params[0] > 0 {
			n = params[0]
		}
		te.cursorX = min(te.width-1, te.cursorX+n)

	case 'D': // Cursor Backward
		n := 1
		if len(params) > 0 && params[0] > 0 {
			n = params[0]
		}
		te.cursorX = max(0, te.cursorX-n)

	case 'H', 'f': // Cursor Position
		row, col := 1, 1
		if len(params) > 0 && params[0] > 0 {
			row = params[0]
		}
		if len(params) > 1 && params[1] > 0 {
			col = params[1]
		}
		te.cursorY = min(te.height-1, max(0, row-1))
		te.cursorX = min(te.width-1, max(0, col-1))

	case 'J': // Erase Display
		mode := 0
		if len(params) > 0 {
			mode = params[0]
		}
		switch mode {
		case 0: // Erase from cursor to end of screen
			te.eraseFromCursorToEnd()
		case 1: // Erase from start of screen to cursor
			te.eraseFromStartToCursor()
		case 2: // Erase entire screen
			te.eraseScreen()
		}

	case 'K': // Erase Line
		mode := 0
		if len(params) > 0 {
			mode = params[0]
		}
		switch mode {
		case 0: // Erase from cursor to end of line
			te.eraseFromCursorToEndOfLine()
		case 1: // Erase from start of line to cursor
			te.eraseFromStartOfLineToCursor()
		case 2: // Erase entire line
			te.eraseEntireLine()
		}

	case 'm': // Select Graphic Rendition (colors/attributes)
		te.processGraphicRendition(params)

	case 'r': // Set Scrolling Region
		top, bottom := 1, te.height
		if len(params) > 0 && params[0] > 0 {
			top = params[0]
		}
		if len(params) > 1 && params[1] > 0 {
			bottom = params[1]
		}
		te.scrollTop = max(0, min(te.height-1, top-1))
		te.scrollBottom = max(0, min(te.height-1, bottom-1))
	}
}

// processGraphicRendition handles color and attribute changes
func (te *TerminalEmulator) processGraphicRendition(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}

	for _, param := range params {
		switch param {
		case 0: // Reset
			te.currentAttr = CellAttributes{Foreground: Color{R: 255, G: 255, B: 255}}
		case 1: // Bold
			te.currentAttr.Bold = true
		case 4: // Underline
			te.currentAttr.Underline = true
		case 7: // Reverse
			te.currentAttr.Reverse = true
		case 22: // Normal intensity
			te.currentAttr.Bold = false
		case 24: // Not underlined
			te.currentAttr.Underline = false
		case 27: // Not reversed
			te.currentAttr.Reverse = false
		case 30, 31, 32, 33, 34, 35, 36, 37: // Foreground colors
			te.currentAttr.Foreground = getANSIColor(param - 30)
		case 40, 41, 42, 43, 44, 45, 46, 47: // Background colors
			te.currentAttr.Background = getANSIColor(param - 40)
		case 38: // Extended foreground color (handled in extended parsing)
		case 48: // Extended background color (handled in extended parsing)
		}
	}
}

// putChar places a character at the current cursor position
func (te *TerminalEmulator) putChar(ch rune) {
	if te.cursorY >= 0 && te.cursorY < te.height && te.cursorX >= 0 && te.cursorX < te.width {
		te.screen[te.cursorY][te.cursorX] = Cell{Char: ch, Attr: te.currentAttr}
		te.cursorX++
		if te.cursorX >= te.width {
			te.newline()
		}
	}
}

// newline moves to the next line, scrolling if necessary
func (te *TerminalEmulator) newline() {
	te.cursorX = 0
	te.cursorY++
	if te.cursorY > te.scrollBottom {
		te.scroll()
		te.cursorY = te.scrollBottom
	}
}

// reverseNewline moves up one line
func (te *TerminalEmulator) reverseNewline() {
	te.cursorY--
	if te.cursorY < te.scrollTop {
		te.reverseScroll()
		te.cursorY = te.scrollTop
	}
}

// scroll scrolls the screen up by one line
func (te *TerminalEmulator) scroll() {
	for y := te.scrollTop; y < te.scrollBottom; y++ {
		copy(te.screen[y], te.screen[y+1])
	}
	// Clear the bottom line
	for x := 0; x < te.width; x++ {
		te.screen[te.scrollBottom][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

// reverseScroll scrolls the screen down by one line
func (te *TerminalEmulator) reverseScroll() {
	for y := te.scrollBottom; y > te.scrollTop; y-- {
		copy(te.screen[y], te.screen[y-1])
	}
	// Clear the top line
	for x := 0; x < te.width; x++ {
		te.screen[te.scrollTop][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

// Erase functions
func (te *TerminalEmulator) eraseScreen() {
	for y := 0; y < te.height; y++ {
		for x := 0; x < te.width; x++ {
			te.screen[y][x] = Cell{Char: ' ', Attr: te.currentAttr}
		}
	}
}

func (te *TerminalEmulator) eraseFromCursorToEnd() {
	// Clear from cursor to end of current line
	for x := te.cursorX; x < te.width; x++ {
		te.screen[te.cursorY][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
	// Clear all lines below
	for y := te.cursorY + 1; y < te.height; y++ {
		for x := 0; x < te.width; x++ {
			te.screen[y][x] = Cell{Char: ' ', Attr: te.currentAttr}
		}
	}
}

func (te *TerminalEmulator) eraseFromStartToCursor() {
	// Clear all lines above
	for y := 0; y < te.cursorY; y++ {
		for x := 0; x < te.width; x++ {
			te.screen[y][x] = Cell{Char: ' ', Attr: te.currentAttr}
		}
	}
	// Clear from start of current line to cursor
	for x := 0; x <= te.cursorX; x++ {
		te.screen[te.cursorY][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

func (te *TerminalEmulator) eraseEntireLine() {
	for x := 0; x < te.width; x++ {
		te.screen[te.cursorY][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

func (te *TerminalEmulator) eraseFromCursorToEndOfLine() {
	for x := te.cursorX; x < te.width; x++ {
		te.screen[te.cursorY][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

func (te *TerminalEmulator) eraseFromStartOfLineToCursor() {
	for x := 0; x <= te.cursorX; x++ {
		te.screen[te.cursorY][x] = Cell{Char: ' ', Attr: te.currentAttr}
	}
}

// reset resets the terminal to initial state
func (te *TerminalEmulator) reset() {
	te.cursorX = 0
	te.cursorY = 0
	te.scrollTop = 0
	te.scrollBottom = te.height - 1
	te.currentAttr = CellAttributes{Foreground: Color{R: 255, G: 255, B: 255}}
	te.eraseScreen()
}

// GetScreen returns a copy of the current screen state
func (te *TerminalEmulator) GetScreen() [][]Cell {
	te.mu.RLock()
	defer te.mu.RUnlock()

	screen := make([][]Cell, te.height)
	for i := range screen {
		screen[i] = make([]Cell, te.width)
		copy(screen[i], te.screen[i])
	}
	return screen
}

// GetCursor returns the current cursor position
func (te *TerminalEmulator) GetCursor() (int, int) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.cursorX, te.cursorY
}

// Resize changes the terminal dimensions
func (te *TerminalEmulator) Resize(width, height int) {
	te.mu.Lock()
	defer te.mu.Unlock()

	// Create new screen buffer
	newScreen := make([][]Cell, height)
	for i := range newScreen {
		newScreen[i] = make([]Cell, width)
		for j := range newScreen[i] {
			newScreen[i][j] = Cell{Char: ' ', Attr: te.currentAttr}
		}
	}

	// Copy existing content
	copyHeight := min(height, te.height)
	copyWidth := min(width, te.width)
	for y := 0; y < copyHeight; y++ {
		copy(newScreen[y][:copyWidth], te.screen[y][:copyWidth])
	}

	te.screen = newScreen
	te.width = width
	te.height = height
	te.scrollBottom = height - 1

	// Adjust cursor position
	te.cursorX = min(te.cursorX, width-1)
	te.cursorY = min(te.cursorY, height-1)
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getANSIColor returns the color for standard ANSI color codes
func getANSIColor(code int) Color {
	colors := []Color{
		{R: 0, G: 0, B: 0},       // Black
		{R: 128, G: 0, B: 0},     // Red
		{R: 0, G: 128, B: 0},     // Green
		{R: 128, G: 128, B: 0},   // Yellow
		{R: 0, G: 0, B: 128},     // Blue
		{R: 128, G: 0, B: 128},   // Magenta
		{R: 0, G: 128, B: 128},   // Cyan
		{R: 192, G: 192, B: 192}, // White
	}
	if code >= 0 && code < len(colors) {
		return colors[code]
	}
	return Color{R: 255, G: 255, B: 255}
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/tui/emulator_test.go`:

```go
package tui

import (
	"testing"
)

func TestNewTerminalEmulator(t *testing.T) {
	width, height := 80, 24
	te := NewTerminalEmulator(width, height)

	if te.width != width {
		t.Errorf("Expected width %d, got %d", width, te.width)
	}

	if te.height != height {
		t.Errorf("Expected height %d, got %d", height, te.height)
	}

	if len(te.screen) != height {
		t.Errorf("Expected screen height %d, got %d", height, len(te.screen))
	}

	if len(te.screen[0]) != width {
		t.Errorf("Expected screen width %d, got %d", width, len(te.screen[0]))
	}

	if te.cursorX != 0 || te.cursorY != 0 {
		t.Errorf("Expected cursor at (0,0), got (%d,%d)", te.cursorX, te.cursorY)
	}
}

func TestProcessDataSimpleText(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	text := "Hello World"
	te.ProcessData([]byte(text))

	screen := te.GetScreen()

	// Check that characters were placed correctly
	for i, ch := range text {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}

	// Check cursor position
	cursorX, cursorY := te.GetCursor()
	if cursorX != len(text) || cursorY != 0 {
		t.Errorf("Expected cursor at (%d,0), got (%d,%d)", len(text), cursorX, cursorY)
	}
}

func TestProcessDataNewline(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	te.ProcessData([]byte("Line1\nLine2"))

	screen := te.GetScreen()

	// Check first line
	expectedLine1 := "Line1"
	for i, ch := range expectedLine1 {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Line 1: Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}

	// Check second line
	expectedLine2 := "Line2"
	for i, ch := range expectedLine2 {
		if screen[1][i].Char != rune(ch) {
			t.Errorf("Line 2: Expected char '%c' at position %d, got '%c'", ch, i, screen[1][i].Char)
		}
	}

	// Check cursor position
	cursorX, cursorY := te.GetCursor()
	if cursorX != 5 || cursorY != 1 {
		t.Errorf("Expected cursor at (5,1), got (%d,%d)", cursorX, cursorY)
	}
}

func TestProcessDataCarriageReturn(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	te.ProcessData([]byte("Hello\rWorld"))

	screen := te.GetScreen()

	// After carriage return, "World" should overwrite "Hello"
	expected := "World"
	for i, ch := range expected {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}
}

func TestProcessDataANSIEscape(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	// Clear screen and move cursor to (5,5)
	te.ProcessData([]byte("\x1b[2J\x1b[6;6H"))

	cursorX, cursorY := te.GetCursor()
	if cursorX != 5 || cursorY != 5 {
		t.Errorf("Expected cursor at (5,5), got (%d,%d)", cursorX, cursorY)
	}

	// Check that screen was cleared
	screen := te.GetScreen()
	for y := 0; y < te.height; y++ {
		for x := 0; x < te.width; x++ {
			if screen[y][x].Char != ' ' {
				t.Errorf("Expected space at (%d,%d), got '%c'", x, y, screen[y][x].Char)
			}
		}
	}
}

func TestResize(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	// Add some content
	te.ProcessData([]byte("Test content"))

	// Resize to smaller
	newWidth, newHeight := 40, 12
	te.Resize(newWidth, newHeight)

	if te.width != newWidth {
		t.Errorf("Expected width %d after resize, got %d", newWidth, te.width)
	}

	if te.height != newHeight {
		t.Errorf("Expected height %d after resize, got %d", newHeight, te.height)
	}

	screen := te.GetScreen()
	if len(screen) != newHeight {
		t.Errorf("Expected screen height %d after resize, got %d", newHeight, len(screen))
	}

	if len(screen[0]) != newWidth {
		t.Errorf("Expected screen width %d after resize, got %d", newWidth, len(screen[0]))
	}

	// Check that content was preserved (first 12 characters)
	expected := "Test content"
	for i, ch := range expected {
		if i < newWidth && screen[0][i].Char != rune(ch) {
			t.Errorf("Expected preserved char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/client_test.go`:

```go
package dgclient

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)
	defer client.Close()

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.config != config {
		t.Error("Client config not set correctly")
	}

	if client.IsConnected() {
		t.Error("New client should not be connected")
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected ConnectTimeout 30s, got %v", config.ConnectTimeout)
	}

	if config.KeepAliveInterval != 30*time.Second {
		t.Errorf("Expected KeepAliveInterval 30s, got %v", config.KeepAliveInterval)
	}

	if config.MaxReconnectAttempts != 3 {
		t.Errorf("Expected MaxReconnectAttempts 3, got %d", config.MaxReconnectAttempts)
	}

	if config.ReconnectDelay != 5*time.Second {
		t.Errorf("Expected ReconnectDelay 5s, got %v", config.ReconnectDelay)
	}

	if config.DefaultTerminal != "xterm-256color" {
		t.Errorf("Expected DefaultTerminal xterm-256color, got %s", config.DefaultTerminal)
	}
}

func TestClientSetView(t *testing.T) {
	client := NewClient(nil)
	defer client.Close()

	// Mock view implementation
	mockView := &MockView{}

	err := client.SetView(mockView)
	if err != nil {
		t.Fatalf("SetView() failed: %v", err)
	}

	if !mockView.InitCalled {
		t.Error("View.Init() was not called")
	}
}

func TestClientDisconnectWhenNotConnected(t *testing.T) {
	client := NewClient(nil)
	defer client.Close()

	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() on unconnected client should not error, got: %v", err)
	}
}

// MockView implements the View interface for testing
type MockView struct {
	InitCalled   bool
	RenderCalled bool
	InputData    []byte
}

func (m *MockView) Init() error {
	m.InitCalled = true
	return nil
}

func (m *MockView) Render(data []byte) error {
	m.RenderCalled = true
	return nil
}

func (m *MockView) Clear() error {
	return nil
}

func (m *MockView) SetSize(width, height int) error {
	return nil
}

func (m *MockView) GetSize() (width, height int) {
	return 80, 24
}

func (m *MockView) HandleInput() ([]byte, error) {
	return m.InputData, nil
}

func (m *MockView) Close() error {
	return nil
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/auth_test.go`:

```go
package dgclient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordAuth(t *testing.T) {
	password := "testpassword"
	auth := NewPasswordAuth(password)

	if auth.Name() != "password" {
		t.Errorf("Expected name 'password', got '%s'", auth.Name())
	}

	sshAuth, err := auth.GetSSHAuthMethod()
	if err != nil {
		t.Fatalf("GetSSHAuthMethod() failed: %v", err)
	}

	if sshAuth == nil {
		t.Error("GetSSHAuthMethod() returned nil")
	}
}

func TestAgentAuth(t *testing.T) {
	auth := NewAgentAuth()

	if auth.Name() != "agent" {
		t.Errorf("Expected name 'agent', got '%s'", auth.Name())
	}

	// This will fail without SSH_AUTH_SOCK, which is expected
	_, err := auth.GetSSHAuthMethod()
	if err == nil && os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Error("Expected error when SSH_AUTH_SOCK not set")
	}
}

func TestKeyAuth(t *testing.T) {
	// Create a temporary key file for testing
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key")

	// Create a dummy private key (this won't be valid for actual SSH)
	keyContent := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEAwJbykjmz1Q7G8aK1K5f3hG4OlJj5EKy1V8sZ9xbJQZbZoFpgW7
-----END OPENSSH PRIVATE KEY-----`

	err := os.WriteFile(keyPath, []byte(keyContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	auth := NewKeyAuth(keyPath, "")

	if auth.Name() != "key" {
		t.Errorf("Expected name 'key', got '%s'", auth.Name())
	}

	// This will fail with invalid key format, which is expected for our dummy key
	_, err = auth.GetSSHAuthMethod()
	if err == nil {
		t.Error("Expected error with invalid key format")
	}
}

func TestKeyAuthNonexistentFile(t *testing.T) {
	auth := NewKeyAuth("/nonexistent/path", "")

	_, err := auth.GetSSHAuthMethod()
	if err == nil {
		t.Error("Expected error with nonexistent key file")
	}
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/session.go`:

```go
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/client.go`:

```go
package dgclient

import (
	"fmt"
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/run.go`:

```go
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

// ...existing code...

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/view.go`:

```go
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/errors.go`:

```go
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/pkg/dgclient/auth.go`:

```go
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
func NewKeyAuth(keyPath, passphrase string) AuthMethod {
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

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/LICENSE`:

```
MIT License

Copyright (c) 2025 opdai

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/go.mod`:

```mod
module github.com/opd-ai/go-gamelaunch-client

go 1.23.2

require (
	github.com/gdamore/tcell/v2 v2.8.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	golang.org/x/crypto v0.38.0
	golang.org/x/term v0.32.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/README.md`:

```md
# go-gamelaunch-client

A Go client library and command-line application for connecting to dgamelaunch-style SSH servers to play terminal-based roguelike games remotely.

## Features

- **SSH Connection Management**: Password, key, and agent-based authentication
- **Terminal Emulation**: Full PTY support with dynamic resize handling
- **Modular Architecture**: Pluggable view interface for custom GUI implementations
- **Robust Error Handling**: Automatic reconnection and graceful error recovery
- **Configuration Support**: YAML-based configuration for servers and preferences
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### Library

```bash
go get github.com/opd-ai/go-gamelaunch-client/pkg/dgclient
```

### CLI Tool

```bash
go install github.com/opd-ai/go-gamelaunch-client/cmd/dgconnect@latest
```

## Quick Start

### CLI Usage

```bash
# Basic connection
dgconnect user@nethack.example.com

# With custom port and key
dgconnect user@server.example.com --port 2022 --key ~/.ssh/id_rsa

# Using configuration file
dgconnect --config ~/.dgconnect.yaml nethack-server

# Direct game launch
dgconnect user@server.example.com --game nethack
```

### Library Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
    "github.com/opd-ai/go-gamelaunch-client/pkg/tui"
)

func main() {
    // Create client
    client := dgclient.NewClient(nil)
    defer client.Close()
    
    // Set up terminal view
    view, err := tui.NewTerminalView(dgclient.DefaultViewOptions())
    if err != nil {
        log.Fatal(err)
    }
    client.SetView(view)
    
    // Connect with password
    auth := dgclient.NewPasswordAuth("mypassword")
    if err := client.Connect("nethack.example.com", 22, auth); err != nil {
        log.Fatal(err)
    }
    
    // Run game session
    if err := client.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

Create `~/.dgconnect.yaml`:

```yaml
default_server: nethack-server

servers:
  nethack-server:
    host: nethack.example.com
    port: 2022
    username: player1
    auth:
      method: key
      key_path: ~/.ssh/dgamelaunch_rsa
    default_game: nethack
    
  dcss-server:
    host: crawl.example.com
    port: 22
    username: crawler
    auth:
      method: password

preferences:
  terminal: xterm-256color
  reconnect_attempts: 3
  reconnect_delay: 5s
  color_enabled: true
  unicode_enabled: true
```

## Custom View Implementation

Implement the `View` interface to create custom GUI clients:

```go
type MyGUIView struct {
    window *MyWindow
}

func (v *MyGUIView) Init() error {
    // Initialize GUI
    return nil
}

func (v *MyGUIView) Render(data []byte) error {
    // Render terminal data to GUI
    return nil
}

// ... implement other View methods
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   CLI/Client    │────▶│  dgclient lib   │
└─────────────────┘     └────────┬────────┘
                                 │
                        ┌────────┴────────┐
                        │                 │
                   ┌────▼────┐      ┌────▼────┐
                   │   TUI   │      │  Custom │
                   │  View   │      │   View  │
                   └─────────┘      └─────────┘
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
# Build CLI
go build -o dgconnect ./cmd/dgconnect

# Build with version info
go build -ldflags "-X main.version=1.0.0" ./cmd/dgconnect
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by dgamelaunch and various roguelike game servers
- Built with golang.org/x/crypto/ssh for SSH connectivity
- Uses tcell for terminal handling
```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/cmd/dgconnect/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Configuration
	cfgFile string

	// Command flags
	port     int
	keyPath  string
	password string
	gameName string
	debug    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dgconnect [user@]host",
	Short: "Connect to dgamelaunch SSH servers",
	Long: `dgconnect is a client for connecting to dgamelaunch-style SSH servers
to play terminal-based roguelike games remotely.

Examples:
  dgconnect user@nethack.example.com
  dgconnect user@server.example.com --port 2022 --key ~/.ssh/id_rsa
  dgconnect --config ~/.dgconnect.yaml nethack-server
  dgconnect user@server.example.com --game nethack`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConnect,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dgconnect.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Connection flags
	rootCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
	rootCmd.Flags().StringVarP(&keyPath, "key", "k", "", "SSH private key path")
	rootCmd.Flags().StringVar(&password, "password", "", "SSH password (use with caution)")
	rootCmd.Flags().StringVarP(&gameName, "game", "g", "", "game to launch directly")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dgconnect %s (commit: %s, built: %s)\n", version, commit, date)
		},
	})

	// Init command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init [config-file]",
		Short: "Generate example configuration file",
		Long: `Generate an example configuration file with common server settings.
        
If no path is specified, creates ~/.dgconnect.yaml by default.

Examples:
  dgconnect init
  dgconnect init ./my-config.yaml
  dgconnect init ~/.config/dgconnect/config.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInitConfig,
	})
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dgconnect")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if debug {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	var configPath string

	if len(args) > 0 {
		configPath = args[0]
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = fmt.Sprintf("%s/.dgconnect.yaml", home)
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Print("Do you want to overwrite it? (yes/no): ")

		var response string
		fmt.Scanln(&response)

		if response != "yes" && response != "y" {
			fmt.Println("Configuration generation cancelled.")
			return nil
		}
	}

	// Generate example configuration
	config := GenerateExampleConfig()

	// Save configuration
	if err := SaveConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Example configuration created at: %s\n", configPath)
	fmt.Println("\nPlease edit the configuration file to match your server settings.")
	fmt.Println("Key sections to update:")
	fmt.Println("  - servers.*.host: Your server hostname")
	fmt.Println("  - servers.*.username: Your username")
	fmt.Println("  - servers.*.auth: Authentication method and credentials")

	return nil
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/cmd/dgconnect/config_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
default_server: test-server
servers:
  test-server:
    host: example.com
    port: 22
    username: testuser
    auth:
      method: password
preferences:
  terminal: xterm-256color
  reconnect_attempts: 3
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.DefaultServer != "test-server" {
		t.Errorf("Expected default_server 'test-server', got '%s'", config.DefaultServer)
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.Servers))
	}

	server := config.Servers["test-server"]
	if server.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", server.Host)
	}

	if server.Port != 22 {
		t.Errorf("Expected port 22, got %d", server.Port)
	}

	if server.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", server.Username)
	}

	if server.Auth.Method != "password" {
		t.Errorf("Expected auth method 'password', got '%s'", server.Auth.Method)
	}
}

func TestLoadConfigNonexistent(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path")
	if err == nil {
		t.Error("Expected error when loading nonexistent config file")
	}
}

func TestValidateConfig(t *testing.T) {
	validConfig := &Config{
		DefaultServer: "test-server",
		Servers: map[string]ServerConfig{
			"test-server": {
				Host:     "example.com",
				Port:     22,
				Username: "testuser",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateConfig() failed for valid config: %v", err)
	}
}

func TestValidateConfigNilConfig(t *testing.T) {
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestValidateConfigNoServers(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for config with no servers")
	}
}

func TestValidateConfigMissingHost(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Username: "testuser",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for server with missing host")
	}
}

func TestValidateConfigMissingUsername(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Host: "example.com",
				Auth: AuthConfig{
					Method: "password",
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for server with missing username")
	}
}

func TestValidateConfigKeyAuthMissingPath(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"test-server": {
				Host:     "example.com",
				Username: "testuser",
				Auth: AuthConfig{
					Method: "key",
					// KeyPath missing
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for key auth with missing key_path")
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	config := GenerateExampleConfig()

	if config == nil {
		t.Fatal("GenerateExampleConfig() returned nil")
	}

	if len(config.Servers) == 0 {
		t.Error("Example config should have servers")
	}

	if config.DefaultServer == "" {
		t.Error("Example config should have default_server set")
	}

	// Validate that the generated config is valid
	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Generated example config is invalid: %v", err)
	}
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/cmd/dgconnect/commands.go`:

```go
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
	"github.com/opd-ai/go-gamelaunch-client/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

func runConnect(cmd *cobra.Command, args []string) error {
	var host, user string
	var actualPort int

	// Parse connection string or use config
	if len(args) > 0 {
		if err := parseConnectionString(args[0], &user, &host); err != nil {
			return err
		}
		actualPort = port // Use command line port
	} else {
		// Try to use default server from config
		defaultServer := viper.GetString("default_server")
		if defaultServer == "" {
			return fmt.Errorf("no server specified and no default_server in config")
		}

		serverConfig, err := GetServerConfig(defaultServer)
		if err != nil {
			return err
		}

		host = serverConfig.Host
		user = serverConfig.Username
		actualPort = serverConfig.Port
		if actualPort == 0 {
			actualPort = 22
		}
	}

	// Validate required parameters
	if host == "" {
		return fmt.Errorf("host is required")
	}
	if user == "" {
		return fmt.Errorf("username is required")
	}

	// Create client configuration
	clientConfig := dgclient.DefaultClientConfig()
	clientConfig.Debug = debug

	// Set up SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: getHostKeyCallback(),
		Timeout:         clientConfig.ConnectTimeout,
	}
	clientConfig.SSHConfig = sshConfig

	// Create client
	client := dgclient.NewClient(clientConfig)
	defer client.Close()

	// Set up view
	viewOpts := dgclient.DefaultViewOptions()
	view, err := tui.NewTerminalView(viewOpts)
	if err != nil {
		return fmt.Errorf("failed to create terminal view: %w", err)
	}

	if err := client.SetView(view); err != nil {
		return fmt.Errorf("failed to set view: %w", err)
	}

	// Get authentication method
	auth, err := getAuthMethod(user, host)
	if err != nil {
		return fmt.Errorf("failed to get authentication method: %w", err)
	}

	// Connect
	fmt.Printf("Connecting to %s@%s:%d...\n", user, host, actualPort)
	if err := client.Connect(host, actualPort, auth); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	fmt.Println("Connected successfully!")

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, disconnecting...")
		cancel()
	}()

	// Launch game if specified
	if gameName != "" {
		if err := client.SelectGame(gameName); err != nil {
			fmt.Printf("Warning: failed to select game %s: %v\n", gameName, err)
		}
	}

	// Run the client
	if err := client.Run(ctx); err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	return nil
}

func parseConnectionString(conn string, user, host *string) error {
	parts := strings.Split(conn, "@")
	if len(parts) == 2 {
		*user = parts[0]
		*host = parts[1]
	} else if len(parts) == 1 {
		*host = parts[0]
		*user = os.Getenv("USER")
		if *user == "" {
			return fmt.Errorf("no username specified and USER environment variable not set")
		}
	} else {
		return fmt.Errorf("invalid connection string: %s", conn)
	}
	return nil
}

func getAuthMethod(user, host string) (dgclient.AuthMethod, error) {
	// Priority: command line flag > config > SSH agent > default keys > password prompt

	if password != "" {
		return dgclient.NewPasswordAuth(password), nil
	}

	if keyPath != "" {
		return dgclient.NewKeyAuth(keyPath, ""), nil
	}

	// Check config for auth method
	defaultServer := viper.GetString("default_server")
	if defaultServer != "" {
		serverConfig, err := GetServerConfig(defaultServer)
		if err == nil {
			switch serverConfig.Auth.Method {
			case "key":
				if serverConfig.Auth.KeyPath != "" {
					return dgclient.NewKeyAuth(expandPath(serverConfig.Auth.KeyPath), serverConfig.Auth.Passphrase), nil
				}
			case "password":
				// Will fall through to password prompt
			case "agent":
				if os.Getenv("SSH_AUTH_SOCK") != "" {
					return dgclient.NewAgentAuth(), nil
				}
			}
		}
	}

	// Try SSH agent
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return dgclient.NewAgentAuth(), nil
	}

	// Try default key locations
	home, _ := os.UserHomeDir()
	defaultKeys := []string{
		fmt.Sprintf("%s/.ssh/id_rsa", home),
		fmt.Sprintf("%s/.ssh/id_ed25519", home),
		fmt.Sprintf("%s/.ssh/id_ecdsa", home),
	}

	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			return dgclient.NewKeyAuth(keyPath, ""), nil
		}
	}

	// Fall back to password prompt
	fmt.Printf("Password for %s@%s: ", user, host)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	return dgclient.NewPasswordAuth(string(passwordBytes)), nil
}

func getHostKeyCallback() ssh.HostKeyCallback {
	// Try to use known_hosts file first
	home, err := os.UserHomeDir()
	if err != nil {
		if debug {
			fmt.Printf("Warning: Could not get home directory: %v\n", err)
		}
		return createInsecureCallback()
	}

	knownHostsPath := fmt.Sprintf("%s/.ssh/known_hosts", home)
	if _, err := os.Stat(knownHostsPath); err != nil {
		if debug {
			fmt.Printf("Warning: known_hosts file not found at %s, using insecure callback\n", knownHostsPath)
		}
		return createInsecureCallback()
	}

	// Use known_hosts for verification
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		if debug {
			fmt.Printf("Warning: Failed to load known_hosts: %v, using insecure callback\n", err)
		}
		return createInsecureCallback()
	}

	// Wrap the callback to provide better error messages and handle unknown hosts
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := hostKeyCallback(hostname, remote, key)
		if err != nil {
			// Check if this is an unknown host error
			if keyErr, ok := err.(*knownhosts.KeyError); ok && len(keyErr.Want) == 0 {
				// Unknown host - prompt user
				fmt.Printf("\nWarning: Unknown host %s\n", hostname)
				fmt.Printf("Host key fingerprint: %s\n", ssh.FingerprintSHA256(key))
				fmt.Print("Do you want to continue connecting? (yes/no): ")

				var response string
				fmt.Scanln(&response)

				if response == "yes" || response == "y" {
					// Add to known_hosts
					if addErr := addToKnownHosts(knownHostsPath, hostname, key); addErr != nil {
						fmt.Printf("Warning: Could not add host to known_hosts: %v\n", addErr)
					} else {
						fmt.Printf("Host %s added to known_hosts\n", hostname)
					}
					return nil
				}
				return fmt.Errorf("host key verification failed: user rejected unknown host")
			}

			// Host key mismatch or other error
			if keyErr, ok := err.(*knownhosts.KeyError); ok && len(keyErr.Want) > 0 {
				fmt.Printf("\nHost key verification failed for %s!\n", hostname)
				fmt.Printf("Expected fingerprint: %s\n", ssh.FingerprintSHA256(keyErr.Want[0].Key))
				fmt.Printf("Received fingerprint: %s\n", ssh.FingerprintSHA256(key))
				return fmt.Errorf("host key verification failed: key mismatch")
			}

			return fmt.Errorf("host key verification failed: %w", err)
		}

		if debug {
			fmt.Printf("Host key verified for %s\n", hostname)
		}
		return nil
	}
}

func createInsecureCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if debug {
			fmt.Printf("Warning: Using insecure host key callback for %s\n", hostname)
			fmt.Printf("Fingerprint: %s\n", ssh.FingerprintSHA256(key))
		}
		return nil
	}
}

func addToKnownHosts(knownHostsPath, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
	_, err = f.WriteString(line)
	return err
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/cmd/dgconnect/config.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration file structure
type Config struct {
	DefaultServer string                  `yaml:"default_server,omitempty"`
	Servers       map[string]ServerConfig `yaml:"servers"`
	Preferences   PreferencesConfig       `yaml:"preferences,omitempty"`
}

// ServerConfig represents a server configuration
type ServerConfig struct {
	Host        string     `yaml:"host"`
	Port        int        `yaml:"port,omitempty"`
	Username    string     `yaml:"username"`
	Auth        AuthConfig `yaml:"auth"`
	DefaultGame string     `yaml:"default_game,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Method     string `yaml:"method"` // password, key, agent
	KeyPath    string `yaml:"key_path,omitempty"`
	Passphrase string `yaml:"passphrase,omitempty"`
}

// PreferencesConfig represents user preferences
type PreferencesConfig struct {
	Terminal          string `yaml:"terminal,omitempty"`
	ReconnectAttempts int    `yaml:"reconnect_attempts,omitempty"`
	ReconnectDelay    string `yaml:"reconnect_delay,omitempty"`
	KeepAliveInterval string `yaml:"keepalive_interval,omitempty"`
	ColorEnabled      bool   `yaml:"color_enabled"`
	UnicodeEnabled    bool   `yaml:"unicode_enabled"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GenerateExampleConfig creates an example configuration file
func GenerateExampleConfig() *Config {
	return &Config{
		DefaultServer: "nethack-server",
		Servers: map[string]ServerConfig{
			"nethack-server": {
				Host:     "nethack.example.com",
				Port:     2022,
				Username: "player1",
				Auth: AuthConfig{
					Method:  "key",
					KeyPath: "~/.ssh/dgamelaunch_rsa",
				},
				DefaultGame: "nethack",
			},
			"dcss-server": {
				Host:     "crawl.example.com",
				Port:     22,
				Username: "crawler",
				Auth: AuthConfig{
					Method: "password",
				},
			},
			"local-test": {
				Host:     "localhost",
				Port:     22,
				Username: os.Getenv("USER"),
				Auth: AuthConfig{
					Method: "agent",
				},
			},
		},
		Preferences: PreferencesConfig{
			Terminal:          "xterm-256color",
			ReconnectAttempts: 3,
			ReconnectDelay:    "5s",
			KeepAliveInterval: "30s",
			ColorEnabled:      true,
			UnicodeEnabled:    true,
		},
	}
}

// ValidateConfig checks if a configuration is valid
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if len(config.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	for name, server := range config.Servers {
		if server.Host == "" {
			return fmt.Errorf("server '%s' has no host configured", name)
		}
		if server.Username == "" {
			return fmt.Errorf("server '%s' has no username configured", name)
		}
		if server.Auth.Method == "" {
			return fmt.Errorf("server '%s' has no auth method configured", name)
		}
		if server.Auth.Method == "key" && server.Auth.KeyPath == "" {
			return fmt.Errorf("server '%s' uses key auth but no key_path specified", name)
		}
		if server.Port <= 0 {
			server.Port = 22 // Set default
		}
	}

	if config.DefaultServer != "" {
		if _, exists := config.Servers[config.DefaultServer]; !exists {
			return fmt.Errorf("default_server '%s' not found in servers list", config.DefaultServer)
		}
	}

	return nil
}

// GetServerConfig retrieves a server configuration by name
func GetServerConfig(name string) (*ServerConfig, error) {
	serverKey := fmt.Sprintf("servers.%s", name)
	if !viper.IsSet(serverKey) {
		return nil, fmt.Errorf("server '%s' not found in configuration", name)
	}

	var server ServerConfig
	if err := viper.UnmarshalKey(serverKey, &server); err != nil {
		return nil, fmt.Errorf("failed to parse server configuration: %w", err)
	}

	// Set defaults
	if server.Port == 0 {
		server.Port = 22
	}

	return &server, nil
}

```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/Makefile`:

```
fmt:
	find . -name '*.go' -not -path './vendor/*' -exec gofumpt -extra -s -w {} \;

prompt: fmt
	code2prompt --output prompt.md .
```

`/home/user/go/src/github.com/opd-ai/go-gamelaunch-client/go.sum`:

```sum
github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/frankban/quicktest v1.14.6 h1:7Xjx+VpznH+oBnejlPUj8oUpdxnVs4f8XU8WnHkI4W8=
github.com/frankban/quicktest v1.14.6/go.mod h1:4ptaffx2x8+WTWXmUCuVU6aPUX1/Mz7zb5vbUoiM6w0=
github.com/fsnotify/fsnotify v1.8.0 h1:dAwr6QBTBZIkG8roQaJjGof0pp0EeF+tNV7YBP3F/8M=
github.com/fsnotify/fsnotify v1.8.0/go.mod h1:8jBTzvmWwFyi3Pb8djgCCO5IBqzKJ/Jwo8TRcHyHii0=
github.com/gdamore/encoding v1.0.1 h1:YzKZckdBL6jVt2Gc+5p82qhrGiqMdG/eNs6Wy0u3Uhw=
github.com/gdamore/encoding v1.0.1/go.mod h1:0Z0cMFinngz9kS1QfMjCP8TY7em3bZYeeklsSDPivEo=
github.com/gdamore/tcell/v2 v2.8.1 h1:KPNxyqclpWpWQlPLx6Xui1pMk8S+7+R37h3g07997NU=
github.com/gdamore/tcell/v2 v2.8.1/go.mod h1:bj8ori1BG3OYMjmb3IklZVWfZUJ1UBQt9JXrOCOhGWw=
github.com/go-viper/mapstructure/v2 v2.2.1 h1:ZAaOCxANMuZx5RCeg0mBdEZk7DZasvvZIxtHqx8aGss=
github.com/go-viper/mapstructure/v2 v2.2.1/go.mod h1:oJDH3BJKyqBA2TXFhDsKDGDTlndYOZ6rGS0BRZIxGhM=
github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/kr/pretty v0.3.1 h1:flRD4NNwYAUpkphVc1HcthR4KEIFJ65n8Mw5qdRn3LE=
github.com/kr/pretty v0.3.1/go.mod h1:hoEshYVHaxMs3cyo3Yncou5ZscifuDolrwPKZanG3xk=
github.com/kr/text v0.2.0 h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=
github.com/kr/text v0.2.0/go.mod h1:eLer722TekiGuMkidMxC/pM04lWEeraHUUmBw8l2grE=
github.com/lucasb-eyer/go-colorful v1.2.0 h1:1nnpGOrhyZZuNyfu1QjKiUICQ74+3FNCN69Aj6K7nkY=
github.com/lucasb-eyer/go-colorful v1.2.0/go.mod h1:R4dSotOR9KMtayYi1e77YzuveK+i7ruzyGqttikkLy0=
github.com/mattn/go-runewidth v0.0.16 h1:E5ScNMtiwvlvB5paMFdw9p4kSQzbXFikJ5SQO6TULQc=
github.com/mattn/go-runewidth v0.0.16/go.mod h1:Jdepj2loyihRzMpdS35Xk/zdY8IAYHsh153qUoGf23w=
github.com/pelletier/go-toml/v2 v2.2.3 h1:YmeHyLY8mFWbdkNWwpr+qIL2bEqT0o95WSdkNHvL12M=
github.com/pelletier/go-toml/v2 v2.2.3/go.mod h1:MfCQTFTvCcUyyvvwm1+G6H/jORL20Xlb6rzQu9GuUkc=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/rivo/uniseg v0.2.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.4.3 h1:utMvzDsuh3suAEnhH0RdHmoPbU648o6CvXxTx4SBMOw=
github.com/rivo/uniseg v0.4.3/go.mod h1:FN3SvrM+Zdj16jyLfmOkMNblXMcoc8DfTHruCPUcx88=
github.com/rogpeppe/go-internal v1.9.0 h1:73kH8U+JUqXU8lRuOHeVHaa/SZPifC7BkcraZVejAe8=
github.com/rogpeppe/go-internal v1.9.0/go.mod h1:WtVeX8xhTBvf0smdhujwtBcq4Qrzq/fJaraNFVN+nFs=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/sagikazarmark/locafero v0.7.0 h1:5MqpDsTGNDhY8sGp0Aowyf0qKsPrhewaLSsFaodPcyo=
github.com/sagikazarmark/locafero v0.7.0/go.mod h1:2za3Cg5rMaTMoG/2Ulr9AwtFaIppKXTRYnozin4aB5k=
github.com/sourcegraph/conc v0.3.0 h1:OQTbbt6P72L20UqAkXXuLOj79LfEanQ+YQFNpLA9ySo=
github.com/sourcegraph/conc v0.3.0/go.mod h1:Sdozi7LEKbFPqYX2/J+iBAM6HpqSLTASQIKqDmF7Mt0=
github.com/spf13/afero v1.12.0 h1:UcOPyRBYczmFn6yvphxkn9ZEOY65cpwGKb5mL36mrqs=
github.com/spf13/afero v1.12.0/go.mod h1:ZTlWwG4/ahT8W7T0WQ5uYmjI9duaLQGy3Q2OAl4sk/4=
github.com/spf13/cast v1.7.1 h1:cuNEagBQEHWN1FnbGEjCXL2szYEXqfJPbP2HNUaca9Y=
github.com/spf13/cast v1.7.1/go.mod h1:ancEpBxwJDODSW/UG4rDrAqiKolqNNh2DX3mk86cAdo=
github.com/spf13/cobra v1.9.1 h1:CXSaggrXdbHK9CF+8ywj8Amf7PBRmPCOJugH954Nnlo=
github.com/spf13/cobra v1.9.1/go.mod h1:nDyEzZ8ogv936Cinf6g1RU9MRY64Ir93oCnqb9wxYW0=
github.com/spf13/pflag v1.0.6 h1:jFzHGLGAlb3ruxLB8MhbI6A8+AQX/2eW4qeyNZXNp2o=
github.com/spf13/pflag v1.0.6/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/spf13/viper v1.20.1 h1:ZMi+z/lvLyPSCoNtFCpqjy0S4kPbirhpTMwl8BkW9X4=
github.com/spf13/viper v1.20.1/go.mod h1:P9Mdzt1zoHIG8m2eZQinpiBjo6kCmZSKBClNNqjJvu4=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/testify v1.3.0/go.mod h1:M5WIy9Dh21IEIfnGCwXGc5bZfKNJtfHm1UVUgZn+9EI=
github.com/stretchr/testify v1.10.0 h1:Xv5erBjTwe/5IxqUQTdXv5kgmIvbHo3QQyRwhJsOfJA=
github.com/stretchr/testify v1.10.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
github.com/subosito/gotenv v1.6.0 h1:9NlTDc1FTs4qu0DDq7AEtTPNw6SVm7uBMsUCUjABIf8=
github.com/subosito/gotenv v1.6.0/go.mod h1:Dk4QP5c2W3ibzajGcXpNraDfq2IrhjMIvMSWPKKo0FU=
github.com/yuin/goldmark v1.4.13/go.mod h1:6yULJ656Px+3vBD8DxQVa3kxgyrAnzto9xy5taEt/CY=
go.uber.org/atomic v1.9.0 h1:ECmE8Bn/WFTYwEW/bpKD3M8VtR/zQVbavAoalC1PYyE=
go.uber.org/atomic v1.9.0/go.mod h1:fEN4uk6kAWBTFdckzkM89CLk9XfWZrxpCo0nPH17wJc=
go.uber.org/multierr v1.9.0 h1:7fIwc/ZtS0q++VgcfqFDxSBZVv/Xo49/SYnDFupUwlI=
go.uber.org/multierr v1.9.0/go.mod h1:X2jQV1h+kxSjClGpnseKVIxpmcjrj7MNnI0bnlfKTVQ=
golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2/go.mod h1:djNgcEr1/C05ACkg1iLfiJU5Ep61QUkGW8qpdssI0+w=
golang.org/x/crypto v0.0.0-20210921155107-089bfa567519/go.mod h1:GvvjBRRGRdwPK5ydBHafDWAxML/pGHZbMvKqRZ5+Abc=
golang.org/x/crypto v0.13.0/go.mod h1:y6Z2r+Rw4iayiXXAIxJIDAJ1zMW4yaTpebo8fPOliYc=
golang.org/x/crypto v0.19.0/go.mod h1:Iy9bg/ha4yyC70EfRS8jz+B6ybOBKMaSxLj6P6oBDfU=
golang.org/x/crypto v0.23.0/go.mod h1:CKFgDieR+mRhux2Lsu27y0fO304Db0wZe70UKqHu0v8=
golang.org/x/crypto v0.38.0 h1:jt+WWG8IZlBnVbomuhg2Mdq0+BBQaHbtqHEFEigjUV8=
golang.org/x/crypto v0.38.0/go.mod h1:MvrbAqul58NNYPKnOra203SB9vpuZW0e+RRZV+Ggqjw=
golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4/go.mod h1:jJ57K6gSWd91VN4djpZkiMVwK6gcyfeH4XE8wZrZaV4=
golang.org/x/mod v0.8.0/go.mod h1:iBbtSCu2XBx23ZKBPSOrRkjjQPZFPuis4dIYUhu/chs=
golang.org/x/mod v0.12.0/go.mod h1:iBbtSCu2XBx23ZKBPSOrRkjjQPZFPuis4dIYUhu/chs=
golang.org/x/mod v0.15.0/go.mod h1:hTbmBsO62+eylJbnUtE2MGJUyE7QWk4xUqPFrRgJ+7c=
golang.org/x/mod v0.17.0/go.mod h1:hTbmBsO62+eylJbnUtE2MGJUyE7QWk4xUqPFrRgJ+7c=
golang.org/x/net v0.0.0-20190620200207-3b0461eec859/go.mod h1:z5CRVTTTmAJ677TzLLGU+0bjPO0LkuOLi4/5GtJWs/s=
golang.org/x/net v0.0.0-20210226172049-e18ecbb05110/go.mod h1:m0MpNAwzfU5UDzcl9v0D8zg8gWTRqZa9RBIspLL5mdg=
golang.org/x/net v0.0.0-20220722155237-a158d28d115b/go.mod h1:XRhObCWvk6IyKnWLug+ECip1KBveYUHfp+8e9klMJ9c=
golang.org/x/net v0.6.0/go.mod h1:2Tu9+aMcznHK/AK1HMvgo6xiTLG5rD5rZLDS+rp2Bjs=
golang.org/x/net v0.10.0/go.mod h1:0qNGK6F8kojg2nk9dLZ2mShWaEBan6FAoqfSigmmuDg=
golang.org/x/net v0.15.0/go.mod h1:idbUs1IY1+zTqbi8yxTbhexhEEk5ur9LInksu6HrEpk=
golang.org/x/net v0.21.0/go.mod h1:bIjVDfnllIU7BJ2DNgfnXvpSvtn8VRwhlsaeUTyUS44=
golang.org/x/net v0.25.0/go.mod h1:JkAGAh7GEvH74S6FOH42FLoXpXbE/aqXSrIQjXgsiwM=
golang.org/x/sync v0.0.0-20190423024810-112230192c58/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
golang.org/x/sync v0.1.0/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
golang.org/x/sync v0.3.0/go.mod h1:FU7BRWz2tNW+3quACPkgCx/L+uEAv1htQ0V83Z9Rj+Y=
golang.org/x/sync v0.6.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
golang.org/x/sync v0.7.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
golang.org/x/sync v0.10.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
golang.org/x/sys v0.0.0-20201119102817-f84b799fce68/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.5.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.8.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.12.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.17.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
golang.org/x/sys v0.20.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
golang.org/x/sys v0.29.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
golang.org/x/sys v0.33.0 h1:q3i8TbbEz+JRD9ywIRlyRAQbM0qF7hu24q3teo2hbuw=
golang.org/x/sys v0.33.0/go.mod h1:BJP2sWEmIv4KK5OTEluFJCKSidICx8ciO85XgH3Ak8k=
golang.org/x/telemetry v0.0.0-20240228155512-f48c80bd79b2/go.mod h1:TeRTkGYfJXctD9OcfyVLyj2J3IxLnKwHJR8f4D8a3YE=
golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1/go.mod h1:bj7SfCRtBDWHUb9snDiAeCFNEtKQo2Wmx5Cou7ajbmo=
golang.org/x/term v0.0.0-20210927222741-03fcf44c2211/go.mod h1:jbD1KX2456YbFQfuXm/mYQcufACuNUgVhRMnK/tPxf8=
golang.org/x/term v0.5.0/go.mod h1:jMB1sMXY+tzblOD4FWmEbocvup2/aLOaQEp7JmGp78k=
golang.org/x/term v0.8.0/go.mod h1:xPskH00ivmX89bAKVGSKKtLOWNx2+17Eiy94tnKShWo=
golang.org/x/term v0.12.0/go.mod h1:owVbMEjm3cBLCHdkQu9b1opXd4ETQWc3BhuQGKgXgvU=
golang.org/x/term v0.17.0/go.mod h1:lLRBjIVuehSbZlaOtGMbcMncT+aqLLLmKrsjNrUguwk=
golang.org/x/term v0.20.0/go.mod h1:8UkIAJTvZgivsXaD6/pH6U9ecQzZ45awqEOzuCvwpFY=
golang.org/x/term v0.28.0/go.mod h1:Sw/lC2IAUZ92udQNf3WodGtn4k/XoLyZoh8v/8uiwek=
golang.org/x/term v0.32.0 h1:DR4lr0TjUs3epypdhTOkMmuF5CDFJ/8pOnbzMZPQ7bg=
golang.org/x/term v0.32.0/go.mod h1:uZG1FhGx848Sqfsq4/DlJr3xGGsYMu/L5GW4abiaEPQ=
golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
golang.org/x/text v0.3.3/go.mod h1:5Zoc/QRtKVWzQhOtBMvqHzDpF6irO9z98xDceosuGiQ=
golang.org/x/text v0.3.7/go.mod h1:u+2+/6zg+i71rQMx5EYifcz6MCKuco9NR6JIITiCfzQ=
golang.org/x/text v0.7.0/go.mod h1:mrYo+phRRbMaCq/xk9113O4dZlRixOauAjOtrjsXDZ8=
golang.org/x/text v0.9.0/go.mod h1:e1OnstbJyHTd6l/uOt8jFFHp6TRDWZR/bV3emEE/zU8=
golang.org/x/text v0.13.0/go.mod h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=
golang.org/x/text v0.14.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
golang.org/x/text v0.15.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
golang.org/x/text v0.21.0/go.mod h1:4IBbMaMmOPCJ8SecivzSH54+73PCFmPWxNTLm+vZkEQ=
golang.org/x/text v0.25.0 h1:qVyWApTSYLk/drJRO5mDlNYskwQznZmkpV2c8q9zls4=
golang.org/x/text v0.25.0/go.mod h1:WEdwpYrmk1qmdHvhkSTNPm3app7v4rsT8F2UD6+VHIA=
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e/go.mod h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ=
golang.org/x/tools v0.0.0-20191119224855-298f0cb1881e/go.mod h1:b+2E5dAYhXwXZwtnZ6UAqBI28+e2cm9otk0dWdXHAEo=
golang.org/x/tools v0.1.12/go.mod h1:hNGJHUnrk76NpqgfD5Aqm5Crs+Hm0VOH/i9J2+nxYbc=
golang.org/x/tools v0.6.0/go.mod h1:Xwgl3UAJ/d3gWutnCtw505GrjyAbvKui8lOU390QaIU=
golang.org/x/tools v0.13.0/go.mod h1:HvlwmtVNQAhOuCjW7xxvovg8wbNq7LwfXh/k7wXUl58=
golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d/go.mod h1:aiJjzUbINMkxbQROHiO6hDPo2LHcIPhhQsa9DLh0yGk=
golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7/go.mod h1:I/5z698sn9Ka8TeJc9MKroUUfqBBauWjQqLJ2OPfmY0=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 h1:YR8cESwS4TdDjEe65xsg0ogRM/Nc3DYOhEAlW+xobZo=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=

```