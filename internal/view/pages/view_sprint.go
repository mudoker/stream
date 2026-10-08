package pages

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/tasks"

	"github.com/charmbracelet/lipgloss"
)

// RenderSprintView renders the Sprint Kanban view with swimlanes.
func RenderSprintView(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	workspaceW := l.TimelineW
	if workspaceW < 40 {
		workspaceW = 40
	}

	activeSprint, hasSprint := m.GetActiveSprint()
	if !hasSprint {
		return lipgloss.NewStyle().
			Width(workspaceW).
			Height(appContentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(t.Muted).
			Render("No active sprints found. Press 'I' to create a new Sprint.")
	}

	// ── 1. Sprint Header (Clean Single Line) ────────────────────────────
	isSprintFocused := !m.SidebarFocus && !m.TodoShelfFocus

	var prefix string
	var sepColor lipgloss.Color
	if isSprintFocused {
		prefix = lipgloss.NewStyle().Foreground(t.Accent).Render("● ")
		sepColor = t.Accent
	} else {
		prefix = "  "
		sepColor = lipgloss.Color("#2a2c37")
	}

	sprintTitleStr := fmt.Sprintf("◀ %s (%s → %s) ▶",
		activeSprint.Name,
		activeSprint.StartDate.Format("Jan 02, 2006"),
		activeSprint.EndDate.Format("Jan 02, 2006"),
	)
	titleStyled := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(sprintTitleStr)

	// Summary stats
	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	totalTasks := len(defined) + len(inProgress) + len(review) + len(testing) + len(completed)
	totalSP := 0
	completedSP := 0
	for _, tList := range [][]model.Task{defined, inProgress, review, testing, completed} {
		for _, task := range tList {
			totalSP += task.StoryPoints
			if task.LifecycleState == model.StateCompleted {
				completedSP += task.StoryPoints
			}
		}
	}

	pct := 0
	if totalSP > 0 {
		pct = (completedSP * 100) / totalSP
	}

	metricsText := fmt.Sprintf("• %d Tasks • %d SP [%d%%]", totalTasks, totalSP, pct)
	metricsStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(metricsText)

	navHint := lipgloss.NewStyle().Foreground(t.Muted).Render("s ◂ · ▸ S Switch Sprint")

	usedLeft := lipgloss.Width(prefix) + lipgloss.Width(titleStyled)
	usedRight := lipgloss.Width(navHint)

	var leftSide string
	if workspaceW-2-usedLeft-usedRight > lipgloss.Width(metricsStyled)+4 {
		leftSide = prefix + titleStyled + "  " + metricsStyled
	} else {
		leftSide = prefix + titleStyled
	}

	leftW := lipgloss.Width(leftSide)
	padW := (workspaceW - 2) - leftW - usedRight
	if padW < 1 {
		padW = 1
	}
	headerLine := leftSide + strings.Repeat(" ", padW) + navHint
	sep := lipgloss.NewStyle().Foreground(sepColor).Render(strings.Repeat("─", workspaceW-2))

	// ── 2. Swimlane Columns with Horizontal Scrolling ──────────────────
	contentW := workspaceW - 4
	numCols := 5
	minColW := 22 // Comfortable swimlane width before horizontal scrolling kicks in

	totalNeededW := numCols*minColW + (numCols-1)*1
	colWidth := minColW
	if contentW >= totalNeededW {
		// All 5 columns fit; expand to fill width proportionally
		colWidth = (contentW - (numCols-1)*1) / numCols
	}

	visibleCols := (contentW + 1) / (colWidth + 1)
	if visibleCols < 1 {
		visibleCols = 1
	}
	if visibleCols > numCols {
		visibleCols = numCols
	}

	// Auto-scroll horizontal offset
	m.AutoScrollSprintLane()
	startCol := m.SprintScrollColOffset
	endCol := startCol + visibleCols
	if endCol > numCols {
		endCol = numCols
		startCol = endCol - visibleCols
		if startCol < 0 {
			startCol = 0
		}
	}

	laneHeight := appContentHeight - 4
	if laneHeight < 8 {
		laneHeight = 8
	}

	type laneDef struct {
		name      string
		shortName string
		icon      string
		tasks     []model.Task
		color     lipgloss.Color
	}

	swimlanes := []laneDef{
		{name: "DEFINED", shortName: "DEF", icon: "📋", tasks: defined, color: lipgloss.Color("#b4befe")},
		{name: "IN PROGRESS", shortName: "IN PROG", icon: "⚡", tasks: inProgress, color: lipgloss.Color("#f9e2af")},
		{name: "REVIEW", shortName: "REV", icon: "🔍", tasks: review, color: lipgloss.Color("#89dceb")},
		{name: "TESTING", shortName: "TEST", icon: "🧪", tasks: testing, color: lipgloss.Color("#cba6f7")},
		{name: "COMPLETED", shortName: "DONE", icon: "✓", tasks: completed, color: lipgloss.Color("#a6e3a1")},
	}

	var renderedColumns []string
	for idx := startCol; idx < endCol; idx++ {
		lane := swimlanes[idx]
		isLaneActive := isSprintFocused && m.SprintSwimlaneIdx == idx

		laneSP := 0
		for _, task := range lane.tasks {
			laneSP += task.StoryPoints
		}

		headerTitle := formatSprintHeader(lane.icon, lane.name, lane.shortName, len(lane.tasks), laneSP, colWidth)

		var headerStyle lipgloss.Style
		var colSepColor lipgloss.Color
		if isLaneActive {
			headerStyle = lipgloss.NewStyle().
				Foreground(t.FocusPurple).
				Bold(true).
				Width(colWidth).
				Align(lipgloss.Center)
			colSepColor = t.FocusPurple
		} else {
			headerStyle = lipgloss.NewStyle().
				Foreground(t.Muted).
				Bold(true).
				Width(colWidth).
				Align(lipgloss.Center)
			colSepColor = lipgloss.Color("#45475a")
		}

		colHeader := headerStyle.Render(headerTitle)
		colSep := lipgloss.NewStyle().Foreground(colSepColor).Render(strings.Repeat("─", colWidth))

		// Column Cards
		var allCardLines []string
		selectedCardStart := -1
		selectedCardEnd := -1

		if len(lane.tasks) == 0 {
			emptyMsg := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Width(colWidth).Align(lipgloss.Center).Render("(No tasks)")
			allCardLines = append(allCardLines, emptyMsg)
		} else {
			for _, task := range lane.tasks {
				isSelected := isLaneActive && task.UUID == m.SelectedTaskUUID
				if isSelected {
					selectedCardStart = len(allCardLines)
				}
				cardLines := renderSprintCard(m, t, task, colWidth, isSelected)
				allCardLines = append(allCardLines, cardLines...)
				if isSelected {
					selectedCardEnd = len(allCardLines)
				}
				allCardLines = append(allCardLines, "") // card spacing
			}
		}

		// Vertical scrolling for lane content
		maxCardsVisibleH := laneHeight - 3
		if maxCardsVisibleH < 4 {
			maxCardsVisibleH = 4
		}

		laneOffset := 0
		if isLaneActive && selectedCardStart != -1 {
			if selectedCardEnd > laneOffset+maxCardsVisibleH {
				laneOffset = selectedCardEnd - maxCardsVisibleH
			}
			if selectedCardStart < laneOffset {
				laneOffset = selectedCardStart
			}
		}
		if laneOffset > len(allCardLines)-maxCardsVisibleH {
			laneOffset = len(allCardLines) - maxCardsVisibleH
		}
		if laneOffset < 0 {
			laneOffset = 0
		}

		var visibleCardLines []string
		if laneOffset > 0 {
			visibleCardLines = append(visibleCardLines, lipgloss.NewStyle().Foreground(t.Muted).Render("  ▲ scroll"))
		} else {
			visibleCardLines = append(visibleCardLines, "")
		}

		sliceEnd := laneOffset + maxCardsVisibleH - 1
		if sliceEnd > len(allCardLines) {
			sliceEnd = len(allCardLines)
		}
		if laneOffset < len(allCardLines) {
			visibleCardLines = append(visibleCardLines, allCardLines[laneOffset:sliceEnd]...)
		}

		for len(visibleCardLines) < maxCardsVisibleH {
			visibleCardLines = append(visibleCardLines, "")
		}

		if sliceEnd < len(allCardLines) {
			visibleCardLines = append(visibleCardLines, lipgloss.NewStyle().Foreground(t.Muted).Render("  ▼ scroll"))
		}

		var columnRows []string
		columnRows = append(columnRows, colHeader, colSep)
		columnRows = append(columnRows, visibleCardLines...)

		colContent := strings.Join(columnRows, "\n")
		colBox := lipgloss.NewStyle().
			Width(colWidth).
			Height(laneHeight).
			MaxHeight(laneHeight).
			Render(colContent)

		renderedColumns = append(renderedColumns, colBox)
	}

	// Vertical line separator matching Week Lanes design
	var sepLines []string
	for h := 0; h < laneHeight; h++ {
		sepLines = append(sepLines, "│")
	}
	sepStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#45475a")).
		Render(strings.Join(sepLines, "\n"))

	var colsToJoin []string
	for idx, col := range renderedColumns {
		if idx > 0 {
			colsToJoin = append(colsToJoin, sepStr)
		}
		colsToJoin = append(colsToJoin, col)
	}

	joinedLanes := lipgloss.JoinHorizontal(lipgloss.Top, colsToJoin...)

	var fullView []string
	fullView = append(fullView,
		headerLine,
		sep,
		joinedLanes,
	)

	return strings.Join(fullView, "\n")
}

func formatSprintHeader(icon, name, shortName string, count int, sp int, maxW int) string {
	// Level 1: Full name + count + SP
	opt1 := fmt.Sprintf("%s %s (%d • %d SP)", icon, name, count, sp)
	if lipgloss.Width(opt1) <= maxW {
		return opt1
	}

	// Level 2: Full name + count
	opt2 := fmt.Sprintf("%s %s (%d)", icon, name, count)
	if lipgloss.Width(opt2) <= maxW {
		return opt2
	}

	// Level 3: Short name + count
	opt3 := fmt.Sprintf("%s %s (%d)", icon, shortName, count)
	if lipgloss.Width(opt3) <= maxW {
		return opt3
	}

	// Level 4: Short name only
	opt4 := fmt.Sprintf("%s %s", icon, shortName)
	if lipgloss.Width(opt4) <= maxW {
		return opt4
	}

	// Level 5: Truncated
	if maxW > 3 {
		runes := []rune(opt4)
		if len(runes) > maxW {
			return string(runes[:maxW])
		}
		return opt4
	}
	return icon
}

func getTaskDurationMinutes(task model.Task) int {
	if !task.TimeWindow.Start.IsZero() && !task.TimeWindow.End.IsZero() && task.TimeWindow.End.After(task.TimeWindow.Start) {
		mins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		if mins > 0 {
			return mins
		}
	}
	if task.EstimatedDurationMins > 0 {
		return task.EstimatedDurationMins
	}
	if task.StoryPoints > 0 {
		return task.StoryPoints * 30
	}
	return 30
}

func formatDurationBadge(mins int) string {
	if mins <= 0 {
		return ""
	}
	if mins < 60 {
		return fmt.Sprintf("󱑂 %dm", mins)
	}
	h := mins / 60
	remM := mins % 60
	if remM == 0 {
		return fmt.Sprintf("󱑂 %dh", h)
	}
	return fmt.Sprintf("󱑂 %dh%dm", h, remM)
}

func renderSprintCard(m *viewmodel.Model, t theme.Theme, task model.Task, colW int, isSelected bool) []string {
	innerW := colW - 2
	if innerW < 10 {
		innerW = 10
	}

	pColor := t.PriorityColor(task.Priority)
	isDone := task.LifecycleState == model.StateCompleted

	borderColor := lipgloss.Color("#313244")
	if isSelected {
		borderColor = t.FocusPurple
	} else if isDone {
		borderColor = lipgloss.Color("#4c644f")
	} else {
		borderColor = pColor
	}

	cursor := "  "
	if isSelected {
		cursor = "▶ "
	}

	durMins := getTaskDurationMinutes(task)
	durBadge := formatDurationBadge(durMins)

	// Line 1: Priority + SP + Duration badge + Today badge
	var badges []string
	badges = append(badges, string(task.Priority))
	if task.StoryPoints > 0 {
		badges = append(badges, fmt.Sprintf("%d SP", task.StoryPoints))
	}
	if durBadge != "" {
		badges = append(badges, durBadge)
	}
	if task.AddedToToday {
		badges = append(badges, "⚡ Today")
	}
	topLine := cursor + strings.Join(badges, " • ")
	if len([]rune(topLine)) > innerW {
		topLine = string([]rune(topLine)[:innerW])
	}

	// Line 2: Title
	chk := "☐"
	if isDone {
		chk = "☑"
	}
	title := theme.SentenceCase(task.Title)
	maxTitleW := innerW - 4
	if len([]rune(title)) > maxTitleW {
		if maxTitleW > 2 {
			title = string([]rune(title)[:maxTitleW-1]) + "…"
		} else {
			title = string([]rune(title)[:maxTitleW])
		}
	}
	titleLine := "  " + chk + " " + title

	// Line 3: Time Window (if scheduled/anchored)
	var timeLine string
	if !task.TimeWindow.Start.IsZero() && !task.TimeWindow.End.IsZero() {
		timeLine = fmt.Sprintf("  🕒 %s - %s",
			task.TimeWindow.Start.Format("15:04"),
			task.TimeWindow.End.Format("15:04"),
		)
		if len([]rune(timeLine)) > innerW {
			timeLine = string([]rune(timeLine)[:innerW])
		}
	}

	// Line 4: Tags or description
	var tagLine string
	if len(task.Tags) > 0 {
		tagStr := strings.Join(task.Tags, ", ")
		maxTagW := innerW - 4
		if len([]rune(tagStr)) > maxTagW {
			if maxTagW > 2 {
				tagStr = string([]rune(tagStr)[:maxTagW-1]) + "…"
			} else {
				tagStr = string([]rune(tagStr)[:maxTagW])
			}
		}
		tagLine = "  🏷 " + tagStr
	}

	var descLine string
	if strings.TrimSpace(task.Description) != "" {
		desc := strings.TrimSpace(task.Description)
		maxDescW := innerW - 4
		if len([]rune(desc)) > maxDescW {
			if maxDescW > 2 {
				desc = string([]rune(desc)[:maxDescW-1]) + "…"
			} else {
				desc = string([]rune(desc)[:maxDescW])
			}
		}
		descLine = "  " + desc
	}

	// Card content rows
	var cardLines []string
	topStyle := lipgloss.NewStyle().Foreground(pColor).Bold(true)
	if isSelected {
		topStyle = topStyle.Foreground(t.FocusPurple)
	}
	cardLines = append(cardLines, topStyle.Render(topLine))

	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(isSelected)
	if isDone {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true)
	}
	cardLines = append(cardLines, titleStyle.Render(titleLine))

	if timeLine != "" {
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(t.Muted).Render(timeLine))
	}
	if descLine != "" && durMins >= 45 {
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(t.Muted).Render(descLine))
	}
	if tagLine != "" {
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(t.Muted).Render(tagLine))
	}

	// Dynamic height proportional to task duration:
	// <= 30m: 3 content rows
	// 45-60m: 4 content rows
	// 90-120m: 5-6 content rows
	// > 120m: 7-8 content rows
	targetContentHeight := 3
	if durMins > 30 && durMins <= 60 {
		targetContentHeight = 4
	} else if durMins > 60 && durMins <= 120 {
		targetContentHeight = 5
	} else if durMins > 120 {
		targetContentHeight = 6
	}

	for len(cardLines) < targetContentHeight {
		cardLines = append(cardLines, "")
	}

	content := strings.Join(cardLines, "\n")
	cardBox := lipgloss.NewStyle().
		Width(colW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 0).
		Render(content)

	return strings.Split(cardBox, "\n")
}
