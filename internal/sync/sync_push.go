package sync

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

func (s *SyncEngine) ManualPush() {
	if s.isRateLimited() {
		s.logCallback("GCal Sync: Rate limited. Please wait.")
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		s.logCallback(fmt.Sprintf("GCal Sync Error: %v", err))
		return
	}

	s.logCallback("GCal Sync: Pushing local updates...")
	if err := s.pushLocalUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Push failed: %v", err))
	} else {
		s.logCallback("GCal Sync: Push complete.")
	}
}

func (s *SyncEngine) pushLocalUpdates(srv *calendar.Service) error {
	// 1. Process deletions from the ledger first to remove deleted local tasks from GCal
	ledger := s.localDB.GetLedger()
	for _, entry := range ledger {
		if entry.Op == "DELETE" {
			if entry.Task.GCalMetadata.EventID != "" {
				s.logCallback(fmt.Sprintf("Sync: Deleting GCal event '%s'...", entry.Task.Title))
				_ = s.deleteRemoteEvent(srv, entry.Task)
			}
		}
		_ = s.localDB.RemoveLedgerEntry(entry.ID)
	}

	// 2. Fetch GCal events to avoid duplicates
	timeMin := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	events, err := srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).Do()
	if err != nil {
		return err
	}

	gcalByEventID := make(map[string]*calendar.Event)
	gcalByTitleTime := make(map[string]*calendar.Event)
	for _, item := range events.Items {
		if item.Status == "cancelled" {
			continue
		}
		gcalByEventID[item.Id] = item
		var start, end time.Time
		if item.Start != nil {
			start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		}
		if item.End != nil {
			end, _ = time.Parse(time.RFC3339, item.End.DateTime)
		}
		if !start.IsZero() && !end.IsZero() {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(item.Summary), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
			gcalByTitleTime[key] = item
		}
	}

	// 3. Scan local database for ANCHORED tasks and push
	localTasks := s.localDB.GetTasks()
	for _, t := range localTasks {
		if !model.IsGCalSyncable(t) {
			continue
		}

		var matchedEvent *calendar.Event
		if t.GCalMetadata.EventID != "" {
			matchedEvent = gcalByEventID[t.GCalMetadata.EventID]
		}
		if matchedEvent == nil {
			// Try title+time matching
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(t.Title), t.TimeWindow.Start.UTC().Format(time.RFC3339), t.TimeWindow.End.UTC().Format(time.RFC3339))
			matchedEvent = gcalByTitleTime[key]
		}

		if matchedEvent != nil {
			// Exists on GCal: Update it (local is source of truth)
			t.GCalMetadata.EventID = matchedEvent.Id
			if err := s.updateRemoteEvent(srv, t); err != nil {
				s.logCallback(fmt.Sprintf("Sync: Failed to update GCal event '%s': %v", t.Title, err))
			}
		} else {
			// Does not exist on GCal: Create it
			if err := s.createRemoteEvent(srv, t); err != nil {
				s.logCallback(fmt.Sprintf("Sync: Failed to create GCal event '%s': %v", t.Title, err))
			}
		}
	}

	return nil
}

func (s *SyncEngine) createRemoteEvent(srv *calendar.Service, task model.Task) error {
	if !model.IsGCalSyncable(task) {
		return nil
	}
	event := s.taskToEvent(task)
	res, err := srv.Events.Insert("primary", event).Do()
	if err != nil {
		return err
	}

	task.GCalMetadata.EventID = res.Id
	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence

	return s.localDB.UpdateTaskNoLedger(task)
}

func (s *SyncEngine) updateRemoteEvent(srv *calendar.Service, task model.Task) error {
	if !model.IsGCalSyncable(task) {
		return nil
	}
	if task.GCalMetadata.EventID == "" {
		return s.createRemoteEvent(srv, task)
	}

	event := s.taskToEvent(task)
	res, err := srv.Events.Update("primary", task.GCalMetadata.EventID, event).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return s.createRemoteEvent(srv, task)
		}
		return err
	}

	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence
	return s.localDB.UpdateTaskNoLedger(task)
}

func (s *SyncEngine) deleteRemoteEvent(srv *calendar.Service, task model.Task) error {
	if task.GCalMetadata.EventID == "" {
		return nil
	}
	err := srv.Events.Delete("primary", task.GCalMetadata.EventID).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return nil
		}
		return err
	}
	return nil
}
