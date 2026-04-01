# AUDIT — 2026-04-01

## Project Goals

**go-gamelaunch-client** claims to be:
> A Go client library and command-line application for connecting to dgamelaunch-style SSH servers to play terminal-based roguelike games remotely.

### Stated Features (from README.md)
1. **SSH Connection Management**: Password, key, and agent-based authentication
2. **Terminal Emulation**: Full PTY support with dynamic resize handling
3. **Modular Architecture**: Pluggable view interface for custom GUI implementations
4. **Robust Error Handling**: Automatic reconnection and graceful error recovery
5. **Configuration Support**: YAML-based configuration for servers and preferences
6. **Cross-Platform**: Works on Linux, macOS, and Windows

### Target Audience
- Developers wanting to integrate gamelaunch connectivity into Go applications
- End-users needing a CLI tool for accessing remote roguelike gaming servers
- Roguelike gaming community accessing dgamelaunch servers (NetHack, DCSS, etc.)

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| SSH Connection Management | ✅ Achieved | `pkg/dgclient/auth.go:23-125` implements `PasswordAuth`, `KeyAuth`, `AgentAuth`, `InteractiveAuth` |
| Password Authentication | ✅ Achieved | `pkg/dgclient/auth.go:23-39` — `PasswordAuth` struct with `GetSSHAuthMethod()` |
| Key Authentication | ✅ Achieved | `pkg/dgclient/auth.go:41-77` — `KeyAuth` with passphrase support |
| Agent Authentication | ✅ Achieved | `pkg/dgclient/auth.go:79-107` — `AgentAuth` using `SSH_AUTH_SOCK` |
| Terminal Emulation | ⚠️ Partial | `pkg/tui/emulator.go` — ANSI parsing, CSI commands, 8-color support; missing 256-color (SGR 38/48) |
| Full PTY Support | ✅ Achieved | `pkg/dgclient/session.go:69-95` — PTY request with terminal modes |
| Dynamic Terminal Resizing | ✅ Achieved | `pkg/tui/tui.go:139-151`, `emulator.go:448-476` — resize handling with content preservation |
| Modular View Interface | ✅ Achieved | `pkg/dgclient/view.go:34-57` — 7-method `View` interface, clean abstraction |
| Automatic Reconnection | ✅ Achieved | `pkg/dgclient/run.go:241-294` — exponential backoff, configurable attempts |
| Graceful Error Recovery | ⚠️ Partial | Custom errors in `errors.go`; missing session state persistence across reconnects |
| YAML Configuration | ✅ Achieved | `cmd/dgconnect/config.go` — full struct definitions with validation |
| Cross-Platform Support | ⚠️ Partial | tcell abstracts terminals; no Windows-specific CI validation |
| Host Key Verification | ✅ Achieved | `pkg/dgclient/auth.go:127-166` — `KnownHostsCallback`; CLI prompts for unknown hosts |
| CLI Interface | ✅ Achieved | `cmd/dgconnect/main.go` — Cobra-based with `version`, `init` commands |
| Library Usage | ✅ Achieved | `pkg/dgclient/` — importable, matches README examples |

**Overall: 12/15 goals fully achieved, 3 partially achieved**

---

## Findings

### CRITICAL

*No critical findings identified.*

### HIGH

- [ ] **Low Test Coverage (10-21%)** — `pkg/dgclient/` 10.5%, `pkg/tui/` 21.3%, `cmd/dgconnect/` 13.3% — Production library with insufficient test coverage risks regressions and makes refactoring dangerous. — **Remediation:** Add integration tests for `Client.Connect()` and `Client.Run()` using mock SSH servers (`golang.org/x/crypto/ssh/testdata`). Add unit tests for ANSI sequence edge cases in `pkg/tui/emulator_test.go`. Target: ≥60% coverage per package. **Validation:** `go test -cover ./... | grep -v "coverage:"` shows ≥60% for all packages.

- [ ] **High Complexity: `runSession` (36.6)** — `pkg/dgclient/run.go:107-210` — 102 lines with 3 concurrent goroutines and complex error handling. Difficult to test, maintain, and reason about. — **Remediation:** Extract goroutine handlers into named functions: `handleStdoutLoop()`, `handleStdinLoop()`, `handleResizeLoop()`. Each function should be <30 lines with complexity <10. **Validation:** `go-stats-generator analyze . --format json | jq '.functions[] | select(.name=="runSession") | .complexity.overall'` returns <15.

- [ ] **High Complexity: `Run` (24.1)** — `pkg/dgclient/run.go:16-104` — 87 lines with main loop and reconnection logic interleaved. — **Remediation:** Extract reconnection decision logic into a separate `shouldAttemptReconnect()` method. Simplify main loop to single responsibility. **Validation:** Complexity reduced to <15.

- [ ] **Code Duplication: Connect/ConnectWithConn (24 lines)** — `pkg/dgclient/run.go:298-321` and `run.go:350-373` — SSH handshake setup duplicated between two methods, risking drift. — **Remediation:** Extract shared logic into `performSSHHandshake(conn net.Conn, config *ssh.ClientConfig) (*ssh.Client, error)`. Call from both `Connect()` and `ConnectWithConn()`. **Validation:** `go-stats-generator analyze . --format json | jq '.duplication.clone_pairs[] | select(.lines >= 20)'` returns empty.

### MEDIUM

- [ ] **Incomplete Terminal Emulation: 256-color Support Missing** — `pkg/tui/emulator.go:281-310` — Only handles basic 8-color SGR codes (30-37, 40-47). Extended color (SGR 38/48 with 5;N or 2;R;G;B) not implemented. Many modern roguelikes use 256-color mode. — **Remediation:** Implement 256-color and true-color parsing in `processGraphicRendition()`. Add test cases for `\x1b[38;5;196m` (256-color red) and `\x1b[38;2;255;0;0m` (true-color red). **Validation:** `go test ./pkg/tui/... -run TestExtendedColors -v` passes.

- [ ] **High Complexity: `getHostKeyCallback` (18.9)** — `cmd/dgconnect/commands.go:206-275` — 68 lines handling host key verification with user prompts and file I/O. — **Remediation:** Split into `verifyKnownHost()`, `promptForUnknownHost()`, and `handleHostKeyMismatch()`. **Validation:** Each extracted function <25 lines, complexity <10.

- [ ] **High Complexity: `getAuthMethod` (17.6)** — `cmd/dgconnect/commands.go:145-204` — 58 lines with multiple fallback paths for authentication. — **Remediation:** Use a chain-of-responsibility pattern: create `[]AuthProvider` and iterate until one succeeds. **Validation:** `getAuthMethod()` complexity <12.

- [ ] **High Complexity: `ValidateConfig` (15.3)** — `cmd/dgconnect/config.go:124-158` — 33 lines validating nested configuration structure. — **Remediation:** Extract `validateServer()` for individual server validation. Use early returns consistently. **Validation:** Complexity <10.

- [ ] **Missing Package Documentation** — `pkg/dgclient/`, `pkg/tui/`, `cmd/dgconnect/` — Package doc coverage: 0% per go-stats-generator. No `doc.go` files with package-level overview. — **Remediation:** Create `doc.go` in each package with package overview, key types, and usage examples. **Validation:** `go doc ./pkg/dgclient` shows comprehensive package description.

- [ ] **Method Documentation Gap (33.9%)** — 22 exported methods without GoDoc comments — Missing docs on `Session` interface methods, `View` interface implementation details. — **Remediation:** Add GoDoc comments to all exported methods in `session.go`, `view.go`, `errors.go`. Focus on: `Wait()` (session.go:198), `CreateView()` (view.go:67), `Unwrap()` (errors.go:54), `GetSSHAuthMethod()` (auth.go:33). **Validation:** `go-stats-generator analyze . --format json | jq '.documentation.method_coverage'` returns >70%.

- [ ] **Low Cohesion: `commands.go` (0.07)** — `cmd/dgconnect/commands.go` — File handles connection logic, authentication, host key verification, and path expansion. Multiple responsibilities. — **Remediation:** Split into `connect.go` (connection orchestration), `hostkey.go` (host key verification), `authprovider.go` (authentication methods). **Validation:** Each file has cohesion >0.5.

### LOW

- [ ] **Code Duplication: Pipe Methods (14 lines × 3)** — `pkg/dgclient/session.go:116-163` — `StdinPipe()`, `StdoutPipe()`, `StderrPipe()` have nearly identical structure. — **Remediation:** Consider a generic pipe getter or accept the duplication as acceptable for clarity. Low priority due to small size. **Validation:** N/A — acceptable if team prefers explicit methods.

- [ ] **File Naming: `tui.go` stutters** — `pkg/tui/tui.go` — go-stats-generator flags "tui" in package "tui" as stuttering. — **Remediation:** Rename to `view.go` or `terminal.go` if desired; low impact. **Validation:** Optional.

- [ ] **Hardcoded Fallback Games** — `pkg/dgclient/client.go:243-258` — Returns hardcoded NetHack/DCSS when parsing fails. May not match actual server offerings. — **Remediation:** Return empty list or configurable defaults instead of hardcoded values. Document behavior in GoDoc. **Validation:** Review confirms intentional fallback behavior.

- [ ] **Dependency Version: golang.org/x/crypto v0.38.0** — `go.mod:13` — Version is current but CVE-2024-45337 (SSH auth bypass) affects versions <0.31.0; project at v0.38.0 is safe. CVE-2025-58181 (GSSAPI DoS) affects <0.45.0. — **Remediation:** Upgrade to v0.45.0+ to address CVE-2025-58181 if GSSAPI auth is used. **Validation:** `go list -m golang.org/x/crypto` shows ≥v0.45.0.

---

## Metrics Snapshot

| Metric | Value | Assessment |
|--------|-------|------------|
| **Lines of Code** | 1,611 | Small, focused codebase |
| **Files** | 12 | Well-organized |
| **Packages** | 3 | Clean separation of concerns |
| **Functions** | 33 | |
| **Methods** | 89 | |
| **Structs** | 29 | |
| **Interfaces** | 5 | |
| **Average Function Length** | 16.2 lines | Acceptable |
| **Average Complexity** | 4.9 | Good baseline |
| **High Complexity (>10)** | 7 functions | Risk areas identified |
| **Test Coverage** | 10.5-21.3% | ⚠️ Below 60% threshold |
| **Documentation Coverage** | 53.2% overall | Package: 0%, Function: 89.5%, Method: 33.9% |
| **Duplication** | 1.75% (52 lines) | Acceptable |
| **Circular Dependencies** | 0 | Clean architecture |

### Top Complex Functions

| Rank | Function | File | Lines | Complexity |
|------|----------|------|-------|------------|
| 1 | `runSession` | `pkg/dgclient/run.go` | 102 | 36.6 |
| 2 | `Run` | `pkg/dgclient/run.go` | 87 | 24.1 |
| 3 | `runConnect` | `cmd/dgconnect/commands.go` | 103 | 20.5 |
| 4 | `getHostKeyCallback` | `cmd/dgconnect/commands.go` | 68 | 18.9 |
| 5 | `getAuthMethod` | `cmd/dgconnect/commands.go` | 58 | 17.6 |
| 6 | `handleReconnection` | `pkg/dgclient/run.go` | 51 | 15.8 |
| 7 | `ValidateConfig` | `cmd/dgconnect/config.go` | 33 | 15.3 |

---

## Validation Commands

```bash
# Run all tests with race detection and coverage
go test -race -cover ./...

# Check for regressions
go vet ./...

# Verify complexity reduction (after remediation)
go-stats-generator analyze . --format json | jq '[.functions[] | select(.complexity.overall > 15)] | length'

# Check documentation coverage
go-stats-generator analyze . --format json | jq '.documentation'

# Verify dependency versions
go list -m all | grep crypto
```

---

*Generated by go-stats-generator v1.0.0 analysis and manual code review.*
