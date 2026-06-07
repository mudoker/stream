package tui

import (
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPartitionTask(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected []Session
	}{
		{
			name:     "180 minutes task (>= 110m)",
			duration: 180 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 90 * time.Minute},
				{Type: BreakSession, Duration: 20 * time.Minute},
				{Type: FocusSession, Duration: 60 * time.Minute}, // 10m trailing merged
				{Type: BreakSession, Duration: 10 * time.Minute},
			},
		},
		{
			name:     "80 minutes task (60m - 110m)",
			duration: 80 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 70 * time.Minute}, // 20m trailing merged
				{Type: BreakSession, Duration: 10 * time.Minute},
			},
		},
		{
			name:     "40 minutes task (< 60m)",
			duration: 40 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 35 * time.Minute},
				{Type: BreakSession, Duration: 5 * time.Minute},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := PartitionTask(tt.duration)
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
	// Create a Floating task with SP=2 (90 minutes duration)
	task := model.Task{
		UUID:        "test-task",
		Title:       "Test Task",
		StoryPoints: 2,
	}

	zt := NewZenTimer(task)
	// SP=2 is 90 minutes. Partitioned into:
	// rem=90m. >= 60m => Focus: 50m, Break: 10m. rem=30m.
	// rem=30m. Not >= 60m, < 60m, >= 30m => Focus: 25m, Break: 5m. rem=0.
	// Sessions:
	// 0: Focus, 50m
	// 1: Break, 10m
	// 2: Focus, 25m
	// 3: Break, 5m

	if len(zt.Sessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(zt.Sessions))
	}

	// 1. Simulate starting work. Spent 20 minutes in the first session.
	zt.TimeRemaining = 30 * time.Minute // 50m - 20m spent = 30m remaining
	zt.TotalDuration = 50 * time.Minute

	// Now increase task duration by editing the task (SP=3, which is 135 minutes)
	newTask := task
	newTask.StoryPoints = 3

	zt.UpdateTaskDuration(newTask)

	// Expected new total duration: 135 minutes.
	// We spent 20 minutes. So 115 minutes remaining to schedule.
	// Current session has 30m remaining.
	// subRemaining = 115 - 30 = 85 minutes.
	// Existing subsequent sessions:
	// index 1: Break, 10m (10 <= 85 => keep, subRemaining = 75)
	// index 2: Focus, 25m (25 <= 75 => keep, subRemaining = 50)
	// index 3: Break, 5m  (5 <= 50  => keep, subRemaining = 45)
	// We have 45 minutes leftover.
	// PartitionTask(45m) -> Focus: 40m, Break: 5m.
	// Total sessions:
	// 0: Focus, 50m (current, 30m remaining)
	// 1: Break, 10m
	// 2: Focus, 25m
	// 3: Break, 5m
	// 4: Focus, 40m
	// 5: Break, 5m
	if len(zt.Sessions) != 6 {
		t.Fatalf("expected 6 sessions, got %d", len(zt.Sessions))
	}
	if zt.TimeRemaining != 30*time.Minute {
		t.Errorf("expected TimeRemaining to be preserved at 30m, got %v", zt.TimeRemaining)
	}

	// 2. Simulate decreasing duration.
	// Current state:
	// We spent 20m of the first session.
	// Total done so far = 20m.
	// Decrease task duration to 45m (e.g., SP=1, 45m duration).
	newTask2 := task
	newTask2.StoryPoints = 1 // 45m

	zt.UpdateTaskDuration(newTask2)

	// newDur = 45m.
	// elapsedTotal = 20m.
	// remainingToSchedule = 45 - 20 = 25m.
	// remainingToSchedule (25m) <= TimeRemaining (30m):
	// Shorten current session to 20m + 25m = 45m.
	// TimeRemaining = 25m.
	// Discard subsequent sessions.
	if len(zt.Sessions) != 1 {
		t.Fatalf("expected 1 session after shortening, got %d", len(zt.Sessions))
	}
	if zt.Sessions[0].Duration != 45*time.Minute {
		t.Errorf("expected session 0 duration to be 45m, got %v", zt.Sessions[0].Duration)
	}
	if zt.TimeRemaining != 25*time.Minute {
		t.Errorf("expected TimeRemaining to be 25m, got %v", zt.TimeRemaining)
	}

	// 3. Simulate decreasing duration below elapsed time.
	// Current state:
	// We spent 20m of the first session.
	// Total done so far = 20m.
	// Decrease task duration to 15m.
	// Since 15m <= 20m (elapsedTotal), we truncate the session to what was done (20m) and stop.
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
		StoryPoints: 1, // 45 minutes
	}

	m := &Model{
		ZenTimer: NewZenTimer(task),
	}

	// 1. Initial State: TimeRemaining = 40m, TotalDuration = 40m (due to 5m Break block partition)
	if m.ZenTimer.TimeRemaining != 40*time.Minute {
		t.Fatalf("expected 40m initial focus block, got %v", m.ZenTimer.TimeRemaining)
	}

	// 2. Press "1" then "0" then "+" (should add 10 * 30s = 5 minutes)
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	if m.ZenPrefix != "10" {
		t.Fatalf("expected ZenPrefix to be '10', got %q", m.ZenPrefix)
	}

	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if m.ZenPrefix != "" {
		t.Fatalf("expected ZenPrefix to be cleared after applying '+'")
	}
	if m.ZenTimer.TimeRemaining != 45*time.Minute {
		t.Errorf("expected TimeRemaining to be 45m after adding 5m, got %v", m.ZenTimer.TimeRemaining)
	}

	// 3. Press "4" then "-" (should subtract 4 * 30s = 2 minutes)
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.ZenTimer.TimeRemaining != 43*time.Minute {
		t.Errorf("expected TimeRemaining to be 43m after subtracting 2m, got %v", m.ZenTimer.TimeRemaining)
	}

	// 4. Press "-" without prefix (should subtract 1 * 30s = 30 seconds)
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.ZenTimer.TimeRemaining != 42*time.Minute+30*time.Second {
		t.Errorf("expected TimeRemaining to be 42m30s after subtracting 30s, got %v", m.ZenTimer.TimeRemaining)
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

	m := NewModel(database, syncEngine)
	task := model.Task{
		UUID:           "test-task-stop-resume",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		Title:          "Stop and Resume Task",
		StoryPoints:    1, // 45 minutes
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	database.AddTask(task)
	m.refreshTasks()

	// 1. Enter Zen mode
	m.startZenMode(task)
	if m.ZenTimer == nil {
		t.Fatal("expected ZenTimer to be created")
	}
	
	// Simulate time passing (say 5 minutes elapsed)
	m.ZenTimer.TimeRemaining = 40 * time.Minute
	m.ZenTimer.TotalDuration = 45 * time.Minute

	// 2. Press "q" to stop
	m.handleZenKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.CurrentMode != ModeNormal {
		t.Errorf("expected mode to revert to Normal, got %v", m.CurrentMode)
	}
	if m.ZenTimer == nil {
		t.Fatal("expected ZenTimer NOT to be nil after stopping")
	}
	if m.ZenTimer.Running {
		t.Error("expected ZenTimer to be stopped (Running = false)")
	}

	// 3. Press "z" on the same task to resume
	m.TodoShelfFocus = true
	m.SelectedTaskUUID = "test-task-stop-resume"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	mRes := res.(*Model)

	if mRes.CurrentMode != ModeZen {
		t.Errorf("expected mode to return to Zen, got %v", mRes.CurrentMode)
	}
	if mRes.ZenTimer.TimeRemaining != 40*time.Minute {
		t.Errorf("expected TimeRemaining to be preserved at 40m, got %v", mRes.ZenTimer.TimeRemaining)
	}
}


