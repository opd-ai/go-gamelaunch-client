# webui
--
    import "github.com/opd-ai/go-gamelaunch-client/pkg/webui"


## Usage

```go
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)
```
Standard JSON-RPC error codes

#### func  CreateWebView

```go
func CreateWebView(opts dgclient.ViewOptions) (dgclient.View, error)
```
CreateWebView creates a new WebView that implements dgclient.View

#### func  SaveTilesetConfig

```go
func SaveTilesetConfig(tileset *TilesetConfig, path string) error
```
SaveTilesetConfig saves a tileset configuration to a YAML file

#### type Cell

```go
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
```

Cell represents a single character cell with rendering attributes

#### type CellDiff

```go
type CellDiff struct {
	X    int  `json:"x"`
	Y    int  `json:"y"`
	Cell Cell `json:"cell"`
}
```

CellDiff represents a change to a specific cell

#### type GameInputParams

```go
type GameInputParams struct {
	Events []InputEvent `json:"events"`
}
```

GameInputParams represents parameters for game.sendInput method

#### type GamePollParams

```go
type GamePollParams struct {
	Version uint64 `json:"version"`
	Timeout int    `json:"timeout,omitempty"`
}
```

GamePollParams represents parameters for game.poll method

#### type GameState

```go
type GameState struct {
	Buffer    [][]Cell `json:"buffer"`
	Width     int      `json:"width"`
	Height    int      `json:"height"`
	CursorX   int      `json:"cursor_x"`
	CursorY   int      `json:"cursor_y"`
	Version   uint64   `json:"version"`
	Timestamp int64    `json:"timestamp"`
}
```

GameState represents the current state of the game screen

#### type InputEvent

```go
type InputEvent struct {
	Type      string `json:"type"`
	Key       string `json:"key,omitempty"`
	KeyCode   int    `json:"keyCode,omitempty"`
	Data      string `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}
```

InputEvent represents a user input event

#### type RPCError

```go
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
```

RPCError represents a JSON-RPC 2.0 error

#### type RPCHandler

```go
type RPCHandler struct {
}
```

RPCHandler handles JSON-RPC method calls

#### func  NewRPCHandler

```go
func NewRPCHandler(webui *WebUI) *RPCHandler
```
NewRPCHandler creates a new RPC handler

#### func (*RPCHandler) HandleRequest

```go
func (h *RPCHandler) HandleRequest(ctx context.Context, req *RPCRequest) *RPCResponse
```
HandleRequest processes a JSON-RPC request

#### type RPCRequest

```go
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}
```

RPCRequest represents a JSON-RPC 2.0 request

#### type RPCResponse

```go
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}
```

RPCResponse represents a JSON-RPC 2.0 response

#### type SpecialTile

```go
type SpecialTile struct {
	ID    string    `yaml:"id"`
	Tiles []TileRef `yaml:"tiles"`
}
```

SpecialTile represents multi-tile entities

#### type StateDiff

```go
type StateDiff struct {
	Version   uint64     `json:"version"`
	Changes   []CellDiff `json:"changes"`
	CursorX   int        `json:"cursor_x"`
	CursorY   int        `json:"cursor_y"`
	Timestamp int64      `json:"timestamp"`
}
```

StateDiff represents changes between game states

#### type StateManager

```go
type StateManager struct {
}
```

StateManager manages game state versions and change tracking

#### func  NewStateManager

```go
func NewStateManager() *StateManager
```
NewStateManager creates a new state manager

#### func (*StateManager) GetCurrentState

```go
func (sm *StateManager) GetCurrentState() *GameState
```
GetCurrentState returns the current state

#### func (*StateManager) GetCurrentVersion

```go
func (sm *StateManager) GetCurrentVersion() uint64
```
GetCurrentVersion returns the current version number

#### func (*StateManager) PollChanges

```go
func (sm *StateManager) PollChanges(clientVersion uint64, timeout time.Duration) (*StateDiff, error)
```
PollChanges waits for changes since the specified version

#### func (*StateManager) UpdateState

```go
func (sm *StateManager) UpdateState(state *GameState)
```
UpdateState updates the current state and notifies waiters

#### type TileMapping

```go
type TileMapping struct {
	Char    string `yaml:"char"`
	X       int    `yaml:"x"`
	Y       int    `yaml:"y"`
	FgColor string `yaml:"fg_color,omitempty"`
	BgColor string `yaml:"bg_color,omitempty"`
}
```

TileMapping maps characters to tile coordinates

#### type TileRef

```go
type TileRef struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
}
```

TileRef references a specific tile

#### type TilesetConfig

```go
type TilesetConfig struct {
	Name         string        `yaml:"name"`
	Version      string        `yaml:"version"`
	TileWidth    int           `yaml:"tile_width"`
	TileHeight   int           `yaml:"tile_height"`
	SourceImage  string        `yaml:"source_image"`
	Mappings     []TileMapping `yaml:"mappings"`
	SpecialTiles []SpecialTile `yaml:"special_tiles"`
}
```

TilesetConfig represents a tileset configuration

#### func  DefaultTilesetConfig

```go
func DefaultTilesetConfig() *TilesetConfig
```
DefaultTilesetConfig returns a basic ASCII tileset configuration

#### func  LoadTilesetConfig

```go
func LoadTilesetConfig(path string) (*TilesetConfig, error)
```
LoadTilesetConfig loads a tileset from a YAML file

#### func (*TilesetConfig) Clone

```go
func (tc *TilesetConfig) Clone() *TilesetConfig
```
Clone creates a deep copy of the tileset configuration

#### func (*TilesetConfig) GetImageData

```go
func (tc *TilesetConfig) GetImageData() image.Image
```
GetImageData returns the loaded image data

#### func (*TilesetConfig) GetMapping

```go
func (tc *TilesetConfig) GetMapping(char rune) *TileMapping
```
GetMapping returns the tile mapping for a character

#### func (*TilesetConfig) GetTileCount

```go
func (tc *TilesetConfig) GetTileCount() (int, int)
```
GetTileCount returns the number of tiles in the tileset

#### func (*TilesetConfig) ToJSON

```go
func (tc *TilesetConfig) ToJSON() map[string]interface{}
```
ToJSON returns a JSON representation for client-side use

#### type WebUI

```go
type WebUI struct {
}
```

WebUI provides a web-based interface for dgclient

#### func  NewWebUI

```go
func NewWebUI(opts WebUIOptions) (*WebUI, error)
```
NewWebUI creates a new WebUI instance

#### func (*WebUI) GetTileset

```go
func (w *WebUI) GetTileset() *TilesetConfig
```
GetTileset returns the current tileset configuration

#### func (*WebUI) GetView

```go
func (w *WebUI) GetView() *WebView
```
GetView returns the current view

#### func (*WebUI) ServeHTTP

```go
func (w *WebUI) ServeHTTP(rw http.ResponseWriter, r *http.Request)
```
ServeHTTP implements http.Handler

#### func (*WebUI) SetView

```go
func (w *WebUI) SetView(view *WebView)
```
SetView sets the view for the WebUI

#### func (*WebUI) Start

```go
func (w *WebUI) Start(addr string) error
```
Start starts the WebUI server

#### func (*WebUI) StartWithContext

```go
func (w *WebUI) StartWithContext(ctx context.Context, addr string) error
```
StartWithContext starts the WebUI server with context for graceful shutdown

#### func (*WebUI) UpdateTileset

```go
func (w *WebUI) UpdateTileset(tileset *TilesetConfig) error
```
UpdateTileset updates the tileset configuration

#### type WebUIOptions

```go
type WebUIOptions struct {
	// View to use for rendering
	View *WebView

	// Tileset configuration
	TilesetPath string
	Tileset     *TilesetConfig

	// Server configuration
	ListenAddr  string
	PollTimeout time.Duration

	// CORS settings
	AllowOrigins []string

	// Static file serving
	StaticPath string // Optional: override embedded files
}
```

WebUIOptions contains configuration for WebUI

#### type WebView

```go
type WebView struct {
}
```

WebView implements dgclient.View for web browser rendering

#### func  NewWebView

```go
func NewWebView(opts dgclient.ViewOptions) (*WebView, error)
```
NewWebView creates a new web-based view

#### func (*WebView) Clear

```go
func (v *WebView) Clear() error
```
Clear clears the display

#### func (*WebView) Close

```go
func (v *WebView) Close() error
```
Close cleans up resources

#### func (*WebView) GetCurrentState

```go
func (v *WebView) GetCurrentState() *GameState
```
GetCurrentState returns the current game state

#### func (*WebView) GetSize

```go
func (v *WebView) GetSize() (width, height int)
```
GetSize returns current dimensions

#### func (*WebView) HandleInput

```go
func (v *WebView) HandleInput() ([]byte, error)
```
HandleInput reads and returns user input

#### func (*WebView) Init

```go
func (v *WebView) Init() error
```
Init initializes the web view

#### func (*WebView) Render

```go
func (v *WebView) Render(data []byte) error
```
Render processes terminal data and updates the screen buffer

#### func (*WebView) SendInput

```go
func (v *WebView) SendInput(data []byte)
```
SendInput queues input from web client

#### func (*WebView) SetSize

```go
func (v *WebView) SetSize(width, height int) error
```
SetSize updates the view dimensions

#### func (*WebView) SetTileset

```go
func (v *WebView) SetTileset(tileset *TilesetConfig)
```
SetTileset updates the tileset configuration

#### func (*WebView) WaitForUpdate

```go
func (v *WebView) WaitForUpdate(timeout time.Duration) bool
```
WaitForUpdate waits for the next screen update
