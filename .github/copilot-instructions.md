# Project Overview

go-gamelaunch-client is a Go client library and command-line application designed for connecting to dgamelaunch-style SSH servers to play terminal-based roguelike games remotely. The project targets both developers who want to integrate gamelaunch connectivity into their applications and end-users who need a reliable CLI tool for accessing remote gaming servers. The library provides a modular architecture with pluggable view interfaces, enabling custom GUI implementations while maintaining robust SSH connection management and terminal emulation capabilities.

The project emphasizes reliability and user experience through automatic reconnection, graceful error recovery, and comprehensive configuration support. It serves the roguelike gaming community by providing a modern, well-maintained client for accessing traditional dgamelaunch servers that host games like NetHack, Dungeon Crawl Stone Soup, and other terminal-based games.

## Technical Stack

- **Primary Language**: Go 1.23.2
- **Frameworks**: 
  - gdamore/tcell/v2 v2.8.1 (terminal UI and emulation)
  - spf13/cobra v1.9.1 (CLI command structure)
  - spf13/viper v1.20.1 (configuration management)
- **SSH/Crypto**: golang.org/x/crypto v0.38.0, golang.org/x/term v0.32.0
- **Testing**: Go built-in testing package with testify for enhanced assertions
- **Build/Deploy**: Standard Go toolchain with Makefile for formatting and documentation
- **Configuration**: YAML-based configuration using gopkg.in/yaml.v3

## Code Assistance Guidelines

1. **SSH Authentication Patterns**: Implement the [`AuthMethod`](pkg/dgclient/auth.go) interface for new authentication methods. Follow the pattern established in [`PasswordAuth`](pkg/dgclient/auth.go), [`KeyAuth`](pkg/dgclient/auth.go), and [`AgentAuth`](pkg/dgclient/auth.go) with proper error handling and resource cleanup.

2. **View Interface Implementation**: When creating custom views, implement all methods of the [`View`](pkg/dgclient/view.go) interface. Use [`DefaultViewOptions()`](pkg/dgclient/view.go) as a starting point and ensure proper terminal size handling and data rendering.

3. **Configuration Structure**: Follow the established pattern in [`cmd/dgconnect/config.go`](cmd/dgconnect/config.go) for configuration management. Use struct tags for YAML marshaling and implement validation methods similar to [`ValidateConfig`](cmd/dgconnect/config.go).

4. **Error Handling**: Use wrapped errors with context (`fmt.Errorf("operation failed: %w", err)`) throughout the codebase. Implement graceful degradation for network issues and provide meaningful error messages for user-facing operations.

5. **Testing Standards**: Write table-driven tests for business logic functions. Include setup and teardown for SSH connections in integration tests. Use mock implementations like [`MockView`](prompt.md) for unit testing client functionality.

6. **Terminal Emulation**: When working with terminal data, respect the [`TerminalEmulator`](pkg/tui/emulator.go) interface and handle resize events properly. Maintain screen state consistency and support ANSI escape sequences correctly.

7. **CLI Command Structure**: Use cobra's command hierarchy and follow the pattern in [`cmd/dgconnect/main.go`](cmd/dgconnect/main.go). Implement proper flag validation and provide helpful usage examples in command descriptions.

## Project Context

- **Domain**: Terminal-based roguelike gaming infrastructure with focus on dgamelaunch server compatibility. Key concepts include PTY (pseudo-terminal) handling, SSH session management, and terminal emulation for game display.

- **Architecture**: Modular client-server architecture with [`dgclient`](pkg/dgclient/) as the core library, [`tui`](pkg/tui/) for terminal interface handling, and [`dgconnect`](cmd/dgconnect/) as the CLI implementation. The [`View`](pkg/dgclient/view.go) interface enables pluggable display backends.

- **Key Directories**: 
  - [`pkg/dgclient/`](pkg/dgclient/) - Core client library with SSH and session management
  - [`pkg/tui/`](pkg/tui/) - Terminal user interface and emulation
  - [`cmd/dgconnect/`](cmd/dgconnect/) - CLI application with configuration management
  - [`pkg/webui/`](pkg/webui/) - Web-based interface components

- **Configuration**: YAML-based configuration in `~/.dgconnect.yaml` with server definitions, authentication methods, and user preferences. Support for multiple authentication types (password, key, agent) and connection parameters.

## Quality Standards

- **Testing Requirements**: Maintain comprehensive test coverage using Go's built-in testing package. Write unit tests for all public interfaces and integration tests for SSH connectivity. Include table-driven tests for configuration validation and terminal emulation functions.

- **Code Review Criteria**: Ensure proper error handling with wrapped errors, resource cleanup (especially SSH connections), and adherence to Go formatting standards. Validate that new features include appropriate documentation and examples.

- **Documentation Standards**: Update README.md for user-facing changes, maintain godoc comments for all public functions, and include usage examples in code comments. Keep configuration schema documentation current with any struct changes.