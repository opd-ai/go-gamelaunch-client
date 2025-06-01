# tui

A terminal user interface package providing ANSI terminal emulation and display management for dgamelaunch-style roguelike games.

---

## Installation

```bash
go get github.com/opd-ai/go-gamelaunch-client/pkg/tui
```

---

## Features

### Terminal Emulation
- **ANSI Escape Sequence Processing** - Complete support for cursor movement, color codes, and screen control sequences
- **Screen Buffer Management** - Efficient character-based display buffer with proper memory management
- **Terminal State Tracking** - Maintains cursor position, attributes, and screen dimensions
- **Character Attribute Handling** - Bold, inverse, blinking, and color attribute processing
- **Terminal Type Support** - Compatible with xterm, vt100, and other standard terminal types

### Display Management
- **tcell Integration** - Built on gdamore/tcell/v2 for cross-platform terminal handling
- **Real-Time Rendering** - Efficient screen updates with minimal flicker and proper refresh management
- **Color Support** - Full 256-color palette with RGB true color support where available
- **Unicode Handling** - Complete UTF-8 character support including wide characters and combining marks
- **Screen Resize Handling** - Dynamic terminal size adjustment with proper content reflow

### Input Processing
- **Keyboard Event Handling** - Complete keyboard input capture including special keys and modifiers
- **Mouse Support** - Mouse click and movement event processing for interactive gameplay
- **Input Buffering** - Efficient input event queuing with configurable buffer sizes
- **Key Mapping** - Customizable key bindings for game-specific control schemes
- **Focus Management** - Proper input focus handling for terminal applications

### View Interface Implementation
- **dgclient.View Compatibility** - Implements the View interface for seamless integration with SSH clients
- **Terminal Dimensions** - Automatic terminal size detection with resize event propagation
- **Data Rendering** - Efficient conversion of game data to terminal display format
- **State Synchronization** - Maintains consistency between game state and terminal display
- **Error Handling** - Graceful degradation with informative error messages

### Terminal Emulator Features
- **Cursor Management** - Full cursor control including visibility, position, and style
- **Scroll Region Support** - Proper handling of scrollable areas and viewport management
- **Character Sets** - Support for alternate character sets and special symbols
- **Terminal Modes** - Application mode, alternate screen buffer, and other terminal modes
- **Line Discipline** - Proper handling of line endings, wrapping, and character encoding

### Performance Optimizations
- **Incremental Updates** - Only renders changed screen regions for optimal performance
- **Buffer Recycling** - Efficient memory usage with buffer pooling and reuse
- **Event Batching** - Groups related events to reduce processing overhead
- **Lazy Rendering** - Deferred screen updates until refresh is needed
- **Memory Management** - Proper cleanup and resource management for long-running sessions

### Cross-Platform Support
- **Windows Compatibility** - Full Windows terminal support through tcell abstraction
- **Unix/Linux Support** - Native terminal handling on Unix-like systems
- **Terminal Detection** - Automatic capability detection and fallback handling
- **Encoding Support** - Proper character encoding handling across different platforms
- **Signal Handling** - Graceful shutdown and resize signal processing

### Developer Features
- **Clean API Design** - Intuitive interfaces following Go best practices
- **Comprehensive Testing** - Unit tests for terminal emulation and display functions
- **Debug Support** - Built-in debugging capabilities with verbose logging
- **Mock Implementations** - Test utilities for development and unit testing
- **Documentation** - Complete godoc documentation with usage examples

### Integration Capabilities
- **Modular Architecture** - Clean separation between emulation and display concerns
- **Plugin Support** - Extensible design for custom terminal behavior
- **CLI Foundation** - Solid base for command-line application development
- **Library Embedding** - Easy integration into larger Go applications
- **Configuration Support** - Flexible configuration options for terminal behavior

---

## Architecture

The tui package implements a layered architecture designed for robust terminal emulation and display management:

**Emulation Layer**: The TerminalEmulator interface provides ANSI escape sequence processing, screen buffer management, and terminal state tracking. Handles cursor positioning, character attributes, and screen control commands with full compatibility for standard terminal types.

**Display Layer**: Built on tcell for cross-platform terminal handling, providing real-time rendering with efficient screen updates. Manages color support, Unicode handling, and terminal capability detection while maintaining consistent display across different platforms.

**Input Layer**: Comprehensive input event processing with keyboard and mouse support. Handles special key combinations, modifier keys, and provides configurable key mapping for game-specific controls with proper focus management.

**View Layer**: Implements the dgclient.View interface for seamless integration with SSH clients. Manages terminal dimensions, data rendering, and state synchronization between game sessions and terminal display with automatic resize handling.

The package serves as the foundation for terminal-based game clients, providing reliable terminal emulation while maintaining compatibility with traditional roguelike games. The modular design enables custom implementations while ensuring robust display management and comprehensive input handling for optimal gaming experience.