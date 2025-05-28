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
