package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

// RenderDashboard renders the main dashboard page view.
func RenderDashboard(m *viewmodel.Model, t theme.Theme, height int) string {
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
			Foreground(t.Accent).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(t.Fg).
			Bold(true).
			Render(today.Format("2006"))
	} else {
		headerDate = lipgloss.NewStyle().
			Foreground(t.Muted).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(t.Muted).
			Bold(true).
			Render(today.Format("2006"))
	}
	headerLine := headerDate + "  " + subDate

	agendaTasks := m.GetAgendaTasks()
	completedCount := 0
	plannedFocusSecs := 0
	elapsedFocusSecs := 0
	for _, task := range agendaTasks {
		if task.LifecycleState == model.StateCompleted {
			completedCount++
		}
		plannedFocusSecs += task.StoryPoints * 45 * 60
		elapsedFocusSecs += task.ExecutionMetrics.ElapsedFocusSeconds
	}

	completionPct := 0.0
	if len(agendaTasks) > 0 {
		completionPct = float64(completedCount) / float64(len(agendaTasks)) * 100
	}

	bannerItems := []string{
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("PLANNED"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(planned(plannedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("LOGGED"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(elapsed(elapsedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("DONE"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%d Tasks", completedCount))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("COMPLETION"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", completionPct))),
	}
	bullet := lipgloss.NewStyle().Foreground(t.Muted).Render("   •   ")
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
		leftHeights = viewmodel.PartitionHeights(availH, 3)
		rightHeights = viewmodel.PartitionHeights(availH, 3)
	} else {
		leftHeights = []int{15, 15, 15}
		rightHeights = []int{11, 17, 17}
	}

	var leftPanels []string
	leftPanels = append(leftPanels,
		renderAgendaPanel(m, t, leftColW, leftHeights[0]),
		renderUpcomingPanel(m, t, leftColW, leftHeights[1]),
		renderRecentActivityPanel(m, t, leftColW, leftHeights[2]),
	)

	var rightPanels []string
	rightPanels = append(rightPanels,
		renderCapacityPanel(m, t, rightColW, rightHeights[0]),
		renderBacklogHealthPanel(m, t, rightColW, rightHeights[1]),
		renderTelemetryPanel(m, t, rightColW, rightHeights[2]),
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

func planned(secs int) string {
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, min)
	}
	return fmt.Sprintf("%dm", min)
}

func elapsed(secs int) string {
	return planned(secs)
}
