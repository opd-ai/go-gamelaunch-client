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
