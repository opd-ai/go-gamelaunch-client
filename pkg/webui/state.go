package webui

import (
	"context"
	"sync"
	"time"
)

// StateManager manages game state versions and change tracking
type StateManager struct {
	mu           sync.RWMutex
	currentState *GameState
	version      uint64
	waiters      map[uint64]chan *StateDiff
	waitersMu    sync.Mutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		waiters: make(map[uint64]chan *StateDiff),
	}
}

// UpdateState updates the current state and notifies waiters
func (sm *StateManager) UpdateState(state *GameState) {
	sm.mu.Lock()

	// Increment version
	sm.version++
	state.Version = sm.version

	// Generate diff if we have a previous state
	var diff *StateDiff
	if sm.currentState != nil {
		diff = sm.generateDiff(sm.currentState, state)
	}

	sm.currentState = state
	sm.mu.Unlock()

	// Notify waiters
	if diff != nil {
		sm.notifyWaiters(diff)
	}
}

// GetCurrentState returns the current state
func (sm *StateManager) GetCurrentState() *GameState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentState == nil {
		return nil
	}

	// Return a copy
	stateCopy := *sm.currentState
	return &stateCopy
}

// GetCurrentVersion returns the current version number
func (sm *StateManager) GetCurrentVersion() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.version
}

// PollChanges waits for changes since the specified version
func (sm *StateManager) PollChanges(clientVersion uint64, timeout time.Duration) (*StateDiff, error) {
	sm.mu.RLock()
	currentVersion := sm.version
	sm.mu.RUnlock()

	// If client is behind, return immediate diff
	if clientVersion < currentVersion {
		return sm.generateDiffFromVersion(clientVersion)
	}

	// Wait for next change
	waiterCh := make(chan *StateDiff, 1)

	sm.waitersMu.Lock()
	sm.waiters[clientVersion] = waiterCh
	sm.waitersMu.Unlock()

	defer func() {
		sm.waitersMu.Lock()
		delete(sm.waiters, clientVersion)
		sm.waitersMu.Unlock()
	}()

	select {
	case diff := <-waiterCh:
		return diff, nil
	case <-time.After(timeout):
		return nil, nil // Timeout
	}
}

// PollChangesWithContext waits for changes with a context
// It is a context-aware version of PollChanges
func (sm *StateManager) PollChangesWithContext(pollCtx context.Context, version uint64) (*StateDiff, error) {
	sm.mu.RLock()
	currentVersion := sm.version
	sm.mu.RUnlock()

	// If client is behind, return immediate diff
	if version < currentVersion {
		return sm.generateDiffFromVersion(version)
	}

	// Wait for next change
	waiterCh := make(chan *StateDiff, 1)

	sm.waitersMu.Lock()
	sm.waiters[version] = waiterCh
	sm.waitersMu.Unlock()

	defer func() {
		sm.waitersMu.Lock()
		delete(sm.waiters, version)
		sm.waitersMu.Unlock()
	}()

	select {
	case diff := <-waiterCh:
		return diff, nil
	case <-pollCtx.Done():
		return nil, pollCtx.Err() // Context cancelled or deadline exceeded
	}
}

// generateDiff creates a diff between two states
func (sm *StateManager) generateDiff(oldState, newState *GameState) *StateDiff {
	diff := &StateDiff{
		Version:   newState.Version,
		CursorX:   newState.CursorX,
		CursorY:   newState.CursorY,
		Timestamp: newState.Timestamp,
		Changes:   make([]CellDiff, 0),
	}

	// Compare buffers
	maxY := newState.Height
	if oldState.Height < maxY {
		maxY = oldState.Height
	}

	for y := 0; y < maxY; y++ {
		maxX := newState.Width
		if oldState.Width < maxX {
			maxX = oldState.Width
		}

		for x := 0; x < maxX; x++ {
			oldCell := oldState.Buffer[y][x]
			newCell := newState.Buffer[y][x]

			if sm.cellsDiffer(oldCell, newCell) {
				diff.Changes = append(diff.Changes, CellDiff{
					X:    x,
					Y:    y,
					Cell: newCell,
				})
			}
		}
	}

	// Handle size changes
	if newState.Height > oldState.Height || newState.Width > oldState.Width {
		// Add new cells
		for y := 0; y < newState.Height; y++ {
			for x := 0; x < newState.Width; x++ {
				if y >= oldState.Height || x >= oldState.Width {
					diff.Changes = append(diff.Changes, CellDiff{
						X:    x,
						Y:    y,
						Cell: newState.Buffer[y][x],
					})
				}
			}
		}
	}

	return diff
}

// generateDiffFromVersion generates diff from a specific version to current
func (sm *StateManager) generateDiffFromVersion(fromVersion uint64) (*StateDiff, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentState == nil {
		return nil, nil
	}

	// For simplicity, return full state as diff if version is old
	// In production, you'd want to store historical states or deltas
	diff := &StateDiff{
		Version:   sm.currentState.Version,
		CursorX:   sm.currentState.CursorX,
		CursorY:   sm.currentState.CursorY,
		Timestamp: sm.currentState.Timestamp,
		Changes:   make([]CellDiff, 0),
	}

	// Add all cells as changes
	for y := 0; y < sm.currentState.Height; y++ {
		for x := 0; x < sm.currentState.Width; x++ {
			diff.Changes = append(diff.Changes, CellDiff{
				X:    x,
				Y:    y,
				Cell: sm.currentState.Buffer[y][x],
			})
		}
	}

	return diff, nil
}

// cellsDiffer compares two cells for differences
func (sm *StateManager) cellsDiffer(a, b Cell) bool {
	return a.Char != b.Char ||
		a.FgColor != b.FgColor ||
		a.BgColor != b.BgColor ||
		a.Bold != b.Bold ||
		a.Inverse != b.Inverse ||
		a.Blink != b.Blink ||
		a.TileX != b.TileX ||
		a.TileY != b.TileY
}

// notifyWaiters sends diff to all waiting clients
func (sm *StateManager) notifyWaiters(diff *StateDiff) {
	sm.waitersMu.Lock()
	defer sm.waitersMu.Unlock()

	for version, waiterCh := range sm.waiters {
		if version < diff.Version {
			select {
			case waiterCh <- diff:
			default:
				// Channel full, skip
			}
		}
	}
}
