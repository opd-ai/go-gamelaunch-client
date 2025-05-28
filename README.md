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