package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

type LedgerEntry struct {
	ID        string     `json:"id"`
	Op        string     `json:"op"` // "CREATE", "UPDATE", "DELETE"
	TaskUUID  string     `json:"task_uuid"`
	Task      model.Task `json:"task"`
	Timestamp time.Time  `json:"timestamp"`
}

type JSONDB struct {
	mu         sync.RWMutex
	configDir  string
	dataPath   string
	ledgerPath string
	tasks      map[string]model.Task
	ledger     []LedgerEntry
}

func NewJSONDB() (*JSONDB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get home dir: %w", err)
	}
	configDir := filepath.Join(home, ".config", "stream")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create config dir: %w", err)
	}

	db := &JSONDB{
		configDir:  configDir,
		dataPath:   filepath.Join(configDir, "data.json"),
		ledgerPath: filepath.Join(configDir, "ledger.json"),
		tasks:      make(map[string]model.Task),
		ledger:     []LedgerEntry{},
	}

	if err := db.load(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *JSONDB) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Load Tasks
	if _, err := os.Stat(db.dataPath); err == nil {
		data, err := os.ReadFile(db.dataPath)
		if err != nil {
			return fmt.Errorf("could not read data file: %w", err)
		}
		var list []model.Task
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("could not unmarshal tasks: %w", err)
		}
		for _, t := range list {
			db.tasks[t.UUID] = t
		}
	}

	// Load Ledger
	if _, err := os.Stat(db.ledgerPath); err == nil {
		data, err := os.ReadFile(db.ledgerPath)
		if err != nil {
			return fmt.Errorf("could not read ledger file: %w", err)
		}
		if err := json.Unmarshal(data, &db.ledger); err != nil {
			return fmt.Errorf("could not unmarshal ledger: %w", err)
		}
	}

	return nil
}

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

func (db *JSONDB) saveLedger() error {
	data, err := json.MarshalIndent(db.ledger, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal ledger: %w", err)
	}
	if err := os.WriteFile(db.ledgerPath, data, 0644); err != nil {
		return fmt.Errorf("could not write ledger file: %w", err)
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

	// Record in ledger
	db.ledger = append(db.ledger, LedgerEntry{
		ID:        uuid.New().String(),
		Op:        "CREATE",
		TaskUUID:  t.UUID,
		Task:      t,
		Timestamp: now,
	})
	return db.saveLedger()
}

func (db *JSONDB) UpdateTask(t model.Task) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.tasks[t.UUID]; !exists {
		return errors.New("task not found")
	}
	t.UpdatedAt = time.Now()
	db.tasks[t.UUID] = t
	if err := db.saveTasks(); err != nil {
		return err
	}

	// Record in ledger
	db.ledger = append(db.ledger, LedgerEntry{
		ID:        uuid.New().String(),
		Op:        "UPDATE",
		TaskUUID:  t.UUID,
		Task:      t,
		Timestamp: t.UpdatedAt,
	})
	return db.saveLedger()
}

func (db *JSONDB) DeleteTask(taskUUID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	task, exists := db.tasks[taskUUID]
	if !exists {
		return errors.New("task not found")
	}

	delete(db.tasks, taskUUID)
	if err := db.saveTasks(); err != nil {
		return err
	}

	// Record in ledger
	db.ledger = append(db.ledger, LedgerEntry{
		ID:        uuid.New().String(),
		Op:        "DELETE",
		TaskUUID:  taskUUID,
		Task:      task,
		Timestamp: time.Now(),
	})
	return db.saveLedger()
}

// GetLedger returns a copy of the ledger entries
func (db *JSONDB) GetLedger() []LedgerEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	list := make([]LedgerEntry, len(db.ledger))
	copy(list, db.ledger)
	return list
}

// ClearLedger clears all entries from the transaction ledger
func (db *JSONDB) ClearLedger() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ledger = []LedgerEntry{}
	return db.saveLedger()
}

func (db *JSONDB) GetConfigDir() string {
	return db.configDir
}
