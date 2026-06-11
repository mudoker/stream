package db

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

func (db *JSONDB) saveTasks() error {
	var list []model.Task
	for _, t := range db.tasks {
		list = append(list, t)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal tasks: %w", err)
	}
	if err := os.WriteFile(db.dataPath, data, 0644); err != nil {
		return fmt.Errorf("could not write data file: %w", err)
	}
	return nil
}

func (db *JSONDB) GetTasks() []model.Task {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var list []model.Task
	for _, t := range db.tasks {
		list = append(list, t)
	}
	return list
}

func (db *JSONDB) GetTask(uuid string) (model.Task, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	t, exists := db.tasks[uuid]
	return t, exists
}

func (db *JSONDB) AddTask(t model.Task) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if t.UUID == "" {
		t.UUID = uuid.New().String()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	db.tasks[t.UUID] = t
	if err := db.saveTasks(); err != nil {
		return err
	}

	if model.IsGCalSyncable(t) {
		return db.appendLedgerLocked("CREATE", t.UUID, t)
	}
	return nil
}

// AddTaskNoLedger persists a task without recording a sync ledger entry.
func (db *JSONDB) AddTaskNoLedger(t model.Task) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if t.UUID == "" {
		t.UUID = uuid.New().String()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	db.tasks[t.UUID] = t
	return db.saveTasks()
}

func (db *JSONDB) UpdateTask(t model.Task) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.updateTaskLocked(t, true)
}

// UpdateTaskNoLedger persists task changes without appending to the sync ledger.
func (db *JSONDB) UpdateTaskNoLedger(t model.Task) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.updateTaskLocked(t, false)
}

func (db *JSONDB) updateTaskLocked(t model.Task, recordLedger bool) error {
	prev, exists := db.tasks[t.UUID]
	if !exists {
		return ErrTaskNotFound
	}
	t.UpdatedAt = time.Now()
	db.tasks[t.UUID] = t
	if err := db.saveTasks(); err != nil {
		return err
	}

	if !recordLedger {
		return nil
	}
	return db.recordTaskChangeLocked(prev, t)
}

func (db *JSONDB) TaskExists(taskUUID string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, exists := db.tasks[taskUUID]
	return exists
}

func (db *JSONDB) DeleteTask(taskUUID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	task, exists := db.tasks[taskUUID]
	if !exists {
		return ErrTaskNotFound
	}

	delete(db.tasks, taskUUID)
	if err := db.saveTasks(); err != nil {
		return err
	}

	if model.IsGCalSyncable(task) {
		return db.appendLedgerLocked("DELETE", taskUUID, task)
	}
	return nil
}
