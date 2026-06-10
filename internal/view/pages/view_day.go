package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/components"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderDayTimeline renders the 24-hour timeline grid for the day view.
func RenderDayTimeline(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	now := time.Now()
	isToday := viewmodel.SameDay(m.SelectedDay, now)

	// Gutter / timestamp lane is exactly 7 characters wide: " HH:MM "
	const timestampLaneW = 7
	leftSpacerW := 4
	rightSpacerW := 2

	// Grid content width (excluding timestamp gutter)
	gridW := l.TimelineW - timestampLaneW
	if gridW < 10 {
		gridW = 10
	}

	// ── Header ──────────────────────────────────────────────────────
	isTimelineFocused := !m.SidebarFocus && !m.TodoShelfFocus

	var dayName, dayDate string
	var prefix string
	var sepColor lipgloss.Color
	if isTimelineFocused {
		prefix = lipgloss.NewStyle().Foreground(t.Accent).Render("● ")
		dayName = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
			Render(m.SelectedDay.Format("Monday"))
		dayDate = lipgloss.NewStyle().Foreground(t.Fg).Bold(true).
			Render(m.SelectedDay.Format("January 2, 2006"))
		sepColor = t.Accent
	} else {
		prefix = "  "
		dayName = lipgloss.NewStyle().Foreground(t.Muted).Bold(true).
			Render(m.SelectedDay.Format("Monday"))
		dayDate = lipgloss.NewStyle().Foreground(t.Muted).Bold(true).
			Render(m.SelectedDay.Format("January 2, 2006"))
		sepColor = lipgloss.Color("#2a2c37")
	}

	sep := lipgloss.NewStyle().
		Foreground(sepColor).
		Render(strings.Repeat("─", l.TimelineW-2))

	navHint := lipgloss.NewStyle().Foreground(t.Muted).
		Render("H ◂ · ▸ L")

	// Right-align navHint in the header line
	usedW := lipgloss.Width(prefix) + lipgloss.Width(dayName) + 2 + lipgloss.Width(dayDate)
	padW := (l.TimelineW - 2) - usedW - lipgloss.Width(navHint)
	if padW < 1 {
		padW = 1
	}
	headerLine := prefix + dayName + "  " + dayDate + strings.Repeat(" ", padW) + navHint

	// ── Resolve overlapping tasks and overlay cards ──────────────────
	var anchoredTasks []model.Task
	for _, task := range m.Tasks {
		if task.SchedulingType == model.Anchored && viewmodel.SameDay(task.TimeWindow.Start, m.SelectedDay) {
			anchoredTasks = append(anchoredTasks, task)
		}
	}
	cols := viewmodel.ResolveOverlaps(anchoredTasks)

	// Find number of overlapping columns
	numCols := 1
	for _, rc := range cols {
		if rc.TotalCol > numCols {
			numCols = rc.TotalCol
		}
	}

	// Columns area width (gridW - left spacer - right spacer)
	colsAreaW := gridW - leftSpacerW - rightSpacerW
	colW := colsAreaW / numCols
	if colW < 8 {
		colW = 8
	}

	// Rounding remainder is added to the right spacer
	actualRightSpacerW := rightSpacerW + (colsAreaW - (numCols * colW))

	nowRow := -1
	if isToday {
		nowRow = viewmodel.TimeToRow(now)
	}

	// ── Initialize Columns ───────────────────────────────────────────
	gutterRows := make([]string, viewmodel.TotalRows)
	leftSpacerRows := make([]string, viewmodel.TotalRows)
	rightSpacerRows := make([]string, viewmodel.TotalRows)
	taskRows := make([][]string, numCols)
	for c := 0; c < numCols; c++ {
		taskRows[c] = make([]string, viewmodel.TotalRows)
	}

	for r := 0; r < viewmodel.TotalRows; r++ {
		isHourRow := r%viewmodel.RowsPerHour == 0
		hour := r / viewmodel.RowsPerHour

		// 1. Gutter / Timestamp Lane (7 chars: " HH:MM ")
		var label string
		if r == nowRow {
			label = fmt.Sprintf(" %02d:%02d ", now.Hour(), now.Minute())
			gutterRows[r] = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render(label)
		} else if isHourRow {
			label = fmt.Sprintf(" %02d:00 ", hour)
			isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == hour
			if isSelectedHour {
				gutterRows[r] = lipgloss.NewStyle().
					Foreground(t.Accent).
					Bold(true).
					Render(label)
			} else {
				gutterRows[r] = lipgloss.NewStyle().Foreground(t.Muted).Render(label)
			}
		} else {
			gutterRows[r] = "       "
		}

		// 2. Left Spacer Column (leftSpacerW chars)
		if r == nowRow {
			leftSpacerRows[r] = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render(strings.Repeat("─", leftSpacerW))
		} else if isHourRow {
			leftSpacerRows[r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", leftSpacerW))
		} else {
			leftSpacerRows[r] = strings.Repeat(" ", leftSpacerW)
		}

		// 3. Task Columns (colW chars each)
		for c := 0; c < numCols; c++ {
			if r == nowRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render(strings.Repeat("─", colW))
			} else if isHourRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", colW))
			} else {
				taskRows[c][r] = strings.Repeat(" ", colW)
			}
		}

		// 4. Right Spacer Column (actualRightSpacerW chars)
		if r == nowRow {
			rightSpacerRows[r] = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render(strings.Repeat("─", actualRightSpacerW))
		} else if isHourRow {
			rightSpacerRows[r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", actualRightSpacerW))
		} else {
			rightSpacerRows[r] = strings.Repeat(" ", actualRightSpacerW)
		}
	}

	// Dynamic layout variables tracking position targets for auto-clamping visibility
	selectedStartRow := -1
	selectedEndRow := -1

	// Overlay tasks onto the columns
	for _, rc := range cols {
		startRow := viewmodel.TimeToRow(rc.Task.TimeWindow.Start)
		endRow := viewmodel.TimeToRow(rc.Task.TimeWindow.End)

		// Map structural height accurately across row milestones
		h := endRow - startRow + 1

		// Compensate for top and bottom border frames generated by Lipgloss
		if h < 1 {
			h = 1
		}

		if startRow+h > viewmodel.TotalRows {
			h = viewmodel.TotalRows - startRow
		}

		colIndex := rc.ColIndex
		if colIndex >= numCols {
			colIndex = numCols - 1
		}

		isActive := isToday && now.After(rc.Task.TimeWindow.Start) && now.Before(rc.Task.TimeWindow.End)
		isSelected := !m.TodoShelfFocus && !m.SidebarFocus && rc.Task.UUID == m.SelectedTaskUUID

		// Capture exact coordinates of focused task card
		if isSelected {
			selectedStartRow = startRow
			selectedEndRow = startRow + h
		}

		cardStr := components.RenderTaskCard(m, t, rc.Task, colW, h, isActive, isSelected)
		cardLines := strings.Split(cardStr, "\n")

		for i, line := range cardLines {
			r := startRow + i
			if r >= viewmodel.TotalRows {
				break
			}
			taskRows[colIndex][r] = line
		}

		restDur := viewmodel.CalculateTaskRestTime(rc.Task)
		restRows := durationToRows(restDur)
		if restRows > 0 {
			restStr := RenderRestBlock(t, colW, restRows, int(restDur.Minutes()))
			restLines := strings.Split(restStr, "\n")
			for i, line := range restLines {
				r := startRow + h + i
				if r >= viewmodel.TotalRows {
					break
				}
				taskRows[colIndex][r] = line
			}
			if isSelected {
				selectedEndRow = startRow + h + restRows
			}
		}
	}

	// ── Assemble all rows ────────────────────────────────────────────
	allRows := make([]string, viewmodel.TotalRows)
	for r := 0; r < viewmodel.TotalRows; r++ {
		var sb strings.Builder
		sb.WriteString(gutterRows[r])
		sb.WriteString(leftSpacerRows[r])
		for c := 0; c < numCols; c++ {
			sb.WriteString(taskRows[c][r])
		}
		sb.WriteString(rightSpacerRows[r])
		allRows[r] = sb.String()
	}

	// ── Seamless Viewport Looping Slicing ────────────────────────────
	visibleH := appContentHeight
	if visibleH < 8 {
		visibleH = 8
	}

	centerRow := m.TimelineHour * viewmodel.RowsPerHour
	startR := centerRow - visibleH/2

	// ── Smart Focus Tracking Adjustment Mechanism ───────────────────
	if selectedStartRow != -1 {
		endR := startR + visibleH

		if selectedStartRow < startR {
			startR = selectedStartRow
		}
		if selectedEndRow > endR {
			startR = selectedEndRow - visibleH
		}
	}

	var visible []string
	visible = append(visible, headerLine, sep, "")

	for i := 0; i < visibleH; i++ {
		rowIndex := startR + i
		r := (rowIndex%viewmodel.TotalRows + viewmodel.TotalRows) % viewmodel.TotalRows
		visible = append(visible, allRows[r])
	}

	return strings.Join(visible, "\n")
}

func embedTextInLine(leftBorder, rightBorder, fillChar, text string, width int, borderStyle, textStyle lipgloss.Style) string {
	textW := lipgloss.Width(text)
	contentW := width - 2 // Left and right borders are 1 character each
	if contentW < 1 {
		contentW = 1
	}

	var content string
	if textW > contentW {
		if contentW > 1 {
			content = textStyle.Render(text[:contentW-1] + "…")
		} else {
			content = textStyle.Render(text[:contentW])
		}
	} else {
		leftPad := (contentW - textW) / 2
		rightPad := contentW - textW - leftPad
		content = borderStyle.Render(strings.Repeat(fillChar, leftPad)) +
			textStyle.Render(text) +
			borderStyle.Render(strings.Repeat(fillChar, rightPad))
	}

	return borderStyle.Render(leftBorder) + content + borderStyle.Render(rightBorder)
}

func RenderRestBlock(t theme.Theme, w, h int, restMins int) string {
	if w < 3 {
		w = 3
	}
	if h < 1 {
		h = 1
	}

	borderColor := lipgloss.Color("#a6e3a1")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(borderColor).Italic(true)

	topLeft := "┌"
	topRight := "┐"
	bottomLeft := "└"
	bottomRight := "┘"
	horizChar := "╌"
	vertChar := "┊"

	restText := fmt.Sprintf("󰔛 Rest %dm", restMins)

	var lines []string

	if h == 1 {
		// For h = 1, show text in one line with side borders: ┊╌ Rest 15m ╌┊
		line := embedTextInLine(vertChar, vertChar, horizChar, restText, w, borderStyle, textStyle)
		lines = append(lines, line)
	} else if h == 2 {
		// For h = 2, embed text in the top line, and render bottom line normally
		topLine := embedTextInLine(topLeft, topRight, horizChar, restText, w, borderStyle, textStyle)
		bottomLine := borderStyle.Render(bottomLeft + strings.Repeat(horizChar, w-2) + bottomRight)
		lines = append(lines, topLine)
		lines = append(lines, bottomLine)
	} else {
		// For h > 2, render top line normally, put text in center row, and bottom line normally
		topLine := borderStyle.Render(topLeft + strings.Repeat(horizChar, w-2) + topRight)
		bottomLine := borderStyle.Render(bottomLeft + strings.Repeat(horizChar, w-2) + bottomRight)
		lines = append(lines, topLine)

		centerRow := (h - 2) / 2
		for i := 0; i < h-2; i++ {
			var line string
			if i == centerRow {
				line = embedTextInLine(vertChar, vertChar, " ", restText, w, borderStyle, textStyle)
			} else {
				line = borderStyle.Render(vertChar) + strings.Repeat(" ", w-2) + borderStyle.Render(vertChar)
			}
			lines = append(lines, line)
		}
		lines = append(lines, bottomLine)
	}

	return strings.Join(lines, "\n")
}

func durationToRows(dur time.Duration) int {
	mins := int(dur.Minutes())
	if mins <= 0 {
		return 0
	}
	return (mins*viewmodel.RowsPerHour + 59) / 60
}
