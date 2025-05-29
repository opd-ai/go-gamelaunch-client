package tui

import (
	"testing"
)

func TestNewTerminalEmulator(t *testing.T) {
	width, height := 80, 24
	te := NewTerminalEmulator(width, height)

	if te.width != width {
		t.Errorf("Expected width %d, got %d", width, te.width)
	}

	if te.height != height {
		t.Errorf("Expected height %d, got %d", height, te.height)
	}

	if len(te.screen) != height {
		t.Errorf("Expected screen height %d, got %d", height, len(te.screen))
	}

	if len(te.screen[0]) != width {
		t.Errorf("Expected screen width %d, got %d", width, len(te.screen[0]))
	}

	if te.cursorX != 0 || te.cursorY != 0 {
		t.Errorf("Expected cursor at (0,0), got (%d,%d)", te.cursorX, te.cursorY)
	}
}

func TestProcessDataSimpleText(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	text := "Hello World"
	te.ProcessData([]byte(text))

	screen := te.GetScreen()

	// Check that characters were placed correctly
	for i, ch := range text {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}

	// Check cursor position
	cursorX, cursorY := te.GetCursor()
	if cursorX != len(text) || cursorY != 0 {
		t.Errorf("Expected cursor at (%d,0), got (%d,%d)", len(text), cursorX, cursorY)
	}
}

func TestProcessDataNewline(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	te.ProcessData([]byte("Line1\nLine2"))

	screen := te.GetScreen()

	// Check first line
	expectedLine1 := "Line1"
	for i, ch := range expectedLine1 {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Line 1: Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}

	// Check second line
	expectedLine2 := "Line2"
	for i, ch := range expectedLine2 {
		if screen[1][i].Char != rune(ch) {
			t.Errorf("Line 2: Expected char '%c' at position %d, got '%c'", ch, i, screen[1][i].Char)
		}
	}

	// Check cursor position
	cursorX, cursorY := te.GetCursor()
	if cursorX != 5 || cursorY != 1 {
		t.Errorf("Expected cursor at (5,1), got (%d,%d)", cursorX, cursorY)
	}
}

func TestProcessDataCarriageReturn(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	te.ProcessData([]byte("Hello\rWorld"))

	screen := te.GetScreen()

	// After carriage return, "World" should overwrite "Hello"
	expected := "World"
	for i, ch := range expected {
		if screen[0][i].Char != rune(ch) {
			t.Errorf("Expected char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}
}

func TestProcessDataANSIEscape(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	// Clear screen and move cursor to (5,5)
	te.ProcessData([]byte("\x1b[2J\x1b[6;6H"))

	cursorX, cursorY := te.GetCursor()
	if cursorX != 5 || cursorY != 5 {
		t.Errorf("Expected cursor at (5,5), got (%d,%d)", cursorX, cursorY)
	}

	// Check that screen was cleared
	screen := te.GetScreen()
	for y := 0; y < te.height; y++ {
		for x := 0; x < te.width; x++ {
			if screen[y][x].Char != ' ' {
				t.Errorf("Expected space at (%d,%d), got '%c'", x, y, screen[y][x].Char)
			}
		}
	}
}

func TestResize(t *testing.T) {
	te := NewTerminalEmulator(80, 24)

	// Add some content
	te.ProcessData([]byte("Test content"))

	// Resize to smaller
	newWidth, newHeight := 40, 12
	te.Resize(newWidth, newHeight)

	if te.width != newWidth {
		t.Errorf("Expected width %d after resize, got %d", newWidth, te.width)
	}

	if te.height != newHeight {
		t.Errorf("Expected height %d after resize, got %d", newHeight, te.height)
	}

	screen := te.GetScreen()
	if len(screen) != newHeight {
		t.Errorf("Expected screen height %d after resize, got %d", newHeight, len(screen))
	}

	if len(screen[0]) != newWidth {
		t.Errorf("Expected screen width %d after resize, got %d", newWidth, len(screen[0]))
	}

	// Check that content was preserved (first 12 characters)
	expected := "Test content"
	for i, ch := range expected {
		if i < newWidth && screen[0][i].Char != rune(ch) {
			t.Errorf("Expected preserved char '%c' at position %d, got '%c'", ch, i, screen[0][i].Char)
		}
	}
}
