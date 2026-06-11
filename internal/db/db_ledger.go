package db

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

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

func (db *JSONDB) GetLedger() []LedgerEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	list := make([]LedgerEntry, len(db.ledger))
	copy(list, db.ledger)
	return list
}

func (db *JSONDB) ClearLedger() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ledger = []LedgerEntry{}
	return db.saveLedger()
}

func (db *JSONDB) RemoveLedgerEntry(entryID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	filtered := db.ledger[:0]
	for _, entry := range db.ledger {
		if entry.ID != entryID {
			filtered = append(filtered, entry)
		}
	}
	db.ledger = filtered
	return db.saveLedger()
}

func (db *JSONDB) appendLedgerLocked(op, taskUUID string, task model.Task) error {
	db.ledger = append(db.ledger, LedgerEntry{
		ID:        uuid.New().String(),
		Op:        op,
		TaskUUID:  taskUUID,
		Task:      task,
		Timestamp: time.Now(),
	})
	return db.saveLedger()
}

func calendarRelevantChange(prev, next model.Task) bool {
	if prev.SchedulingType != next.SchedulingType {
		return true
	}
	if !model.IsGCalSyncable(next) {
		return false
	}
	return prev.Title != next.Title ||
		prev.Description != next.Description ||
		!prev.TimeWindow.Start.Equal(next.TimeWindow.Start) ||
		!prev.TimeWindow.End.Equal(next.TimeWindow.End)
}

func (db *JSONDB) recordTaskChangeLocked(prev, next model.Task) error {
	wasAnchored := model.IsGCalSyncable(prev)
	isAnchored := model.IsGCalSyncable(next)

	switch {
	case isAnchored && calendarRelevantChange(prev, next):
		return db.appendLedgerLocked("UPDATE", next.UUID, next)
	case wasAnchored && !isAnchored:
		return db.appendLedgerLocked("DELETE", prev.UUID, prev)
	default:
		return nil
	}
}
