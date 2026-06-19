package pages

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderUpcomingPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string
	today := time.Now()

	var upcoming []model.Task
	for _, task := range m.Tasks {
		if task.LifecycleState == model.StateCompleted {
			continue
		}
		isFuture := false
		if task.SchedulingType == model.Anchored {
			isFuture = task.TimeWindow.Start.After(today) && !viewmodel.SameDay(task.TimeWindow.Start, today)
		} else {
			isFuture = !viewmodel.SameDay(task.CreatedAt, today)
		}
		if isFuture {
			upcoming = append(upcoming, task)
		}
	}

	sort.Slice(upcoming, func(i, j int) bool {
		if upcoming[i].Priority != upcoming[j].Priority {
			return upcoming[i].Priority < upcoming[j].Priority
		}
		return upcoming[i].Title < upcoming[j].Title
	})

	if len(upcoming) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(" • No future tasks scheduled."))
	} else {
		maxCount := innerH / 2 - 2
		if maxCount < 2 {
			maxCount = 2
		}
		for idx, task := range upcoming {
			if idx >= maxCount {
				break
			}
			pColor := t.PriorityColor(task.Priority)
			pBadge := lipgloss.NewStyle().Foreground(pColor).Render(fmt.Sprintf("[%s]", string(task.Priority)))

			fixedW := 8
			suffixStr := ""
			if task.SchedulingType == model.Anchored {
				suffixStr = fmt.Sprintf(" (%s)", task.TimeWindow.Start.Format("Mon Jan _2"))
				fixedW = 21
			}

			title := theme.SentenceCase(task.Title)
			maxTitleW := innerW - fixedW
			if maxTitleW < 5 {
				maxTitleW = 5
			}

			titleRunes := []rune(title)
			if len(titleRunes) > maxTitleW {
				title = string(titleRunes[:maxTitleW-1]) + "…"
			}

			row := fmt.Sprintf(" • %s %s%s", pBadge, title, suffixStr)
			lines = append(lines, row)
		}
	}

	remaining := innerH - len(lines) - 2
	if remaining > 5 {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW)),
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("🔥 DAILY LOAD DISTRIBUTION:"),
		)

		pCounts := make(map[model.Priority]int)
		for _, task := range m.Tasks {
			if task.LifecycleState != model.StateCompleted {
				pCounts[task.Priority]++
			}
		}

		priorities := []model.Priority{model.P0, model.P1, model.P2, model.P3}
		pNames := []string{"P0 Critical", "P1 High    ", "P2 Medium  ", "P3 Low     "}
		pColors := []lipgloss.Color{t.P0Color, t.P1Color, t.P2Color, t.P3Color}

		maxVal := 1
		for _, p := range priorities {
			if pCounts[p] > maxVal {
				maxVal = pCounts[p]
			}
		}

		barMax := innerW - 18
		if barMax < 5 {
			barMax = 5
		}

		for idx, p := range priorities {
			cnt := pCounts[p]
			fillW := int(math.Round(float64(cnt) * float64(barMax) / float64(maxVal)))
			if fillW > barMax {
				fillW = barMax
			}
			if fillW == 0 && cnt > 0 {
				fillW = 1
			}
			bar := strings.Repeat("█", fillW) + strings.Repeat("░", barMax-fillW)
			barStyled := lipgloss.NewStyle().Foreground(pColors[idx]).Render(bar)
			row := fmt.Sprintf("  %s %s %2d tasks", pNames[idx], barStyled, cnt)
			lines = append(lines, row)
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 0 && m.DashboardFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🎯 TARGETS & LOAD DISTRIBUTION", lines, w, h, borderCol)
}
