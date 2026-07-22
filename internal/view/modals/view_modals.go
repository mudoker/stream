package modals

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

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

func ComputeTaskMetricsInfo(m *viewmodel.Model, t theme.Theme, task model.Task) TaskMetricsInfo {
	var plannedDur time.Duration
	if task.SchedulingType == model.Anchored || task.SchedulingType == model.Event {
		plannedDur = task.TimeWindow.End.Sub(task.TimeWindow.Start)
	} else {
		plannedDur = time.Duration(task.StoryPoints) * 45 * time.Minute
	}

	focusDur := time.Duration(task.ExecutionMetrics.ElapsedFocusSeconds) * time.Second
	breakDur := time.Duration(task.ExecutionMetrics.ElapsedBreakSeconds) * time.Second

	ratioStr := "0% / 0%"
	totalSessionSecs := task.ExecutionMetrics.ElapsedFocusSeconds + task.ExecutionMetrics.ElapsedBreakSeconds
	if totalSessionSecs > 0 {
		focusPct := (task.ExecutionMetrics.ElapsedFocusSeconds * 100) / totalSessionSecs
		breakPct := 100 - focusPct
		ratioStr = fmt.Sprintf("%d%% / %d%%", focusPct, breakPct)
	}

	efficiencyStr := "N/A"
	if task.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		efficiencyPct := int(plannedDur.Seconds() * 100 / float64(task.ExecutionMetrics.ElapsedFocusSeconds))
		efficiencyStr = fmt.Sprintf("%d%%", efficiencyPct)
	}

	qualityScore := 100 - (task.ExecutionMetrics.InterruptionCount * 15)
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
		ratingColor = t.SuccessColor
	} else if qualityScore >= 70 {
		qualityRating = "Focused (Good)"
		ratingColor = lipgloss.Color("#a6e3a1")
	} else if qualityScore >= 50 {
		qualityRating = "Distracted (Fair)"
		ratingColor = lipgloss.Color("#f9e2af")
	} else {
		qualityRating = "Fragmented (Poor)"
		ratingColor = t.P0Color
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

func ModalSep(w int) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", w))
}

func RenderExecutionMetrics(task model.Task, info TaskMetricsInfo, headerIndent, itemIndent string, includePomodoros bool) string {
	var sb strings.Builder
	sb.WriteString(headerIndent + "EXECUTION METRICS\n")
	sb.WriteString(fmt.Sprintf("%s• Planned Time:    %v\n", itemIndent, info.PlannedDur))
	sb.WriteString(fmt.Sprintf("%s• Focus Logged:    %v\n", itemIndent, info.FocusDur))
	if task.SchedulingType != model.Event {
		sb.WriteString(fmt.Sprintf("%s• Rest Logged:     %v\n", itemIndent, info.BreakDur))
		sb.WriteString(fmt.Sprintf("%s• Focus/Rest:      %s\n", itemIndent, info.RatioStr))
	}
	sb.WriteString(fmt.Sprintf("%s• Efficiency:      %s\n", itemIndent, info.EfficiencyStr))
	if includePomodoros {
		sb.WriteString(fmt.Sprintf("%s• Pomodoros:       %d / %d\n", itemIndent, task.ExecutionMetrics.TotalCompletedPomodoros, task.ExecutionMetrics.TargetPomodoros))
	}
	sb.WriteString(fmt.Sprintf("%s• Interruptions:   %d\n", itemIndent, task.ExecutionMetrics.InterruptionCount))
	if task.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		sb.WriteString(fmt.Sprintf("%s• Focus Quality:   %s\n", itemIndent, info.QualityStyled))
	}
	return sb.String()
}

func RenderDetailPanel(m *viewmodel.Model, t theme.Theme, height int) string {
	task := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(strings.ToUpper(task.Title)) + "\n")
	sb.WriteString(strings.Repeat("─", 32) + "\n\n")

	sb.WriteString(fmt.Sprintf("Priority      %s\n", task.Priority))
	if task.SchedulingType != model.Reminder {
		sb.WriteString(fmt.Sprintf("Story Points  %d\n", task.StoryPoints))
	}
	sb.WriteString(fmt.Sprintf("Lifecycle     %s\n", task.LifecycleState))
	sb.WriteString(fmt.Sprintf("Schedule      %s\n\n", task.SchedulingType))

	if task.SchedulingType == model.Anchored || task.SchedulingType == model.Event {
		if task.IsAllDay {
			sb.WriteString(fmt.Sprintf("Start Date    %s (All Day)\n\n", task.TimeWindow.Start.Format("2006-01-02")))
		} else {
			sb.WriteString(fmt.Sprintf("Start Time    %s\n", task.TimeWindow.Start.Format("2006-01-02 15:04")))
			sb.WriteString(fmt.Sprintf("End Time      %s\n\n", task.TimeWindow.End.Format("15:04")))
		}
		if task.SchedulingType == model.Event {
			if task.Location != "" {
				sb.WriteString(fmt.Sprintf("Location      %s\n", task.Location))
			}
			if task.CommuteBuffer > 0 {
				sb.WriteString(fmt.Sprintf("Commute Buf   %d mins\n", task.CommuteBuffer))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("DESCRIPTION\n")
	desc := task.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render(desc) + "\n\n")

	info := ComputeTaskMetricsInfo(m, t, task)
	sb.WriteString(RenderExecutionMetrics(task, info, "", " ", false))

	return lipgloss.NewStyle().
		Foreground(t.Fg).
		Padding(1, 2).
		Height(height - 2).
		Render(sb.String())
}

func RenderDetailModal(m *viewmodel.Model, t theme.Theme) string {
	task := m.DetailTask
	const innerW = 52

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Task Inspector") + "\n")
	sb.WriteString(ModalSep(innerW) + "\n\n")

	titleStr := theme.SentenceCase(task.Title)
	titleRunes := []rune(titleStr)
	maxTitleW := innerW - 4
	if len(titleRunes) > maxTitleW {
		titleStr = string(titleRunes[:maxTitleW-3]) + "..."
	}
	titleRendered := lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(titleStr)
	sb.WriteString(fmt.Sprintf("  %s\n\n", titleRendered))

	pColor := t.PriorityColor(task.Priority)
	pBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render(fmt.Sprintf("▲ %s", task.Priority))
	if task.SchedulingType == model.Reminder {
		sb.WriteString(fmt.Sprintf("  %s  •  %s\n", pBadge, task.LifecycleState))
	} else {
		sb.WriteString(fmt.Sprintf("  %s  •  %d SP  •  %s\n", pBadge, task.StoryPoints, task.LifecycleState))
	}
	sb.WriteString(fmt.Sprintf("  Schedule: %s\n", task.SchedulingType))

	if task.SchedulingType == model.Anchored || task.SchedulingType == model.Event {
		sb.WriteString("\n")
		if task.IsAllDay {
			sb.WriteString(fmt.Sprintf("  %s (All Day)\n",
				task.TimeWindow.Start.Format("Mon Jan _2")))
		} else {
			sb.WriteString(fmt.Sprintf("  %s  →  %s\n",
				task.TimeWindow.Start.Format("Mon Jan _2  15:04"),
				task.TimeWindow.End.Format("15:04")))
		}
		if task.SchedulingType == model.Event {
			if task.Location != "" {
				sb.WriteString(fmt.Sprintf("  Location: %s\n", task.Location))
			}
			if task.CommuteBuffer > 0 {
				sb.WriteString(fmt.Sprintf("  Commute:  %d mins\n", task.CommuteBuffer))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(ModalSep(innerW) + "\n\n")

	desc := task.Description
	if desc == "" {
		desc = "(no description)"
	}
	wrapped := theme.WrapText(desc, innerW-2)
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render(theme.IndentText(wrapped, "  ")) + "\n\n")
	sb.WriteString(ModalSep(innerW) + "\n\n")

	info := ComputeTaskMetricsInfo(m, t, task)
	sb.WriteString(RenderExecutionMetrics(task, info, "  ", "  ", true))

	sb.WriteString("\n")
	sb.WriteString(ModalSep(innerW) + "\n")
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("z focus  x complete  e edit  d delete  Esc close")
	sb.WriteString("  " + hint)

	return t.ModalStyle.Render(PrepareModalContent(sb.String(), innerW))
}

func PrepareModalContent(content string, innerW int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		cells := theme.ParseLineToCells(line)
		w := len(cells)
		if w < innerW {
			for len(cells) < innerW {
				cells = append(cells, theme.Cell{Text: " "})
			}
			lines[i] = theme.CellsToLine(cells)
		}
	}
	return strings.Join(lines, "\n")
}
