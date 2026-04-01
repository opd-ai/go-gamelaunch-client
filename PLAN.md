# Ebiten Graphical Tileset Rendering — Implementation Plan

## 1. Architecture Overview

### Current Architecture

```
cmd/dgconnect/          ← CLI entry point (cobra)
├── main.go             ← rootCmd, flags, initConfig
├── commands.go         ← runConnect() creates TerminalView, wires Client
└── config.go           ← Config structs, YAML parsing

pkg/dgclient/           ← Core library
├── view.go             ← View interface (7 methods), ViewOptions, ViewFactory
├── client.go           ← Client struct, SetView(), connection management
├── run.go              ← Run() loop: SSH session → PTY → stdout→Render / HandleInput→stdin
├── session.go          ← Session interface (PTY, pipes, shell)
├── auth.go             ← AuthMethod implementations
└── errors.go           ← Custom error types

pkg/tui/                ← Terminal UI (only existing View implementation)
├── tui.go              ← TerminalView: tcell screen + TerminalEmulator
├── emulator.go         ← ANSI parser, Cell/CellAttributes, screen buffer [][]Cell
└── input.go            ← InputHandler: tcell events → bytes
```

### Proposed Architecture with Ebiten View

```
pkg/gfx/                    ← NEW: Ebiten graphical view
├── view.go                  ← EbitenView implementing dgclient.View
├── tileset.go               ← Tileset loading, sprite atlas management
├── tilemap.go               ← Character-to-tile mapping engine
├── renderer.go              ← Screen buffer → Ebiten draw calls
├── input.go                 ← Ebiten keyboard/mouse → byte sequences
└── config.go                ← GraphicsConfig, tileset definitions

cmd/dgconnect/
├── main.go                  ← Add --graphics / --view flag
└── commands.go              ← ViewFactory dispatch: "terminal" | "ebiten"
```

### Data Flow (Ebiten Mode)

```
SSH stdout bytes
    │
    ▼
EbitenView.Render(data)
    │
    ▼
TerminalEmulator.ProcessData(data)     ← Reuses existing pkg/tui emulator
    │
    ▼
Screen buffer: [][]Cell                ← Each Cell has Char + CellAttributes
    │
    ▼
TileMapper.MapCell(cell) → TileID      ← Maps (rune, fg color) → sprite index
    │
    ▼
Renderer draws tiles to Ebiten screen  ← Tile atlas + sprite batching
    │
    ▼
Ebiten window displays graphical output

User keyboard input
    │
    ▼
ebiten.Key events → InputHandler → byte sequences
    │
    ▼
EbitenView.HandleInput() returns bytes → SSH stdin
```

### Key Design Decisions

1. **Reuse `pkg/tui.TerminalEmulator`** — The emulator already parses ANSI sequences and maintains a `[][]Cell` screen buffer. The Ebiten view embeds this emulator and reads its screen state, rather than duplicating terminal parsing logic.

2. **`View` interface unchanged** — `EbitenView` implements `dgclient.View` exactly. No changes to the interface contract. The `Client` is agnostic to which view is active.

3. **ViewOptions.Config for graphics settings** — The existing `Config map[string]interface{}` field on `ViewOptions` carries tileset configuration without adding new fields to the core library.

4. **Terminal mode remains default** — The `--view` flag defaults to `"terminal"`. Ebiten is opt-in via `--view ebiten` or `preferences.view_mode: ebiten` in config.

---

## 2. Implementation Checklist

### Phase 1: Core Package Scaffolding

- [ ] **1.1** Add `github.com/hajimehoshi/ebiten/v2` dependency to `go.mod`
- [ ] **1.2** Create `pkg/gfx/` package directory
- [ ] **1.3** Create `pkg/gfx/doc.go` — Package-level documentation describing the Ebiten view
- [ ] **1.4** Create `pkg/gfx/config.go` — `GraphicsConfig` struct with tileset path, tile size, scale factor, fallback font settings

### Phase 2: Tileset Loading

- [ ] **2.1** Create `pkg/gfx/tileset.go` — `Tileset` struct holding an `*ebiten.Image` atlas, tile dimensions, and column count
- [ ] **2.2** Implement `LoadTileset(path string, tileWidth, tileHeight int) (*Tileset, error)` — Loads PNG atlas, validates dimensions divisible by tile size
- [ ] **2.3** Implement `(*Tileset) TileAt(index int) *ebiten.Image` — Returns sub-image for a tile index using `SubImage()`
- [ ] **2.4** Add built-in fallback: render characters as bitmap text using Ebiten's text package when no tileset is loaded

### Phase 3: Character-to-Tile Mapping

- [ ] **3.1** Create `pkg/gfx/tilemap.go` — `TileMapper` struct with mapping tables
- [ ] **3.2** Define `TileMapping` struct: `{ Char rune, FGColor tui.Color, TileIndex int }`
- [ ] **3.3** Implement `LoadTileMap(path string) (*TileMapper, error)` — Parse YAML/JSON mapping files
- [ ] **3.4** Implement `(*TileMapper) Resolve(char rune, attr tui.CellAttributes) int` — Returns tile index for a cell; falls back to ASCII rendering if unmapped
- [ ] **3.5** Create default mapping files for DCSS (`mappings/dcss.yaml`), NetHack (`mappings/nethack.yaml`), and Caves of Qud (`mappings/qud.yaml`)
- [ ] **3.6** Support color-aware mapping: same character with different foreground colors can map to different tiles (e.g., `@` red vs `@` white = different monsters)

### Phase 4: Ebiten View Implementation

- [ ] **4.1** Create `pkg/gfx/view.go` — `EbitenView` struct embedding `*tui.TerminalEmulator`
- [ ] **4.2** Implement `NewEbitenView(opts dgclient.ViewOptions) (dgclient.View, error)` — Constructor parsing `GraphicsConfig` from `opts.Config`
- [ ] **4.3** Implement `Init()` — Start Ebiten game loop in a goroutine via `ebiten.RunGame()`
- [ ] **4.4** Implement `Render(data []byte)` — Feed data to embedded emulator, mark screen dirty
- [ ] **4.5** Implement `Clear()` — Reset emulator and clear Ebiten screen
- [ ] **4.6** Implement `SetSize(w, h int)` / `GetSize()` — Resize emulator; recalculate Ebiten window dimensions as `(w * tileWidth * scale, h * tileHeight * scale)`
- [ ] **4.7** Implement `HandleInput()` — Return bytes from input channel (populated by Ebiten key events)
- [ ] **4.8** Implement `Close()` — Signal Ebiten game loop termination, clean up resources

### Phase 5: Ebiten Renderer

- [ ] **5.1** Create `pkg/gfx/renderer.go` — `Renderer` struct managing draw state
- [ ] **5.2** Implement `(*EbitenView) Update()` (ebiten.Game interface) — Process pending input events, check for resize
- [ ] **5.3** Implement `(*EbitenView) Draw(screen *ebiten.Image)` — Iterate emulator `GetScreen()`, resolve each cell to a tile, batch-draw using `DrawImage` with `GeoM` translation
- [ ] **5.4** Implement `(*EbitenView) Layout(outsideWidth, outsideHeight int)` — Return logical screen size based on terminal dimensions × tile size
- [ ] **5.5** Optimize rendering: only redraw cells that changed since last frame (dirty rectangle tracking)
- [ ] **5.6** Render cursor: draw blinking cursor overlay at emulator cursor position

### Phase 6: Input Handling

- [ ] **6.1** Create `pkg/gfx/input.go` — `InputHandler` for Ebiten key events
- [ ] **6.2** Map `ebiten.Key` constants to terminal byte sequences (arrows → `\x1b[A-D`, function keys, ctrl combinations)
- [ ] **6.3** Handle text input via `ebiten.AppendInputChars()` for printable characters
- [ ] **6.4** Support mouse events: translate pixel coordinates to cell positions, generate terminal mouse escape sequences if applicable

### Phase 7: CLI Integration

- [ ] **7.1** Add `--view` flag to `cmd/dgconnect/main.go`: `rootCmd.Flags().StringVar(&viewMode, "view", "terminal", "view mode: terminal, ebiten")`
- [ ] **7.2** Add `--tileset` flag: path to tileset PNG
- [ ] **7.3** Add `--tilemap` flag: path to character mapping file
- [ ] **7.4** Add `--scale` flag: integer pixel scale factor (default 1)
- [ ] **7.5** Update `runConnect()` in `commands.go` to dispatch view creation based on `--view` flag
- [ ] **7.6** Add `preferences.view_mode`, `preferences.tileset`, `preferences.tilemap`, `preferences.scale` to `PreferencesConfig` struct in `config.go`

### Phase 8: Testing

- [ ] **8.1** Create `pkg/gfx/tileset_test.go` — Test tileset loading, sub-image extraction, invalid dimensions
- [ ] **8.2** Create `pkg/gfx/tilemap_test.go` — Test mapping resolution, fallback behavior, color-aware mapping
- [ ] **8.3** Create `pkg/gfx/view_test.go` — Test EbitenView satisfies `dgclient.View` interface (compile-time check), Init/Render/Close lifecycle
- [ ] **8.4** Create `pkg/gfx/input_test.go` — Test key-to-byte mapping for special keys, arrow keys, ctrl combinations
- [ ] **8.5** Add integration test: feed known ANSI output, verify tile indices resolved for each cell

### Phase 9: Documentation & Examples

- [ ] **9.1** Update `README.md` — Add "Graphics Mode" section with usage examples
- [ ] **9.2** Create `pkg/gfx/DOC.md` — Package documentation via godocdown
- [ ] **9.3** Create `examples/ebiten-basic/` — Minimal example connecting with graphical tileset
- [ ] **9.4** Document tileset format requirements and mapping file schema
- [ ] **9.5** Update `Makefile` godoc target to include `pkg/gfx`

---

## 3. Tileset Integration Strategy

### Supported Tileset Formats

All target games use **PNG sprite atlases** — a single image containing tiles arranged in a grid:

| Game | Standard Tileset | Tile Size | Atlas Layout |
|------|-----------------|-----------|--------------|
| DCSS | `dc-dngn.png`, `dc-mon.png`, `dc-item.png` | 32×32 | Multiple category atlases |
| NetHack | `nhtiles.png` (via Slash'EM / UnNetHack) | 16×16 or 32×32 | Single atlas, row-major |
| Caves of Qud | Custom PNG export | 16×24 (CP437) | Single atlas, codepage 437 layout |

### Tileset Loading

```
tileset.png (e.g., 512×512 with 32×32 tiles)
    ├── Loaded as *ebiten.Image
    ├── Grid: 16 columns × 16 rows = 256 tiles
    └── Tile at index N = SubImage at:
            x = (N % cols) * tileWidth
            y = (N / cols) * tileHeight
```

The `Tileset` struct stores the atlas image plus tile dimensions. `TileAt(index)` returns a sub-image without allocating new textures.

### Character Mapping Approach

Terminal output is already parsed into `[][]Cell` by the emulator. Each cell contains:
- `Char rune` — The ASCII/Unicode character (e.g., `@`, `#`, `.`, `D`)
- `Attr.Foreground` — Color (R, G, B)
- `Attr.Bold` — Bold flag (often indicates bright color variant)

The **TileMapper** translates each cell to a tile index using a priority cascade:

1. **Exact match** — `(char, fg_color)` → specific tile index
2. **Char-only match** — `char` → default tile for that character
3. **Fallback** — Render the raw character as bitmap text on a colored background

### Mapping File Format (YAML)

```yaml
# mappings/nethack.yaml
game: nethack
tileset: nethack-32x32.png
tile_width: 32
tile_height: 32

mappings:
  # Player character
  - char: "@"
    tile: 64
    
  # Walls (color differentiates dungeon features)
  - char: "#"
    tile: 3      # corridor
  - char: "-"
    tile: 1      # horizontal wall
  - char: "|"
    tile: 2      # vertical wall
    
  # Floor
  - char: "."
    tile: 0      # floor
    
  # Monsters — color-aware mapping
  - char: "D"
    fg: [255, 0, 0]      # red
    tile: 180             # red dragon
  - char: "D"
    fg: [255, 255, 255]   # white
    tile: 181             # white dragon
  - char: "D"
    tile: 178             # generic dragon (fallback)
    
  # Items
  - char: ")"
    tile: 96              # weapon
  - char: "["
    tile: 112             # armor
  - char: "!"
    tile: 128             # potion
  - char: "?"
    tile: 144             # scroll
    
  # Stairs
  - char: ">"
    tile: 10              # downstairs
  - char: "<"
    tile: 11              # upstairs
    
  # Doors
  - char: "+"
    tile: 8               # closed door
```

### Color Matching Strategy

Roguelikes use foreground color to distinguish entities sharing the same character. The mapper uses **nearest-color matching** with a configurable tolerance:

1. If `fg` is specified in the mapping, compute Euclidean distance in RGB space
2. Match if distance < threshold (default: 30)
3. If multiple color entries match, prefer the closest

This handles the common case where terminal colors vary slightly between implementations (e.g., "red" might be `(255,0,0)` or `(205,0,0)`).

### Codepage 437 Mode (Caves of Qud)

Caves of Qud and some other games use CP437 tilesets where tile index = character code. For these:

```yaml
game: qud
tileset: cp437-16x24.png
tile_width: 16
tile_height: 24
mode: cp437   # tile index = codepoint (0-255), color applied as tint
```

In CP437 mode, the mapper doesn't use a mapping table. Instead:
- Tile index = `int(cell.Char)` for chars 0–255
- The cell's foreground color is applied as an Ebiten `ColorScale` tint to the white-on-transparent CP437 glyph
- Background color fills the cell rect before drawing the glyph

---

## 4. Configuration Schema

### New ViewOptions Usage

The existing `ViewOptions.Config map[string]interface{}` carries graphics configuration:

```go
// In cmd/dgconnect/commands.go, when --view=ebiten:
viewOpts := dgclient.DefaultViewOptions()
viewOpts.Config["graphics"] = gfx.GraphicsConfig{
    TilesetPath:  tilesetFlag,   // --tileset flag
    TileMapPath:  tilemapFlag,   // --tilemap flag
    TileWidth:    32,
    TileHeight:   32,
    ScaleFactor:  scaleFlag,     // --scale flag
    WindowTitle:  "dgconnect",
    VSync:        true,
    ShowFPS:      false,
}
```

### GraphicsConfig Struct (pkg/gfx/config.go)

```go
type GraphicsConfig struct {
    // Tileset atlas image path (PNG)
    TilesetPath string `yaml:"tileset_path"`
    
    // Character-to-tile mapping file path (YAML)
    TileMapPath string `yaml:"tilemap_path"`
    
    // Individual tile dimensions in pixels
    TileWidth  int `yaml:"tile_width"`
    TileHeight int `yaml:"tile_height"`
    
    // Integer scale factor for pixel-perfect scaling (1 = native, 2 = double, etc.)
    ScaleFactor int `yaml:"scale_factor"`
    
    // Ebiten window title
    WindowTitle string `yaml:"window_title"`
    
    // Enable vertical sync
    VSync bool `yaml:"vsync"`
    
    // Show FPS counter overlay
    ShowFPS bool `yaml:"show_fps"`
    
    // Fallback font for unmapped characters (path to TTF)
    FallbackFontPath string `yaml:"fallback_font_path"`
    
    // Fallback font size in points
    FallbackFontSize float64 `yaml:"fallback_font_size"`
    
    // CP437 mode: treat tile index = codepoint, apply color as tint
    CP437Mode bool `yaml:"cp437_mode"`
}
```

### Updated Configuration File Schema

```yaml
# ~/.dgconnect.yaml
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

preferences:
  terminal: xterm-256color
  reconnect_attempts: 3
  reconnect_delay: 5s
  keepalive_interval: 30s
  color_enabled: true
  unicode_enabled: true
  
  # NEW: Graphics settings
  view_mode: terminal           # "terminal" (default) or "ebiten"
  graphics:
    tileset_path: ~/.dgconnect/tilesets/nethack-32x32.png
    tilemap_path: ~/.dgconnect/mappings/nethack.yaml
    tile_width: 32
    tile_height: 32
    scale_factor: 2
    vsync: true
    show_fps: false
    fallback_font_path: ""      # empty = use built-in
    fallback_font_size: 16.0
    cp437_mode: false
```

### New CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--view` | string | `"terminal"` | View mode: `terminal` or `ebiten` |
| `--tileset` | string | `""` | Path to tileset PNG atlas |
| `--tilemap` | string | `""` | Path to character mapping YAML |
| `--scale` | int | `1` | Pixel scale factor |

Flags override config file values. Config file values override defaults.

---

## 5. Testing Plan

### Unit Tests

| Test File | What It Validates |
|-----------|-------------------|
| `pkg/gfx/tileset_test.go` | PNG loading, tile extraction by index, out-of-bounds index handling, non-square tiles, invalid file paths |
| `pkg/gfx/tilemap_test.go` | YAML parsing, exact char match, color-aware match, fallback to char-only, unmapped char returns sentinel, CP437 mode index calculation |
| `pkg/gfx/view_test.go` | Compile-time `var _ dgclient.View = (*EbitenView)(nil)` check, config parsing from `ViewOptions.Config`, Init/Close lifecycle |
| `pkg/gfx/input_test.go` | Arrow keys produce `\x1b[A`–`\x1b[D`, Enter produces `\r`, Ctrl+C produces `\x03`, printable chars pass through |

### Integration Tests

| Test | Procedure | Expected Result |
|------|-----------|-----------------|
| **Emulator → Tile Mapping** | Feed ANSI sequence `\x1b[31mD\x1b[0m` (red "D") to emulator, read cell, resolve tile | Returns red dragon tile index from NetHack mapping |
| **Full Render Pipeline** | Create EbitenView with test tileset, call `Render()` with known output, inspect draw calls | Correct tiles drawn at correct grid positions |
| **Resize Handling** | Call `SetSize(120, 40)`, verify Ebiten window resizes to `120*tileW*scale × 40*tileH*scale` | Window dimensions update, emulator resized |
| **Fallback Rendering** | Load tilemap missing entry for `%`, render a cell with `%` | Character rendered as bitmap text instead of tile |

### Game-Specific Verification

Each game has distinct character conventions. Verification uses captured terminal output from real sessions:

| Game | Test Characters | Verification |
|------|----------------|--------------|
| **NetHack** | `@` (player), `D` (dragon), `#` (corridor), `.` (floor), `>` `<` (stairs), `)` `[` `!` `?` (items) | Tiles render for all standard dungeon features; monsters differentiated by color |
| **DCSS** | `@` (player), `#` (wall), `.` (floor), various monster letters with color | Multiple atlas files load correctly; tile categories (dungeon, monster, item) resolve from separate atlases |
| **Caves of Qud** | CP437 glyphs (box-drawing, special symbols) | CP437 mode correctly maps codepoints 0–255; colors applied as tints |

### Backwards Compatibility Verification

| Test | Procedure | Expected Result |
|------|-----------|-----------------|
| **Default mode unchanged** | Run `dgconnect user@host` without `--view` flag | Terminal mode (tcell) activates, no Ebiten dependency loaded at runtime |
| **View interface contract** | Run full `go test ./pkg/dgclient/...` | All existing client tests pass with no changes |
| **Config backwards compat** | Load existing `~/.dgconnect.yaml` without `graphics` section | No errors; `view_mode` defaults to `"terminal"` |
| **Build without CGo** | Build with `CGO_ENABLED=0` (if Ebiten allows) or verify build tag isolation | Terminal-only build succeeds without Ebiten's OpenGL dependencies |

### Build Tag Strategy

To avoid forcing Ebiten's OpenGL/CGo dependencies on terminal-only users:

```go
// pkg/gfx/view.go
//go:build ebiten

package gfx
```

```go
// pkg/gfx/stub.go
//go:build !ebiten

package gfx

// NewEbitenView returns an error when built without ebiten tag
func NewEbitenView(opts dgclient.ViewOptions) (dgclient.View, error) {
    return nil, fmt.Errorf("ebiten view not available: rebuild with -tags ebiten")
}
```

This allows `go build ./cmd/dgconnect` to work without OpenGL, and `go build -tags ebiten ./cmd/dgconnect` to include graphical support.

---

## Appendix: Dependency Assessment

### Ebiten v2

- **Module**: `github.com/hajimehoshi/ebiten/v2`
- **Latest stable**: v2.8.x (as of early 2026)
- **License**: Apache 2.0 (compatible with project)
- **Platform support**: Linux, macOS, Windows, WebAssembly
- **CGo requirement**: Required on Linux (OpenGL), optional on macOS/Windows
- **Go version compatibility**: Requires Go 1.22+ (project uses 1.23.2 ✓)

### Additional Dependencies

| Dependency | Purpose | Notes |
|-----------|---------|-------|
| `golang.org/x/image/font` | Fallback text rendering | Already an indirect dep via ebiten |
| `golang.org/x/image/font/opentype` | TTF/OTF font loading | For custom fallback fonts |

No other new dependencies required. The mapping file parser uses `gopkg.in/yaml.v3` which is already in `go.mod`.
