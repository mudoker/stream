package viewmodel

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleGlobalViewsAndNavigation(key string) (bool, tea.Cmd) {
	switch key {
	case "1":
		m.CurrentView = DashboardView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return true, nil
	case "2":
		m.CurrentView = MonthView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return true, nil
	case "3":
		m.CurrentView = WeekView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return true, nil
	case "4":
		m.CurrentView = DayView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return true, nil
	case "5":
		m.CurrentView = AnalyticsView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return true, nil

	case "tab":
		m.cycleFocus()
		return true, nil
	case "ctrl+d":
		if m.CurrentView == DayView {
			if m.TodoShelfFocus {
				m.ShelfScrollOffset += 2
				shelfTasks := m.GetTodoShelfTasks()
				if m.ShelfScrollOffset > len(shelfTasks)-3 {
					m.ShelfScrollOffset = len(shelfTasks) - 3
				}
				if m.ShelfScrollOffset < 0 {
					m.ShelfScrollOffset = 0
				}
			} else {
				m.TimelineHour = (m.TimelineHour + 2) % 24
			}
		} else if m.CurrentView == MonthView {
			// Scroll forward by colsFit months
			workspaceWidth := m.Layout.WorkspaceW - 4
			colWidth := workspaceWidth / 2
			innerW := colWidth - 6
			colsFit := innerW / 29
			if colsFit < 1 {
				colsFit = 1
			}
			m.ScrollOffset += colsFit
		} else {
			m.ScrollOffset += 2
		}
		return true, nil
	case "ctrl+u":
		if m.CurrentView == DayView {
			if m.TodoShelfFocus {
				m.ShelfScrollOffset -= 2
				if m.ShelfScrollOffset < 0 {
					m.ShelfScrollOffset = 0
				}
			} else {
				m.TimelineHour = (m.TimelineHour - 2 + 24) % 24
			}
		} else if m.CurrentView == MonthView {
			// Scroll backward by colsFit months (indefinitely back)
			workspaceWidth := m.Layout.WorkspaceW - 4
			colWidth := workspaceWidth / 2
			innerW := colWidth - 6
			colsFit := innerW / 29
			if colsFit < 1 {
				colsFit = 1
			}
			m.ScrollOffset -= colsFit
		} else {
			m.ScrollOffset -= 2
			if m.ScrollOffset < 0 {
				m.ScrollOffset = 0
			}
		}
		return true, nil
	case "?":
		m.HelpOpen = true
		m.HelpScrollOffset = 0
		m.StatusMsg = "Help opened. Press Esc/? to exit."
		return true, nil
	case ":":
		m.CurrentMode = ModeCommand
		m.CommandInput.SetValue("")
		m.CommandSelectedIndex = -1
		m.CommandInput.Focus()
		return true, nil
	}
	return false, nil
}
