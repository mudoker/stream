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
	RecurringDaysSelected []bool // Mon, Tue, Wed, Thu, Fri, Sat, Sun
	RecurringDaysSubIdx   int    // Cursor (0-6)
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

	form := TaskForm{
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
		RecurringDaysSelected: make([]bool, 7),
		RecurringDaysSubIdx:   0,
	}
	form.SyncDaysSelectedFromInput()
	return form
}

func (f TaskForm) VisibleFields() []int {
	var fields []int
	fields = append(fields, 0, 1, 2)
	if f.TaskTypeIdx != 2 && f.TaskTypeIdx != 3 && f.TaskTypeIdx != 4 {
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

	if !f.IsEditing {
		if f.TaskTypeIdx == 3 {
			// Habit is always recurring
			fields = append(fields, 12, 13)
		} else if f.TaskTypeIdx == 0 || f.TaskTypeIdx == 1 {
			fields = append(fields, 11)
			if f.IsRecurringIdx == 1 {
				fields = append(fields, 12, 13)
			}
		}
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
