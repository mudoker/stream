package db

import (
	"os"
	"path/filepath"
	"testing"

	"stream/internal/model"
)

func TestNewJSONDBAndBasicCRUD(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	db, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// Verify GetTasks and GetConfigDir
	tList := db.GetTasks()
	if len(tList) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tList))
	}
	if db.GetConfigDir() == "" {
		t.Errorf("expected config dir to be non-empty")
	}

	// 1. User Settings test
	settings := db.GetUserSettings()
	if settings.Username != "Doan Huu Quoc" {
		t.Errorf("expected username Doan Huu Quoc, got %s", settings.Username)
	}

	settings.Username = "New User"
	if err := db.UpdateUserSettings(settings); err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}
	if db.GetUserSettings().Username != "New User" {
		t.Errorf("expected username New User")
	}

	// 2. Workspace CRUD test
	workspaces := db.GetWorkspaces()
	if len(workspaces) != 1 {
		t.Errorf("expected 1 default workspace, got %d", len(workspaces))
	}
	defaultWS := workspaces[0]

	ws, exists := db.GetWorkspace(defaultWS.UUID)
	if !exists || ws.Name != "Aether Workspace" {
		t.Errorf("failed to get default workspace")
	}

	newWS := model.Workspace{
		UUID: "ws-test",
		Name: "Test WS",
	}
	if err := db.AddWorkspace(newWS); err != nil {
		t.Fatalf("failed to add workspace: %v", err)
	}
	if ws, exists = db.GetWorkspace("ws-test"); !exists || ws.Name != "Test WS" {
		t.Errorf("failed to retrieve added workspace")
	}

	newWS.Name = "Updated WS"
	if err := db.UpdateWorkspace(newWS); err != nil {
		t.Fatalf("failed to update workspace: %v", err)
	}
	if ws, _ = db.GetWorkspace("ws-test"); ws.Name != "Updated WS" {
		t.Errorf("workspace update failed")
	}

	if err := db.DeleteWorkspace("ws-test"); err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}
	if _, exists = db.GetWorkspace("ws-test"); exists {
		t.Errorf("expected workspace to be deleted")
	}

	// Test deleting last workspace should fail
	if err := db.DeleteWorkspace(defaultWS.UUID); err == nil {
		t.Errorf("expected error deleting last workspace, got nil")
	}

	// 3. Task CRUD and Ledger test
	task := model.Task{
		UUID:           "task-1",
		WorkspaceUUID:  defaultWS.UUID,
		Title:          "Task 1",
		SchedulingType: model.Anchored,
	}

	if err := db.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	if !db.TaskExists("task-1") {
		t.Errorf("expected task-1 to exist")
	}

	// Verify CREATE ledger entry
	ledger := db.GetLedger()
	if len(ledger) != 1 || ledger[0].Op != "CREATE" || ledger[0].TaskUUID != "task-1" {
		t.Errorf("expected CREATE ledger entry, got %v", ledger)
	}

	// Update Task
	task.Title = "Task 1 Updated"
	if err := db.UpdateTask(task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
	if tk, _ := db.GetTask("task-1"); tk.Title != "Task 1 Updated" {
		t.Errorf("failed to update task in memory")
	}

	// Verify UPDATE ledger entry
	ledger = db.GetLedger()
	if len(ledger) != 2 || ledger[1].Op != "UPDATE" {
		t.Errorf("expected UPDATE ledger entry, got %v", ledger)
	}

	// Update Task No Ledger
	task.Title = "Task 1 No Ledger"
	if err := db.UpdateTaskNoLedger(task); err != nil {
		t.Fatalf("failed to update task no ledger: %v", err)
	}
	ledger = db.GetLedger()
	if len(ledger) != 2 {
		t.Errorf("expected no additional ledger entry, got %d", len(ledger))
	}

	// Remove ledger entry
	if err := db.RemoveLedgerEntry(ledger[0].ID); err != nil {
		t.Fatalf("failed to remove ledger entry: %v", err)
	}
	if len(db.GetLedger()) != 1 {
		t.Errorf("expected 1 ledger entry after removal")
	}

	if err := db.ClearLedger(); err != nil {
		t.Fatalf("failed to clear ledger: %v", err)
	}
	if len(db.GetLedger()) != 0 {
		t.Errorf("expected ledger to be empty")
	}

	// Delete Task
	if err := db.DeleteTask("task-1"); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}
	if db.TaskExists("task-1") {
		t.Errorf("expected task-1 to be deleted")
	}

	// Verify DELETE ledger entry
	ledger = db.GetLedger()
	if len(ledger) != 1 || ledger[0].Op != "DELETE" {
		t.Errorf("expected DELETE ledger entry, got %v", ledger)
	}
}

func TestDBEdgeCases(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	db, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// 1. Get/Update non-existent workspace/task
	_, exists := db.GetWorkspace("non-existent")
	if exists {
		t.Errorf("expected false for non-existent workspace")
	}
	err = db.UpdateWorkspace(model.Workspace{UUID: "non-existent"})
	if err == nil {
		t.Errorf("expected error updating non-existent workspace")
	}

	_, exists = db.GetTask("non-existent")
	if exists {
		t.Errorf("expected false for non-existent task")
	}
	err = db.UpdateTask(model.Task{UUID: "non-existent"})
	if err == nil {
		t.Errorf("expected error updating non-existent task")
	}
	err = db.DeleteTask("non-existent")
	if err == ErrTaskNotFound {
		// correct
	} else {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}

	// 2. Password Hash
	pass := "my-secret-password"
	hash := HashPassword(pass)
	if hash == "" {
		t.Errorf("expected hashed password")
	}

	// 3. Workspace deletion cleans up tasks
	wsUUID := "test-ws-deletion"
	_ = db.AddWorkspace(model.Workspace{UUID: wsUUID, Name: "Delete WS"})
	_ = db.AddTaskNoLedger(model.Task{UUID: "t-deleted", WorkspaceUUID: wsUUID, Title: "WS Deleted Task", SchedulingType: model.Anchored})

	if err := db.DeleteWorkspace(wsUUID); err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}

	if db.TaskExists("t-deleted") {
		t.Errorf("expected task to be cleaned up on workspace deletion")
	}
}

func TestDBLoadErrors(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	dbDir := filepath.Join(tempHome, ".config", "stream")
	_ = os.MkdirAll(dbDir, 0755)

	// 1. Invalid workspaces JSON
	_ = os.WriteFile(filepath.Join(dbDir, "workspaces.json"), []byte("{invalid-json}"), 0644)
	_, err := NewJSONDB()
	if err == nil {
		t.Error("expected error for invalid workspaces.json")
	}
	_ = os.Remove(filepath.Join(dbDir, "workspaces.json"))

	// 2. Invalid tasks JSON
	_ = os.WriteFile(filepath.Join(dbDir, "data.json"), []byte("{invalid-json}"), 0644)
	_, err = NewJSONDB()
	if err == nil {
		t.Error("expected error for invalid data.json")
	}
	_ = os.Remove(filepath.Join(dbDir, "data.json"))

	// 3. Invalid ledger JSON
	_ = os.WriteFile(filepath.Join(dbDir, "ledger.json"), []byte("{invalid-json}"), 0644)
	_, err = NewJSONDB()
	if err == nil {
		t.Error("expected error for invalid ledger.json")
	}
}

func TestDBLoadMigrations(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	dbDir := filepath.Join(tempHome, ".config", "stream")
	_ = os.MkdirAll(dbDir, 0755)

	// Write mock tasks JSON:
	// 1. Valid anchored task
	// 2. Task with empty WorkspaceUUID (should migrate to default WS)
	// 3. Stale task with empty Title (should be filtered out)
	// 4. Stale task with invalid scheduling type and not completed (should be filtered out)
	// 5. Completed task with empty/invalid scheduling type (should be retained)
	tasksJSON := `[
		{
			"uuid": "t-valid",
			"workspace_uuid": "ws-custom",
			"title": "Valid Task",
			"scheduling_type": "ANCHORED",
			"lifecycle_state": "SCHEDULED"
		},
		{
			"uuid": "t-no-workspace",
			"workspace_uuid": "",
			"title": "Migrated Workspace Task",
			"scheduling_type": "FLOATING",
			"lifecycle_state": "SCHEDULED"
		},
		{
			"uuid": "t-empty-title",
			"workspace_uuid": "ws-custom",
			"title": "",
			"scheduling_type": "FLOATING",
			"lifecycle_state": "SCHEDULED"
		},
		{
			"uuid": "t-invalid-sched",
			"workspace_uuid": "ws-custom",
			"title": "Invalid Sched",
			"scheduling_type": "INVALID_TYPE",
			"lifecycle_state": "SCHEDULED"
		},
		{
			"uuid": "t-completed-invalid-sched",
			"workspace_uuid": "ws-custom",
			"title": "Completed Invalid Sched",
			"scheduling_type": "INVALID_TYPE",
			"lifecycle_state": "COMPLETED"
		}
	]`
	_ = os.WriteFile(filepath.Join(dbDir, "data.json"), []byte(tasksJSON), 0644)

	// Write workspaces JSON to establish "ws-custom" and default workspace
	workspacesJSON := `[
		{"uuid": "ws-custom", "name": "Custom WS"}
	]`
	_ = os.WriteFile(filepath.Join(dbDir, "workspaces.json"), []byte(workspacesJSON), 0644)

	// Write user settings JSON with empty username and invalid timeout to test fallbacks
	settingsJSON := `{"username": "", "lock_timeout_minutes": 0}`
	_ = os.WriteFile(filepath.Join(dbDir, "settings.json"), []byte(settingsJSON), 0644)

	db, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to load db: %v", err)
	}

	// Verify t-valid exists
	if _, ok := db.GetTask("t-valid"); !ok {
		t.Errorf("expected t-valid to be loaded")
	}

	// Verify t-no-workspace exists and has custom workspace UUID ("ws-custom")
	tNoWS, ok := db.GetTask("t-no-workspace")
	if !ok {
		t.Errorf("expected t-no-workspace to be loaded")
	} else if tNoWS.WorkspaceUUID != "ws-custom" {
		t.Errorf("expected t-no-workspace WorkspaceUUID to migrate to default workspace 'ws-custom', got %s", tNoWS.WorkspaceUUID)
	}

	// Verify t-empty-title is filtered out
	if _, ok := db.GetTask("t-empty-title"); ok {
		t.Errorf("expected t-empty-title to be filtered out")
	}

	// Verify t-invalid-sched is filtered out
	if _, ok := db.GetTask("t-invalid-sched"); ok {
		t.Errorf("expected t-invalid-sched to be filtered out")
	}

	// Verify t-completed-invalid-sched is retained
	if _, ok := db.GetTask("t-completed-invalid-sched"); !ok {
		t.Errorf("expected completed task with invalid scheduling type to be retained")
	}

	// Verify user settings fallbacks
	settings := db.GetUserSettings()
	if settings.Username != "Doan Huu Quoc" {
		t.Errorf("expected username fallback to Doan Huu Quoc, got %s", settings.Username)
	}
	if settings.LockTimeoutMinutes != 5 {
		t.Errorf("expected lock timeout fallback to 5, got %d", settings.LockTimeoutMinutes)
	}
}

func TestDBLedgerTransitions(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	db, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// 1. Add Anchored task (should append CREATE)
	t1 := model.Task{
		UUID:           "task-1",
		Title:          "Task 1",
		SchedulingType: model.Anchored,
	}
	_ = db.AddTask(t1)
	ledger := db.GetLedger()
	if len(ledger) != 1 || ledger[0].Op != "CREATE" {
		t.Fatalf("expected CREATE ledger entry, got %v", ledger)
	}

	// 2. Change Anchored task to Floating (de-anchoring) -> should append DELETE
	t1.SchedulingType = model.Floating
	_ = db.UpdateTask(t1)
	ledger = db.GetLedger()
	if len(ledger) != 2 || ledger[1].Op != "DELETE" {
		t.Fatalf("expected DELETE ledger entry for de-anchoring, got %v", ledger)
	}

	// 3. Change Floating task to Anchored (anchoring) -> should append UPDATE
	t1.SchedulingType = model.Anchored
	_ = db.UpdateTask(t1)
	ledger = db.GetLedger()
	if len(ledger) != 3 || ledger[2].Op != "UPDATE" {
		t.Fatalf("expected UPDATE ledger entry for anchoring, got %v", ledger)
	}

	// 4. Update title of Anchored task -> should append UPDATE
	t1.Title = "Updated Title"
	_ = db.UpdateTask(t1)
	ledger = db.GetLedger()
	if len(ledger) != 4 || ledger[3].Op != "UPDATE" {
		t.Fatalf("expected UPDATE ledger entry for title change, got %v", ledger)
	}

	// 5. Update title of Floating task -> should NOT append to ledger
	t2 := model.Task{
		UUID:           "task-2",
		Title:          "Floating Task",
		SchedulingType: model.Floating,
	}
	_ = db.AddTask(t2) // should not append to ledger
	ledger1 := db.GetLedger()

	t2.Title = "Floating Task Changed"
	_ = db.UpdateTask(t2) // should not append to ledger
	ledger2 := db.GetLedger()

	if len(ledger1) != len(ledger2) {
		t.Errorf("expected no additional ledger entry for floating task changes")
	}
}

func TestDBSaveErrors(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	db, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// Add an anchored task first to generate a ledger entry
	err = db.AddTask(model.Task{UUID: "t-ledger-err", Title: "Ledger Err Task", SchedulingType: model.Anchored})
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}
	ledger := db.GetLedger()
	if len(ledger) == 0 {
		t.Fatalf("expected ledger entries")
	}
	entryID := ledger[0].ID

	// Now make database directory read-only/unwritable to trigger save errors
	dbDir := filepath.Join(tempHome, ".config", "stream")
	_ = os.Chmod(dbDir, 0000)
	defer os.Chmod(dbDir, 0755) // clean up

	// 1. saveTasks error
	err = db.AddTask(model.Task{UUID: "t-error", Title: "Error task", SchedulingType: model.Anchored})
	if err == nil {
		t.Error("expected write error when adding task to unwritable directory")
	}

	// 2. saveWorkspaces error
	err = db.UpdateWorkspace(model.Workspace{UUID: "ws-error", Name: "Error workspace"})
	if err == nil {
		t.Error("expected write error when updating workspace to unwritable directory")
	}

	// 3. saveSettings error
	settings := db.GetUserSettings()
	settings.Username = "Another name"
	err = db.UpdateUserSettings(settings)
	if err == nil {
		t.Error("expected write error when updating settings to unwritable directory")
	}

	// 4. saveLedger error (RemoveLedgerEntry)
	err = db.RemoveLedgerEntry(entryID)
	if err == nil {
		t.Error("expected write error when removing ledger entry in unwritable directory")
	}

	// 5. saveLedger error (ClearLedger)
	err = db.ClearLedger()
	if err == nil {
		t.Error("expected write error when clearing ledger in unwritable directory")
	}
}
