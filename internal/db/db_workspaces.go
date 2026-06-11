package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

func (db *JSONDB) saveWorkspaces() error {
	var list []model.Workspace
	for _, ws := range db.workspaces {
		list = append(list, ws)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal workspaces: %w", err)
	}
	if err := os.WriteFile(db.workspacesPath, data, 0644); err != nil {
		return fmt.Errorf("could not write workspaces file: %w", err)
	}
	return nil
}

func (db *JSONDB) GetWorkspaces() []model.Workspace {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var list []model.Workspace
	for _, ws := range db.workspaces {
		list = append(list, ws)
	}
	return list
}

func (db *JSONDB) GetWorkspace(uuid string) (model.Workspace, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	ws, exists := db.workspaces[uuid]
	return ws, exists
}

func (db *JSONDB) AddWorkspace(ws model.Workspace) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if ws.UUID == "" {
		ws.UUID = uuid.New().String()
	}
	now := time.Now()
	ws.CreatedAt = now
	ws.UpdatedAt = now

	db.workspaces[ws.UUID] = ws
	return db.saveWorkspaces()
}

func (db *JSONDB) UpdateWorkspace(ws model.Workspace) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.workspaces[ws.UUID]; !exists {
		return errors.New("workspace not found")
	}
	ws.UpdatedAt = time.Now()
	db.workspaces[ws.UUID] = ws
	return db.saveWorkspaces()
}

func (db *JSONDB) DeleteWorkspace(wsUUID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(db.workspaces) <= 1 {
		return errors.New("cannot delete the last workspace")
	}

	if _, exists := db.workspaces[wsUUID]; !exists {
		return errors.New("workspace not found")
	}

	// Remove workspace
	delete(db.workspaces, wsUUID)
	if err := db.saveWorkspaces(); err != nil {
		return err
	}

	// Clean up tasks in deleted workspace and queue remote deletes
	for u, t := range db.tasks {
		if t.WorkspaceUUID == wsUUID {
			if model.IsGCalSyncable(t) {
				if err := db.appendLedgerLocked("DELETE", u, t); err != nil {
					return err
				}
			}
			delete(db.tasks, u)
		}
	}
	return db.saveTasks()
}
