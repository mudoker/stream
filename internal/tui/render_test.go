package tui

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"

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

func TestComputeTaskMetricsInfo(t *testing.T) {
	m := Model{Theme: NewTheme()}

	// Case 1: Floating task with 1 Story Point (planned: 45m).
	// No focus or break logged.
	task1 := model.Task{
		SchedulingType: model.Floating,
		StoryPoints:    1,
	}
	info1 := m.computeTaskMetricsInfo(task1)
	if info1.PlannedDur != 45*time.Minute {
		t.Errorf("Expected 45m planned duration, got %v", info1.PlannedDur)
	}
	if info1.QualityScore != 100 {
		t.Errorf("Expected 100 quality score, got %d", info1.QualityScore)
	}

	// Case 2: 1 Interruption, 10 minutes focus, 5 minutes break
	// 5m break / 10m focus = 50% ratio. Excess break ratio = 30%.
	// Interruption penalty = 15. Break penalty = 30. Total penalty = 45. Quality = 55.
	task2 := model.Task{
		SchedulingType: model.Floating,
		StoryPoints:    1,
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 600,
			ElapsedBreakSeconds: 300,
			InterruptionCount:   1,
		},
	}
	info2 := m.computeTaskMetricsInfo(task2)
	if info2.QualityScore != 55 {
		t.Errorf("Expected 55 quality score, got %d", info2.QualityScore)
	}
}

