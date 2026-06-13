package viewmodel

import (
	"time"

	"stream/constant"
	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel/timer"

	"github.com/charmbracelet/bubbles/textinput"
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
	ModeInsert          UIState = "INSERT"
	ModeZen             UIState = "ZEN"
	ModeCommand         UIState = "COMMAND"
	ModeForm            UIState = "WIZARD"
	ModeTaskMove        UIState = "TASK_MOVE"
	ModeTaskDurationAdjust UIState = "DURATION_ADJUST"
	ModeWorkspaceForm   UIState = "WORKSPACE_WIZARD"
	ModeWorkspacePicker UIState = "WS_PICKER"
	ModeProfileForm     UIState = "PROFILE_WIZARD"
	ModeSyncForm        UIState = "SYNC_SETTINGS"
)

type TickMsg struct {
	Time time.Time
}

type SyncLogMsg struct {
	Message string
}

type AuthCompleteMsg struct{}

type Model struct {
	DB           *db.JSONDB
	Sync         *sync.SyncEngine
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
	ViewFunc     func(*Model) string

	SelectedTaskUUID     string
	PrevSelectedTaskUUID string
	TodoShelfFocus       bool
	SidebarFocus         bool
	TimelineHour         int

	CommandInput textinput.Model

	DetailOpen bool
	DetailTask model.Task

	ZenTimer *timer.ZenTimer

	Form TaskForm

	Workspaces           []model.Workspace
	ActiveWorkspaceUUID  string
	WorkspaceForm        WorkspaceForm
	IsEditingWorkspace   bool
	EditingWorkspaceUUID string

	PromptOpen        bool
	PromptTask        model.Task
	PromptSelectedIdx int

	ReviewOpen           bool
	ReviewTasksCompleted int
	ReviewTasksDeferred  int
	ReviewFocusSeconds   int

	ConfirmOpen bool
	ConfirmTask model.Task
	ConfirmActionType string
	WarningOpen bool
	WarningMsg  string

	AuthNoticeOpen bool
	AuthNoticeMsg  string

	IsEditing       bool
	EditingTaskUUID string

	AnchorPromptOpen    bool
	AnchorPromptTask    model.Task
	AnchorTimeInput     textinput.Model
	AnchorDurationInput textinput.Model
	AnchorActiveField   int

	HelpOpen             bool
	HelpScrollOffset     int
	ScrollOffset         int
	ShelfScrollOffset    int
	CommandSelectedIndex int
	WorkspacePickerIdx   int
	DashboardFocusCol    int
	DashboardFocusRow    int
	AnalyticsFocusCol    int
	AnalyticsFocusRow    int

	TaskMovePrefix             string
	TaskMoveOriginalTimeWindow model.TimeWindow
	ZenPrefix                  string

	ProfileForm                 ProfileForm
	SyncForm                    SyncForm
	IsLocked                    bool
	LockPasswordInput           textinput.Model
	SessionTimeRemainingSeconds int
	SessionExpiryPromptOpen     bool
	LastTodoShelfTaskUUID       string
}

func NewModel(database *db.JSONDB, syncEngine *sync.SyncEngine) Model {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "Search commands, tasks, or settings..."
	cmdInput.Prompt = "🔍  "

	settings := database.GetUserSettings()
	lockPass := textinput.New()
	lockPass.Placeholder = "Enter password to unlock..."
	lockPass.EchoMode = textinput.EchoPassword
	lockPass.EchoCharacter = '•'
	lockPass.Focus()

	m := Model{
		DB:                          database,
		Sync:                        syncEngine,
		Layout:                      ComputeLayout(constant.DefaultLayoutWidth, constant.DefaultLayoutHeight),
		Width:                       constant.DefaultLayoutWidth,
		Height:                      constant.DefaultLayoutHeight,
		CurrentView:                 DayView,
		CurrentMode:                 ModeNormal,
		SelectedDay:                 time.Now(),
		LastSyncTime:                time.Now(),
		CommandInput:                cmdInput,
		TodoShelfFocus:              false,
		TimelineHour:                9,
		Form:                        NewTaskForm(),
		WorkspaceForm:               NewWorkspaceForm(),
		ProfileForm:                 NewProfileForm(settings.Username, settings.LockTimeoutMinutes),
		SyncForm:                    NewSyncForm(settings.NormalizedGCalSync()),
		IsLocked:                    false,
		LockPasswordInput:           lockPass,
		SessionTimeRemainingSeconds: settings.LockTimeoutMinutes * 60,
		SessionExpiryPromptOpen:     false,
		AuthNoticeOpen:              false,
		LastTodoShelfTaskUUID:       "",
	}

	m.refreshWorkspaces()
	m.refreshTasks()
	m.selectDefaultTaskForSelectedDay()
	return m
}

func (m *Model) View() string {
	if m.ViewFunc != nil {
		return m.ViewFunc(m)
	}
	return ""
}
