package viewmodel

import (
	"sort"
	"strconv"
	"strings"
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
	realWS := m.DB.GetWorkspaces()
	allWS := model.Workspace{
		UUID:      "ALL_WORKSPACES",
		Name:      "All",
		Icon:      "🌐",
		Badge:     "[All]",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	m.Workspaces = append([]model.Workspace{allWS}, realWS...)
	if m.ActiveWorkspaceUUID == "" && len(m.Workspaces) > 0 {
		m.ActiveWorkspaceUUID = m.Workspaces[0].UUID
	}
}

func (m *Model) refreshTasks() {
	if m.DB == nil {
		return
	}
	allTasks := m.DB.GetTasks()
	m.Tasks = allTasks

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

func (m *Model) GetConfirmFocusArea() int {
	return m.ConfirmFocusArea
}

func (m *Model) SetConfirmFocusArea(area int) {
	m.ConfirmFocusArea = area
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

func (m *Model) InitLogSessionInputs(focusMins, breakMins int) {
	m.LogSessionFocusInput = textinput.New()
	m.LogSessionFocusInput.SetValue(strconv.Itoa(focusMins))
	m.LogSessionFocusInput.Focus()

	m.LogSessionBreakInput = textinput.New()
	m.LogSessionBreakInput.SetValue(strconv.Itoa(breakMins))

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

func (m *Model) GetWorkspaceName(wsUUID string) string {
	for _, ws := range m.Workspaces {
		if ws.UUID == wsUUID {
			return ws.Name
		}
	}
	return ""
}

func (m *Model) GetTagsAutocompleteSuggestion() string {
	if m.DB == nil {
		return ""
	}
	val := m.Form.TagsInput.Value()
	parts := strings.Split(val, ",")
	if len(parts) == 0 {
		return ""
	}
	lastToken := strings.TrimSpace(parts[len(parts)-1])
	if lastToken == "" {
		return ""
	}

	tags := m.DB.GetTags()
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Frequency > tags[j].Frequency
	})

	for _, tag := range tags {
		if strings.HasPrefix(strings.ToLower(tag.Name), strings.ToLower(lastToken)) && len(tag.Name) > len(lastToken) {
			return tag.Name[len(lastToken):]
		}
	}
	return ""
}

func (m *Model) AutocompleteTag(sug string) {
	val := m.Form.TagsInput.Value()
	lastCommaIdx := strings.LastIndex(val, ",")
	if lastCommaIdx == -1 {
		m.Form.TagsInput.SetValue(val + sug + ", ")
	} else {
		m.Form.TagsInput.SetValue(val[:lastCommaIdx+1] + " " + strings.TrimSpace(val[lastCommaIdx+1:]) + sug + ", ")
	}
	m.Form.TagsInput.SetCursor(len(m.Form.TagsInput.Value()))
}


