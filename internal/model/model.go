package model

import (
	"time"
)

type Priority string

const (
	P0 Priority = "P0" // Urgent
	P1 Priority = "P1" // High
	P2 Priority = "P2" // Medium
	P3 Priority = "P3" // Low
)

type SchedulingType string

const (
	Anchored  SchedulingType = "ANCHORED"  // Fixed time block
	Floating  SchedulingType = "FLOATING"  // Todo Shelf/unscheduled
	Reminder  SchedulingType = "REMINDER"  // Reminder with due time
	Recurring SchedulingType = "RECURRING" // Recurring task template
	Habit     SchedulingType = "HABIT"     // Habit repeatable daily
	Event     SchedulingType = "EVENT"     // Calendar event
)

type LifecycleState string

const (
	StateBacklog   LifecycleState = "BACKLOG"
	StateScheduled LifecycleState = "SCHEDULED"
	StateReady     LifecycleState = "READY"
	StateActive    LifecycleState = "ACTIVE"
	StatePaused    LifecycleState = "PAUSED"
	StateCompleted LifecycleState = "COMPLETED"
	StateArchived  LifecycleState = "ARCHIVED"
	StateOverdue   LifecycleState = "OVERDUE"
)

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ExecutionMetrics struct {
	ElapsedFocusSeconds     int `json:"elapsed_focus_seconds"`
	ElapsedBreakSeconds     int `json:"elapsed_break_seconds"`
	TotalCompletedPomodoros int `json:"total_completed_pomodoros"`
	TargetPomodoros         int `json:"target_pomodoros"`
	InterruptionCount       int `json:"interruption_count"`
}

type GCalMetadata struct {
	SyncToken  string `json:"sync_token,omitempty"`
	ETag       string `json:"etag,omitempty"`
	SequenceID int64  `json:"sequence_id,omitempty"`
	EventID    string `json:"event_id,omitempty"` // Google Calendar Event ID
}

type Workspace struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Badge     string    `json:"badge"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	UUID             string           `json:"uuid"`
	WorkspaceUUID    string           `json:"workspace_uuid,omitempty"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	Priority         Priority         `json:"priority"`
	StoryPoints      int              `json:"story_points"`
	SchedulingType   SchedulingType   `json:"scheduling_type"`
	TimeWindow       TimeWindow       `json:"time_window"`
	LifecycleState   LifecycleState   `json:"lifecycle_state"`
	ExecutionMetrics ExecutionMetrics `json:"execution_metrics"`
	GCalMetadata     GCalMetadata     `json:"gcal_metadata"`
	Location         string           `json:"location,omitempty"`
	CommuteBuffer    int              `json:"commute_buffer,omitempty"` // in minutes
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Tags             []string         `json:"tags,omitempty"`
	Notes            string           `json:"notes,omitempty"`
	CompletedDates   []string         `json:"completed_dates,omitempty"`
	RecurringParentUUID   string           `json:"recurring_parent_uuid,omitempty"`
	EstimatedDurationMins int              `json:"estimated_duration_mins,omitempty"` // explicit duration for floating tasks
}

// SortingWeight computes the priority execution weight: (Priority Value * 1000) + Story Points
func (t *Task) SortingWeight() int {
	var pVal int
	switch t.Priority {
	case P0:
		pVal = 4000
	case P1:
		pVal = 3000
	case P2:
		pVal = 2000
	case P3:
		pVal = 1000
	default:
		pVal = 0
	}
	return pVal + t.StoryPoints
}

func IsTaskAnchored(t Task) bool {
	return t.SchedulingType == Anchored || t.SchedulingType == Event || (t.SchedulingType == Habit && !t.TimeWindow.Start.IsZero() && !t.TimeWindow.End.IsZero())
}

type GCalSyncMode string

const (
	GCalSyncNone   GCalSyncMode = "none"
	GCalSyncPush   GCalSyncMode = "push" // one-way: local → Google Calendar
	GCalSyncTwoWay GCalSyncMode = "two-way"
)

type UserSettings struct {
	Username                 string       `json:"username"`
	PasswordHash             string       `json:"password_hash"`
	LockTimeoutMinutes       int          `json:"lock_timeout_minutes"`
	GCalSyncMode             GCalSyncMode `json:"gcal_sync_mode,omitempty"`
	GCalSyncIntervalSeconds  int          `json:"gcal_sync_interval_seconds,omitempty"`
	GCalSyncIntervalMinutes  int          `json:"gcal_sync_interval_minutes,omitempty"` // legacy, migrated on load
	UpdateSnoozedUntil       time.Time    `json:"update_snoozed_until,omitempty"`
}

func IsGCalSyncable(task Task) bool {
	return task.SchedulingType == Anchored || task.SchedulingType == Event
}

func (s UserSettings) NormalizedGCalSync() UserSettings {
	if s.GCalSyncMode == "" {
		s.GCalSyncMode = GCalSyncPush
	}
	if s.GCalSyncIntervalSeconds <= 0 {
		if s.GCalSyncIntervalMinutes > 0 {
			s.GCalSyncIntervalSeconds = s.GCalSyncIntervalMinutes * 60
		} else {
			s.GCalSyncIntervalSeconds = 5
		}
	}
	return s
}
