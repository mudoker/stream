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

		s.logCallback("Sync Engine: Online. Replaying anchored task ledger...")
		ledger := s.localDB.GetLedger()
		if len(ledger) > 0 {
			s.logCallback(fmt.Sprintf("Sync Engine: Replaying %d operations...", len(ledger)))
			replayed := 0
			for _, entry := range ledger {
				err := s.replayEntry(srv, entry)
				if err != nil {
					if isRateLimitError(err) {
						s.handleRateLimit(err)
						return
					}
					if s.handleStaleLedgerEntry(srv, entry, err) {
						_ = s.localDB.RemoveLedgerEntry(entry.ID)
						replayed++
						continue
					}
					if s.handleSkippableLedgerEntry(entry, err) {
						_ = s.localDB.RemoveLedgerEntry(entry.ID)
						replayed++
						continue
					}
					s.logCallback(fmt.Sprintf("Sync Ledger Error on %s (%s): %s", entry.Op, entry.TaskUUID, formatSyncError(err)))
					return
				}
				_ = s.localDB.RemoveLedgerEntry(entry.ID)
				replayed++
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
	normTitle := strings.TrimSpace(strings.ToLower(title))
	return fmt.Sprintf("%s|%s|%s", normTitle, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
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
