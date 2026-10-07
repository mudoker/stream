package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/view"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func setupTestFactoryResetDB(t *testing.T) (*db.JSONDB, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "stream_reset_test_*")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	database, err := db.NewJSONDB()
	if err != nil {
		os.RemoveAll(tempDir)
		os.Setenv("HOME", origHome)
		t.Fatalf("could not init db: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
		os.Setenv("HOME", origHome)
	}

	return database, cleanup
}

func TestDBFactoryResetDirect(t *testing.T) {
	database, cleanup := setupTestFactoryResetDB(t)
	defer cleanup()

	// Add custom workspace, sprint, task
	customWS := model.Workspace{
		UUID: uuid.New().String(),
		Name: "Secret Project",
	}
	_ = database.AddWorkspace(customWS)

	customSprint := model.Sprint{
		UUID: uuid.New().String(),
		Name: "Q4 Sprint",
	}
	_ = database.AddSprint(customSprint)

	task := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Sensitive Task",
		SchedulingType: model.Floating,
	}
	_ = database.AddTask(task)

	if len(database.GetTasks()) != 1 {
		t.Fatalf("expected 1 task before reset, got %d", len(database.GetTasks()))
	}

	// Trigger Factory Reset
	err := database.FactoryReset()
	if err != nil {
		t.Fatalf("FactoryReset returned error: %v", err)
	}

	// Verify tasks wiped
	if len(database.GetTasks()) != 0 {
		t.Fatalf("expected 0 tasks after reset, got %d", len(database.GetTasks()))
	}

	// Verify default workspace restored
	workspaces := database.GetWorkspaces()
	if len(workspaces) != 1 || workspaces[0].Name != "Aether Workspace" {
		t.Fatalf("expected default Aether Workspace, got %+v", workspaces)
	}

	// Verify default sprint restored
	sprints := database.GetSprints()
	if len(sprints) != 1 || sprints[0].Name != "Sprint 1" {
		t.Fatalf("expected default Sprint 1, got %+v", sprints)
	}
}

func TestFactoryResetCommandAndCountdownFlow(t *testing.T) {
	database, cleanup := setupTestFactoryResetDB(t)
	defer cleanup()

	// Add test task
	task := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Task Before Reset",
		SchedulingType: model.Floating,
	}
	_ = database.AddTask(task)

	m := viewmodel.NewModel(database, nil)
	v := view.NewView(&m)

	// 1. Run command ":factory-reset"
	m.RunCommand("factory-reset")

	if !m.ConfirmOpen || m.ConfirmActionType != "factory_reset" {
		t.Fatalf("expected ConfirmOpen=true with ConfirmActionType='factory_reset', got open=%v type=%s", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.FactoryResetCountdown != 10 {
		t.Fatalf("expected 10 seconds countdown, got %d", m.FactoryResetCountdown)
	}

	// 2. Render view check - should display countdown in confirmation modal
	rendered := v.Render()
	if !strings.Contains(rendered, "FACTORY RESET") || !strings.Contains(rendered, "10 seconds") {
		t.Fatalf("modal should contain countdown and title: %s", rendered)
	}

	// 3. Attempting to confirm during countdown (countdown > 0) should be blocked
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = *res.(*viewmodel.Model)

	if !m.ConfirmOpen {
		t.Fatalf("dialog should remain open when confirming before countdown reaches 0")
	}
	if !strings.Contains(m.StatusMsg, "Please wait") {
		t.Fatalf("expected warning status message, got: %s", m.StatusMsg)
	}
	if len(database.GetTasks()) != 1 {
		t.Fatalf("tasks should NOT be wiped while countdown > 0")
	}

	// 4. Tick down 10 seconds via Bubble Tea messages
	for i := 0; i < 10; i++ {
		res, _ = m.Update(viewmodel.TickMsg{Time: time.Now()})
		m = *res.(*viewmodel.Model)
	}

	if m.FactoryResetCountdown != 0 {
		t.Fatalf("expected countdown to reach 0 after 10 ticks, got %d", m.FactoryResetCountdown)
	}

	// 5. Now confirm after countdown reached 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = *res.(*viewmodel.Model)

	if m.ConfirmOpen {
		t.Fatalf("dialog should close after successful confirmation")
	}
	if !strings.Contains(m.StatusMsg, "Factory reset complete") {
		t.Fatalf("expected success status message, got: %s", m.StatusMsg)
	}

	// Verify all tasks were wiped
	if len(m.Tasks) != 0 || len(database.GetTasks()) != 0 {
		t.Fatalf("expected all tasks to be wiped, in-memory=%d db=%d", len(m.Tasks), len(database.GetTasks()))
	}
}

func TestFactoryResetCancelFlow(t *testing.T) {
	database, cleanup := setupTestFactoryResetDB(t)
	defer cleanup()

	task := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Preserved Task",
		SchedulingType: model.Floating,
	}
	_ = database.AddTask(task)

	m := viewmodel.NewModel(database, nil)

	// Run command ":reset"
	m.RunCommand("reset")

	if !m.ConfirmOpen || m.ConfirmActionType != "factory_reset" {
		t.Fatalf("expected confirmation open for factory_reset")
	}

	// Cancel with Esc
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*viewmodel.Model)

	if m.ConfirmOpen {
		t.Fatalf("expected dialog to close on Esc")
	}
	if !strings.Contains(m.StatusMsg, "cancelled") {
		t.Fatalf("expected cancelled status, got: %s", m.StatusMsg)
	}
	if len(database.GetTasks()) != 1 {
		t.Fatalf("task should be preserved after cancel")
	}
}
