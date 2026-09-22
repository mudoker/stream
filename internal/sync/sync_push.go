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
		s.logCallback(fmt.Sprintf("GCal Sync Error: %s", formatSyncError(err)))
		return
	}

	s.logCallback("GCal Sync: Pushing local updates...")
	if err := s.pushLocalUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Push failed: %s", formatSyncError(err)))
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
				if err := s.deleteRemoteEvent(srv, entry.Task); err != nil {
					if isRateLimitError(err) {
						return err
					}
				}
				s.throttle()
			}
		}
		_ = s.localDB.RemoveLedgerEntry(entry.ID)
	}

	// 2. Fetch GCal events to avoid duplicates
	timeMin := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	gcalByEventID := make(map[string]*calendar.Event)
	gcalByTitleTime := make(map[string]*calendar.Event)
	gcalByUUID := make(map[string][]*calendar.Event)

	pageToken := ""
	for {
		call := srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).SingleEvents(true)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		events, err := call.Do()
		if err != nil {
			return err
		}

		for _, item := range events.Items {
			if item.Status == "cancelled" {
				continue
			}
			gcalByEventID[item.Id] = item
			if item.ExtendedProperties != nil && item.ExtendedProperties.Private != nil {
				if u, ok := item.ExtendedProperties.Private["uuid"]; ok && u != "" {
					gcalByUUID[u] = append(gcalByUUID[u], item)
				}
			}
			var start, end time.Time
			if item.Start != nil {
				start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			}
			if item.End != nil {
				end, _ = time.Parse(time.RFC3339, item.End.DateTime)
			}
			if !start.IsZero() && !end.IsZero() {
				key := normalizeTitleTimeKey(item.Summary, start, end)
				gcalByTitleTime[key] = item
			}
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			break
		}
		s.throttle()
	}

	// 3. Scan local database for ANCHORED tasks and push
	windowStart := time.Now().AddDate(0, 0, -30)
	localTasks := s.localDB.GetTasks()
	for _, t := range localTasks {
		if !model.IsGCalSyncable(t) {
			continue
		}
		if !t.TimeWindow.End.IsZero() && t.TimeWindow.End.Before(windowStart) {
			continue
		}

		// Check if GCal has multiple ghost events with this task's UUID -> clean them up!
		if evts, hasDups := gcalByUUID[t.UUID]; hasDups && len(evts) > 1 {
			bestIdx := 0
			for i, e := range evts {
				if t.GCalMetadata.EventID != "" && e.Id == t.GCalMetadata.EventID {
					bestIdx = i
					break
				}
			}
			// Delete the duplicate ghost events from GCal
			for i, e := range evts {
				if i != bestIdx {
					_ = s.deleteRemoteEvent(srv, model.Task{GCalMetadata: model.GCalMetadata{EventID: e.Id}})
					delete(gcalByEventID, e.Id)
				}
			}
			gcalByUUID[t.UUID] = []*calendar.Event{evts[bestIdx]}
		}

		var matchedEvent *calendar.Event
		if t.GCalMetadata.EventID != "" {
			matchedEvent = gcalByEventID[t.GCalMetadata.EventID]
		}
		if matchedEvent == nil && t.UUID != "" {
			if evts, ok := gcalByUUID[t.UUID]; ok && len(evts) > 0 {
				matchedEvent = evts[0]
			}
		}
		if matchedEvent == nil {
			// Try title+time matching
			key := normalizeTitleTimeKey(t.Title, t.TimeWindow.Start, t.TimeWindow.End)
			matchedEvent = gcalByTitleTime[key]
		}

		if matchedEvent != nil {
			t.GCalMetadata.EventID = matchedEvent.Id
			if matchedEvent.EventType != "" && matchedEvent.EventType != "default" && matchedEvent.EventType != "focusTime" {
				// Special Google events (flight reservations, hotel, outOfOffice) are read-only
				_ = s.localDB.UpdateTaskNoLedger(t)
				continue
			}

			// If event is already identical, skip sending redundant API requests
			if t.GCalMetadata.ETag == matchedEvent.Etag && t.Title == matchedEvent.Summary && t.Description == matchedEvent.Description {
				continue
			}

			// Exists on GCal: Update it (local is source of truth)
			if err := s.updateRemoteEvent(srv, t); err != nil {
				if isRateLimitError(err) {
					return err
				}
				s.logCallback(fmt.Sprintf("Sync: Failed to update GCal event '%s': %s", t.Title, formatSyncError(err)))
			}
		} else {
			// Does not exist on GCal: Create it
			if err := s.createRemoteEvent(srv, t); err != nil {
				if isRateLimitError(err) {
					return err
				}
				s.logCallback(fmt.Sprintf("Sync: Failed to create GCal event '%s': %s", t.Title, formatSyncError(err)))
			}
		}
		s.throttle()
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
	res, err := srv.Events.Patch("primary", task.GCalMetadata.EventID, event).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) {
			if apiErr.Code == 404 || apiErr.Code == 410 {
				return s.createRemoteEvent(srv, task)
			}
			if apiErr.Code == 400 && strings.Contains(apiErr.Message, "Event type cannot be changed") {
				// Special Google events (flights, reservations, etc.) are read-only from Gmail
				return nil
			}
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
		if errors.As(err, &apiErr) && (apiErr.Code == 404 || apiErr.Code == 410) {
			return nil
		}
		return err
	}
	return nil
}
