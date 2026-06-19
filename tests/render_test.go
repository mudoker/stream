package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"
	"stream/internal/view/modals"
	"stream/internal/view/pages"

	"github.com/charmbracelet/lipgloss"
)

func TestSliceAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Hello World")
	sliced := theme.SliceAnsi(red, 1, 7)
	visualLen := lipgloss.Width(sliced)
	if visualLen != 6 {
		t.Errorf("Expected visual length 6, got %d for '%s'", visualLen, sliced)
	}
	if !strings.Contains(sliced, "ello W") {
		t.Errorf("Expected sliced string to contain 'ello W', got '%s'", sliced)
	}

	slicedOut := theme.SliceAnsi(red, 20, 30)
	if lipgloss.Width(slicedOut) != 0 {
		t.Errorf("Expected visual length 0, got %d", lipgloss.Width(slicedOut))
	}
}

func TestOverlayString(t *testing.T) {
	base := "Line 1: Hello World\nLine 2: Goodbye World"
	overlay := "OVERLAY"
	res := theme.OverlayString(base, overlay, 8, 0, 30)
	expectedLine0 := "Line 1: OVERLAYorld"
	lines := strings.Split(res, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	visualLine0 := theme.SliceAnsi(lines[0], 0, 30)
	if !strings.HasPrefix(visualLine0, expectedLine0) {
		t.Errorf("Expected line 0 to start with '%s', got '%s'", expectedLine0, visualLine0)
	}

	if lines[1] != "Line 2: Goodbye World" {
		t.Errorf("Expected line 1 to be unmodified, got '%s'", lines[1])
	}
}

func TestComputeTaskMetricsInfo(t *testing.T) {
	m := &viewmodel.Model{}
	th := theme.NewTheme()

	task1 := model.Task{
		SchedulingType: model.Floating,
		StoryPoints:    1,
	}
	info1 := modals.ComputeTaskMetricsInfo(m, th, task1)
	if info1.PlannedDur != 45*time.Minute {
		t.Errorf("Expected 45m planned duration, got %v", info1.PlannedDur)
	}
	if info1.QualityScore != 100 {
		t.Errorf("Expected 100 quality score, got %d", info1.QualityScore)
	}

	task2 := model.Task{
		SchedulingType: model.Floating,
		StoryPoints:    1,
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 600,
			ElapsedBreakSeconds: 300,
			InterruptionCount:   1,
		},
	}
	info2 := modals.ComputeTaskMetricsInfo(m, th, task2)
	if info2.QualityScore != 55 {
		t.Errorf("Expected 55 quality score, got %d", info2.QualityScore)
	}
}

func TestPartitionHeightsDistributesSmallSpace(t *testing.T) {
	availH := 20
	rowHeights := viewmodel.PartitionHeights(availH, 3)
	sum := rowHeights[0] + rowHeights[1] + rowHeights[2]
	if sum != availH {
		t.Fatalf("Expected partition sum %d, got %d", availH, sum)
	}
	if rowHeights[0] == 15 && rowHeights[1] == 15 && rowHeights[2] == 15 {
		t.Fatalf("Expected small available space to use distributed heights, not fixed 15s")
	}
}

func TestModalOverlayBorderAlignment(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{Width: 120, Height: 60}

	helpStr := modals.RenderHelpModal(m, th)
	helpW := lipgloss.Width(helpStr)
	helpH := lipgloss.Height(helpStr)
	topPad := (m.Height - helpH) / 2
	leftPad := (m.Width - helpW) / 2

	baseLines := make([]string, m.Height)
	for i := range baseLines {
		baseLines[i] = strings.Repeat(" ", m.Width)
	}
	base := strings.Join(baseLines, "\n")

	res := theme.OverlayString(base, helpStr, leftPad, topPad, m.Width)
	lines := strings.Split(res, "\n")

	if topPad < 0 || leftPad < 0 {
		t.Fatalf("invalid pads: top=%d left=%d", topPad, leftPad)
	}

	topLine := theme.SliceAnsi(lines[topPad], leftPad, leftPad+helpW)
	topCells := theme.ParseLineToCells(topLine)
	if len(topCells) == 0 || topCells[0].Text != "╭" {
		t.Errorf("expected top-left corner '╭', got '%s'", topLine)
	}
	if len(topCells) == 0 || topCells[len(topCells)-1].Text != "╮" {
		t.Errorf("expected top-right corner '╮', got '%s'", topLine)
	}

	bottomIdx := topPad + helpH - 1
	bottomLine := theme.SliceAnsi(lines[bottomIdx], leftPad, leftPad+helpW)
	bottomCells := theme.ParseLineToCells(bottomLine)
	if len(bottomCells) == 0 || bottomCells[0].Text != "╰" {
		t.Errorf("expected bottom-left corner '╰', got '%s'", bottomLine)
	}
	if len(bottomCells) == 0 || bottomCells[len(bottomCells)-1].Text != "╯" {
		t.Errorf("expected bottom-right corner '╯', got '%s'", bottomLine)
	}

	for y := topPad + 1; y < bottomIdx; y++ {
		mid := theme.SliceAnsi(lines[y], leftPad, leftPad+helpW)
		midCells := theme.ParseLineToCells(mid)
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

	m.DetailTask = model.Task{
		Title:       "Task Inspector",
		Description: "Test task",
		Priority:    "P1",
		StoryPoints: 3,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{Start: time.Now(), End: time.Now().Add(45 * time.Minute)},
	}
	detStr := modals.RenderDetailModal(m, th)
	detW := lipgloss.Width(detStr)
	detH := lipgloss.Height(detStr)
	topPad = (m.Height - detH) / 2
	leftPad = (m.Width - detW) / 2

	res = theme.OverlayString(base, detStr, leftPad, topPad, m.Width)
	lines = strings.Split(res, "\n")

	topLine = theme.SliceAnsi(lines[topPad], leftPad, leftPad+detW)
	detTopCells := theme.ParseLineToCells(topLine)
	if len(detTopCells) == 0 || detTopCells[0].Text != "╭" || detTopCells[len(detTopCells)-1].Text != "╮" {
		t.Errorf("detail modal top corners incorrect: '%s'", topLine)
	}
	bottomIdx = topPad + detH - 1
	bottomLine = theme.SliceAnsi(lines[bottomIdx], leftPad, leftPad+detW)
	detBottomCells := theme.ParseLineToCells(bottomLine)
	if len(detBottomCells) == 0 || detBottomCells[0].Text != "╰" || detBottomCells[len(detBottomCells)-1].Text != "╯" {
		t.Errorf("detail modal bottom corners incorrect: '%s'", bottomLine)
	}
}

func TestRenderRestBlock(t *testing.T) {
	th := theme.NewTheme()
	testTime := time.Date(2026, 6, 10, 10, 15, 0, 0, time.UTC) // Format will be 10:15

	// Test height 1 (not completed, not focused)
	h1 := cleanAnsi(pages.RenderRestBlock(th, 20, 1, 15, testTime, false, false))
	if !strings.Contains(h1, "Rest 15m (10:15)") {
		t.Errorf("Expected height-1 rest block to contain 'Rest 15m (10:15)', got: %q", h1)
	}
	if !strings.HasPrefix(h1, "┌") || !strings.HasSuffix(h1, "┐") {
		t.Errorf("Expected height-1 rest block to have top corners '┌'/'┐', got: %q", h1)
	}

	// Test height 2 (not completed, not focused)
	h2 := cleanAnsi(pages.RenderRestBlock(th, 20, 2, 15, testTime, false, false))
	h2Lines := strings.Split(h2, "\n")
	if len(h2Lines) != 2 {
		t.Fatalf("Expected height-2 rest block to have 2 lines, got %d", len(h2Lines))
	}
	if !strings.Contains(h2Lines[0], "Rest 15m (10:15)") {
		t.Errorf("Expected first line of height-2 rest block to contain 'Rest 15m (10:15)', got: %q", h2Lines[0])
	}
	if !strings.HasPrefix(h2Lines[0], "┌") || !strings.HasSuffix(h2Lines[0], "┐") {
		t.Errorf("Expected top line of height-2 rest block to have top corners '┌'/'┐', got: %q", h2Lines[0])
	}
	if !strings.HasPrefix(h2Lines[1], "└") || !strings.HasSuffix(h2Lines[1], "┘") {
		t.Errorf("Expected bottom line of height-2 rest block to have bottom corners '└'/'┘', got: %q", h2Lines[1])
	}

	// Test height 3 (not completed, not focused)
	h3 := cleanAnsi(pages.RenderRestBlock(th, 20, 3, 15, testTime, false, false))
	h3Lines := strings.Split(h3, "\n")
	if len(h3Lines) != 3 {
		t.Fatalf("Expected height-3 rest block to have 3 lines, got %d", len(h3Lines))
	}
	if !strings.HasPrefix(h3Lines[0], "┌") || !strings.HasSuffix(h3Lines[0], "┐") {
		t.Errorf("Expected top line of height-3 rest block to have top corners '┌'/'┐', got: %q", h3Lines[0])
	}
	if !strings.Contains(h3Lines[1], "Rest 15m (10:15)") {
		t.Errorf("Expected center line of height-3 rest block to contain 'Rest 15m (10:15)', got: %q", h3Lines[1])
	}
	if !strings.HasPrefix(h3Lines[1], "┊") || !strings.HasSuffix(h3Lines[1], "┊") {
		t.Errorf("Expected center line of height-3 rest block to have vertical side borders '┊', got: %q", h3Lines[1])
	}
	if !strings.HasPrefix(h3Lines[2], "└") || !strings.HasSuffix(h3Lines[2], "┘") {
		t.Errorf("Expected bottom line of height-3 rest block to have bottom corners '└'/'┘', got: %q", h3Lines[2])
	}

	// Test completed state (not focused)
	hCompleted := cleanAnsi(pages.RenderRestBlock(th, 26, 1, 15, testTime, true, false))
	if !strings.Contains(hCompleted, "Rest 15m (10:15) ✔") {
		t.Errorf("Expected completed rest block to contain 'Rest 15m (10:15) ✔', got: %q", hCompleted)
	}

	// Test focused state (not completed)
	hFocused := cleanAnsi(pages.RenderRestBlock(th, 20, 1, 15, testTime, false, true))
	if !strings.Contains(hFocused, "Rest 15m (10:15)") {
		t.Errorf("Expected focused rest block to contain 'Rest 15m (10:15)', got: %q", hFocused)
	}
}

func cleanAnsi(s string) string {
	var sb strings.Builder
	var inEscape = false
	var runes = []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
