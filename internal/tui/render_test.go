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



func TestPartitionHeightsDistributesSmallSpace(t *testing.T) {
	availH := 28 - 8
	rowHeights := partitionHeights(availH, 3)
	sum := rowHeights[0] + rowHeights[1] + rowHeights[2]
	if sum != availH {
		t.Fatalf("Expected partition sum %d, got %d", availH, sum)
	}
	if rowHeights[0] == 15 && rowHeights[1] == 15 && rowHeights[2] == 15 {
		t.Fatalf("Expected small available space to use distributed heights, not fixed 15s")
	}
}

func TestModalOverlayBorderAlignment(t *testing.T) {
	// Prepare a model with fixed terminal size
	// Use a larger virtual terminal to ensure modals fit during tests.
	m := Model{Theme: NewTheme(), Width: 120, Height: 60}

	// Test Help modal
	helpStr := m.renderHelpModal()
	helpW := lipgloss.Width(helpStr)
	helpH := lipgloss.Height(helpStr)
	topPad := (m.Height - helpH) / 2
	leftPad := (m.Width - helpW) / 2

	// Blank base canvas
	baseLines := make([]string, m.Height)
	for i := range baseLines {
		baseLines[i] = strings.Repeat(" ", m.Width)
	}
	base := strings.Join(baseLines, "\n")

	res := overlayString(base, helpStr, leftPad, topPad, m.Width)
	lines := strings.Split(res, "\n")

	if topPad < 0 || leftPad < 0 {
		t.Fatalf("invalid pads: top=%d left=%d", topPad, leftPad)
	}

	// Top border corners
	topLine := sliceAnsi(lines[topPad], leftPad, leftPad+helpW)
	topCells := parseLineToCells(topLine)
	if len(topCells) == 0 || topCells[0].Text != "╭" {
		t.Errorf("expected top-left corner '╭', got '%s'", topLine)
	}
	if len(topCells) == 0 || topCells[len(topCells)-1].Text != "╮" {
		t.Errorf("expected top-right corner '╮', got '%s'", topLine)
	}

	// Bottom border corners
	bottomIdx := topPad + helpH - 1
	bottomLine := sliceAnsi(lines[bottomIdx], leftPad, leftPad+helpW)
	bottomCells := parseLineToCells(bottomLine)
	if len(bottomCells) == 0 || bottomCells[0].Text != "╰" {
		t.Errorf("expected bottom-left corner '╰', got '%s'", bottomLine)
	}
	if len(bottomCells) == 0 || bottomCells[len(bottomCells)-1].Text != "╯" {
		t.Errorf("expected bottom-right corner '╯', got '%s'", bottomLine)
	}

	// Middle lines should have vertical borders at edges
	for y := topPad + 1; y < bottomIdx; y++ {
		mid := sliceAnsi(lines[y], leftPad, leftPad+helpW)
		midCells := parseLineToCells(mid)
		if len(midCells) < 2 {
			t.Fatalf("modal middle line too short: '%s'", mid)
		}
		if midCells[0].Text != "│" {
			t.Errorf("expected left vertical border '│' at line %d, got %q", y, mid)
		}
		if midCells[len(midCells)-1].Text != "│" {
			t.Errorf("expected right vertical border '│' at line %d, got %q", y, mid)
		}
	}

	// Test Detail modal (Task Inspector)
	// Provide a sample detail task so renderDetailModal has content
	m.DetailTask = model.Task{
		Title:       "Task Inspector",
		Description: "Test task",
		Priority:    "P1",
		StoryPoints: 3,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{Start: time.Now(), End: time.Now().Add(45 * time.Minute)},
	}
	detStr := m.renderDetailModal()
	detW := lipgloss.Width(detStr)
	detH := lipgloss.Height(detStr)
	topPad = (m.Height - detH) / 2
	leftPad = (m.Width - detW) / 2

	res = overlayString(base, detStr, leftPad, topPad, m.Width)
	lines = strings.Split(res, "\n")

	topLine = sliceAnsi(lines[topPad], leftPad, leftPad+detW)
	detTopCells := parseLineToCells(topLine)
	if len(detTopCells) == 0 || detTopCells[0].Text != "╭" || detTopCells[len(detTopCells)-1].Text != "╮" {
		t.Errorf("detail modal top corners incorrect: '%s'", topLine)
	}
	bottomIdx = topPad + detH - 1
	bottomLine = sliceAnsi(lines[bottomIdx], leftPad, leftPad+detW)
	detBottomCells := parseLineToCells(bottomLine)
	if len(detBottomCells) == 0 || detBottomCells[0].Text != "╰" || detBottomCells[len(detBottomCells)-1].Text != "╯" {
		t.Errorf("detail modal bottom corners incorrect: '%s'", bottomLine)
	}
}

