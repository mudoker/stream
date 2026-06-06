package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

type TaskMetricsInfo struct {
	PlannedDur    time.Duration
	FocusDur      time.Duration
	BreakDur      time.Duration
	RatioStr      string
	EfficiencyStr string
	QualityScore  int
	QualityStyled string
}

func (m Model) computeTaskMetricsInfo(t model.Task) TaskMetricsInfo {
	var plannedDur time.Duration
	if t.SchedulingType == model.Anchored {
		plannedDur = t.TimeWindow.End.Sub(t.TimeWindow.Start)
	} else {
		plannedDur = time.Duration(t.StoryPoints) * 45 * time.Minute
	}

	focusDur := time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds) * time.Second
	breakDur := time.Duration(t.ExecutionMetrics.ElapsedBreakSeconds) * time.Second

	ratioStr := "0% / 0%"
	totalSessionSecs := t.ExecutionMetrics.ElapsedFocusSeconds + t.ExecutionMetrics.ElapsedBreakSeconds
	if totalSessionSecs > 0 {
		focusPct := (t.ExecutionMetrics.ElapsedFocusSeconds * 100) / totalSessionSecs
		breakPct := 100 - focusPct
		ratioStr = fmt.Sprintf("%d%% / %d%%", focusPct, breakPct)
	}

	efficiencyStr := "N/A"
	if t.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		efficiencyPct := int(plannedDur.Seconds() * 100 / float64(t.ExecutionMetrics.ElapsedFocusSeconds))
		efficiencyStr = fmt.Sprintf("%d%%", efficiencyPct)
	}

	qualityScore := 100 - (t.ExecutionMetrics.InterruptionCount * 15)
	if focusDur > 0 && breakDur > 0 {
		breakRatio := breakDur.Seconds() / focusDur.Seconds()
		if breakRatio > 0.25 {
			excess := breakRatio - 0.20
			penalty := int(excess * 100)
			qualityScore -= penalty
		}
	}
	if qualityScore < 0 {
		qualityScore = 0
	}
	qualityRating := "Optimal"
	var ratingColor lipgloss.Color
	if qualityScore >= 90 {
		qualityRating = "Optimal (Excellent)"
		ratingColor = m.Theme.SuccessColor
	} else if qualityScore >= 70 {
		qualityRating = "Focused (Good)"
		ratingColor = lipgloss.Color("#a6e3a1")
	} else if qualityScore >= 50 {
		qualityRating = "Distracted (Fair)"
		ratingColor = lipgloss.Color("#f9e2af")
	} else {
		qualityRating = "Fragmented (Poor)"
		ratingColor = m.Theme.P0Color
	}
	qualityStyled := lipgloss.NewStyle().Foreground(ratingColor).Bold(true).Render(qualityRating)

	return TaskMetricsInfo{
		PlannedDur:    plannedDur,
		FocusDur:      focusDur,
		BreakDur:      breakDur,
		RatioStr:      ratioStr,
		EfficiencyStr: efficiencyStr,
		QualityScore:  qualityScore,
		QualityStyled: qualityStyled,
	}
}

// modalSep returns a styled horizontal rule for use inside modals.
func (m Model) modalSep(w int) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", w))
}

func (m Model) renderDetailPanel(height int) string {
	t := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(strings.ToUpper(t.Title)) + "\n")
	sb.WriteString(strings.Repeat("─", 32) + "\n\n")

	sb.WriteString(fmt.Sprintf("Priority      %s\n", t.Priority))
	sb.WriteString(fmt.Sprintf("Story Points  %d\n", t.StoryPoints))
	sb.WriteString(fmt.Sprintf("Lifecycle     %s\n", t.LifecycleState))
	sb.WriteString(fmt.Sprintf("Schedule      %s\n\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("Start Time    %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("End Time      %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("DESCRIPTION\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(desc) + "\n\n")

	sb.WriteString("EXECUTION METRICS\n")
	info := m.computeTaskMetricsInfo(t)
	sb.WriteString(fmt.Sprintf(" • Planned Time:    %v\n", info.PlannedDur))
	sb.WriteString(fmt.Sprintf(" • Focus Logged:    %v\n", info.FocusDur))
	sb.WriteString(fmt.Sprintf(" • Rest Logged:     %v\n", info.BreakDur))
	sb.WriteString(fmt.Sprintf(" • Focus/Rest:      %s\n", info.RatioStr))
	sb.WriteString(fmt.Sprintf(" • Efficiency:      %s\n", info.EfficiencyStr))
	sb.WriteString(fmt.Sprintf(" • Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))
	if t.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		sb.WriteString(fmt.Sprintf(" • Focus Quality:   %s\n", info.QualityStyled))
	}

	return lipgloss.NewStyle().
		Foreground(m.Theme.Fg).
		Padding(1, 2).
		Height(height - 2).
		Render(sb.String())
}

func (m Model) renderDetailModal() string {
	t := m.DetailTask
	const innerW = 52

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Task Inspector") + "\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	titleStr := sentenceCase(t.Title)
	titleRunes := []rune(titleStr)
	maxTitleW := innerW - 4
	if len(titleRunes) > maxTitleW {
		titleStr = string(titleRunes[:maxTitleW-3]) + "..."
	}
	titleRendered := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(titleStr)
	sb.WriteString(fmt.Sprintf("  %s\n\n", titleRendered))

	pColor := m.priorityColor(t.Priority)
	pBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render(fmt.Sprintf("▲ %s", t.Priority))
	sb.WriteString(fmt.Sprintf("  %s  •  %d SP  •  %s\n", pBadge, t.StoryPoints, t.LifecycleState))
	sb.WriteString(fmt.Sprintf("  Schedule: %s\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s  →  %s\n",
			t.TimeWindow.Start.Format("Mon Jan 2  15:04"),
			t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	desc := t.Description
	if desc == "" {
		desc = "(no description)"
	}
	wrapped := wrapText(desc, innerW-2)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(indentText(wrapped, "  ")) + "\n\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	info := m.computeTaskMetricsInfo(t)
	sb.WriteString("  EXECUTION METRICS\n")
	sb.WriteString(fmt.Sprintf("  • Planned Time:    %v\n", info.PlannedDur))
	sb.WriteString(fmt.Sprintf("  • Focus Logged:    %v\n", info.FocusDur))
	sb.WriteString(fmt.Sprintf("  • Rest Logged:     %v\n", info.BreakDur))
	sb.WriteString(fmt.Sprintf("  • Focus/Rest:      %s\n", info.RatioStr))
	sb.WriteString(fmt.Sprintf("  • Efficiency:      %s\n", info.EfficiencyStr))
	sb.WriteString(fmt.Sprintf("  • Pomodoros:       %d / %d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf("  • Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))
	if t.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		sb.WriteString(fmt.Sprintf("  • Focus Quality:   %s\n", info.QualityStyled))
	}

	sb.WriteString("\n")
	sb.WriteString(m.modalSep(innerW) + "\n")
	hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("z focus  x complete  e edit  d delete  Esc close")
	sb.WriteString("  " + hint)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(sb.String(), innerW))
}

func (m Model) prepareModalContent(content string, innerW int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		cells := parseLineToCells(line)
		w := len(cells)
		if w < innerW {
			for len(cells) < innerW {
				cells = append(cells, Cell{Text: " "})
			}
			lines[i] = cellsToLine(cells)
		}
	}
	return strings.Join(lines, "\n")
}
