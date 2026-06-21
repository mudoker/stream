package viewmodel

import (
	"strconv"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/viewmodel/timer"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

func (m *Model) refreshWorkspaces() {
	if m.DB == nil {
		return
	}
	m.Workspaces = m.DB.GetWorkspaces()
	if m.ActiveWorkspaceUUID == "" && len(m.Workspaces) > 0 {
		m.ActiveWorkspaceUUID = m.Workspaces[0].UUID
	}
}

func (m *Model) refreshTasks() {
	if m.DB == nil {
		return
	}
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
		if model.IsTaskAnchored(t) && t.SchedulingType != model.Event &&
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
			dayTasks := m.GetDayTasks()
			if len(dayTasks) > 0 {
				m.SelectedTaskUUID = dayTasks[0].UUID
				m.TimelineHour = dayTasks[0].TimeWindow.Start.Hour()
			} else {
				m.SelectedTaskUUID = ""
			}
		} else if m.TodoShelfFocus {
			m.LastTodoShelfTaskUUID = m.SelectedTaskUUID
			m.SidebarFocus = true
			m.TodoShelfFocus = false
		} else {
			m.SidebarFocus = false
			m.TodoShelfFocus = true
			shelf := m.GetTodoShelfTasks()
			if len(shelf) > 0 {
				found := false
				if m.LastTodoShelfTaskUUID != "" {
					for _, t := range shelf {
						if t.UUID == m.LastTodoShelfTaskUUID {
							found = true
							break
						}
					}
				}
				if found {
					m.SelectedTaskUUID = m.LastTodoShelfTaskUUID
				} else {
					m.SelectedTaskUUID = shelf[0].UUID
				}
			} else {
				m.SelectedTaskUUID = ""
			}
		}
	} else {
		m.SidebarFocus = !m.SidebarFocus
	}
}

func (m *Model) CycleFocus() {
	m.cycleFocus()
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

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
		m.CheckForUpdatesCmd(),
	)
}

func (m *Model) RefreshTasks() {
	m.refreshTasks()
}

func (m *Model) RefreshWorkspaces() {
	m.refreshWorkspaces()
}

func (m *Model) GetDB() *db.JSONDB {
	return m.DB
}

func (m *Model) SetStatusMsg(msg string) {
	m.StatusMsg = msg
}

func (m *Model) SetConfirmOpen(open bool) {
	m.ConfirmOpen = open
}

func (m *Model) SetConfirmActionType(actionType string) {
	m.ConfirmActionType = actionType
}

func (m *Model) SetConfirmTask(task model.Task) {
	m.ConfirmTask = task
}

func (m *Model) SetConfirmSelectedIndex(idx int) {
	m.ConfirmSelectedIndex = idx
}

func (m *Model) GetConfirmTask() model.Task {
	return m.ConfirmTask
}

func (m *Model) GetConfirmSelectedIndex() int {
	return m.ConfirmSelectedIndex
}

func (m *Model) GetZenTimer() *timer.ZenTimer {
	return m.ZenTimer
}

func (m *Model) SetZenTimer(zt *timer.ZenTimer) {
	m.ZenTimer = zt
}

func (m *Model) GetTasks() []model.Task {
	return m.Tasks
}

func (m *Model) SetTasks(tasks []model.Task) {
	m.Tasks = tasks
}

func (m *Model) IsDetailOpen() bool {
	return m.DetailOpen
}

func (m *Model) SetDetailOpen(open bool) {
	m.DetailOpen = open
}

func (m *Model) GetDetailTask() model.Task {
	return m.DetailTask
}

func (m *Model) SetDetailTask(task model.Task) {
	m.DetailTask = task
}

func (m *Model) TriggerGCalPushIfAnchored(task model.Task) {
	m.triggerGCalPushIfAnchored(task)
}

func (m *Model) SetSelectedDay(day time.Time) {
	m.SelectedDay = day
}

func (m *Model) GetSelectedDay() time.Time {
	return m.SelectedDay
}

func (m *Model) SetCurrentMode(mode string) {
	m.CurrentMode = UIState(mode)
}

func (m *Model) SetLogSessionPromptOpen(open bool) {
	m.LogSessionPromptOpen = open
}

func (m *Model) SetLogSessionPromptTask(task model.Task) {
	m.LogSessionPromptTask = task
}

func (m *Model) InitLogSessionInputs(plannedMins int) {
	m.LogSessionFocusInput = textinput.New()
	m.LogSessionFocusInput.SetValue(strconv.Itoa(plannedMins))
	m.LogSessionFocusInput.Focus()

	m.LogSessionBreakInput = textinput.New()
	m.LogSessionBreakInput.SetValue("0")

	m.LogSessionActiveField = 0
}

func (m *Model) AddTask(task model.Task) {
	if m.DB != nil {
		m.DB.AddTask(task)
	} else {
		m.Tasks = append(m.Tasks, task)
	}
}

func (m *Model) UpdateTask(task model.Task) {
	if m.DB != nil {
		m.DB.UpdateTask(task)
	} else {
		m.updateTaskInMemory(task)
	}
}

func (m *Model) DeleteTask(uuid string) {
	if m.DB != nil {
		m.DB.DeleteTask(uuid)
	} else {
		for i, t := range m.Tasks {
			if t.UUID == uuid {
				m.Tasks = append(m.Tasks[:i], m.Tasks[i+1:]...)
				break
			}
		}
	}
}

