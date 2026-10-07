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

	"github.com/google/uuid"
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
	sprintsPath    string
	settingsPath   string
	tasks          map[string]model.Task
	workspaces     map[string]model.Workspace
	sprints        map[string]model.Sprint
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
		sprintsPath:    filepath.Join(configDir, "sprints.json"),
		settingsPath:   filepath.Join(configDir, "settings.json"),
		tasks:          make(map[string]model.Task),
		workspaces:     make(map[string]model.Workspace),
		sprints:        make(map[string]model.Sprint),
		ledger:         []LedgerEntry{},
	}

	if err := db.load(); err != nil {
		return nil, err
	}

	if err := db.saveWorkspaces(); err != nil {
		return nil, err
	}
	if err := db.saveSprints(); err != nil {
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

func (db *JSONDB) GetTags() []model.TagInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.userSettings.Tags
}

func (db *JSONDB) SaveTags(tags []model.TagInfo) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.userSettings.Tags = tags
	return db.saveSettings()
}

func (db *JSONDB) FactoryReset() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Clear in-memory state
	db.tasks = make(map[string]model.Task)
	db.workspaces = make(map[string]model.Workspace)
	db.sprints = make(map[string]model.Sprint)
	db.ledger = []LedgerEntry{}
	db.userSettings = model.UserSettings{}

	// Remove physical files
	_ = os.Remove(db.dataPath)
	_ = os.Remove(db.ledgerPath)
	_ = os.Remove(db.workspacesPath)
	_ = os.Remove(db.sprintsPath)
	_ = os.Remove(db.settingsPath)

	// Recreate default workspace
	defaultWS := model.Workspace{
		UUID:      uuid.New().String(),
		Name:      "Aether Workspace",
		Icon:      "🚀",
		Badge:     "[Dev]",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.workspaces[defaultWS.UUID] = defaultWS

	// Recreate default sprint
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 14)
	defaultSprint := model.Sprint{
		UUID:          uuid.New().String(),
		WorkspaceUUID: defaultWS.UUID,
		Name:          "Sprint 1",
		StartDate:     start,
		EndDate:       end,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	db.sprints[defaultSprint.UUID] = defaultSprint

	if err := db.saveWorkspaces(); err != nil {
		return err
	}
	if err := db.saveSprints(); err != nil {
		return err
	}
	if err := db.saveTasks(); err != nil {
		return err
	}
	if err := db.saveLedger(); err != nil {
		return err
	}
	if err := db.saveSettings(); err != nil {
		return err
	}
	return nil
}

func (db *JSONDB) GetConfigDir() string {
	return db.configDir
}

func HashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
