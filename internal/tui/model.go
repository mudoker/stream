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
	sidebarW := w * 15 / 100
	if sidebarW < 14 {
		sidebarW = 14
	} else if sidebarW > 20 {
		sidebarW = 20
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

	// Auto-activation Task Prompt
	PromptOpen bool
	PromptTask model.Task

	// Daily Shutdown Review
	ReviewOpen           bool
	ReviewTasksCompleted int
	ReviewTasksDeferred  int
	ReviewFocusSeconds   int

	// Scrolling & Help View States
	HelpOpen          bool
	ScrollOffset      int
	ShelfScrollOffset int
}

func NewModel(database *db.JSONDB, syncEngine *sync.SyncEngine) Model {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "Enter command (e.g. create task, day, week, month, sync, q)..."
	cmdInput.Prompt = ": "

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
	}

	m.refreshTasks()
	return m
}

func (m *Model) refreshTasks() {
	m.Tasks = m.DB.GetTasks()

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

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
	)
}
