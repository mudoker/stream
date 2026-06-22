package db

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

func (db *JSONDB) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Load Workspaces
	if _, err := os.Stat(db.workspacesPath); err == nil {
		data, err := os.ReadFile(db.workspacesPath)
		if err != nil {
			return fmt.Errorf("could not read workspaces file: %w", err)
		}
		var list []model.Workspace
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("could not unmarshal workspaces: %w", err)
		}
		for _, ws := range list {
			db.workspaces[ws.UUID] = ws
		}
	}

	// Initialize default workspace if none exist
	if len(db.workspaces) == 0 {
		defaultWS := model.Workspace{
			UUID:      uuid.New().String(),
			Name:      "Aether Workspace",
			Icon:      "🚀",
			Badge:     "[Dev]",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.workspaces[defaultWS.UUID] = defaultWS
	}

	// Get first/default workspace UUID
	var defaultWSUUID string
	for u := range db.workspaces {
		defaultWSUUID = u
		break
	}

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
			if t.WorkspaceUUID == "" {
				t.WorkspaceUUID = defaultWSUUID
			}
			// Remove/skip stale tasks that are invalid or not showed on screen (except completed ones)
			if t.Title == "" || (t.LifecycleState != model.StateCompleted &&
				t.SchedulingType != model.Anchored &&
				t.SchedulingType != model.Floating &&
				t.SchedulingType != model.Reminder &&
				t.SchedulingType != model.Habit &&
				t.SchedulingType != model.Event) {
				continue
			}
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

	// Load User Settings
	db.userSettings = model.UserSettings{
		Username:                "Doan Huu Quoc",
		PasswordHash:            "",
		LockTimeoutMinutes:      5,
		GCalSyncMode:            model.GCalSyncPush,
		GCalSyncIntervalSeconds: 5,
	}
	if _, err := os.Stat(db.settingsPath); err == nil {
		data, err := os.ReadFile(db.settingsPath)
		if err == nil {
			var s model.UserSettings
			if err := json.Unmarshal(data, &s); err == nil {
				if s.Username == "" {
					s.Username = "Doan Huu Quoc"
				}
				if s.LockTimeoutMinutes <= 0 {
					s.LockTimeoutMinutes = 5
				}
				db.userSettings = s.NormalizedGCalSync()
			}
		}
	}

	if len(db.userSettings.Tags) == 0 {
		db.userSettings.Tags = []model.TagInfo{
			{Name: "work", Frequency: 5},
			{Name: "personal", Frequency: 3},
			{Name: "learning", Frequency: 2},
			{Name: "health", Frequency: 1},
			{Name: "admin", Frequency: 1},
		}
	}

	return nil
}
