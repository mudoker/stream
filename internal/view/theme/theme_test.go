package theme

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func TestThemeHelpers(t *testing.T) {
	th := NewTheme()

	// 1. PriorityColor
	p0 := th.PriorityColor(model.P0)
	p1 := th.PriorityColor(model.P1)
	p2 := th.PriorityColor(model.P2)
	p3 := th.PriorityColor(model.P3)
	pDefault := th.PriorityColor("")
	if p0 == "" || p1 == "" || p2 == "" || p3 == "" || pDefault == "" {
		t.Errorf("expected non-empty priority colors")
	}

	// 2. SentenceCase
	s1 := SentenceCase("hello world")
	if s1 != "Hello world" {
		t.Errorf("expected Hello world, got %q", s1)
	}
	if SentenceCase("") != "" {
		t.Errorf("expected empty string")
	}

	// 3. SliceAnsi
	boldHello := lipgloss.NewStyle().Bold(true).Render("Hello")
	sliced := SliceAnsi(boldHello, 1, 3)
	if !strings.Contains(sliced, "el") {
		t.Errorf("expected sliced ANSI text to contain 'el'")
	}

	// 4. Cells and Line Conversion
	cells := ParseLineToCells(boldHello)
	if len(cells) != 5 {
		t.Errorf("expected 5 cells, got %d", len(cells))
	}
	line := CellsToLine(cells)
	if len(line) == 0 {
		t.Errorf("expected line to be converted back")
	}

	// 5. OverlayString
	base := "Hello World"
	overlay := "Gopher"
	result := OverlayString(base, overlay, 6, 0, 15)
	if !strings.Contains(result, "Gopher") {
		t.Errorf("expected overlayed text to contain 'Gopher'")
	}

	// 6. IndentText and WrapText
	indented := IndentText("Hello\nWorld", "  ")
	if indented != "  Hello\n  World" {
		t.Errorf("expected indented text, got %q", indented)
	}

	wrapped := WrapText("Hello World", 5)
	if !strings.Contains(wrapped, "\n") {
		t.Errorf("expected wrapped text to contain newlines")
	}

	// WrapText edge cases
	if WrapText("", 5) != "" {
		t.Errorf("expected empty string wrap to be empty")
	}
	if WrapText("short", 10) != "short" {
		t.Errorf("expected no-wrap for short text")
	}
	if WrapText("superlongword", 5) != "superlongword" {
		t.Errorf("expected long word wrap to remain as is")
	}

	// 7. RenderProgressBar
	progress := RenderProgressBar(10, 0.5)
	if len(progress) == 0 {
		t.Errorf("expected progress bar")
	}

	// RenderProgressBar edge cases
	pNeg := RenderProgressBar(10, -0.5)
	pPos := RenderProgressBar(10, 1.5)
	pZero := RenderProgressBar(10, 0)
	pFull := RenderProgressBar(10, 1)
	if len(pNeg) == 0 || len(pPos) == 0 || len(pZero) == 0 || len(pFull) == 0 {
		t.Errorf("expected valid progress bar bounds")
	}

	// 8. RenderLargeTime
	largeTime := RenderLargeTime(25 * time.Minute)
	if len(largeTime) == 0 {
		t.Errorf("expected large time render")
	}
	largeTime3 := RenderLargeTime3(12 * time.Hour)
	if len(largeTime3) == 0 {
		t.Errorf("expected large time 3 render")
	}

	// 9. ParseLineToCells combining chars & CellsToLine empty cell
	cellsCombined := ParseLineToCells("a\u0300")
	if len(cellsCombined) != 1 {
		t.Errorf("expected 1 cell for base+combining character, got %d", len(cellsCombined))
	}
	emptyTextCells := []Cell{{Text: "", Style: ""}}
	if CellsToLine(emptyTextCells) == "" {
		t.Errorf("expected non-empty output for empty text cell (space padding)")
	}

	// 10. OverlayString out of bounds
	oOutLeft := OverlayString("Hello", "World", -2, 0, 5)
	oOutRight := OverlayString("Hello", "World", 10, 0, 5)
	oOutDown := OverlayString("Hello", "World", 0, 5, 5)
	if oOutLeft == "" || oOutRight == "" || oOutDown == "" {
		t.Errorf("expected out-of-bounds overlays to fail gracefully without crashing")
	}

	// 11. SliceAnsi extra edge cases
	sliceLeftRight := SliceAnsi(boldHello, 2, 4)
	if !strings.Contains(sliceLeftRight, "ll") {
		t.Errorf("expected sliceLeftRight to contain 'll'")
	}
	sliceNoAnsi := SliceAnsi("abcdef", 1, 4)
	if sliceNoAnsi != "bcd" {
		t.Errorf("expected 'bcd', got %q", sliceNoAnsi)
	}

	// 12. ParseLineToCells extra edge cases
	cellsWide := ParseLineToCells("hello 界")
	if len(cellsWide) != 8 {
		t.Errorf("expected 8 cells, got %d", len(cellsWide))
	}

	// 13. CellsToLine style transitions
	cellsStyleChange := []Cell{
		{Text: "a", Style: "\x1b[31m"},
		{Text: "b", Style: "\x1b[31m"},
		{Text: "c", Style: ""},
	}
	lineStyle := CellsToLine(cellsStyleChange)
	if !strings.Contains(lineStyle, "\x1b[0m") {
		t.Errorf("expected reset escape sequence in lineStyle")
	}
}
