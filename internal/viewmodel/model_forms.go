package viewmodel

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

var SyncModeOptions = []string{"No Sync", "Push Only (Local → GCal)", "Two-Way Sync"}

type TaskForm struct {
	Title                 string
	Description           string
	PriorityIdx           int // 0: P0, 1: P1, 2: P2, 3: P3
	SPIdx                 int // index in []int{1, 2, 3, 5, 8, 13}
	TaskTypeIdx           int // 0: Anchored, 1: Floating, 2: Reminder, 3: Habit, 4: Event
	StartHour             int
	StartMin              int
	DurationMins          int
	ActiveField           int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Type, 5: Start/Due Time, 6: Duration, 7: Location, 8: Commute Buffer, 9: Tags, 10: Submit, 11: Is Recurring, 12: Recurring End Date, 13: Recurring Days
	TitleInput            textinput.Model
	DescInput             textinput.Model
	StartTimeInput        textinput.Model
	DurationInput         textinput.Model
	TagsInput             textinput.Model
	DueDateInput          textinput.Model
	LocationInput         textinput.Model
	CommuteInput          textinput.Model
	IsRecurringIdx        int // 0: No, 1: Yes
	RecurringEndDateInput textinput.Model
	RecurringDaysInput    textinput.Model
	IsEditing             bool
}

func NewTaskForm() TaskForm {
	t := textinput.New()
	t.Placeholder = "Refactor auth engine..."
	t.Focus()

	d := textinput.New()
	d.Placeholder = "Fix memory leak in pool..."

	now := time.Now()
	st := textinput.New()
	st.Placeholder = now.Format("15:04")
	st.SetValue(now.Format("15:04"))

	dur := textinput.New()
	dur.Placeholder = "60"
	dur.SetValue("60")

	tags := textinput.New()
	tags.Placeholder = "engineering, refactor, admin"

	dd := textinput.New()
	dd.Placeholder = now.Format("2006-01-02")
	dd.SetValue(now.Format("2006-01-02"))

	loc := textinput.New()
	loc.Placeholder = "Office / Headquarters / Zoom"

	commute := textinput.New()
	commute.Placeholder = "Commute buffer (min)"
	commute.SetValue("0")

	reEnd := textinput.New()
	reEnd.Placeholder = now.AddDate(0, 0, 7).Format("2006-01-02")
	reEnd.SetValue(now.AddDate(0, 0, 7).Format("2006-01-02"))

	reDays := textinput.New()
	reDays.Placeholder = "Mon, Wed, Fri"
	reDays.SetValue("Mon, Tue, Wed, Thu, Fri")

	return TaskForm{
		PriorityIdx:           2,
		SPIdx:                 2,
		TaskTypeIdx:           0,
		StartHour:             now.Hour(),
		StartMin:              now.Minute(),
		DurationMins:          60,
		ActiveField:           0,
		TitleInput:            t,
		DescInput:             d,
		StartTimeInput:        st,
		DurationInput:         dur,
		TagsInput:             tags,
		DueDateInput:          dd,
		LocationInput:         loc,
		CommuteInput:          commute,
		IsRecurringIdx:        0,
		RecurringEndDateInput: reEnd,
		RecurringDaysInput:    reDays,
		IsEditing:             false,
	}
}

func (f TaskForm) VisibleFields() []int {
	var fields []int
	fields = append(fields, 0, 1, 2)
	if f.TaskTypeIdx != 2 {
		fields = append(fields, 3)
	}
	fields = append(fields, 4)

	if f.TaskTypeIdx == 0 || f.TaskTypeIdx == 4 {
		fields = append(fields, 5, 6)
	} else if f.TaskTypeIdx == 2 {
		fields = append(fields, 5, 6)
	}

	if f.TaskTypeIdx == 4 {
		fields = append(fields, 7)
		if strings.TrimSpace(f.LocationInput.Value()) != "" {
			fields = append(fields, 8)
		}
	}

	if !f.IsEditing && (f.TaskTypeIdx == 0 || f.TaskTypeIdx == 1 || f.TaskTypeIdx == 3) {
		fields = append(fields, 11)
		if f.IsRecurringIdx == 1 {
			fields = append(fields, 12, 13)
		}
	}

	fields = append(fields, 9, 10)
	return fields
}
