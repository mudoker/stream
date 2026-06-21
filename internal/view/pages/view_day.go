package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/components"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

// RenderDayTimeline renders the 24-hour timeline grid for the day view.
func RenderDayTimeline(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	now := time.Now()
	isToday := viewmodel.SameDay(m.SelectedDay, now)

	const scale = 5
	visualRows := viewmodel.TotalRows / scale         // 288
	visualRowsPerHour := viewmodel.RowsPerHour / scale // 12

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
	clones := make(map[string]bool)
	for _, task := range m.Tasks {
		if strings.HasSuffix(task.UUID, "_moving") {
			clones[strings.TrimSuffix(task.UUID, "_moving")] = true
		} else if strings.HasSuffix(task.UUID, "_adjusting") {
			clones[strings.TrimSuffix(task.UUID, "_adjusting")] = true
		}
	}

	var anchoredTasks []model.Task
	for _, task := range m.Tasks {
		if clones[task.UUID] {
			continue
		}
		if model.IsTaskAnchored(task) && viewmodel.SameDay(task.TimeWindow.Start, m.SelectedDay) {
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

	// Right spacer remains constant since remainder is added to the last column in cellWidths
	actualRightSpacerW := rightSpacerW

	// Pre-calculate cell widths based on task overlaps
	cellWidths := make([][]int, numCols)
	for c := 0; c < numCols; c++ {
		cellWidths[c] = make([]int, visualRows)
		for r := 0; r < visualRows; r++ {
			if c == numCols-1 {
				cellWidths[c][r] = colsAreaW - (numCols-1)*colW
			} else {
				cellWidths[c][r] = colW
			}
		}
	}

	lastOccupiedRowFirst := make([]int, numCols)
	for c := 0; c < numCols; c++ {
		lastOccupiedRowFirst[c] = -1
	}

	for _, rc := range cols {
		if strings.HasSuffix(rc.Task.UUID, "_moving") || strings.HasSuffix(rc.Task.UUID, "_adjusting") {
			continue
		}
		if rc.TotalCol == 1 {
			startRow := viewmodel.TimeToRow(rc.Task.TimeWindow.Start) / scale
			endRow := viewmodel.TimeToRow(rc.Task.TimeWindow.End) / scale

			// Determine visual start row based on commute buffer
			commuteRows := 0
			if rc.Task.SchedulingType == model.Event && strings.TrimSpace(rc.Task.Location) != "" && rc.Task.CommuteBuffer > 0 {
				commuteRows = durationToRows(time.Duration(rc.Task.CommuteBuffer) * time.Minute)
			}

			topStartRow := startRow - commuteRows
			if topStartRow < 0 {
				topStartRow = 0
			}

			// Prevent visual overlap in the same column by ensuring the task starts after the predecessor's visual block
			if lastOccupiedRowFirst[0] != -1 && topStartRow < lastOccupiedRowFirst[0] {
				topStartRow = lastOccupiedRowFirst[0]
				startRow = topStartRow + commuteRows
			}

			// Adjust endRow for commute buffer and rest buffer
			if rc.Task.SchedulingType == model.Event && strings.TrimSpace(rc.Task.Location) != "" && rc.Task.CommuteBuffer > 0 {
				endRow += commuteRows
			}
			restDur := viewmodel.CalculateTaskRestTime(rc.Task)
			restRows := durationToRows(restDur)
			if restRows > 0 {
				endRow += restRows
			}

			if startRow < 0 {
				startRow = 0
			}
			limitRow := endRow - 1
			if limitRow >= visualRows {
				limitRow = visualRows - 1
			}

			for r := startRow; r <= limitRow; r++ {
				cellWidths[0][r] = colsAreaW
				for c := 1; c < numCols; c++ {
					cellWidths[c][r] = 0
				}
			}

			h := endRow - startRow + 1
			if h < 1 {
				h = 1
			}
			maxRowOccupied := startRow + h - 1
			lastOccupiedRowFirst[0] = maxRowOccupied
		}
	}

	nowRow := -1
	if isToday {
		nowRow = viewmodel.TimeToRow(now) / scale
	}

	// ── Initialize Columns ───────────────────────────────────────────
	gutterRows := make([]string, visualRows)
	leftSpacerRows := make([]string, visualRows)
	rightSpacerRows := make([]string, visualRows)
	taskRows := make([][]string, numCols)
	for c := 0; c < numCols; c++ {
		taskRows[c] = make([]string, visualRows)
	}

	for r := 0; r < visualRows; r++ {
		isHourRow := r%visualRowsPerHour == 0
		hour := r / visualRowsPerHour

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

		// 3. Task Columns
		for c := 0; c < numCols; c++ {
			w := cellWidths[c][r]
			if w == 0 {
				taskRows[c][r] = ""
				continue
			}
			if r == nowRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render(strings.Repeat("─", w))
			} else if isHourRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", w))
			} else {
				taskRows[c][r] = strings.Repeat(" ", w)
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

	// Track the last occupied row for each column to handle visual spacing of consecutive blocks
	lastOccupiedRow := make([]int, numCols)
	for c := 0; c < numCols; c++ {
		lastOccupiedRow[c] = -1
	}

	// Overlay tasks onto the columns
	for _, rc := range cols {
		isSpecial := strings.HasSuffix(rc.Task.UUID, "_moving") || strings.HasSuffix(rc.Task.UUID, "_adjusting")

		startRow := viewmodel.TimeToRow(rc.Task.TimeWindow.Start) / scale
		endRow := viewmodel.TimeToRow(rc.Task.TimeWindow.End) / scale

		colIndex := rc.ColIndex
		if colIndex >= numCols {
			colIndex = numCols - 1
		}

		// Determine visual start row based on commute buffer
		commuteRows := 0
		if rc.Task.SchedulingType == model.Event && strings.TrimSpace(rc.Task.Location) != "" && rc.Task.CommuteBuffer > 0 {
			commuteRows = durationToRows(time.Duration(rc.Task.CommuteBuffer) * time.Minute)
		}

		topStartRow := startRow - commuteRows
		if topStartRow < 0 {
			topStartRow = 0
		}

		// Prevent visual overlap in the same column by ensuring the task starts after the predecessor's visual block
		if !isSpecial && lastOccupiedRow[colIndex] != -1 && topStartRow < lastOccupiedRow[colIndex] {
			topStartRow = lastOccupiedRow[colIndex]
			startRow = topStartRow + commuteRows
		}

		// Map structural height accurately across row milestones
		h := endRow - startRow + 1
		if h < 1 {
			h = 1
		}

		if startRow+h > visualRows {
			h = visualRows - startRow
		}

		isActive := isToday && now.After(rc.Task.TimeWindow.Start) && now.Before(rc.Task.TimeWindow.End)
		isSelected := !m.TodoShelfFocus && !m.SidebarFocus && rc.Task.UUID == m.SelectedTaskUUID

		// Capture exact coordinates of focused task card
		if isSelected {
			selectedStartRow = startRow
			selectedEndRow = startRow + h
		}

		cardW := cellWidths[colIndex][startRow]
		if isSpecial {
			cardW = colsAreaW
		}

		// Render Top Commute Buffer
		if rc.Task.SchedulingType == model.Event && strings.TrimSpace(rc.Task.Location) != "" && rc.Task.CommuteBuffer > 0 {
			commuteDur := time.Duration(rc.Task.CommuteBuffer) * time.Minute
			commuteRows := durationToRows(commuteDur)
			topStartRow := startRow - commuteRows
			if topStartRow < 0 {
				topStartRow = 0
			}
			topH := startRow - topStartRow
			if topH > 0 {
				topCommuteTime := rc.Task.TimeWindow.Start.Add(-commuteDur)
				topCommuteStr := RenderTopCommuteBlock(t, cardW, topH, rc.Task.CommuteBuffer, topCommuteTime, isSelected)
				topCommuteLines := strings.Split(topCommuteStr, "\n")
				for i, line := range topCommuteLines {
					r := topStartRow + i
					if r >= visualRows {
						break
					}
					taskRows[colIndex][r] = line
					if isSpecial {
						for c := 1; c < numCols; c++ {
							taskRows[c][r] = ""
						}
					}
				}
				if isSelected {
					selectedStartRow = topStartRow
				}
			}
		}

		// Render main task card
		cardStr := components.RenderCard(m, t, rc.Task, cardW, h, isActive, isSelected)
		cardLines := strings.Split(strings.TrimSpace(cardStr), "\n")

		// Allow full string write to completely clear layout lines without visual truncation drops
		actualCardHeightWritten := 0
		for i, line := range cardLines {
			r := startRow + i
			if r >= visualRows {
				break
			}
			if i == len(cardLines)-1 && cellWidths[colIndex][r] == 0 && r > startRow {
				taskRows[colIndex][r-1] = line
			} else {
				taskRows[colIndex][r] = line
			}
			if isSpecial {
				for c := 1; c < numCols; c++ {
					taskRows[c][r] = ""
				}
			}
			actualCardHeightWritten++
		}

		// Dynamic cursor tracking to stack elements cleanly underneath the task card boundary
		currentRowOffset := actualCardHeightWritten

		// Render Bottom Commute Buffer
		if rc.Task.SchedulingType == model.Event && strings.TrimSpace(rc.Task.Location) != "" && rc.Task.CommuteBuffer > 0 {
			commuteDur := time.Duration(rc.Task.CommuteBuffer) * time.Minute
			commuteRows := durationToRows(commuteDur)
			bottomEndRow := startRow + h + commuteRows
			if bottomEndRow > viewmodel.TotalRows {
				bottomEndRow = viewmodel.TotalRows
			}
			bottomH := bottomEndRow - (startRow + h)
			if bottomH > 0 {
				bottomCommuteTime := rc.Task.TimeWindow.End.Add(commuteDur)
				bottomCommuteStr := RenderBottomCommuteBlock(t, cardW, bottomH, rc.Task.CommuteBuffer, bottomCommuteTime, isSelected)
				bottomCommuteLines := strings.Split(strings.TrimSpace(bottomCommuteStr), "\n")
				for i, line := range bottomCommuteLines {
					r := startRow + currentRowOffset + i
					if r >= visualRows {
						break
					}
					taskRows[colIndex][r] = line
					if isSpecial {
						for c := 1; c < numCols; c++ {
							taskRows[c][r] = ""
						}
					}
				}
				currentRowOffset += len(bottomCommuteLines)
				if isSelected {
					selectedEndRow = startRow + currentRowOffset
				}
			}
		}

		// Render Rest Buffer
		isCompleted := rc.Task.LifecycleState == model.StateCompleted
		restDur := viewmodel.CalculateTaskRestTime(rc.Task)
		restRows := durationToRows(restDur)
		if restRows > 0 {
			restEndTime := rc.Task.TimeWindow.End.Add(restDur)
			restStr := RenderRestBlock(t, cardW, restRows, int(restDur.Minutes()), restEndTime, isCompleted, isSelected)
			restLines := strings.Split(strings.TrimSpace(restStr), "\n")
			for i, line := range restLines {
				r := startRow + currentRowOffset + i
				if r >= visualRows {
					break
				}
				if i == len(restLines)-1 && cellWidths[colIndex][r] == 0 && r > startRow+currentRowOffset {
					taskRows[colIndex][r-1] = line
				} else {
					taskRows[colIndex][r] = line
				}
				if isSpecial {
					for c := 1; c < numCols; c++ {
						taskRows[c][r] = ""
					}
				}
			}
			currentRowOffset += len(restLines)
			if isSelected {
				selectedEndRow = startRow + currentRowOffset
			}
		}

		// Track the actual final row occupied by this visual block
		maxRowOccupied := startRow + h - 1
		if startRow+actualCardHeightWritten-1 > maxRowOccupied {
			maxRowOccupied = startRow + actualCardHeightWritten - 1
		}
		if startRow+currentRowOffset-1 > maxRowOccupied {
			maxRowOccupied = startRow + currentRowOffset - 1
		}
		if !isSpecial {
			lastOccupiedRow[colIndex] = maxRowOccupied
		}
	}

	// ── Assemble all rows ────────────────────────────────────────────
	allRows := make([]string, visualRows)
	for r := 0; r < visualRows; r++ {
		var sb strings.Builder
		sb.WriteString(gutterRows[r])
		sb.WriteString(leftSpacerRows[r])

		var colsSb strings.Builder
		for c := 0; c < numCols; c++ {
			if cellWidths[c][r] == 0 {
				colsSb.WriteString("")
			} else {
				colsSb.WriteString(taskRows[c][r])
			}
		}
		colsStr := colsSb.String()
		colsWidth := lipgloss.Width(colsStr)
		if colsWidth > colsAreaW {
			colsStr = theme.SliceAnsi(colsStr, 0, colsAreaW)
		} else if colsWidth < colsAreaW {
			colsStr = colsStr + strings.Repeat(" ", colsAreaW-colsWidth)
		}

		sb.WriteString(colsStr)
		sb.WriteString(rightSpacerRows[r])
		allRows[r] = sb.String()
	}

	// ── Seamless Viewport Looping Slicing ────────────────────────────
	visibleH := appContentHeight - 3
	if visibleH < 8 {
		visibleH = 8
	}

	centerRow := m.TimelineHour * visualRowsPerHour
	startR := centerRow - visibleH/2

	// ── Smart Focus Tracking Adjustment Mechanism ───────────────────
	if selectedStartRow != -1 && (m.SelectedTaskUUID != m.PrevSelectedTaskUUID || m.CurrentMode == viewmodel.ModeTaskMove) {
		if selectedEndRow-selectedStartRow < visibleH {
			endR := startR + visibleH

			if selectedStartRow < startR {
				startR = selectedStartRow
			}
			if selectedEndRow > endR {
				startR = selectedEndRow - visibleH
			}
		}

		// Sync back to m.TimelineHour so manual scroll starts from this adjusted position
		m.TimelineHour = (startR + visibleH/2) / visualRowsPerHour
		if m.TimelineHour < 0 {
			m.TimelineHour = 0
		}
		if m.TimelineHour > 23 {
			m.TimelineHour = 23
		}
	}

	maxStartR := visualRows - visibleH
	if maxStartR < 0 {
		maxStartR = 0
	}
	if startR < 0 {
		startR = 0
	}
	if startR > maxStartR {
		startR = maxStartR
	}

	var visible []string
	visible = append(visible, headerLine, sep, "")

	for i := 0; i < visibleH; i++ {
		r := startR + i
		if r >= 0 && r < len(allRows) {
			visible = append(visible, allRows[r])
		} else {
			visible = append(visible, "")
		}
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
			content = textStyle.Render(theme.SliceAnsi(text, 0, contentW-1) + "…")
		} else {
			content = textStyle.Render(theme.SliceAnsi(text, 0, contentW))
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

func RenderRestBlock(t theme.Theme, w, h int, restMins int, endTime time.Time, isCompleted bool, isFocused bool) string {
	var color lipgloss.Color
	if isFocused {
		color = t.FocusPurple
	} else if isCompleted {
		color = lipgloss.Color("#5b8e5d")
	} else {
		color = lipgloss.Color("#94e2d5")
	}
	restEndTimeStr := endTime.Format("15:04")
	text := fmt.Sprintf("󰔛 Rest %dm (%s)", restMins, restEndTimeStr)
	if isCompleted {
		text = fmt.Sprintf("󰔛 Rest %dm (%s) ✔", restMins, restEndTimeStr)
	}
	return renderTimelineBufferBlock(w, h, text, false, color)
}

func durationToRows(dur time.Duration) int {
	mins := int(dur.Minutes())
	if mins <= 0 {
		return 0
	}
	return (mins*(viewmodel.RowsPerHour/5) + 59) / 60
}


