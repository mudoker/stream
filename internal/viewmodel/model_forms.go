package viewmodel

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
)

var SyncModeOptions = []string{"No Sync", "Push Only (Local → GCal)", "Two-Way Sync"}

type TaskForm struct {
	Title                 string
	Description           string
	PriorityIdx           int // 0: P0, 1: P1, 2: P2, 3: P3
	SPIdx                 int // index in []int{0, 1, 2, 3, 5, 8, 13}
	TaskTypeIdx           int // 0: Feature, 1: Defect, 2: Improvement, 3: Task, 4: Reminder, 5: Habit, 6: Event
	IsAnchoredIdx         int // 0: No, 1: Yes
	IsAllDayIdx           int // 0: No, 1: Yes
	LinkSprintIdx         int // 0: No, 1: Yes
	AddToTodayIdx         int // 0: No, 1: Yes
	LinkedFeatureIdx      int // 0: None, 1..N: index into AvailableFeatures
	AvailableFeatures     []model.Task
	BlockedByIdx          int // 0: None, 1..N: index into AvailableBlockers
	AvailableBlockers     []model.Task
	StartHour             int
	StartMin              int
	DurationMins          int
	ActiveField           int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Type, 5: Start/Due Time, 6: Duration, 7: Location, 8: Commute Buffer, 9: Tags, 10: Submit, 11: Is Recurring, 12: Recurring End Date, 13: Recurring Days, 14: Start Date, 15: End Date, 16: Is Anchored, 17: Is All Day, 18: Link to Sprint, 20: Link to Feature, 21: Blocked By, 22: Add to Today
	TitleInput            textinput.Model
	DescInput             textinput.Model
	StartTimeInput        textinput.Model
	DurationInput         textinput.Model
	TagsInput             textinput.Model
	DueDateInput          textinput.Model
	StartDateInput        textinput.Model
	EndDateInput          textinput.Model
	LocationInput         textinput.Model
	CommuteInput          textinput.Model
	IsRecurringIdx        int // 0: No, 1: Yes
	RecurringEndDateInput textinput.Model
	RecurringDaysInput    textinput.Model
	IsEditing             bool
	RecurringDaysSelected []bool // Mon, Tue, Wed, Thu, Fri, Sat, Sun
	RecurringDaysSubIdx   int    // Cursor (0-6)
}

func NewTaskForm() TaskForm {
	return NewTaskFormWithDate(time.Now())
}

// smartDefaultTime returns the current time rounded up to the next 30-minute mark.
// This gives users a sensible default start time when opening the new task form.
func smartDefaultTime() string {
	now := time.Now()
	min := now.Minute()
	var h, m int
	if min < 30 {
		h, m = now.Hour(), 30
	} else {
		h = (now.Hour() + 1) % 24
		m = 0
	}
	return fmt.Sprintf("%02d:%02d", h, m)
}

func NewTaskFormWithDate(baseDate time.Time) TaskForm {
	t := textinput.New()
	t.Placeholder = "Refactor auth engine..."
	t.Focus()

	d := textinput.New()
	d.Placeholder = "Fix memory leak in pool..."

	defaultTime := smartDefaultTime()
	st := textinput.New()
	st.Placeholder = defaultTime
	st.SetValue(defaultTime)

	dur := textinput.New()
	dur.Placeholder = "60"
	dur.SetValue("60")

	tags := textinput.New()
	tags.Placeholder = "engineering, refactor, admin"
	tags.SetValue("Misc.")

	dd := textinput.New()
	dd.Placeholder = baseDate.Format("2006-01-02")
	dd.SetValue(baseDate.Format("2006-01-02"))

	loc := textinput.New()
	loc.Placeholder = "Office / Headquarters / Zoom"

	commute := textinput.New()
	commute.Placeholder = "Commute buffer (min)"
	commute.SetValue("0")

	reEnd := textinput.New()
	reEnd.Placeholder = baseDate.AddDate(0, 0, 7).Format("2006-01-02")
	reEnd.SetValue(baseDate.AddDate(0, 0, 7).Format("2006-01-02"))

	reDays := textinput.New()
	reDays.Placeholder = "Mon, Wed, Fri"
	reDays.SetValue("Mon, Tue, Wed, Thu, Fri")

	startDate := textinput.New()
	startDate.Placeholder = baseDate.Format("2006-01-02")
	startDate.SetValue(baseDate.Format("2006-01-02"))

	endDate := textinput.New()
	endDate.Placeholder = baseDate.Format("2006-01-02")
	endDate.SetValue(baseDate.Format("2006-01-02"))

	form := TaskForm{
		PriorityIdx:           2,
		SPIdx:                 2,
		TaskTypeIdx:           3, // default: Task
		IsAnchoredIdx:         0,
		IsAllDayIdx:           0,
		LinkSprintIdx:         0,
		AddToTodayIdx:         0,
		LinkedFeatureIdx:      0,
		BlockedByIdx:          0,
		StartHour:             baseDate.Hour(),
		StartMin:              baseDate.Minute(),
		DurationMins:          60,
		ActiveField:           0,
		TitleInput:            t,
		DescInput:             d,
		StartTimeInput:        st,
		DurationInput:         dur,
		TagsInput:             tags,
		DueDateInput:          dd,
		StartDateInput:        startDate,
		EndDateInput:          endDate,
		LocationInput:         loc,
		CommuteInput:          commute,
		IsRecurringIdx:        0,
		RecurringEndDateInput: reEnd,
		RecurringDaysInput:    reDays,
		IsEditing:             false,
		RecurringDaysSelected: make([]bool, 7),
		RecurringDaysSubIdx:   0,
	}
	form.SyncDaysSelectedFromInput()
	return form
}

func (f TaskForm) VisibleFields() []int {
	var fields []int
	fields = append(fields, 0, 1, 4, 2) // Title, Description, Type, Priority

	// Feature (0), Defect (1), Improvement (2)
	if f.TaskTypeIdx == 0 || f.TaskTypeIdx == 1 || f.TaskTypeIdx == 2 {
		fields = append(fields, 3, 21, 9, 10) // Story Points, Blocked By, Tags, Submit
		return fields
	}

	// Task (3)
	if f.TaskTypeIdx == 3 {
		fields = append(fields, 3, 16) // Story Points, Is Anchored
		if f.IsAnchoredIdx == 1 {
			fields = append(fields, 5, 6) // Start Time, Duration
		} else {
			fields = append(fields, 6) // Est Duration
		}
		fields = append(fields, 22, 20, 21, 11) // Add to Today, Link to Feature, Blocked By, Is Recurring
		if f.IsRecurringIdx == 1 {
			fields = append(fields, 12, 13)
		}
		fields = append(fields, 9, 10)
		return fields
	}

	// Reminder (4)
	if f.TaskTypeIdx == 4 {
		fields = append(fields, 5, 6, 11)
		if f.IsRecurringIdx == 1 {
			fields = append(fields, 12, 13)
		}
		fields = append(fields, 9, 10)
		return fields
	}

	// Habit (5)
	if f.TaskTypeIdx == 5 {
		fields = append(fields, 5, 6, 7)
		if strings.TrimSpace(f.LocationInput.Value()) != "" {
			fields = append(fields, 8)
		}
		fields = append(fields, 12, 13, 9, 10)
		return fields
	}

	// Event (6)
	if f.TaskTypeIdx == 6 {
		fields = append(fields, 14, 17)
		if f.IsAllDayIdx == 0 {
			fields = append(fields, 5, 6)
		}
		fields = append(fields, 7)
		if strings.TrimSpace(f.LocationInput.Value()) != "" {
			fields = append(fields, 8)
		}
		fields = append(fields, 11)
		if f.IsRecurringIdx == 1 {
			fields = append(fields, 12, 13)
		}
		fields = append(fields, 9, 10)
		return fields
	}

	fields = append(fields, 9, 10)
	return fields
}

func (f *TaskForm) SyncDaysSelectedFromInput() {
	if len(f.RecurringDaysSelected) != 7 {
		f.RecurringDaysSelected = make([]bool, 7)
	}
	val := strings.ToLower(f.RecurringDaysInput.Value())
	f.RecurringDaysSelected[0] = strings.Contains(val, "mon") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[1] = strings.Contains(val, "tue") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[2] = strings.Contains(val, "wed") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[3] = strings.Contains(val, "thu") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[4] = strings.Contains(val, "fri") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[5] = strings.Contains(val, "sat") || strings.Contains(val, "daily")
	f.RecurringDaysSelected[6] = strings.Contains(val, "sun") || strings.Contains(val, "daily")
}

func (f *TaskForm) updateDaysInputValue() {
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var selected []string
	for i, val := range f.RecurringDaysSelected {
		if val {
			selected = append(selected, days[i])
		}
	}
	f.RecurringDaysInput.SetValue(strings.Join(selected, ", "))
}
