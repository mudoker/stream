package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSliceAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Hello World")
	// "Hello World" is 11 chars visual width.
	// Let's slice it to get "ello W" (visual indices 1 to 7)
	sliced := sliceAnsi(red, 1, 7)
	visualLen := lipgloss.Width(sliced)
	if visualLen != 6 {
		t.Errorf("Expected visual length 6, got %d for '%s'", visualLen, sliced)
	}
	if !strings.Contains(sliced, "ello W") {
		t.Errorf("Expected sliced string to contain 'ello W', got '%s'", sliced)
	}

	// Slicing out of bounds
	slicedOut := sliceAnsi(red, 20, 30)
	if lipgloss.Width(slicedOut) != 0 {
		t.Errorf("Expected visual length 0, got %d", lipgloss.Width(slicedOut))
	}
}

func TestOverlayString(t *testing.T) {
	base := "Line 1: Hello World\nLine 2: Goodbye World"
	overlay := "OVERLAY"
	// Overlay at x=8, y=0
	res := overlayString(base, overlay, 8, 0, 30)
	expectedLine0 := "Line 1: OVERLAYorld"
	lines := strings.Split(res, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	visualLine0 := sliceAnsi(lines[0], 0, 30)
	if !strings.HasPrefix(visualLine0, expectedLine0) {
		t.Errorf("Expected line 0 to start with '%s', got '%s'", expectedLine0, visualLine0)
	}

	// Check line 1 is unmodified
	if lines[1] != "Line 2: Goodbye World" {
		t.Errorf("Expected line 1 to be unmodified, got '%s'", lines[1])
	}
}
