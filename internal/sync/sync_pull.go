package sync

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
	"google.golang.org/api/calendar/v3"
)

func (s *SyncEngine) ManualPull() {
	if s.isRateLimited() {
		s.logCallback("GCal Sync: Rate limited. Please wait.")
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		s.logCallback(fmt.Sprintf("GCal Sync Error: %s", formatSyncError(err)))
		return
	}

	s.logCallback("GCal Sync: Pulling remote updates...")
	if err := s.pullRemoteUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Pull failed: %s", formatSyncError(err)))
	} else {
		s.logCallback("GCal Sync: Pull complete.")
	}
}

func (s *SyncEngine) defaultWorkspaceUUID() string {
	workspaces := s.localDB.GetWorkspaces()
	if len(workspaces) > 0 {
		return workspaces[0].UUID
	}
	return ""
}

func (s *SyncEngine) pullRemoteUpdates(srv *calendar.Service) error {
	timeMin := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)
	timeMax := time.Now().AddDate(0, 0, 180).Format(time.RFC3339)

	ledger := s.localDB.GetLedger()
	pendingDeleteUUIDs := make(map[string]bool)
	pendingDeleteEventIDs := make(map[string]bool)
	pendingLocalEdits := make(map[string]bool)

	for _, entry := range ledger {
		if entry.Op == "DELETE" {
			if entry.TaskUUID != "" {
				pendingDeleteUUIDs[entry.TaskUUID] = true
			}
			if entry.Task.GCalMetadata.EventID != "" {
				pendingDeleteEventIDs[entry.Task.GCalMetadata.EventID] = true
			}
		} else if entry.Op == "UPDATE" || entry.Op == "CREATE" {
			if entry.TaskUUID != "" {
				pendingLocalEdits[entry.TaskUUID] = true
			}
		}
	}

	localTasks := s.localDB.GetTasks()
	localByGCalID := make(map[string]model.Task)
	localByTitleTime := make(map[string]model.Task)
	localByUUID := make(map[string]model.Task)
	for _, t := range localTasks {
		if t.UUID != "" {
			localByUUID[t.UUID] = t
		}
		if t.GCalMetadata.EventID != "" {
			localByGCalID[t.GCalMetadata.EventID] = t
		}
		if model.IsGCalSyncable(t) {
			key := normalizeTitleTimeKey(t.Title, t.TimeWindow.Start, t.TimeWindow.End)
			localByTitleTime[key] = t
		}
	}

	defaultWS := s.defaultWorkspaceUUID()

	// Track processed UUIDs and TitleTimes from GCal to detect and clean up ghost duplicates
	seenGCalUUIDs := make(map[string]string)     // uuid -> primary eventID
	seenGCalTitleTime := make(map[string]string) // key -> primary eventID

	pageToken := ""
	for {
		call := srv.Events.List("primary").
			TimeMin(timeMin).
			TimeMax(timeMax).
			MaxResults(250).
			ShowDeleted(true).
			SingleEvents(true)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		events, err := call.Do()
		if err != nil {
			return err
		}

		for _, item := range events.Items {
			uuidVal := ""
			sourceVal := model.SourceGCal
			priorityVal := model.P2
			spVal := 0
			stateVal := model.StateScheduled
			schedVal := model.Event
			hasExplicitSched := false

			if item.ExtendedProperties != nil && item.ExtendedProperties.Private != nil {
				uuidVal = item.ExtendedProperties.Private["uuid"]
				if src := item.ExtendedProperties.Private["source"]; src == "stream" || uuidVal != "" {
					sourceVal = model.SourceStream
				}
				if p, ok := item.ExtendedProperties.Private["priority"]; ok && p != "" {
					priorityVal = model.Priority(p)
				}
				if spStr, ok := item.ExtendedProperties.Private["story_points"]; ok && spStr != "" {
					if sp, err := strconv.Atoi(spStr); err == nil {
						spVal = sp
					}
				}
				if st, ok := item.ExtendedProperties.Private["lifecycle_state"]; ok && st != "" {
					stateVal = model.LifecycleState(st)
				}
				if sc, ok := item.ExtendedProperties.Private["scheduling_type"]; ok && sc != "" {
					schedVal = model.SchedulingType(sc)
					hasExplicitSched = true
				}
			}

			if item.Status == "cancelled" {
				if local, exists := localByGCalID[item.Id]; exists {
					var remoteUpdated time.Time
					if item.Updated != "" {
						remoteUpdated, _ = time.Parse(time.RFC3339, item.Updated)
					}
					// If local has pending unpushed edits or local is strictly newer than remote cancellation, keep local
					if pendingLocalEdits[local.UUID] || (!remoteUpdated.IsZero() && !local.UpdatedAt.IsZero() && local.UpdatedAt.After(remoteUpdated)) {
						continue
					}

					// For imported GCal events, delete them from local DB
					if local.Source == model.SourceGCal || local.SchedulingType == model.Event {
						_ = s.localDB.DeleteTaskNoLedger(local.UUID)
						delete(localByUUID, local.UUID)
						delete(localByGCalID, item.Id)
					} else {
						// For Stream user tasks: NEVER delete user tasks! Safely de-anchor to Todo Shelf
						local.SchedulingType = model.Floating
						local.GCalMetadata = model.GCalMetadata{}
						_ = s.localDB.UpdateTaskNoLedger(local)
						localByUUID[local.UUID] = local
						delete(localByGCalID, item.Id)
					}
				}
				continue
			}

			// If this task was deleted locally and pending delete on GCal, do not resurrect it
			if (uuidVal != "" && pendingDeleteUUIDs[uuidVal]) || pendingDeleteEventIDs[item.Id] {
				_ = s.deleteRemoteEvent(srv, model.Task{GCalMetadata: model.GCalMetadata{EventID: item.Id}})
				continue
			}

			var isAllDay bool
			var start, end time.Time
			if item.Start != nil {
				start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
				if start.IsZero() && item.Start.Date != "" {
					start, _ = time.Parse("2006-01-02", item.Start.Date)
					isAllDay = true
				}
			}
			if item.End != nil {
				end, _ = time.Parse(time.RFC3339, item.End.DateTime)
				if end.IsZero() && item.End.Date != "" {
					end, _ = time.Parse("2006-01-02", item.End.Date)
					isAllDay = true
				} else if end.IsZero() && item.Start != nil && item.Start.Date != "" {
					end = start.Add(24 * time.Hour)
					isAllDay = true
				}
			}

			if !model.IsGCalSyncable(model.Task{SchedulingType: schedVal}) || start.IsZero() {
				continue
			}

			var remoteUpdated time.Time
			if item.Updated != "" {
				remoteUpdated, _ = time.Parse(time.RFC3339, item.Updated)
			}

			titleTimeKey := normalizeTitleTimeKey(item.Summary, start, end)

			// Track seen UUIDs and TitleTimes from GCal to avoid duplicate processing in the same pull
			if uuidVal != "" {
				if primID, alreadySeen := seenGCalUUIDs[uuidVal]; alreadySeen && primID != item.Id {
					continue
				}
				seenGCalUUIDs[uuidVal] = item.Id
			} else {
				if primID, alreadySeen := seenGCalTitleTime[titleTimeKey]; alreadySeen && primID != item.Id {
					continue
				}
				seenGCalTitleTime[titleTimeKey] = item.Id
			}

			var localTask model.Task
			exists := false

			if uuidVal != "" {
				if t, ok := localByUUID[uuidVal]; ok {
					localTask = t
					exists = true
				}
			}
			if !exists {
				if t, ok := localByGCalID[item.Id]; ok {
					localTask = t
					exists = true
				}
			}
			if !exists && uuidVal == "" {
				if t, ok := localByTitleTime[titleTimeKey]; ok && t.GCalMetadata.EventID == "" {
					localTask = t
					exists = true
				}
			}

			if exists {
				hasPendingEdit := localTask.UUID != "" && pendingLocalEdits[localTask.UUID]

				if hasPendingEdit {
					// Conflict resolution: Local has pending unpushed edits in ledger.
					// Keep local data intact and link GCal metadata.
					if localTask.GCalMetadata.EventID != item.Id || localTask.GCalMetadata.ETag != item.Etag {
						localTask.GCalMetadata.EventID = item.Id
						localTask.GCalMetadata.ETag = item.Etag
						localTask.GCalMetadata.SequenceID = item.Sequence
						_ = s.localDB.UpdateTaskNoLedger(localTask)
						localByUUID[localTask.UUID] = localTask
						localByGCalID[item.Id] = localTask
					}
				} else if !remoteUpdated.IsZero() && !localTask.UpdatedAt.IsZero() && localTask.UpdatedAt.After(remoteUpdated) {
					// Stale handling: Local task is strictly newer than remote event timestamp.
					// Preserve local changes and link metadata.
					if localTask.GCalMetadata.EventID != item.Id || localTask.GCalMetadata.ETag != item.Etag {
						localTask.GCalMetadata.EventID = item.Id
						localTask.GCalMetadata.ETag = item.Etag
						localTask.GCalMetadata.SequenceID = item.Sequence
						_ = s.localDB.UpdateTaskNoLedger(localTask)
						localByUUID[localTask.UUID] = localTask
						localByGCalID[item.Id] = localTask
					}
				} else {
					// Remote is newer or equal: apply remote updates to local task
					if !hasExplicitSched && localTask.SchedulingType != "" {
						schedVal = localTask.SchedulingType
					}
					if localTask.Priority != "" && (item.ExtendedProperties == nil || item.ExtendedProperties.Private == nil || item.ExtendedProperties.Private["priority"] == "") {
						priorityVal = localTask.Priority
					}
					if item.ExtendedProperties == nil || item.ExtendedProperties.Private == nil || item.ExtendedProperties.Private["story_points"] == "" {
						spVal = localTask.StoryPoints
					}
					if localTask.LifecycleState != "" && (item.ExtendedProperties == nil || item.ExtendedProperties.Private == nil || item.ExtendedProperties.Private["lifecycle_state"] == "") {
						stateVal = localTask.LifecycleState
					}
					if localTask.Source != "" {
						sourceVal = localTask.Source
					}

					localTask.Title = item.Summary
					localTask.Description = item.Description
					localTask.TimeWindow.Start = start
					localTask.TimeWindow.End = end
					localTask.Location = item.Location
					localTask.Priority = priorityVal
					localTask.StoryPoints = spVal
					localTask.LifecycleState = stateVal
					localTask.SchedulingType = schedVal
					localTask.IsAllDay = isAllDay
					localTask.GCalMetadata.EventID = item.Id
					localTask.GCalMetadata.ETag = item.Etag
					localTask.GCalMetadata.SequenceID = item.Sequence
					if localTask.Source == "" {
						localTask.Source = sourceVal
					}
					if !remoteUpdated.IsZero() {
						localTask.UpdatedAt = remoteUpdated
					}

					_ = s.localDB.UpdateTaskNoLedger(localTask)
					localByUUID[localTask.UUID] = localTask
					localByGCalID[item.Id] = localTask
				}
			} else {
				if uuidVal == "" {
					uuidVal = uuid.New().String()
				}
				createdAt := time.Now()
				if !remoteUpdated.IsZero() {
					createdAt = remoteUpdated
				}
				newTask := model.Task{
					UUID:           uuidVal,
					WorkspaceUUID:  defaultWS,
					Title:          item.Summary,
					Description:    item.Description,
					Location:       item.Location,
					Priority:       priorityVal,
					StoryPoints:    spVal,
					SchedulingType: schedVal,
					Source:         sourceVal,
					IsAllDay:       isAllDay,
					TimeWindow: model.TimeWindow{
						Start: start,
						End:   end,
					},
					LifecycleState: stateVal,
					GCalMetadata: model.GCalMetadata{
						EventID:    item.Id,
						ETag:       item.Etag,
						SequenceID: item.Sequence,
					},
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				}
				_ = s.localDB.AddTaskNoLedger(newTask)
				localByUUID[newTask.UUID] = newTask
				localByGCalID[item.Id] = newTask
			}
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			break
		}
		s.throttle()
	}

	return nil
}
