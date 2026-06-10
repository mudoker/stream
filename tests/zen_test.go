package tests

import (
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
				{Type: timer.FocusSession, Duration: 90 * time.Minute},
				{Type: timer.BreakSession, Duration: 20 * time.Minute},
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 40 * time.Minute},
				{Type: timer.BreakSession, Duration: 5 * time.Minute},
			},
		},
		{
			name:     "80 minutes task (60m - 110m)",
			duration: 80 * time.Minute,
			expected: []timer.Session{
				{Type: timer.FocusSession, Duration: 50 * time.Minute},
				{Type: timer.BreakSession, Duration: 10 * time.Minute},
				{Type: timer.FocusSession, Duration: 30 * time.Minute},
				{Type: timer.BreakSession, Duration: 5 * time.Minute},
			},
		},
		{
			name:     "40 minutes task (< 60m)",
			duration: 40 * time.Minute,
			expected: []timer.Session{
				{Type: timer.FocusSession, Duration: 40 * time.Minute},
				{Type: timer.BreakSession, Duration: 5 * time.Minute},
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

	if len(zt.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(zt.Sessions))
	}

	zt.TimeRemaining = 70 * time.Minute
	zt.TotalDuration = 90 * time.Minute

	newTask := task
	newTask.StoryPoints = 3

	zt.UpdateTaskDuration(newTask)

	if len(zt.Sessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(zt.Sessions))
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
	syncEngine, err := sync.NewSyncEngine(database, nil)
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
