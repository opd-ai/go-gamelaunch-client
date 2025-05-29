package webui
package webui

import (
    "fmt"
    "image"
    _ "image/png" // Import for PNG support
    "os"

    "gopkg.in/yaml.v3"
)

// TilesetConfig represents a tileset configuration
type TilesetConfig struct {
    Name         string         `yaml:"name"`
    Version      string         `yaml:"version"`
    TileWidth    int            `yaml:"tile_width"`
    TileHeight   int            `yaml:"tile_height"`
    SourceImage  string         `yaml:"source_image"`
    Mappings     []TileMapping  `yaml:"mappings"`
    SpecialTiles []SpecialTile  `yaml:"special_tiles"`
    
    // Runtime data
    mappingIndex map[rune]*TileMapping
    imageData    image.Image
}

// TileMapping maps characters to tile coordinates
type TileMapping struct {
    Char     string `yaml:"char"`
    X        int    `yaml:"x"`
    Y        int    `yaml:"y"`
    FgColor  string `yaml:"fg_color,omitempty"`
    BgColor  string `yaml:"bg_color,omitempty"`
    
    // Runtime data
    charRune rune
}

// SpecialTile represents multi-tile entities
type SpecialTile struct {
    ID    string      `yaml:"id"`
    Tiles []TileRef   `yaml:"tiles"`
}

// TileRef references a specific tile
type TileRef struct {
    X int `yaml:"x"`
    Y int `yaml:"y"`
}

// LoadTilesetConfig loads a tileset from a YAML file
func LoadTilesetConfig(path string) (*TilesetConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read tileset file: %w", err)
    }
    
    var config struct {
        Tileset TilesetConfig `yaml:"tileset"`
    }
    
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse tileset YAML: %w", err)
    }
    
    tileset := &config.Tileset
    if err := tileset.validate(); err != nil {
        return nil, fmt.Errorf("invalid tileset configuration: %w", err)
    }
    
    if err := tileset.buildIndex(); err != nil {
        return nil, fmt.Errorf("failed to build tileset index: %w", err)
    }
    
    if err := tileset.loadImage(); err != nil {
        return nil, fmt.Errorf("failed to load tileset image: %w", err)
    }
    
    return tileset, nil
}

// validate checks if the tileset configuration is valid
func (tc *TilesetConfig) validate() error {
    if tc.Name == "" {
        return fmt.Errorf("tileset name is required")
    }
    
    if tc.TileWidth <= 0 || tc.TileHeight <= 0 {
        return fmt.Errorf("tile dimensions must be positive")
    }
    
    if tc.SourceImage == "" {
        return fmt.Errorf("source image is required")
    }
    
    // Validate mappings
    charSet := make(map[string]bool)
    for i, mapping := range tc.Mappings {
        if mapping.Char == "" {
            return fmt.Errorf("mapping %d: character is required", i)
        }
        
        if charSet[mapping.Char] {
            return fmt.Errorf("mapping %d: duplicate character '%s'", i, mapping.Char)
        }
        charSet[mapping.Char] = true
        
        if mapping.X < 0 || mapping.Y < 0 {
            return fmt.Errorf("mapping %d: tile coordinates must be non-negative", i)
        }
    }
    
    return nil
}

// buildIndex creates the character-to-mapping lookup table
func (tc *TilesetConfig) buildIndex() error {
    tc.mappingIndex = make(map[rune]*TileMapping)
    
    for i := range tc.Mappings {
        mapping := &tc.Mappings[i]
        
        // Convert string to rune
        runes := []rune(mapping.Char)
        if len(runes) != 1 {
            return fmt.Errorf("character '%s' must be a single rune", mapping.Char)
        }
        
        mapping.charRune = runes[0]
        tc.mappingIndex[mapping.charRune] = mapping
    }
    
    return nil
}

// loadImage loads the tileset source image
func (tc *TilesetConfig) loadImage() error {
    file, err := os.Open(tc.SourceImage)
    if err != nil {
        return fmt.Errorf("failed to open image file: %w", err)
    }
    defer file.Close()
    
    img, _, err := image.Decode(file)
    if err != nil {
        return fmt.Errorf("failed to decode image: %w", err)
    }
    
    tc.imageData = img
    
    // Validate tile coordinates against image dimensions
    bounds := img.Bounds()
    maxTileX := bounds.Dx() / tc.TileWidth
    maxTileY := bounds.Dy() / tc.TileHeight
    
    for _, mapping := range tc.Mappings {
        if mapping.X >= maxTileX || mapping.Y >= maxTileY {
            return fmt.Errorf("tile coordinates (%d, %d) for character '%s' exceed image bounds", 
                mapping.X, mapping.Y, mapping.Char)
        }
    }
    
    return nil
}

// GetMapping returns the tile mapping for a character
func (tc *TilesetConfig) GetMapping(char rune) *TileMapping {
    return tc.mappingIndex[char]
}

// GetImageData returns the loaded image data
func (tc *TilesetConfig) GetImageData() image.Image {
    return tc.imageData
}

// GetTileCount returns the number of tiles in the tileset
func (tc *TilesetConfig) GetTileCount() (int, int) {
    if tc.imageData == nil {
        return 0, 0
    }
    
    bounds := tc.imageData.Bounds()
    tilesX := bounds.Dx() / tc.TileWidth
    tilesY := bounds.Dy() / tc.TileHeight
    
    return tilesX, tilesY
}

// ToJSON returns a JSON representation for client-side use
func (tc *TilesetConfig) ToJSON() map[string]interface{} {
    mappings := make([]map[string]interface{}, len(tc.Mappings))
    for i, mapping := range tc.Mappings {
        mappings[i] = map[string]interface{}{
            "char":     mapping.Char,
            "x":        mapping.X,
            "y":        mapping.Y,
            "fg_color": mapping.FgColor,
            "bg_color": mapping.BgColor,
        }
    }
    
    tilesX, tilesY := tc.GetTileCount()
    
    return map[string]interface{}{
        "name":         tc.Name,
        "version":      tc.Version,
        "tile_width":   tc.TileWidth,
        "tile_height":  tc.TileHeight,
        "tiles_x":      tilesX,
        "tiles_y":      tilesY,
        "mappings":     mappings,
        "special_tiles": tc.SpecialTiles,
    }
}

// DefaultTilesetConfig returns a basic ASCII tileset configuration
func DefaultTilesetConfig() *TilesetConfig {
    config := &TilesetConfig{
        Name:       "ASCII Default",
        Version:    "1.0.0",
        TileWidth:  8,
        TileHeight: 16,
        Mappings: []TileMapping{
            {Char: "@", X: 0, Y: 0, FgColor: "#FFFFFF"},
            {Char: ".", X: 1, Y: 0, FgColor: "#888888"},
            {Char: "#", X: 2, Y: 0, FgColor: "#AAAAAA"},
            {Char: "+", X: 3, Y: 0, FgColor: "#8B4513"},
            {Char: "d", X: 0, Y: 1, FgColor: "#FF0000"},
            {Char: "k", X: 1, Y: 1, FgColor: "#00FF00"},
            {Char: "D", X: 2, Y: 1, FgColor: "#FF4500"},
        },
    }
    
    config.buildIndex()
    return config
}