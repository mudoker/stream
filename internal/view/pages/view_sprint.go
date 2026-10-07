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

	// Conditionally include metrics if width allows
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

	// ── 2. Swimlane Columns ──────────────────────────────────────────────
	contentW := workspaceW - 4
	numCols := 5
	colWidth := (contentW - (numCols-1)*2) / numCols
	if colWidth < 10 {
		colWidth = 10
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
	for idx, lane := range swimlanes {
		isLaneActive := isSprintFocused && m.SprintSwimlaneIdx == idx

		laneSP := 0
		for _, task := range lane.tasks {
			laneSP += task.StoryPoints
		}

		// Responsive single-line column header without background
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
		var cardRows []string
		cardRows = append(cardRows, colHeader, colSep, "")

		if len(lane.tasks) == 0 {
			emptyMsg := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Width(colWidth).Align(lipgloss.Center).Render("(No tasks)")
			cardRows = append(cardRows, emptyMsg)
		} else {
			for _, task := range lane.tasks {
				isSelected := isLaneActive && task.UUID == m.SelectedTaskUUID
				cardRows = append(cardRows, renderSprintCard(m, t, task, colWidth, isSelected)...)
			}
		}

		colContent := strings.Join(cardRows, "\n")
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

	// Line 1: Priority + SP + Today badge
	var badges []string
	badges = append(badges, string(task.Priority))
	if task.StoryPoints > 0 {
		badges = append(badges, fmt.Sprintf("%d SP", task.StoryPoints))
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

	// Line 3: Tags or description
	var bottomLine string
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
		bottomLine = "  🏷 " + tagStr
	}

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

	if bottomLine != "" {
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(t.Muted).Render(bottomLine))
	}

	content := strings.Join(cardLines, "\n")
	cardBox := lipgloss.NewStyle().
		Width(colW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 0).
		Render(content)

	return []string{cardBox}
}
