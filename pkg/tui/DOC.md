# tui
--
    import "github.com/opd-ai/go-gamelaunch-client/pkg/tui"


## Usage

#### func  NewTerminalView

```go
func NewTerminalView(opts dgclient.ViewOptions) (dgclient.View, error)
```
NewTerminalView creates a new terminal-based view

#### type AnsiParser

```go
type AnsiParser struct {
}
```

AnsiParser handles ANSI escape sequence parsing

#### type BufferedReader

```go
type BufferedReader struct {
}
```

BufferedReader provides a buffered input reader

#### func  NewBufferedReader

```go
func NewBufferedReader(inputCh <-chan []byte) *BufferedReader
```
NewBufferedReader creates a new buffered reader

#### func (*BufferedReader) Read

```go
func (r *BufferedReader) Read(p []byte) (n int, err error)
```
Read implements io.Reader

#### type Cell

```go
type Cell struct {
	Char rune
	Attr CellAttributes
}
```

Cell represents a single character cell with attributes

#### type CellAttributes

```go
type CellAttributes struct {
	Foreground Color
	Background Color
	Bold       bool
	Underline  bool
	Reverse    bool
}
```

CellAttributes stores text formatting information

#### type Color

```go
type Color struct {
	R, G, B uint8
	IsIndex bool
	Index   uint8
}
```

Color represents a terminal color

#### type InputBuffer

```go
type InputBuffer struct {
}
```

InputBuffer provides input buffering with line editing capabilities

#### func  NewInputBuffer

```go
func NewInputBuffer() *InputBuffer
```
NewInputBuffer creates a new input buffer

#### func (*InputBuffer) AddChar

```go
func (b *InputBuffer) AddChar(r rune)
```
AddChar adds a character at the current position

#### func (*InputBuffer) DeleteChar

```go
func (b *InputBuffer) DeleteChar() bool
```
DeleteChar deletes the character before the cursor

#### func (*InputBuffer) GetLine

```go
func (b *InputBuffer) GetLine() string
```
GetLine returns the current line and resets the buffer

#### func (*InputBuffer) HistoryDown

```go
func (b *InputBuffer) HistoryDown() bool
```
HistoryDown moves down in history

#### func (*InputBuffer) HistoryUp

```go
func (b *InputBuffer) HistoryUp() bool
```
HistoryUp moves up in history

#### func (*InputBuffer) MoveLeft

```go
func (b *InputBuffer) MoveLeft() bool
```
MoveLeft moves the cursor left

#### func (*InputBuffer) MoveRight

```go
func (b *InputBuffer) MoveRight() bool
```
MoveRight moves the cursor right

#### type InputHandler

```go
type InputHandler struct {
}
```

InputHandler processes user input with different modes

#### func  NewInputHandler

```go
func NewInputHandler() *InputHandler
```
NewInputHandler creates a new input handler

#### func (*InputHandler) ProcessKey

```go
func (h *InputHandler) ProcessKey(ev *tcell.EventKey) ([]byte, bool)
```
ProcessKey processes a key event based on current mode

#### func (*InputHandler) SetMode

```go
func (h *InputHandler) SetMode(mode InputMode)
```
SetMode changes the input processing mode

#### type InputMode

```go
type InputMode int
```

InputMode defines how input is processed

```go
const (
	// InputModeNormal processes input normally
	InputModeNormal InputMode = iota

	// InputModeRaw sends all input without processing
	InputModeRaw

	// InputModePassword hides input
	InputModePassword
)
```

#### type ParserState

```go
type ParserState int
```


```go
const (
	StateNormal ParserState = iota
	StateEscape
	StateCSI
	StateOSC
)
```

#### type TerminalEmulator

```go
type TerminalEmulator struct {
}
```

TerminalEmulator provides a proper terminal emulation layer

#### func  NewTerminalEmulator

```go
func NewTerminalEmulator(width, height int) *TerminalEmulator
```
NewTerminalEmulator creates a new terminal emulator

#### func (*TerminalEmulator) GetCursor

```go
func (te *TerminalEmulator) GetCursor() (int, int)
```
GetCursor returns the current cursor position

#### func (*TerminalEmulator) GetScreen

```go
func (te *TerminalEmulator) GetScreen() [][]Cell
```
GetScreen returns a copy of the current screen state

#### func (*TerminalEmulator) ProcessData

```go
func (te *TerminalEmulator) ProcessData(data []byte)
```
ProcessData processes incoming terminal data and updates the screen

#### func (*TerminalEmulator) Resize

```go
func (te *TerminalEmulator) Resize(width, height int)
```
Resize changes the terminal dimensions

#### type TerminalView

```go
type TerminalView struct {
}
```

TerminalView implements dgclient.View using tcell for terminal rendering

#### func (*TerminalView) Clear

```go
func (v *TerminalView) Clear() error
```
Clear clears the display

#### func (*TerminalView) Close

```go
func (v *TerminalView) Close() error
```
Close cleans up resources

#### func (*TerminalView) GetSize

```go
func (v *TerminalView) GetSize() (width, height int)
```
GetSize returns current dimensions

#### func (*TerminalView) HandleInput

```go
func (v *TerminalView) HandleInput() ([]byte, error)
```
HandleInput reads and returns user input

#### func (*TerminalView) Init

```go
func (v *TerminalView) Init() error
```
Init initializes the terminal view

#### func (*TerminalView) Render

```go
func (v *TerminalView) Render(data []byte) error
```
Render displays the provided data

#### func (*TerminalView) SetSize

```go
func (v *TerminalView) SetSize(width, height int) error
```
SetSize updates the view dimensions
