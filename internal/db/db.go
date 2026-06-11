package db

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"stream/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

type LedgerEntry struct {
	ID        string     `json:"id"`
	Op        string     `json:"op"` // "CREATE", "UPDATE", "DELETE"
	TaskUUID  string     `json:"task_uuid"`
	Task      model.Task `json:"task"`
	Timestamp time.Time  `json:"timestamp"`
}

type JSONDB struct {
	mu             sync.RWMutex
	configDir      string
	dataPath       string
	ledgerPath     string
	workspacesPath string
	settingsPath   string
	tasks          map[string]model.Task
	workspaces     map[string]model.Workspace
	ledger         []LedgerEntry
	userSettings   model.UserSettings
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
		configDir:      configDir,
		dataPath:       filepath.Join(configDir, "data.json"),
		ledgerPath:     filepath.Join(configDir, "ledger.json"),
		workspacesPath: filepath.Join(configDir, "workspaces.json"),
		settingsPath:   filepath.Join(configDir, "settings.json"),
		tasks:          make(map[string]model.Task),
		workspaces:     make(map[string]model.Workspace),
		ledger:         []LedgerEntry{},
	}

	if err := db.load(); err != nil {
		return nil, err
	}

	if err := db.saveWorkspaces(); err != nil {
		return nil, err
	}
	if err := db.saveTasks(); err != nil {
		return nil, err
	}
	if err := db.saveSettings(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *JSONDB) saveSettings() error {
	data, err := json.MarshalIndent(db.userSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal settings: %w", err)
	}
	if err := os.WriteFile(db.settingsPath, data, 0644); err != nil {
		return fmt.Errorf("could not write settings file: %w", err)
	}
	return nil
}

func (db *JSONDB) GetUserSettings() model.UserSettings {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.userSettings
}

func (db *JSONDB) UpdateUserSettings(s model.UserSettings) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.userSettings = s.NormalizedGCalSync()
	return db.saveSettings()
}

func (db *JSONDB) GetConfigDir() string {
	return db.configDir
}

func HashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
