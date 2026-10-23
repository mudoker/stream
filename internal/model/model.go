package model

import (
	"strings"
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
	StateReview    LifecycleState = "REVIEW"
	StateTesting   LifecycleState = "TESTING"
	StatePaused    LifecycleState = "PAUSED"
	StateCompleted LifecycleState = "COMPLETED"
	StateArchived  LifecycleState = "ARCHIVED"
	StateOverdue   LifecycleState = "OVERDUE"
)

type Sprint struct {
	UUID          string    `json:"uuid"`
	WorkspaceUUID string    `json:"workspace_uuid,omitempty"`
	Name          string    `json:"name"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NormalizeDate returns a time.Time at midnight UTC representing the calendar date of t.
func NormalizeDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// SprintDateBounds returns the normalized start and end calendar dates for a sprint.
// If EndDate is before StartDate or zero, it defaults to StartDate + 13 days.
func SprintDateBounds(s Sprint) (time.Time, time.Time) {
	var start time.Time
	if s.StartDate.IsZero() {
		now := time.Now()
		start = NormalizeDate(now)
	} else {
		start = NormalizeDate(s.StartDate)
	}
	end := NormalizeDate(s.EndDate)
	if end.Before(start) {
		end = start.AddDate(0, 0, 13)
	}
	return start, end
}

// IsSprintActive returns true if the sprint date range covers the date of `now`.
func IsSprintActive(s Sprint, now time.Time) bool {
	if s.StartDate.IsZero() {
		return false
	}
	today := NormalizeDate(now)
	start, end := SprintDateBounds(s)
	return !today.Before(start) && !today.After(end)
}

// SprintsOverlap checks if two sprints share at least one calendar day.
func SprintsOverlap(s1, s2 Sprint) bool {
	start1, end1 := SprintDateBounds(s1)
	start2, end2 := SprintDateBounds(s2)
	return !start1.After(end2) && !start2.After(end1)
}

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

type TaskSource string

const (
	SourceStream TaskSource = "stream"
	SourceGCal   TaskSource = "gcal"
)

type WorkItemType string

const (
	WorkItemFeature     WorkItemType = "Feature"
	WorkItemDefect      WorkItemType = "Defect"
	WorkItemImprovement WorkItemType = "Improvement"
	WorkItemTask        WorkItemType = "Task"
)

type Task struct {
	UUID                  string           `json:"uuid"`
	ID                    string           `json:"id,omitempty"` // Concise ID, e.g. "FEAT-1", "DEF-2"
	WorkItemType          WorkItemType     `json:"work_item_type,omitempty"`
	LinkedFeatureID       string           `json:"linked_feature_id,omitempty"` // ID of linked feature
	BlockedBy             string           `json:"blocked_by,omitempty"`        // Concise ID of blocker
	WorkspaceUUID         string           `json:"workspace_uuid,omitempty"`
	SprintUUID            string           `json:"sprint_uuid,omitempty"`
	AddedToToday          bool             `json:"added_to_today,omitempty"`
	InitiateDate          time.Time        `json:"initiate_date,omitempty"`
	Title                 string           `json:"title"`
	Description           string           `json:"description"`
	Priority              Priority         `json:"priority"`
	StoryPoints           int              `json:"story_points"`
	SchedulingType        SchedulingType   `json:"scheduling_type"`
	TimeWindow            TimeWindow       `json:"time_window"`
	LifecycleState        LifecycleState   `json:"lifecycle_state"`
	ExecutionMetrics      ExecutionMetrics `json:"execution_metrics"`
	GCalMetadata          GCalMetadata     `json:"gcal_metadata"`
	Source                TaskSource       `json:"source,omitempty"`
	Location              string           `json:"location,omitempty"`
	CommuteBuffer         int              `json:"commute_buffer,omitempty"` // in minutes
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
	Tags                  []string         `json:"tags,omitempty"`
	Notes                 string           `json:"notes,omitempty"`
	CompletedDates        []string         `json:"completed_dates,omitempty"`
	RecurringParentUUID   string           `json:"recurring_parent_uuid,omitempty"`
	EstimatedDurationMins int              `json:"estimated_duration_mins,omitempty"` // explicit duration for floating tasks
	SprintOrder           int              `json:"sprint_order,omitempty"`           // manual order within sprint swimlane
	IsAllDay              bool             `json:"is_all_day"`
}

func (t Task) GetInitiateDate() time.Time {
	if !t.InitiateDate.IsZero() {
		return t.InitiateDate
	}
	if !t.TimeWindow.Start.IsZero() {
		return t.TimeWindow.Start
	}
	if !t.CreatedAt.IsZero() {
		return t.CreatedAt
	}
	return time.Now()
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

func (t Task) HasCommuteBuffer() bool {
	return (t.SchedulingType == Event || t.SchedulingType == Habit) && strings.TrimSpace(t.Location) != "" && t.CommuteBuffer > 0
}

func IsTaskAnchored(t Task) bool {
	if t.SchedulingType == Anchored || t.SchedulingType == Event {
		return true
	}
	if !t.TimeWindow.Start.IsZero() && !t.TimeWindow.End.IsZero() && t.SchedulingType != Recurring && t.SchedulingType != Floating {
		return true
	}
	return false
}

type GCalSyncMode string

const (
	GCalSyncNone   GCalSyncMode = "none"
	GCalSyncPush   GCalSyncMode = "push" // one-way: local → Google Calendar
	GCalSyncTwoWay GCalSyncMode = "two-way"
)

type TagInfo struct {
	Name      string `json:"name"`
	Frequency int    `json:"frequency"`
}

type UserSettings struct {
	Username                 string       `json:"username"`
	PasswordHash             string       `json:"password_hash"`
	LockTimeoutMinutes       int          `json:"lock_timeout_minutes"`
	GCalSyncMode             GCalSyncMode `json:"gcal_sync_mode,omitempty"`
	GCalSyncIntervalSeconds  int          `json:"gcal_sync_interval_seconds,omitempty"`
	GCalSyncIntervalMinutes  int          `json:"gcal_sync_interval_minutes,omitempty"` // legacy, migrated on load
	UpdateSnoozedUntil       time.Time    `json:"update_snoozed_until,omitempty"`
	Tags                     []TagInfo    `json:"tags,omitempty"`
}

func IsGCalSyncable(task Task) bool {
	// Only tasks, habits, and events can sync to GCal as long as they are anchored.
	// Features, defects, improvements, and other sprint/feature-level items must NEVER sync to GCal.
	if task.WorkItemType == WorkItemFeature || task.WorkItemType == WorkItemDefect || task.WorkItemType == WorkItemImprovement {
		return false
	}
	if task.WorkItemType != "" && task.WorkItemType != WorkItemTask {
		return false
	}
	return IsTaskAnchored(task)
}

func (s UserSettings) NormalizedGCalSync() UserSettings {
	if s.GCalSyncMode == "" {
		s.GCalSyncMode = GCalSyncTwoWay
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

// InferStoryPoints infers story points from estimated duration in minutes:
// 1 point is 1 to 45 minutes, 2 points is 46 to 90 minutes, and so on.
func InferStoryPoints(durationMins int) int {
	if durationMins <= 0 {
		return 0
	}
	if durationMins <= 45 {
		return 1
	}
	if durationMins <= 90 {
		return 2
	}
	if durationMins <= 135 {
		return 3
	}
	if durationMins <= 225 {
		return 5
	}
	if durationMins <= 360 {
		return 8
	}
	return 13
}

// InferDurationFromPoints returns estimated duration in minutes for points (45 mins per point).
func InferDurationFromPoints(points int) int {
	if points <= 0 {
		return 0
	}
	return points * 45
}
