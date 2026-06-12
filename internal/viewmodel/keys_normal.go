package viewmodel

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) HandleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.SidebarFocus {
		if handled, cmd := m.handleSidebarNormalKeys(key); handled {
			return m, cmd
		}
	}

	// Try global views and navigation keys
	if handled, cmd := m.handleGlobalViewsAndNavigation(key); handled {
		return m, cmd
	}

	// Try global actions keys
	if handled, cmd := m.handleGlobalActions(key); handled {
		return m, cmd
	}

	// Navigation keys depending on active view
	switch m.CurrentView {
	case MonthView:
		m.handleMonthNav(key)
	case WeekView:
		m.handleWeekNav(key)
	case DayView:
		m.handleDayNav(key)
	case DashboardView, AnalyticsView:
		m.handleDashboardOrAnalyticsNav(key)
	}

	return m, nil
}

func (m *Model) handleSidebarNormalKeys(key string) (bool, tea.Cmd) {
	switch key {
	case "j":
		m.moveSidebarView(1)
		return true, nil
	case "k":
		m.moveSidebarView(-1)
		return true, nil
	case "tab":
		m.cycleFocus()
		return true, nil
	}
	return false, nil
}

func isFutureDay(day time.Time) bool {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	target := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	return target.After(today)
}
