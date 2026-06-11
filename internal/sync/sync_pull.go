package sync

import (
	"fmt"
	"strconv"
	"strings"
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
		s.logCallback(fmt.Sprintf("GCal Sync Error: %v", err))
		return
	}

	s.logCallback("GCal Sync: Pulling remote updates...")
	if err := s.pullRemoteUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Pull failed: %v", err))
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
	timeMin := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	events, err := srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).Do()
	if err != nil {
		return err
	}

	localTasks := s.localDB.GetTasks()
	localByGCalID := make(map[string]model.Task)
	localByTitleTime := make(map[string]model.Task)
	for _, t := range localTasks {
		if t.GCalMetadata.EventID != "" {
			localByGCalID[t.GCalMetadata.EventID] = t
		}
		if model.IsGCalSyncable(t) {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(t.Title), t.TimeWindow.Start.UTC().Format(time.RFC3339), t.TimeWindow.End.UTC().Format(time.RFC3339))
			localByTitleTime[key] = t
		}
	}

	defaultWS := s.defaultWorkspaceUUID()

	for _, item := range events.Items {
		if item.Status == "cancelled" {
			if local, exists := localByGCalID[item.Id]; exists && model.IsGCalSyncable(local) {
				_ = s.localDB.DeleteTask(local.UUID)
			}
			continue
		}

		uuidVal := ""
		priorityVal := model.P2
		spVal := 1
		stateVal := model.StateScheduled
		schedVal := model.Anchored

		if item.ExtendedProperties != nil && item.ExtendedProperties.Private != nil {
			uuidVal = item.ExtendedProperties.Private["uuid"]
			if p, ok := item.ExtendedProperties.Private["priority"]; ok {
				priorityVal = model.Priority(p)
			}
			if spStr, ok := item.ExtendedProperties.Private["story_points"]; ok {
				if sp, err := strconv.Atoi(spStr); err == nil {
					spVal = sp
				}
			}
			if st, ok := item.ExtendedProperties.Private["lifecycle_state"]; ok {
				stateVal = model.LifecycleState(st)
			}
			if sc, ok := item.ExtendedProperties.Private["scheduling_type"]; ok {
				schedVal = model.SchedulingType(sc)
			}
		}

		var start, end time.Time
		if item.Start != nil {
			start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			if start.IsZero() && item.Start.Date != "" {
				start, _ = time.Parse("2006-01-02", item.Start.Date)
			}
		}
		if item.End != nil {
			end, _ = time.Parse(time.RFC3339, item.End.DateTime)
			if end.IsZero() && item.Start != nil && item.Start.Date != "" {
				end = start.Add(24 * time.Hour)
			}
		}

		if schedVal != model.Anchored || start.IsZero() {
			continue
		}

		localTask, exists := localByGCalID[item.Id]
		if !exists {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(item.Summary), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
			if t, ok := localByTitleTime[key]; ok {
				localTask = t
				exists = true
			}
		}

		if exists {
			// GCal is source of truth, override local
			localTask.Title = item.Summary
			localTask.Description = item.Description
			localTask.TimeWindow.Start = start
			localTask.TimeWindow.End = end
			localTask.Priority = priorityVal
			localTask.StoryPoints = spVal
			localTask.LifecycleState = stateVal
			localTask.SchedulingType = schedVal
			localTask.GCalMetadata.EventID = item.Id
			localTask.GCalMetadata.ETag = item.Etag
			localTask.GCalMetadata.SequenceID = item.Sequence

			_ = s.localDB.UpdateTaskNoLedger(localTask)
		} else {
			if uuidVal == "" {
				uuidVal = uuid.New().String()
			}
			newTask := model.Task{
				UUID:           uuidVal,
				WorkspaceUUID:  defaultWS,
				Title:          item.Summary,
				Description:    item.Description,
				Priority:       priorityVal,
				StoryPoints:    spVal,
				SchedulingType: schedVal,
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
			}
			_ = s.localDB.AddTaskNoLedger(newTask)
		}
	}

	return nil
}
