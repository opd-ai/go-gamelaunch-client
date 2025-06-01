# dgclient

A Go client library for connecting to dgamelaunch-style SSH servers with modular view interfaces and comprehensive authentication support.

---

## Installation

```bash
go get github.com/opd-ai/go-gamelaunch-client/pkg/dgclient
```

---

## Features

### SSH Connection Management
- **Multiple Authentication Methods** - Password, SSH key (RSA, Ed25519, ECDSA), and SSH agent authentication
- **Automatic Reconnection** - Exponential backoff with configurable retry limits and delay intervals
- **Connection Health Monitoring** - Real-time status tracking with graceful error recovery
- **Session Persistence** - Maintains game state across temporary network interruptions
- **Host Key Verification** - Secure host key validation with known_hosts support

### Terminal Emulation and PTY Handling
- **Full PTY Support** - Complete pseudo-terminal implementation with proper signal handling
- **Dynamic Terminal Resizing** - Automatic resize propagation to remote sessions
- **ANSI Escape Sequence Processing** - Complete support for color codes, cursor movement, and screen control
- **Screen Buffer Management** - Efficient memory handling with configurable buffer sizes
- **Terminal Type Detection** - Automatic TERM environment variable configuration

### Modular View Architecture
- **Pluggable View Interface** - Clean abstraction for custom display implementations
- **Multiple View Backends** - Terminal UI (tcell), web interface, and custom implementations
- **View State Synchronization** - Efficient state updates with change detection
- **Concurrent View Support** - Multiple simultaneous view connections to single client
- **View Configuration** - Flexible options for terminal dimensions and rendering preferences

### Authentication System
- **AuthMethod Interface** - Extensible authentication framework for custom methods
- **Key-Based Authentication** - Support for encrypted and unencrypted private keys
- **SSH Agent Integration** - Automatic agent discovery with fallback authentication
- **Password Authentication** - Secure password handling with memory clearing
- **Multi-Factor Support** - Keyboard-interactive authentication for complex login flows

### Session Management
- **Game Server Integration** - Seamless connection to dgamelaunch-style servers
- **Game Selection Automation** - Programmatic game launching and menu navigation
- **Session State Tracking** - Connection status monitoring with detailed error reporting
- **Resource Management** - Proper cleanup of SSH connections and terminal resources
- **Concurrent Session Handling** - Multiple simultaneous connections with independent state

### Configuration and Flexibility
- **Comprehensive Client Options** - Configurable timeouts, buffer sizes, and connection parameters
- **Environment Variable Support** - Standard SSH environment variable handling
- **Debug and Logging** - Detailed logging with configurable verbosity levels
- **Error Context** - Rich error information with operation context and recovery suggestions
- **Performance Tuning** - Adjustable polling intervals and buffer management

### Network Resilience
- **Connection Recovery** - Automatic detection and recovery from network failures
- **Timeout Management** - Configurable timeouts for different operation types
- **Keepalive Support** - SSH keepalive packets to maintain long-running connections
- **Bandwidth Optimization** - Efficient data transfer with compression support
- **IPv4/IPv6 Support** - Dual-stack networking with preference configuration

### Developer Experience
- **Clean API Design** - Intuitive interfaces with comprehensive documentation
- **Example Implementations** - Reference implementations for common use cases
- **Testing Infrastructure** - Mock implementations and test utilities for development
- **Error Handling Patterns** - Consistent error wrapping with actionable messages
- **Thread Safety** - Safe concurrent usage with proper synchronization

### Integration Capabilities
- **Library Integration** - Easy embedding in larger applications with minimal dependencies
- **CLI Tool Foundation** - Solid base for command-line tool development
- **GUI Framework Support** - Compatible with various Go GUI frameworks through view interface
- **Service Integration** - Suitable for daemon and service implementations
- **Monitoring Integration** - Metrics and health check endpoints for operational monitoring

---

## Architecture

The dgclient package implements a layered architecture designed for modularity and extensibility:

**Connection Layer**: Manages SSH connections with authentication, host key verification, and connection lifecycle. Handles network failures with automatic reconnection and maintains session state across interruptions.

**Session Layer**: Provides PTY management with terminal emulation, handling resize events and maintaining screen buffer state. Processes ANSI escape sequences and manages terminal input/output streams.

**View Layer**: Defines the View interface for pluggable display backends, enabling integration with terminal UIs, web interfaces, or custom rendering systems. Supports multiple concurrent views with independent state management.

**Authentication Layer**: Implements the AuthMethod interface with support for password, key-based, and SSH agent authentication. Provides secure credential handling with proper memory management and cleanup.

The package serves as the foundation for both CLI tools and GUI applications, providing reliable SSH connectivity to dgamelaunch servers while maintaining compatibility with traditional terminal-based roguelike games. The modular design enables custom implementations while ensuring robust connection management and comprehensive error handling.