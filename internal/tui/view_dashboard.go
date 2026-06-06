package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard(height int) string {
	today := time.Now()
	
	workspaceWidth := m.Layout.WorkspaceW - 4
	appContentHeight := height - 8 // minus Header + Banner + spacing
	if appContentHeight < 10 {
		appContentHeight = 10
	}

	availH := appContentHeight
	
	var headerDate, subDate string
	if !m.SidebarFocus {
		headerDate = lipgloss.NewStyle().
			Foreground(m.Theme.Accent).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(m.Theme.Fg).
			Bold(true).
			Render(today.Format("2006"))
	} else {
		headerDate = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render(today.Format("2006"))
	}
	headerLine := headerDate + "  " + subDate

	agendaTasks := m.getAgendaTasks()
	completedCount := 0
	plannedFocusSecs := 0
	elapsedFocusSecs := 0
	for _, t := range agendaTasks {
		if t.LifecycleState == model.StateCompleted {
			completedCount++
		}
		plannedFocusSecs += t.StoryPoints * 45 * 60
		elapsedFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
	}

	completionPct := 0.0
	if len(agendaTasks) > 0 {
		completionPct = float64(completedCount) / float64(len(agendaTasks)) * 100
	}

	bannerItems := []string{
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("PLANNED"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(planned(plannedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("LOGGED"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(elapsed(elapsedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("DONE"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%d Tasks", completedCount))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("COMPLETION"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", completionPct))),
	}
	bullet := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("   •   ")
	bannerStr := strings.Join(bannerItems, bullet)

	bannerContainer := lipgloss.NewStyle().
		Width(workspaceWidth).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(bannerStr)

	leftColW := (workspaceWidth * 5) / 10
	rightColW := workspaceWidth - leftColW

	var leftHeights []int
	var rightHeights []int
	
	defaultH := 45
	if availH > defaultH {
		leftHeights = partitionHeights(availH, 3)
		rightHeights = partitionHeights(availH, 3)
	} else {
		leftHeights = []int{15, 15, 15}
		rightHeights = []int{11, 17, 17}
	}

	var leftPanels []string
	leftPanels = append(leftPanels,
		m.renderAgendaPanel(leftColW, leftHeights[0]),
		m.renderUpcomingPanel(leftColW, leftHeights[1]),
		m.renderRecentActivityPanel(leftColW, leftHeights[2]),
	)

	var rightPanels []string
	rightPanels = append(rightPanels,
		m.renderCapacityPanel(rightColW, rightHeights[0]),
		m.renderBacklogHealthPanel(rightColW, rightHeights[1]),
		m.renderTelemetryPanel(rightColW, rightHeights[2]),
	)

	leftJoined := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)
	rightJoined := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftJoined, rightJoined)

	gridLines := strings.Split(columns, "\n")
	maxScroll := len(gridLines) - availH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.ScrollOffset > maxScroll {
		m.ScrollOffset = maxScroll
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}

	endIdx := m.ScrollOffset + availH
	if endIdx > len(gridLines) {
		endIdx = len(gridLines)
	}
	visibleGridLines := gridLines[m.ScrollOffset : endIdx]
	
	for len(visibleGridLines) < availH {
		visibleGridLines = append(visibleGridLines, strings.Repeat(" ", workspaceWidth))
	}
	
	visibleGrid := strings.Join(visibleGridLines, "\n")

	var out strings.Builder
	out.WriteString(headerLine + "\n\n")
	out.WriteString(bannerContainer + "\n\n")
	out.WriteString(visibleGrid)

	return out.String()
}

func (m Model) renderPanel(title string, lines []string, w, h int, borderCol lipgloss.Color) string {
	innerW := w - 6
	innerH := h - 2
	if innerW < 4 {
		innerW = 4
	}
	if innerH < 2 {
		innerH = 2
	}

	var contentLines []string
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(title))
	contentLines = append(contentLines, "")

	for _, l := range lines {
		if len(contentLines) >= innerH {
			break
		}
		contentLines = append(contentLines, l)
	}

	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}

	for i, line := range contentLines {
		rawW := lipgloss.Width(line)
		if rawW < innerW {
			contentLines[i] = line + strings.Repeat(" ", innerW-rawW)
		} else if rawW > innerW {
			contentLines[i] = sliceAnsi(line, 0, innerW)
		}
	}

	joined := strings.Join(contentLines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Width(w - 2).
		Height(innerH).
		Padding(0, 2).
		Render(joined)
}
