package sync

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

type CompactedOp struct {
	Op             string // "CREATE", "UPDATE", "DELETE", "NOOP"
	TaskUUID       string
	Task           model.Task
	SourceEntryIDs []string
}

// compactLedger coalesces sequential operations on the same task into a minimal,
// idempotent batch to eliminate redundant network calls and prevent stale updates.
func compactLedger(entries []db.LedgerEntry) []CompactedOp {
	if len(entries) == 0 {
		return nil
	}

	var result []CompactedOp
	taskIndex := make(map[string]int) // taskUUID -> index in result

	for _, entry := range entries {
		idx, exists := taskIndex[entry.TaskUUID]
		if !exists {
			op := CompactedOp{
				Op:             entry.Op,
				TaskUUID:       entry.TaskUUID,
				Task:           entry.Task,
				SourceEntryIDs: []string{entry.ID},
			}
			taskIndex[entry.TaskUUID] = len(result)
			result = append(result, op)
			continue
		}

		curr := &result[idx]
		curr.SourceEntryIDs = append(curr.SourceEntryIDs, entry.ID)

		switch curr.Op {
		case "CREATE":
			switch entry.Op {
			case "UPDATE":
				curr.Task = entry.Task
			case "DELETE":
				if curr.Task.GCalMetadata.EventID != "" || entry.Task.GCalMetadata.EventID != "" {
					curr.Op = "DELETE"
					if entry.Task.GCalMetadata.EventID != "" {
						curr.Task = entry.Task
					}
				} else {
					curr.Op = "NOOP"
				}
			case "CREATE":
				curr.Task = entry.Task
			}

		case "UPDATE":
			switch entry.Op {
			case "UPDATE":
				curr.Task = entry.Task
			case "DELETE":
				curr.Op = "DELETE"
				curr.Task = entry.Task
			case "CREATE":
				curr.Task = entry.Task
			}

		case "DELETE":
			switch entry.Op {
			case "CREATE", "UPDATE":
				if curr.Task.GCalMetadata.EventID != "" || entry.Task.GCalMetadata.EventID != "" {
					curr.Op = "UPDATE"
					if entry.Task.GCalMetadata.EventID != "" {
						curr.Task = entry.Task
					} else {
						curr.Task.Title = entry.Task.Title
						curr.Task.Description = entry.Task.Description
						curr.Task.Location = entry.Task.Location
						curr.Task.TimeWindow = entry.Task.TimeWindow
						curr.Task.Priority = entry.Task.Priority
						curr.Task.StoryPoints = entry.Task.StoryPoints
						curr.Task.LifecycleState = entry.Task.LifecycleState
						curr.Task.SchedulingType = entry.Task.SchedulingType
						curr.Task.IsAllDay = entry.Task.IsAllDay
					}
				} else {
					curr.Op = "CREATE"
					curr.Task = entry.Task
				}
			case "DELETE":
				curr.Task = entry.Task
			}

		case "NOOP":
			curr.Op = entry.Op
			curr.Task = entry.Task
		}
	}

	return result
}

func (s *SyncEngine) ensureService() (*calendar.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		return s.srv, nil
	}

	if err := s.initOAuth(); err != nil {
		s.isOnline = false
		return nil, err
	}
	return s.srv, nil
}

func (s *SyncEngine) sync(includePull bool) {
	if s.isRateLimited() {
		return
	}

	mode := s.getSyncMode()
	if mode == model.GCalSyncNone && !includePull {
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		if includePull {
			s.logCallback("GCal Sync: credentials unavailable, staying in local mode.")
		}
		return
	}

	if mode != model.GCalSyncNone {
		s.logCallback("Sync Engine: Checking connection...")
		_, err = srv.Calendars.Get("primary").Do()
		if err != nil {
			if isRateLimitError(err) {
				s.handleRateLimit(err)
				return
			}
			s.setOnline(false)
			s.logCallback("Sync Engine: Offline. Anchored changes queued locally.")
			return
		}
		s.setOnline(true)

		ledger := s.localDB.GetLedger()
		if len(ledger) > 0 {
			compacted := compactLedger(ledger)
			s.logCallback(fmt.Sprintf("Sync Engine: Replaying %d operations (compacted from %d)...", len(compacted), len(ledger)))

			replayed, err := s.replayCompactedOps(srv, compacted)
			if err != nil {
				if isRateLimitError(err) {
					s.handleRateLimit(err)
					return
				}
				s.logCallback(fmt.Sprintf("Sync Ledger Replay Error: %s", formatSyncError(err)))
				return
			}
			if replayed > 0 {
				s.logCallback(fmt.Sprintf("Sync Engine: Processed %d ledger operations.", replayed))
			}
		}
	}

	if includePull {
		s.logCallback("Sync Engine: Pulling remote updates (manual sync)...")
		if err := s.pullRemoteUpdates(srv); err != nil {
			if isRateLimitError(err) {
				s.handleRateLimit(err)
				return
			}
			s.logCallback(fmt.Sprintf("Sync Engine Pull Error: %s", formatSyncError(err)))
		} else {
			s.logCallback("Sync Engine: Remote pull complete.")
		}
	}

	s.resetRateLimitBackoff()
}

func (s *SyncEngine) replayCompactedOps(srv *calendar.Service, ops []CompactedOp) (int, error) {
	replayed := 0

	// 1. Purge NOOP operations immediately in a single batch
	var noopIDs []string
	for _, op := range ops {
		if op.Op == "NOOP" {
			noopIDs = append(noopIDs, op.SourceEntryIDs...)
			replayed += len(op.SourceEntryIDs)
		}
	}
	if len(noopIDs) > 0 {
		_ = s.localDB.RemoveLedgerEntries(noopIDs)
	}

	// 2. Replay active operations
	for _, op := range ops {
		if op.Op == "NOOP" {
			continue
		}

		if s.isRateLimited() {
			return replayed, &googleapi.Error{Code: 429, Message: "rate limit backoff active"}
		}

		var opErr error
		switch op.Op {
		case "CREATE":
			task, exists := s.localDB.GetTask(op.TaskUUID)
			if !exists {
				s.logCallback(fmt.Sprintf("Sync: skipping stale CREATE for deleted task %s.", op.TaskUUID))
				_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
				replayed += len(op.SourceEntryIDs)
				continue
			}
			if !model.IsGCalSyncable(task) {
				_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
				replayed += len(op.SourceEntryIDs)
				continue
			}
			if task.GCalMetadata.EventID != "" {
				opErr = s.updateRemoteEvent(srv, task)
			} else {
				opErr = s.createRemoteEvent(srv, task)
			}

		case "UPDATE":
			task, exists := s.localDB.GetTask(op.TaskUUID)
			if !exists {
				if op.Task.GCalMetadata.EventID != "" {
					s.logCallback(fmt.Sprintf("Sync: task %s deleted locally, removing remote event.", op.TaskUUID))
					if delErr := s.deleteRemoteEvent(srv, op.Task); delErr != nil && !s.isSkippableError(delErr) {
						s.logCallback(fmt.Sprintf("Sync: remote cleanup failed for %s: %s", op.TaskUUID, formatSyncError(delErr)))
						return replayed, delErr
					}
				}
				_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
				replayed += len(op.SourceEntryIDs)
				continue
			}
			if !model.IsGCalSyncable(task) {
				if task.GCalMetadata.EventID != "" {
					_ = s.deleteRemoteEvent(srv, task)
					task.GCalMetadata = model.GCalMetadata{}
					_ = s.localDB.UpdateTaskNoLedger(task)
				}
				_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
				replayed += len(op.SourceEntryIDs)
				continue
			}
			opErr = s.updateRemoteEvent(srv, task)

		case "DELETE":
			opErr = s.deleteRemoteEvent(srv, op.Task)
		}

		if opErr != nil {
			if isRateLimitError(opErr) {
				return replayed, opErr
			}
			if s.isSkippableError(opErr) {
				s.logCallback(fmt.Sprintf("Sync: skipping non-syncable operation for %s: %s", op.TaskUUID, formatSyncError(opErr)))
				_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
				replayed += len(op.SourceEntryIDs)
				continue
			}
			return replayed, opErr
		}

		_ = s.localDB.RemoveLedgerEntries(op.SourceEntryIDs)
		replayed += len(op.SourceEntryIDs)
		s.throttle()
	}

	return replayed, nil
}

func (s *SyncEngine) isSkippableError(err error) bool {
	if err == nil {
		return false
	}
	if err.Error() == "non-anchored task" {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 400 || apiErr.Code == 404 || apiErr.Code == 410 {
			return true
		}
	}
	return false
}

func (s *SyncEngine) replayEntry(srv *calendar.Service, entry db.LedgerEntry) error {
	if entry.Op != "DELETE" && !model.IsGCalSyncable(entry.Task) {
		return fmt.Errorf("non-anchored task")
	}

	switch entry.Op {
	case "CREATE":
		if !s.localDB.TaskExists(entry.TaskUUID) {
			return db.ErrTaskNotFound
		}
		return s.createRemoteEvent(srv, entry.Task)
	case "UPDATE":
		if !s.localDB.TaskExists(entry.TaskUUID) {
			return db.ErrTaskNotFound
		}
		return s.updateRemoteEvent(srv, entry.Task)
	case "DELETE":
		return s.deleteRemoteEvent(srv, entry.Task)
	}
	return nil
}

func (s *SyncEngine) handleSkippableLedgerEntry(entry db.LedgerEntry, err error) bool {
	if err == nil {
		return false
	}
	if err.Error() == "non-anchored task" {
		s.logCallback(fmt.Sprintf("Sync: skipping non-anchored ledger entry for %s.", entry.TaskUUID))
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 400 || apiErr.Code == 404 || apiErr.Code == 410 {
			s.logCallback(fmt.Sprintf("Sync: skipping non-syncable entry for %s: %s", entry.TaskUUID, apiErr.Message))
			return true
		}
	}
	return false
}

func (s *SyncEngine) handleStaleLedgerEntry(srv *calendar.Service, entry db.LedgerEntry, err error) bool {
	if !errors.Is(err, db.ErrTaskNotFound) {
		return false
	}

	switch entry.Op {
	case "CREATE":
		s.logCallback(fmt.Sprintf("Sync: skipping stale CREATE for deleted task %s.", entry.TaskUUID))
		return true
	case "UPDATE":
		if entry.Task.GCalMetadata.EventID != "" {
			s.logCallback(fmt.Sprintf("Sync: task %s deleted locally, removing remote event.", entry.TaskUUID))
			if delErr := s.deleteRemoteEvent(srv, entry.Task); delErr != nil {
				s.logCallback(fmt.Sprintf("Sync: remote cleanup failed for %s: %s", entry.TaskUUID, formatSyncError(delErr)))
				return false
			}
			return true
		}
		s.logCallback(fmt.Sprintf("Sync: skipping stale UPDATE for deleted task %s.", entry.TaskUUID))
		return true
	default:
		return false
	}
}

func normalizeTitleTimeKey(title string, start, end time.Time) string {
	normTitle := strings.Join(strings.Fields(strings.ToLower(title)), " ")
	startUTC := start.UTC().Truncate(time.Second).Format(time.RFC3339)
	endUTC := end.UTC().Truncate(time.Second).Format(time.RFC3339)
	return fmt.Sprintf("%s|%s|%s", normTitle, startUTC, endUTC)
}

func isTaskAndEventEqual(task model.Task, event *calendar.Event) bool {
	if event == nil {
		return false
	}

	// 1. Text fields
	if strings.TrimSpace(task.Title) != strings.TrimSpace(event.Summary) {
		return false
	}
	if strings.TrimSpace(task.Description) != strings.TrimSpace(event.Description) {
		return false
	}
	if strings.TrimSpace(task.Location) != strings.TrimSpace(event.Location) {
		return false
	}

	// 2. Date / Time fields
	if task.IsAllDay {
		if event.Start == nil || event.Start.Date != task.TimeWindow.Start.Format("2006-01-02") {
			return false
		}
		if event.End == nil || event.End.Date != task.TimeWindow.End.Format("2006-01-02") {
			return false
		}
	} else {
		if event.Start == nil || event.Start.DateTime == "" {
			return false
		}
		evStart, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil || !evStart.Equal(task.TimeWindow.Start) {
			return false
		}
		if event.End == nil || event.End.DateTime == "" {
			return false
		}
		evEnd, err := time.Parse(time.RFC3339, event.End.DateTime)
		if err != nil || !evEnd.Equal(task.TimeWindow.End) {
			return false
		}
	}

	// 3. Extended properties
	if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
		priv := event.ExtendedProperties.Private
		if priv["uuid"] != "" && priv["uuid"] != task.UUID {
			return false
		}
		if p, ok := priv["priority"]; ok && p != "" && model.Priority(p) != task.Priority {
			return false
		}
		if spStr, ok := priv["story_points"]; ok && spStr != "" {
			if sp, err := strconv.Atoi(spStr); err == nil && sp != task.StoryPoints {
				return false
			}
		}
		if st, ok := priv["lifecycle_state"]; ok && st != "" && model.LifecycleState(st) != task.LifecycleState {
			return false
		}
		if sc, ok := priv["scheduling_type"]; ok && sc != "" && model.SchedulingType(sc) != task.SchedulingType {
			return false
		}
	}

	return true
}

func (s *SyncEngine) taskToEvent(task model.Task) *calendar.Event {
	event := &calendar.Event{
		Summary:     task.Title,
		Description: task.Description,
		Location:    task.Location,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid":            task.UUID,
				"source":          "stream",
				"priority":        string(task.Priority),
				"story_points":    strconv.Itoa(task.StoryPoints),
				"lifecycle_state": string(task.LifecycleState),
				"scheduling_type": string(task.SchedulingType),
			},
		},
	}

	if task.IsAllDay {
		event.Start = &calendar.EventDateTime{
			Date: task.TimeWindow.Start.Format("2006-01-02"),
		}
		event.End = &calendar.EventDateTime{
			Date: task.TimeWindow.End.Format("2006-01-02"),
		}
	} else {
		event.Start = &calendar.EventDateTime{
			DateTime: task.TimeWindow.Start.Format(time.RFC3339),
		}
		event.End = &calendar.EventDateTime{
			DateTime: task.TimeWindow.End.Format(time.RFC3339),
		}
	}

	return event
}
