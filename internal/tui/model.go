package tui

import (
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	ModeNormal          UIState = "NORMAL"
	ModeInsert          UIState = "INSERT" // For inline text entry
	ModeZen             UIState = "ZEN"    // Zen Mode Pomodoro Focus
	ModeCommand         UIState = "COMMAND"
	ModeForm            UIState = "WIZARD" // Task Creation Form Wizard
	ModeWorkspaceForm   UIState = "WORKSPACE_WIZARD"
	ModeWorkspacePicker UIState = "WS_PICKER"
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

// Layout holds all pre-computed column dimensions for the current terminal size.
// It is the single source of truth — view functions must NOT recompute these.
type Layout struct {
	SidebarW  int // Arc sidebar width
	TimelineW int // Day timeline column width (day view)
	TodoW     int // Todo shelf column width (day view)
	WorkspaceW int // Full workspace width (non-day views = TimelineW + TodoW)
	Height    int // Total terminal height
}

// computeLayout calculates all column widths from terminal dimensions.
// Sidebar: 15% of width, clamped [12, 18]
// Todo shelf: 20% of workspace
// Timeline: remaining workspace space
func computeLayout(w, h int) Layout {
	sidebarW := w * 22 / 100
	if sidebarW < 22 {
		sidebarW = 22
	} else if sidebarW > 28 {
		sidebarW = 28
	}

	workspaceW := w - sidebarW - 1 // 1 for the sidebar right border
	if workspaceW < 40 {
		workspaceW = 40
	}

	todoW := workspaceW * 22 / 100
	if todoW < 22 {
		todoW = 22
	} else if todoW > 36 {
		todoW = 36
	}

	timelineW := workspaceW - todoW - 2 // 2 for gutter between columns
	if timelineW < 30 {
		timelineW = 30
	}

	return Layout{
		SidebarW:   sidebarW,
		TimelineW:  timelineW,
		TodoW:      todoW,
		WorkspaceW: workspaceW,
		Height:     h,
	}
}

type TaskForm struct {
	Title          string
	Description    string
	PriorityIdx    int // 0: P0, 1: P1, 2: P2, 3: P3
	SPIdx          int // index in []int{1, 2, 3, 5, 8, 13}
	IsAnchored     bool
	StartHour      int
	StartMin       int
	DurationMins   int
	ActiveField    int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Anchored (Y/N), 5: Start Time, 6: Duration, 7: Submit
	TitleInput     textinput.Model
	DescInput      textinput.Model
	StartTimeInput textinput.Model
	DurationInput  textinput.Model
}

func NewTaskForm() TaskForm {
	t := textinput.New()
	t.Placeholder = "Refactor auth engine..."
	t.Focus()

	d := textinput.New()
	d.Placeholder = "Fix memory leak in pool..."

	now := time.Now()
	st := textinput.New()
	st.Placeholder = now.Format("15:04")
	st.SetValue(now.Format("15:04"))

	dur := textinput.New()
	dur.Placeholder = "60"
	dur.SetValue("60")

	return TaskForm{
		PriorityIdx:    2,
		SPIdx:          2,
		IsAnchored:     true,
		StartHour:      now.Hour(),
		StartMin:       now.Minute(),
		DurationMins:   60,
		ActiveField:    0,
		TitleInput:     t,
		DescInput:      d,
		StartTimeInput: st,
		DurationInput:  dur,
	}
}

type WorkspaceForm struct {
	Name        string
	Icon        string
	Badge       string
	ActiveField int // 0: Name, 1: Icon, 2: Badge, 3: Submit
	NameInput   textinput.Model
	IconInput   textinput.Model
	BadgeInput  textinput.Model
}

func NewWorkspaceForm() WorkspaceForm {
	name := textinput.New()
	name.Placeholder = "Aether Workspace"
	name.Focus()

	icon := textinput.New()
	icon.Placeholder = "🚀"
	icon.SetValue("🚀")

	badge := textinput.New()
	badge.Placeholder = "[Dev]"

	return WorkspaceForm{
		Name:        "",
		Icon:        "🚀",
		Badge:       "",
		ActiveField: 0,
		NameInput:   name,
		IconInput:   icon,
		BadgeInput:  badge,
	}
}

type Model struct {
	DB           *db.JSONDB
	Sync         *sync.SyncEngine
	Theme        Theme
	Layout       Layout
	CurrentView  ViewType
	CurrentMode  UIState
	Width        int
	Height       int
	Tasks        []model.Task
	SelectedDay  time.Time
	StatusMsg    string
	LastSyncTime time.Time
	SyncLogs     []string

	SelectedTaskUUID string
	TodoShelfFocus   bool // DayView specific: toggle between timeline and todo shelf
	SidebarFocus     bool // Global sidebar navigation focus toggle
	TimelineHour     int  // DayView specific: selected hour on timeline (0-23)

	// Command Palette
	CommandInput textinput.Model

	// Task Detail Inspector
	DetailOpen bool
	DetailTask model.Task

	// Zen Mode Timer
	ZenTimer *ZenTimer

	// Task Creation Wizard
	Form TaskForm

	// Workspace State & Form
	Workspaces           []model.Workspace
	ActiveWorkspaceUUID  string
	WorkspaceForm        WorkspaceForm
	IsEditingWorkspace   bool
	EditingWorkspaceUUID string

	// Auto-activation Task Prompt
	PromptOpen bool
	PromptTask model.Task

	// Daily Shutdown Review
	ReviewOpen           bool
	ReviewTasksCompleted int
	ReviewTasksDeferred  int
	ReviewFocusSeconds   int

	// Confirmation Dialog
	ConfirmOpen          bool
	ConfirmTask          model.Task

	// Task Edit Mode
	IsEditing            bool
	EditingTaskUUID      string

	// Scrolling & Help View States
	HelpOpen             bool
	HelpScrollOffset     int // scroll position inside the help modal
	ScrollOffset         int
	ShelfScrollOffset    int
	CommandSelectedIndex int
	WorkspacePickerIdx   int
}

func NewModel(database *db.JSONDB, syncEngine *sync.SyncEngine) Model {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "Search commands, tasks, or settings..."
	cmdInput.Prompt = "🔍  "

	m := Model{
		DB:             database,
		Sync:           syncEngine,
		Theme:          NewTheme(),
		Layout:         computeLayout(120, 40), // sensible default before first WindowSizeMsg
		CurrentView:    DayView,
		CurrentMode:    ModeNormal,
		SelectedDay:    time.Now(),
		LastSyncTime:   time.Now(),
		CommandInput:   cmdInput,
		TodoShelfFocus: false,
		TimelineHour:   9,
		Form:           NewTaskForm(),
		WorkspaceForm:  NewWorkspaceForm(),
	}

	m.refreshWorkspaces()
	m.refreshTasks()
	m.selectDefaultTaskForSelectedDay()
	return m
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

	// Automatically transition expired incomplete tasks to OVERDUE
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
		} else if m.TodoShelfFocus {
			m.SidebarFocus = true
			m.TodoShelfFocus = false
		} else {
			m.SidebarFocus = false
			m.TodoShelfFocus = true
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
