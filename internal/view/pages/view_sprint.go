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
	var titleStyled string
	if isSprintFocused {
		titleStyled = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(sprintTitleStr)
	} else {
		titleStyled = lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render(sprintTitleStr)
	}

	// Summary stats
	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	totalFeatures := len(defined) + len(inProgress) + len(review) + len(testing) + len(completed)
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

	metricsText := fmt.Sprintf("• %d Features • %d SP [%d%%]", totalFeatures, totalSP, pct)
	metricsStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(metricsText)

	var navHintStr string
	if m.CurrentMode == viewmodel.ModeSprintTaskMove {
		navHintStr = "MOVE: j/k reorder • h/l lane • ↵ save • esc cancel"
	} else {
		navHintStr = "s ◂ · ▸ S Switch Sprint"
	}
	navHint := lipgloss.NewStyle().Foreground(t.Muted).Render(navHintStr)
	if m.CurrentMode == viewmodel.ModeSprintTaskMove {
		navHint = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Bold(true).Render(navHintStr)
	}

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
	availW := workspaceW - 2
	minColW := 28
	if availW < minColW {
		availW = minColW
	}
	numCols := 5

	visibleCols := (availW + 1) / (minColW + 1)
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

	// Distribute available width across visible columns to fill 100% of horizontal space
	availForCols := availW - (visibleCols - 1)
	baseColW := availForCols / visibleCols
	remColW := availForCols % visibleCols

	laneHeight := appContentHeight - 4
	if laneHeight < 8 {
		laneHeight = 8
	}

	type laneDef struct {
		name      string
		shortName string
		tasks     []model.Task
		color     lipgloss.Color
	}

	swimlanes := []laneDef{
		{name: "DEFINED", shortName: "DEF", tasks: defined, color: lipgloss.Color("#b4befe")},
		{name: "IN PROGRESS", shortName: "IN PROG", tasks: inProgress, color: lipgloss.Color("#f9e2af")},
		{name: "REVIEW", shortName: "REV", tasks: review, color: lipgloss.Color("#89dceb")},
		{name: "TESTING", shortName: "TEST", tasks: testing, color: lipgloss.Color("#cba6f7")},
		{name: "COMPLETED", shortName: "DONE", tasks: completed, color: lipgloss.Color("#a6e3a1")},
	}

	var renderedColumns []string
	for idx := startCol; idx < endCol; idx++ {
		lane := swimlanes[idx]
		isLaneActive := isSprintFocused && m.SprintSwimlaneIdx == idx

		visIdx := idx - startCol
		colWidth := baseColW
		if visIdx < remColW {
			colWidth++
		}
		if colWidth < minColW {
			colWidth = minColW
		}

		laneSP := 0
		for _, task := range lane.tasks {
			laneSP += task.StoryPoints
		}

		headerTitle := formatSprintHeader(lane.name, lane.shortName, len(lane.tasks), laneSP, colWidth)

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
			emptyMsg := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Width(colWidth).Align(lipgloss.Center).Render("(No features)")
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

func formatSprintHeader(name, shortName string, count int, sp int, maxW int) string {
	// Level 1: Full name + count + SP
	if sp > 0 {
		opt1 := fmt.Sprintf("%s (%d • %d SP)", name, count, sp)
		if lipgloss.Width(opt1) <= maxW {
			return opt1
		}
	}

	// Level 2: Full name + count
	opt2 := fmt.Sprintf("%s (%d)", name, count)
	if lipgloss.Width(opt2) <= maxW {
		return opt2
	}

	// Level 3: Short name + count
	opt3 := fmt.Sprintf("%s (%d)", shortName, count)
	if lipgloss.Width(opt3) <= maxW {
		return opt3
	}

	// Level 4: Short name only
	opt4 := shortName
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
	if len(name) > 0 {
		return string([]rune(name)[:1])
	}
	return ""
}

func getWorkItemTypeDetails(itemType model.WorkItemType) (label string, color lipgloss.Color) {
	switch itemType {
	case model.WorkItemDefect:
		return "Defect", lipgloss.Color("#f38ba8")
	case model.WorkItemImprovement:
		return "Improvement", lipgloss.Color("#fab387")
	case model.WorkItemTask:
		return "Task", lipgloss.Color("#b4befe")
	default:
		return "Feature", lipgloss.Color("#89dceb")
	}
}

func renderSprintBlockedBufferBlock(w int, text string, isSelected bool) string {
	color := lipgloss.Color("#f38ba8")
	borderStyle := lipgloss.NewStyle().Foreground(color)
	textStyle := lipgloss.NewStyle().Foreground(color).Bold(isSelected)

	return embedTextInLine("└", "┘", "╌", " "+text+" ", w, borderStyle, textStyle)
}

func renderSprintCard(m *viewmodel.Model, t theme.Theme, task model.Task, colW int, isSelected bool) []string {
	innerW := colW - 2
	if innerW < 10 {
		innerW = 10
	}

	_, typeColor := getWorkItemTypeDetails(task.WorkItemType)
	pColor := t.PriorityColor(task.Priority)
	isDone := task.LifecycleState == model.StateCompleted

	borderColor := lipgloss.Color("#313244")
	if isSelected {
		if m.CurrentMode == viewmodel.ModeSprintTaskMove {
			borderColor = lipgloss.Color("#f9e2af")
		} else {
			borderColor = t.FocusPurple
		}
	} else if isDone {
		borderColor = lipgloss.Color("#4c644f")
	} else if task.Priority == model.P0 {
		borderColor = pColor
	} else {
		borderColor = typeColor
	}

	cursor := "  "
	if isSelected {
		if m.CurrentMode == viewmodel.ModeSprintTaskMove {
			cursor = "❖ "
		} else {
			cursor = "▶ "
		}
	}

	// Concise 3-letter ID display
	idStr := task.ID
	if idStr == "" {
		idStr = "FEA-1"
	}

	// Line 1: [ID] Title directly (no checkbox, no spelled-out type)
	idBadge := fmt.Sprintf("%s[%s] ", cursor, idStr)
	idBadgeW := lipgloss.Width(idBadge)
	title := theme.SentenceCase(task.Title)
	maxTitleW := innerW - idBadgeW
	if maxTitleW < 3 {
		maxTitleW = 3
	}
	if lipgloss.Width(title) > maxTitleW {
		runes := []rune(title)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxTitleW {
			runes = runes[:len(runes)-1]
		}
		title = string(runes) + "…"
	}

	idStyle := lipgloss.NewStyle().Foreground(typeColor).Bold(true)
	if isSelected {
		idStyle = idStyle.Foreground(t.FocusPurple)
	}
	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(isSelected)
	if isDone {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true)
	}
	topLine := idStyle.Render(idBadge) + titleStyle.Render(title)

	var cardLines []string
	cardLines = append(cardLines, topLine)

	// Line 2 (optional): Description rendered right below title
	if strings.TrimSpace(task.Description) != "" {
		desc := strings.TrimSpace(task.Description)
		descStr := "  " + desc
		if lipgloss.Width(descStr) > innerW {
			runes := []rune(descStr)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > innerW {
				runes = runes[:len(runes)-1]
			}
			descStr = string(runes) + "…"
		}
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render(descStr))
	}

	// Line 3: Priority, Story Points, Tags
	var metaParts []string
	metaParts = append(metaParts, string(task.Priority))
	if task.StoryPoints > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d SP", task.StoryPoints))
	}
	if len(task.Tags) > 0 {
		metaParts = append(metaParts, "# "+strings.Join(task.Tags, ", "))
	}
	metaStr := "  " + strings.Join(metaParts, " • ")
	if lipgloss.Width(metaStr) > innerW {
		runes := []rune(metaStr)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > innerW {
			runes = runes[:len(runes)-1]
		}
		metaStr = string(runes) + "…"
	}
	metaLine := lipgloss.NewStyle().Foreground(t.Muted).Render(metaStr)
	cardLines = append(cardLines, metaLine)

	// If linked to another feature and not blocked
	if task.LinkedFeatureID != "" && task.BlockedBy == "" {
		linkStr := fmt.Sprintf("  Link: %s", task.LinkedFeatureID)
		if lipgloss.Width(linkStr) > innerW {
			runes := []rune(linkStr)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > innerW {
				runes = runes[:len(runes)-1]
			}
			linkStr = string(runes) + "…"
		}
		cardLines = append(cardLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Render(linkStr))
	}

	content := strings.Join(cardLines, "\n")
	cardBox := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 0).
		Render(content)

	resultLines := strings.Split(cardBox, "\n")

	// If blocked, render an additional continuous card below saying blocked and blocked by
	if task.BlockedBy != "" {
		blockedText := fmt.Sprintf("⛔ Blocked by %s", task.BlockedBy)
		blockedBlock := renderSprintBlockedBufferBlock(innerW+2, blockedText, isSelected)
		resultLines = append(resultLines, blockedBlock)
	}

	return resultLines
}
