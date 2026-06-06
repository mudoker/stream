package tui

import (
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

func (m *Model) refreshWorkspaces() {
	m.Workspaces = m.DB.GetWorkspaces()
	if m.ActiveWorkspaceUUID == "" && len(m.Workspaces) > 0 {
		m.ActiveWorkspaceUUID = m.Workspaces[0].UUID
	}
}

func (m *Model) refreshTasks() {
	allTasks := m.DB.GetTasks()
	m.Tasks = nil
	for _, t := range allTasks {
		if t.WorkspaceUUID == m.ActiveWorkspaceUUID {
			m.Tasks = append(m.Tasks, t)
		}
	}

	now := time.Now()
	updatedAny := false
	for i, t := range m.Tasks {
		if t.SchedulingType == model.Anchored &&
			t.TimeWindow.End.Before(now) &&
			t.LifecycleState != model.StateCompleted &&
			t.LifecycleState != model.StateArchived &&
			t.LifecycleState != model.StateOverdue {

			t.LifecycleState = model.StateOverdue
			m.DB.UpdateTask(t)
			m.Tasks[i] = t
			updatedAny = true
		}
	}
	if updatedAny {
		m.Tasks = m.DB.GetTasks()
	}
}

func (m *Model) cycleFocus() {
	if m.CurrentView == DayView {
		if m.SidebarFocus {
			m.SidebarFocus = false
			m.TodoShelfFocus = false
			dayTasks := m.getDayTasks()
			if len(dayTasks) > 0 {
				m.SelectedTaskUUID = dayTasks[0].UUID
				m.TimelineHour = dayTasks[0].TimeWindow.Start.Hour()
			} else {
				m.SelectedTaskUUID = ""
			}
		} else if m.TodoShelfFocus {
			m.SidebarFocus = true
			m.TodoShelfFocus = false
		} else {
			m.SidebarFocus = false
			m.TodoShelfFocus = true
			shelf := m.getTodoShelfTasks()
			if len(shelf) > 0 {
				m.SelectedTaskUUID = shelf[0].UUID
			} else {
				m.SelectedTaskUUID = ""
			}
		}
	} else {
		m.SidebarFocus = !m.SidebarFocus
	}
}

func (m *Model) moveSidebarView(delta int) {
	viewsOrder := []ViewType{
		DashboardView,
		MonthView,
		WeekView,
		DayView,
		AnalyticsView,
	}
	currentIdx := -1
	for i, v := range viewsOrder {
		if v == m.CurrentView {
			currentIdx = i
			break
		}
	}
	if currentIdx != -1 {
		nextIdx := (currentIdx + delta + len(viewsOrder)) % len(viewsOrder)
		m.CurrentView = viewsOrder[nextIdx]
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
	)
}
