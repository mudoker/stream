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

var ErrSprintOverlap = errors.New("sprint dates cannot overlap with an existing sprint")

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

	if s.StartDate.IsZero() {
		s.StartDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	if s.EndDate.IsZero() || s.EndDate.Before(s.StartDate) {
		s.EndDate = s.StartDate.AddDate(0, 0, 13)
	}

	for _, existing := range db.sprints {
		if existing.UUID == s.UUID {
			continue
		}
		if model.SprintsOverlap(s, existing) {
			return fmt.Errorf("%w: sprint dates (%s to %s) overlap with existing sprint '%s' (%s to %s)",
				ErrSprintOverlap,
				s.StartDate.Format("2006-01-02"), s.EndDate.Format("2006-01-02"),
				existing.Name,
				existing.StartDate.Format("2006-01-02"), existing.EndDate.Format("2006-01-02"))
		}
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

	now := time.Now()
	if s.StartDate.IsZero() {
		s.StartDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	if s.EndDate.IsZero() || s.EndDate.Before(s.StartDate) {
		s.EndDate = s.StartDate.AddDate(0, 0, 13)
	}

	for _, existing := range db.sprints {
		if existing.UUID == s.UUID {
			continue
		}
		if model.SprintsOverlap(s, existing) {
			return fmt.Errorf("%w: sprint dates (%s to %s) overlap with existing sprint '%s' (%s to %s)",
				ErrSprintOverlap,
				s.StartDate.Format("2006-01-02"), s.EndDate.Format("2006-01-02"),
				existing.Name,
				existing.StartDate.Format("2006-01-02"), existing.EndDate.Format("2006-01-02"))
		}
	}

	s.UpdatedAt = now
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

	var toAdd []model.Sprint
	currStart := baseSprint.EndDate.AddDate(0, 0, 1)
	for i := 1; i <= count; i++ {
		currEnd := currStart.AddDate(0, 0, gapDays)
		sprintNum := len(db.sprints) + i
		s := model.Sprint{
			UUID:          uuid.New().String(),
			WorkspaceUUID: baseSprint.WorkspaceUUID,
			Name:          fmt.Sprintf("Sprint %d", sprintNum),
			StartDate:     currStart,
			EndDate:       currEnd,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		for _, existing := range db.sprints {
			if model.SprintsOverlap(s, existing) {
				return nil, fmt.Errorf("%w: generated sprint '%s' (%s to %s) overlaps with existing sprint '%s' (%s to %s)",
					ErrSprintOverlap,
					s.Name, s.StartDate.Format("2006-01-02"), s.EndDate.Format("2006-01-02"),
					existing.Name, existing.StartDate.Format("2006-01-02"), existing.EndDate.Format("2006-01-02"))
			}
		}
		for _, prevGen := range toAdd {
			if model.SprintsOverlap(s, prevGen) {
				return nil, fmt.Errorf("%w: generated sprint '%s' overlaps with generated sprint '%s'",
					ErrSprintOverlap, s.Name, prevGen.Name)
			}
		}

		toAdd = append(toAdd, s)
		currStart = currEnd.AddDate(0, 0, 1)
	}

	for _, s := range toAdd {
		db.sprints[s.UUID] = s
	}

	if err := db.saveSprints(); err != nil {
		return nil, err
	}
	return toAdd, nil
}
