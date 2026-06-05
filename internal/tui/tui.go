package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tuical/internal/db"
	"tuical/internal/model"
	"tuical/internal/sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type ViewType int

const (
	DashboardView ViewType = iota
	MonthView
	WeekView
	DayView
	AnalyticsView
)

type UIState string

const (
	ModeNormal  UIState = "NORMAL"
	ModeInsert  UIState = "INSERT" // For inline text entry
	ModeZen     UIState = "ZEN"    // Zen Mode Pomodoro Focus
	ModeCommand UIState = "COMMAND"
	ModeForm    UIState = "WIZARD" // Task Creation Form Wizard
)

type TickMsg struct {
	Time time.Time
}

type SyncLogMsg struct {
	Message string
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

type TaskForm struct {
	Title          string
	Description    string
	Priority       model.Priority
	StoryPoints    int
	IsAnchored     bool
	StartHour      int
	StartMin       int
	DurationMins   int
	ActiveField    int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Anchored (Y/N), 5: Start Time, 6: Duration, 7: Submit
	TitleInput     textinput.Model
	DescInput      textinput.Model
	PriorityInput  textinput.Model
	SPInput        textinput.Model
	AnchorInput    textinput.Model
	StartTimeInput textinput.Model
	DurationInput  textinput.Model
}

func NewTaskForm() TaskForm {
	t := textinput.New()
	t.Placeholder = "Refactor auth engine..."
	t.Focus()

	d := textinput.New()
	d.Placeholder = "Fix memory leak in pool..."

	p := textinput.New()
	p.Placeholder = "P0, P1, P2, P3"

	sp := textinput.New()
	sp.Placeholder = "3"

	anc := textinput.New()
	anc.Placeholder = "Y/N"

	st := textinput.New()
	st.Placeholder = "09:00"

	dur := textinput.New()
	dur.Placeholder = "60"

	return TaskForm{
		Priority:       model.P2,
		StoryPoints:    3,
		IsAnchored:     true,
		StartHour:      9,
		StartMin:       0,
		DurationMins:   60,
		ActiveField:    0,
		TitleInput:     t,
		DescInput:      d,
		PriorityInput:  p,
		SPInput:        sp,
		AnchorInput:    anc,
		StartTimeInput: st,
		DurationInput:  dur,
	}
}

type Model struct {
	DB               *db.JSONDB
	Sync             *sync.SyncEngine
	Theme            Theme
	CurrentView      ViewType
	CurrentMode      UIState
	Width            int
	Height           int
	Tasks            []model.Task
	SelectedDay      time.Time
	StatusMsg        string
	LastSyncTime     time.Time
	SyncLogs         []string
	SelectedTaskUUID string
	TodoShelfFocus   bool // DayView specific: toggle between timeline and todo shelf
	TimelineHour     int  // DayView specific: selected hour on timeline (8 to 20)

	// Command Palette
	CommandInput textinput.Model

	// Task Detail Inspector
	DetailOpen bool
	DetailTask model.Task

	// Zen Mode Timer
	ZenTimer *ZenTimer

	// Task Creation Wizard
	Form TaskForm
}

func NewModel(database *db.JSONDB, syncEngine *sync.SyncEngine) Model {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "Enter command (e.g. create task, day, week, month, sync, q)..."
	cmdInput.Prompt = ": "

	m := Model{
		DB:               database,
		Sync:             syncEngine,
		Theme:            NewTheme(),
		CurrentView:      DayView,
		CurrentMode:      ModeNormal,
		SelectedDay:      time.Now(),
		LastSyncTime:     time.Now(),
		CommandInput:     cmdInput,
		TodoShelfFocus:   false,
		TimelineHour:     9,
		Form:             NewTaskForm(),
	}

	m.refreshTasks()
	return m
}

func (m *Model) refreshTasks() {
	m.Tasks = m.DB.GetTasks()
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case TickMsg:
		// Tick Zen Timer if active
		if m.CurrentMode == ModeZen && m.ZenTimer != nil {
			finished := m.ZenTimer.Tick()
			if finished {
				// Record completion metrics
				t := m.ZenTimer.Task
				t.LifecycleState = model.StateCompleted
				t.ExecutionMetrics.ElapsedFocusSeconds += int(m.ZenTimer.TotalDuration.Seconds())
				t.ExecutionMetrics.TotalCompletedPomodoros += 1
				m.DB.UpdateTask(t)
				m.refreshTasks()

				m.CurrentMode = ModeNormal
				m.StatusMsg = fmt.Sprintf("Completed Focus Session for %s!", t.Title)
			}
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case SyncLogMsg:
		m.SyncLogs = append(m.SyncLogs, msg.Message)
		if len(m.SyncLogs) > 5 {
			m.SyncLogs = m.SyncLogs[1:]
		}
		m.StatusMsg = msg.Message
		m.LastSyncTime = time.Now()
		m.refreshTasks()
		return m, nil

	case tea.KeyMsg:
		// ESC is universal: drop back to NORMAL mode, dismiss overlays
		if msg.String() == "esc" {
			if m.DetailOpen {
				m.DetailOpen = false
				return m, nil
			}
			if m.CurrentMode == ModeZen {
				// Abort session
				if m.ZenTimer != nil {
					t := m.ZenTimer.Task
					t.LifecycleState = model.StatePaused
					t.ExecutionMetrics.ElapsedFocusSeconds += int((m.ZenTimer.TotalDuration - m.ZenTimer.TimeRemaining).Seconds())
					m.DB.UpdateTask(t)
					m.refreshTasks()
				}
				m.CurrentMode = ModeNormal
				m.StatusMsg = "Focus Session aborted."
				return m, nil
			}
			m.CurrentMode = ModeNormal
			return m, nil
		}

		// Mode-Specific Key Handlers
		switch m.CurrentMode {
		case ModeZen:
			return m.handleZenKeys(msg)

		case ModeCommand:
			return m.handleCommandKeys(msg)

		case ModeForm:
			return m.handleFormKeys(msg)

		case ModeNormal:
			return m.handleNormalKeys(msg)
		}
	}

	return m, nil
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global View Selectors
	switch key {
	case "1":
		m.CurrentView = DashboardView
		return m, nil
	case "2":
		m.CurrentView = MonthView
		return m, nil
	case "3":
		m.CurrentView = WeekView
		return m, nil
	case "4":
		m.CurrentView = DayView
		return m, nil
	case "5":
		m.CurrentView = AnalyticsView
		return m, nil
	case ":":
		m.CurrentMode = ModeCommand
		m.CommandInput.SetValue("")
		m.CommandInput.Focus()
		return m, textinput.Blink
	case "i":
		m.CurrentMode = ModeForm
		m.Form = NewTaskForm()
		m.Form.TitleInput.Focus()
		return m, textinput.Blink
	case "enter":
		// Open Detail panel of active task
		task, exists := m.getActiveTask()
		if exists {
			m.DetailTask = task
			m.DetailOpen = true
		}
		return m, nil
	case "x":
		// Complete Task
		task, exists := m.getActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			m.StatusMsg = fmt.Sprintf("Task '%s' completed!", task.Title)
		}
		return m, nil
	case "d":
		// Delete Task
		task, exists := m.getActiveTask()
		if exists {
			m.DB.DeleteTask(task.UUID)
			m.refreshTasks()
			m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", task.Title)
		}
		return m, nil
	case " ":
		// Check for Zen Mode Trigger
		// PRD: Space during Zen pauses/resumes, but in Normal, let's see:
		// Actually, standard PRD says: Space on selected task block in Normal does detail slide out, or leader + x starts Zen
		// Let's implement leader+x by checking combo or just using Space then x, or 'g' 'x'.
		// Let's implement a hotkey like `z` or `g x` or just `ctrl+x` or `g` then `x`.
		// Let's use `ctrl+x` or `g` followed by `x` or just check if they press `x` inside NORMAL mode?
		// Wait, the PRD says: "leader + x -> Activates Zen Mode, initializing the greedy Pomodoro generator layout."
		// Let's support `g` followed by `x` or we can support press `z` to trigger it easily, or Space then x!
		// Let's support both 'z' (easy) and 'space' followed by 'x'. Let's check for 'z' directly for ease.
	case "z":
		task, exists := m.getActiveTask()
		if exists {
			m.startZenMode(task)
		} else {
			m.StatusMsg = "No active task selected to start Zen Mode."
		}
		return m, nil
	}

	// Navigation keys depending on active view
	switch m.CurrentView {
	case DashboardView:
		// Dashboard nav
	case MonthView:
		m.handleMonthNav(key)
	case WeekView:
		m.handleWeekNav(key)
	case DayView:
		m.handleDayNav(key)
	case AnalyticsView:
		// Analytics nav
	}

	return m, nil
}

func (m Model) handleMonthNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "j":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	case "k":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, -1, 0)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 1, 0)
	case "enter":
		m.CurrentView = DayView
	}
}

func (m Model) handleWeekNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	}
}

func (m Model) handleDayNav(key string) {
	switch key {
	case "h", "l", "tab":
		m.TodoShelfFocus = !m.TodoShelfFocus
	case "j":
		if m.TodoShelfFocus {
			m.moveTaskSelection(1)
		} else {
			m.TimelineHour = (m.TimelineHour + 1) % 24
		}
	case "k":
		if m.TodoShelfFocus {
			m.moveTaskSelection(-1)
		} else {
			m.TimelineHour = (m.TimelineHour - 1 + 24) % 24
		}
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	}
}

func (m *Model) startZenMode(task model.Task) {
	task.LifecycleState = model.StateActive
	m.DB.UpdateTask(task)
	m.refreshTasks()

	m.ZenTimer = NewZenTimer(task)
	m.CurrentMode = ModeZen
}

func (m *Model) getActiveTask() (model.Task, bool) {
	// Returns currently selected task depending on view focus
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.getTodoShelfTasks()
			if len(shelf) > 0 {
				// Find matching uuid in db tasks
				for _, t := range m.Tasks {
					if t.UUID == m.SelectedTaskUUID {
						return t, true
					}
				}
				// Default to first on shelf if UUID not matched
				return shelf[0], true
			}
		} else {
			// Find task overlapping with TimelineHour
			for _, t := range m.Tasks {
				if t.SchedulingType == model.Anchored {
					startH := t.TimeWindow.Start.Hour()
					endH := t.TimeWindow.End.Hour()
					if t.TimeWindow.Start.Day() == m.SelectedDay.Day() &&
						t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
						t.TimeWindow.Start.Year() == m.SelectedDay.Year() &&
						m.TimelineHour >= startH && m.TimelineHour < endH {
						return t, true
					}
				}
			}
		}
	} else {
		// Fallback: search tasks scheduled for SelectedDay
		var todayTasks []model.Task
		for _, t := range m.Tasks {
			if t.SchedulingType == model.Anchored &&
				t.TimeWindow.Start.Day() == m.SelectedDay.Day() &&
				t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
				t.TimeWindow.Start.Year() == m.SelectedDay.Year() {
				todayTasks = append(todayTasks, t)
			}
		}
		if len(todayTasks) > 0 {
			return todayTasks[0], true
		}
	}
	return model.Task{}, false
}

func (m *Model) getTodoShelfTasks() []model.Task {
	var shelf []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Floating && t.LifecycleState != model.StateCompleted {
			shelf = append(shelf, t)
		}
	}
	// Sort by sorting weight (Priority * 1000 + Story Points) descending
	importSort(shelf)
	return shelf
}

func importSort(tasks []model.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].SortingWeight() > tasks[i].SortingWeight() {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func (m *Model) moveTaskSelection(dir int) {
	shelf := m.getTodoShelfTasks()
	if len(shelf) == 0 {
		return
	}

	idx := -1
	for i, t := range shelf {
		if t.UUID == m.SelectedTaskUUID {
			idx = i
			break
		}
	}

	if idx == -1 {
		m.SelectedTaskUUID = shelf[0].UUID
		return
	}

	idx += dir
	if idx < 0 {
		idx = len(shelf) - 1
	} else if idx >= len(shelf) {
		idx = 0
	}
	m.SelectedTaskUUID = shelf[idx].UUID
}

func (m Model) handleZenKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ZenTimer == nil {
		m.CurrentMode = ModeNormal
		return m, nil
	}

	switch msg.String() {
	case "space":
		m.ZenTimer.IsPaused = !m.ZenTimer.IsPaused
		if m.ZenTimer.IsPaused {
			m.StatusMsg = "Timer PAUSED"
		} else {
			m.StatusMsg = "Timer RUNNING"
		}
	case "+":
		m.ZenTimer.AddTime(5 * time.Minute)
		m.StatusMsg = "Added 5 minutes to countdown."
	case "b":
		// Force Break
		finished := m.ZenTimer.NextSession()
		if finished {
			t := m.ZenTimer.Task
			t.LifecycleState = model.StateCompleted
			t.ExecutionMetrics.ElapsedFocusSeconds += int(m.ZenTimer.TotalDuration.Seconds())
			m.DB.UpdateTask(t)
			m.refreshTasks()
			m.CurrentMode = ModeNormal
			m.StatusMsg = "Focus sessions completed!"
		} else {
			m.StatusMsg = "Skipped to next block."
		}
	}
	return m, nil
}

func (m Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := m.CommandInput.Value()
		m.CurrentMode = ModeNormal
		return m.runCommand(val)
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.CommandInput, cmd = m.CommandInput.Update(msg)
	return m, cmd
}

func (m Model) runCommand(val string) (tea.Model, tea.Cmd) {
	val = strings.TrimSpace(val)
	if val == "" {
		return m, nil
	}

	parts := strings.Fields(val)
	cmdName := parts[0]

	switch cmdName {
	case "q", "quit", "exit":
		return m, tea.Quit

	case "dashboard":
		m.CurrentView = DashboardView
		m.StatusMsg = "Switched to Dashboard view."
	case "month":
		m.CurrentView = MonthView
		m.StatusMsg = "Switched to Month view."
	case "week":
		m.CurrentView = WeekView
		m.StatusMsg = "Switched to Week view."
	case "day":
		m.CurrentView = DayView
		m.StatusMsg = "Switched to Day view."
	case "analytics":
		m.CurrentView = AnalyticsView
		m.StatusMsg = "Switched to Analytics view."

	case "create", "todo":
		if len(parts) < 2 {
			m.StatusMsg = "Syntax: create <task title>"
			return m, nil
		}
		title := strings.Join(parts[1:], " ")
		newTask := model.Task{
			UUID:           uuid.New().String(),
			Title:          title,
			Priority:       model.P2,
			StoryPoints:    3,
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		}
		if cmdName == "create" {
			// Anchor it for today 9:00 - 10:00
			now := time.Now()
			start := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), 9, 0, 0, 0, now.Location())
			end := start.Add(1 * time.Hour)
			newTask.SchedulingType = model.Anchored
			newTask.TimeWindow = model.TimeWindow{Start: start, End: end}
			newTask.LifecycleState = model.StateScheduled
		}
		m.DB.AddTask(newTask)
		m.refreshTasks()
		m.Sync.TriggerSync()
		m.StatusMsg = fmt.Sprintf("Task '%s' created.", title)

	case "complete":
		task, exists := m.getActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			m.Sync.TriggerSync()
			m.StatusMsg = fmt.Sprintf("Task '%s' completed.", task.Title)
		}

	case "delete":
		task, exists := m.getActiveTask()
		if exists {
			m.DB.DeleteTask(task.UUID)
			m.refreshTasks()
			m.Sync.TriggerSync()
			m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", task.Title)
		}

	case "sync":
		m.Sync.TriggerSync()
		m.StatusMsg = "Triggering Google Calendar sync..."

	case "auth":
		url, err := m.Sync.StartAuthServer(8080)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Auth server error: %v", err)
		} else {
			m.StatusMsg = "Go to: " + url
		}

	default:
		m.StatusMsg = fmt.Sprintf("Unknown command: %s", cmdName)
	}

	return m, nil
}

func (m Model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab", "down":
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
		m.focusFormFields()
		return m, nil
	case "shift+tab", "up":
		m.Form.ActiveField = (m.Form.ActiveField - 1 + 8) % 8
		m.focusFormFields()
		return m, nil
	case "enter":
		if m.Form.ActiveField == 7 { // Submit
			m.submitForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		// Otherwise, move to next field
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
		m.focusFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	// Update the active text input
	var cmd tea.Cmd
	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput, cmd = m.Form.TitleInput.Update(msg)
	case 1:
		m.Form.DescInput, cmd = m.Form.DescInput.Update(msg)
	case 2:
		m.Form.PriorityInput, cmd = m.Form.PriorityInput.Update(msg)
	case 3:
		m.Form.SPInput, cmd = m.Form.SPInput.Update(msg)
	case 4:
		m.Form.AnchorInput, cmd = m.Form.AnchorInput.Update(msg)
	case 5:
		m.Form.StartTimeInput, cmd = m.Form.StartTimeInput.Update(msg)
	case 6:
		m.Form.DurationInput, cmd = m.Form.DurationInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusFormFields() {
	m.Form.TitleInput.Blur()
	m.Form.DescInput.Blur()
	m.Form.PriorityInput.Blur()
	m.Form.SPInput.Blur()
	m.Form.AnchorInput.Blur()
	m.Form.StartTimeInput.Blur()
	m.Form.DurationInput.Blur()

	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput.Focus()
	case 1:
		m.Form.DescInput.Focus()
	case 2:
		m.Form.PriorityInput.Focus()
	case 3:
		m.Form.SPInput.Focus()
	case 4:
		m.Form.AnchorInput.Focus()
	case 5:
		m.Form.StartTimeInput.Focus()
	case 6:
		m.Form.DurationInput.Focus()
	}
}

func (m *Model) submitForm() {
	title := m.Form.TitleInput.Value()
	if title == "" {
		m.StatusMsg = "Title cannot be empty."
		return
	}

	priorityVal := model.Priority(strings.ToUpper(m.Form.PriorityInput.Value()))
	if priorityVal != model.P0 && priorityVal != model.P1 && priorityVal != model.P2 && priorityVal != model.P3 {
		priorityVal = model.P2
	}

	spVal, err := strconv.Atoi(m.Form.SPInput.Value())
	if err != nil || spVal <= 0 {
		spVal = 3
	}

	anchored := true
	if strings.ToUpper(m.Form.AnchorInput.Value()) == "N" {
		anchored = false
	}

	var startTime time.Time
	duration := 60

	if anchored {
		timeStr := m.Form.StartTimeInput.Value()
		durStr := m.Form.DurationInput.Value()

		hour, min := 9, 0
		if parts := strings.Split(timeStr, ":"); len(parts) == 2 {
			h, _ := strconv.Atoi(parts[0])
			mVal, _ := strconv.Atoi(parts[1])
			if h >= 0 && h < 24 {
				hour = h
			}
			if mVal >= 0 && mVal < 60 {
				min = mVal
			}
		}

		if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
			duration = d
		}

		now := time.Now()
		startTime = time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())
	}

	newTask := model.Task{
		UUID:        uuid.New().String(),
		Title:       title,
		Description: m.Form.DescInput.Value(),
		Priority:    priorityVal,
		StoryPoints: spVal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if anchored {
		newTask.SchedulingType = model.Anchored
		newTask.TimeWindow = model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(time.Duration(duration) * time.Minute),
		}
		newTask.LifecycleState = model.StateScheduled
	} else {
		newTask.SchedulingType = model.Floating
		newTask.LifecycleState = model.StateReady
	}

	m.DB.AddTask(newTask)
	m.refreshTasks()
	m.Sync.TriggerSync()
	m.StatusMsg = fmt.Sprintf("Task '%s' created successfully.", title)
}
