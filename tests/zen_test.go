package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/timer"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPartitionTask(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected []timer.Session
	}{
		{
			name:     "180 minutes task (>= 110m)",
			duration: 180 * time.Minute,
			expected: []timer.Session{
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 30 * time.Minute},
			},
		},
		{
			name:     "80 minutes task (60m - 110m)",
			duration: 80 * time.Minute,
			expected: []timer.Session{
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 30 * time.Minute},
			},
		},
		{
			name:     "40 minutes task (< 60m)",
			duration: 40 * time.Minute,
			expected: []timer.Session{
				{Type: timer.FocusSession, Duration: 40 * time.Minute},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := timer.PartitionTask(tt.duration)
			if len(sessions) != len(tt.expected) {
				t.Fatalf("Expected %d sessions, got %d", len(tt.expected), len(sessions))
			}
			for i := range sessions {
				if sessions[i].Type != tt.expected[i].Type {
					t.Errorf("Session %d: expected type %v, got %v", i, tt.expected[i].Type, sessions[i].Type)
				}
				if sessions[i].Duration != tt.expected[i].Duration {
					t.Errorf("Session %d: expected duration %v, got %v", i, tt.expected[i].Duration, sessions[i].Duration)
				}
			}
		})
	}
}

func TestZenTimerUpdateTaskDuration(t *testing.T) {
	task := model.Task{
		UUID:        "test-task",
		Title:       "Test Task",
		StoryPoints: 2,
	}

	zt := timer.NewZenTimer(task)

	if len(zt.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(zt.Sessions))
	}

	zt.TimeRemaining = 70 * time.Minute
	zt.TotalDuration = 90 * time.Minute

	newTask := task
	newTask.StoryPoints = 3

	zt.UpdateTaskDuration(newTask)

	if len(zt.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(zt.Sessions))
	}
	if zt.TimeRemaining != 70*time.Minute {
		t.Errorf("expected TimeRemaining to be preserved at 70m, got %v", zt.TimeRemaining)
	}

	newTask2 := task
	newTask2.StoryPoints = 1

	zt.UpdateTaskDuration(newTask2)

	if len(zt.Sessions) != 1 {
		t.Fatalf("expected 1 session after shortening, got %d", len(zt.Sessions))
	}
	if zt.Sessions[0].Duration != 45*time.Minute {
		t.Errorf("expected session 0 duration to be 45m, got %v", zt.Sessions[0].Duration)
	}
	if zt.TimeRemaining != 25*time.Minute {
		t.Errorf("expected TimeRemaining to be 25m, got %v", zt.TimeRemaining)
	}

	newTask3 := task
	newTask3.SchedulingType = model.Anchored
	newTask3.TimeWindow = model.TimeWindow{
		Start: time.Now(),
		End:   time.Now().Add(15 * time.Minute),
	}
	zt.UpdateTaskDuration(newTask3)
	if zt.Sessions[0].Duration != 20*time.Minute {
		t.Errorf("expected session 0 duration to be truncated to 20m, got %v", zt.Sessions[0].Duration)
	}
	if zt.TimeRemaining != 0 {
		t.Errorf("expected TimeRemaining to be 0, got %v", zt.TimeRemaining)
	}
	if zt.Running {
		t.Errorf("expected ZenTimer to stop running")
	}
}

func TestZenTimerAdjustTimeMultiplier(t *testing.T) {
	task := model.Task{
		UUID:        "test-task",
		Title:       "Test Task",
		StoryPoints: 1,
	}

	m := &viewmodel.Model{
		ZenTimer: timer.NewZenTimer(task),
	}

	if m.ZenTimer.TimeRemaining != 45*time.Minute {
		t.Fatalf("expected 45m initial focus block, got %v", m.ZenTimer.TimeRemaining)
	}

	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	if m.ZenPrefix != "10" {
		t.Fatalf("expected ZenPrefix to be '10', got %q", m.ZenPrefix)
	}

	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if m.ZenPrefix != "" {
		t.Fatalf("expected ZenPrefix to be cleared after applying '+'")
	}
	if m.ZenTimer.TimeRemaining != 50*time.Minute {
		t.Errorf("expected TimeRemaining to be 50m after adding 5m, got %v", m.ZenTimer.TimeRemaining)
	}

	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.ZenTimer.TimeRemaining != 48*time.Minute {
		t.Errorf("expected TimeRemaining to be 48m after subtracting 2m, got %v", m.ZenTimer.TimeRemaining)
	}

	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.ZenTimer.TimeRemaining != 47*time.Minute+30*time.Second {
		t.Errorf("expected TimeRemaining to be 47m30s after subtracting 30s, got %v", m.ZenTimer.TimeRemaining)
	}
}

func TestZenTimerStopAndResume(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := viewmodel.NewModel(database, syncEngine)
	task := model.Task{
		UUID:           "test-task-stop-resume",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		Title:          "Stop and Resume Task",
		StoryPoints:    1,
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	database.AddTask(task)
	m.RefreshTasks()

	m.StartZenMode(task)
	if m.ZenTimer == nil {
		t.Fatal("expected ZenTimer to be created")
	}

	m.ZenTimer.TimeRemaining = 40 * time.Minute
	m.ZenTimer.TotalDuration = 45 * time.Minute

	m.HandleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !m.ConfirmOpen || m.ConfirmActionType != "exit_focus" {
		t.Fatal("expected exit_focus confirmation dialog to be open")
	}

	// Choose option 1 (Mark as complete) to save elapsed time and exit
	m.HandleExitFocusOption(0)

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected mode to revert to Normal, got %v", m.CurrentMode)
	}
	if m.ZenTimer != nil {
		t.Fatal("expected ZenTimer to be nil after stopping")
	}

	tOpt, ok := database.GetTask("test-task-stop-resume")
	if !ok {
		t.Fatal("task not found in database")
	}
	if tOpt.ExecutionMetrics.ElapsedFocusSeconds != 300 {
		t.Errorf("expected 300 seconds elapsed focus time, got %d", tOpt.ExecutionMetrics.ElapsedFocusSeconds)
	}

	m.TodoShelfFocus = true
	m.SelectedTaskUUID = "test-task-stop-resume"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	mRes := res.(*viewmodel.Model)

	if mRes.CurrentMode != viewmodel.ModeZen {
		t.Errorf("expected mode to return to Zen, got %v", mRes.CurrentMode)
	}
	if mRes.ZenTimer == nil {
		t.Fatal("expected ZenTimer to be reconstructed, got nil")
	}
	if mRes.ZenTimer.TimeRemaining != 40*time.Minute {
		t.Errorf("expected TimeRemaining to be preserved at 40m, got %v", mRes.ZenTimer.TimeRemaining)
	}
}

func TestExitFocusOptions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	task := model.Task{
		UUID:           "test-exit-focus",
		Title:          "Exit Focus Task",
		StoryPoints:    2,
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	database.AddTask(task)

	m := viewmodel.NewModel(database, nil)
	m.StartZenMode(task)

	m.ZenTimer.TimeRemaining = 60 * time.Minute
	m.ZenTimer.TotalDuration = 90 * time.Minute

	// Option 2: Complete and Resume (index 1)
	m.HandleExitFocusOption(1)

	t1, _ := database.GetTask("test-exit-focus")
	if t1.LifecycleState != model.StateCompleted {
		t.Errorf("expected original task to be completed, got %v", t1.LifecycleState)
	}

	var resumeTask model.Task
	foundResume := false
	tasks := database.GetTasks()
	for _, tk := range tasks {
		if strings.Contains(tk.Title, "Exit Focus Task (Resume)") {
			resumeTask = tk
			foundResume = true
			break
		}
	}

	if !foundResume {
		t.Fatal("expected resuming task to be created")
	}
	if resumeTask.SchedulingType != model.Floating {
		t.Errorf("expected resuming task to be floating, got %v", resumeTask.SchedulingType)
	}
	if resumeTask.StoryPoints != 2 {
		t.Errorf("expected resuming task to have 2 story points, got %d", resumeTask.StoryPoints)
	}
}

func TestStartLateTrimFeature(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	// 1 hour task, scheduled starting 15 minutes ago
	now := time.Now()
	task := model.Task{
		UUID:           "test-start-late",
		Title:          "Start Late Task",
		StoryPoints:    0, // use anchored time window
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: now.Add(-15 * time.Minute),
			End:   now.Add(45 * time.Minute),
		},
	}
	database.AddTask(task)

	m := viewmodel.NewModel(database, nil)

	// 1. Trigger Zen mode, check if confirm modal opens
	m.CheckAndStartZenMode(task)
	if !m.ConfirmOpen || m.ConfirmActionType != "start_late_confirm" {
		t.Fatalf("expected start_late_confirm dialog to be open, got ConfirmOpen=%t Action=%q", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.ConfirmSelectedIndex != 0 {
		t.Errorf("expected default option to be 0 (Full Duration), got %d", m.ConfirmSelectedIndex)
	}

	// 2. Select Option 0 (Full duration)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("expected confirm dialog to close")
	}
	if m.CurrentMode != viewmodel.ModeZen || m.ZenTimer == nil {
		t.Fatal("expected Zen mode to start")
	}
	// Total duration of partitioned sessions for 1h task (Focus 50m, Break 10m, Focus 10m)
	// Should be unchanged
	if len(m.ZenTimer.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(m.ZenTimer.Sessions))
	}
	if m.ZenTimer.Sessions[0].Duration != 50*time.Minute {
		t.Errorf("expected first session to be 50m, got %v", m.ZenTimer.Sessions[0].Duration)
	}

	// 3. Reset and test Option 1 (Trim to current time)
	m.CurrentMode = viewmodel.ModeNormal
	m.ZenTimer = nil
	task.LifecycleState = model.StateReady
	database.UpdateTask(task)

	m.CheckAndStartZenMode(task)
	if !m.ConfirmOpen || m.ConfirmActionType != "start_late_confirm" {
		t.Fatal("expected start_late_confirm dialog to open again")
	}

	// Move to Option 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ConfirmSelectedIndex != 1 {
		t.Errorf("expected index to be 1, got %d", m.ConfirmSelectedIndex)
	}

	// Press Enter to confirm Trimmed Duration
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.CurrentMode != viewmodel.ModeZen || m.ZenTimer == nil {
		t.Fatal("expected Zen mode to start")
	}

	// Shrunk sessions should be: Focus 35m (50 - 15), Break 10m, Focus 10m
	if len(m.ZenTimer.Sessions) != 3 {
		t.Fatalf("expected 3 sessions after trim, got %d", len(m.ZenTimer.Sessions))
	}
	// Since time has ticked slightly, delay will be slightly more than 15 minutes.
	// We can assert the first session is approximately 35 minutes, or between 34 and 35 minutes.
	firstSessDur := m.ZenTimer.Sessions[0].Duration
	if firstSessDur > 35*time.Minute || firstSessDur < 34*time.Minute {
		t.Errorf("expected first session to be trimmed to ~35m (between 34m and 35m), got %v", firstSessDur)
	}

	// Break session and second focus session must remain untouched
	if m.ZenTimer.Sessions[1].Type != timer.BreakSession || m.ZenTimer.Sessions[1].Duration != 10*time.Minute {
		t.Errorf("expected second session to be untouched Break 10m, got type=%s dur=%v", m.ZenTimer.Sessions[1].Type, m.ZenTimer.Sessions[1].Duration)
	}
	if m.ZenTimer.Sessions[2].Type != timer.FocusSession || m.ZenTimer.Sessions[2].Duration != 10*time.Minute {
		t.Errorf("expected third session to be untouched Focus 10m, got type=%s dur=%v", m.ZenTimer.Sessions[2].Type, m.ZenTimer.Sessions[2].Duration)
	}

	// 4. Test starting 55 minutes late (Focus 50m and part of Break 10m are missed)
	m.CurrentMode = viewmodel.ModeNormal
	m.ZenTimer = nil
	task.LifecycleState = model.StateReady
	now55 := time.Now()
	task.TimeWindow.Start = now55.Add(-55 * time.Minute)
	task.TimeWindow.End = now55.Add(5 * time.Minute)
	database.UpdateTask(task)

	m.CheckAndStartZenMode(task)
	// Move to Option 1 (Trim) and confirm
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Remaining session: Break 5m, Focus 10m
	if len(m.ZenTimer.Sessions) != 2 {
		t.Fatalf("expected 2 sessions after 55m trim, got %d", len(m.ZenTimer.Sessions))
	}
	if m.ZenTimer.Sessions[0].Type != timer.BreakSession {
		t.Errorf("expected first session to be BreakSession, got %s", m.ZenTimer.Sessions[0].Type)
	}
	breakDur := m.ZenTimer.Sessions[0].Duration
	if breakDur > 5*time.Minute || breakDur < 4*time.Minute {
		t.Errorf("expected break session to be trimmed to ~5m (between 4m and 5m), got %v", breakDur)
	}
	if m.ZenTimer.Sessions[1].Type != timer.FocusSession || m.ZenTimer.Sessions[1].Duration != 10*time.Minute {
		t.Errorf("expected second session to be untouched Focus 10m, got type=%s dur=%v", m.ZenTimer.Sessions[1].Type, m.ZenTimer.Sessions[1].Duration)
	}
}
