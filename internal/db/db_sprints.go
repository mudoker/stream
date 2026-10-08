package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

func (db *JSONDB) saveSprints() error {
	var list []model.Sprint
	for _, s := range db.sprints {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].StartDate.Before(list[j].StartDate)
	})
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal sprints: %w", err)
	}
	if err := os.WriteFile(db.sprintsPath, data, 0644); err != nil {
		return fmt.Errorf("could not write sprints file: %w", err)
	}
	return nil
}

func (db *JSONDB) GetSprints() []model.Sprint {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var list []model.Sprint
	for _, s := range db.sprints {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].StartDate.Before(list[j].StartDate)
	})
	return list
}

func (db *JSONDB) GetSprint(uuid string) (model.Sprint, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	s, exists := db.sprints[uuid]
	return s, exists
}

func (db *JSONDB) AddSprint(s model.Sprint) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if s.UUID == "" {
		s.UUID = uuid.New().String()
	}
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	if s.EndDate.IsZero() || s.EndDate.Before(s.StartDate) {
		s.EndDate = s.StartDate.AddDate(0, 0, 13)
	}

	db.sprints[s.UUID] = s
	return db.saveSprints()
}

func (db *JSONDB) UpdateSprint(s model.Sprint) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.sprints[s.UUID]; !exists {
		return errors.New("sprint not found")
	}
	s.UpdatedAt = time.Now()
	db.sprints[s.UUID] = s
	return db.saveSprints()
}

func (db *JSONDB) DeleteSprint(sprintUUID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.sprints[sprintUUID]; !exists {
		return errors.New("sprint not found")
	}

	delete(db.sprints, sprintUUID)

	// Unlink tasks from deleted sprint
	for u, t := range db.tasks {
		if t.SprintUUID == sprintUUID {
			t.SprintUUID = ""
			t.UpdatedAt = time.Now()
			db.tasks[u] = t
		}
	}

	if err := db.saveSprints(); err != nil {
		return err
	}
	return db.saveTasks()
}

// GenerateRecurringSprints generates `count` recurring sprints following `baseSprint`.
func (db *JSONDB) GenerateRecurringSprints(baseSprint model.Sprint, count int) ([]model.Sprint, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	gapDays := int(baseSprint.EndDate.Sub(baseSprint.StartDate).Hours() / 24)
	if gapDays <= 0 {
		gapDays = 13
	}

	var generated []model.Sprint
	currStart := baseSprint.EndDate.AddDate(0, 0, 1)
	for i := 1; i <= count; i++ {
		currEnd := currStart.AddDate(0, 0, gapDays)
		sprintNum := len(db.sprints) + 1
		s := model.Sprint{
			UUID:          uuid.New().String(),
			WorkspaceUUID: baseSprint.WorkspaceUUID,
			Name:          fmt.Sprintf("Sprint %d", sprintNum),
			StartDate:     currStart,
			EndDate:       currEnd,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		db.sprints[s.UUID] = s
		generated = append(generated, s)
		currStart = currEnd.AddDate(0, 0, 1)
	}

	if err := db.saveSprints(); err != nil {
		return nil, err
	}
	return generated, nil
}
