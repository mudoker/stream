package viewmodel

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

var SyncModeOptions = []string{"No Sync", "Push Only (Local → GCal)", "Two-Way Sync"}

type TaskForm struct {
	Title          string
	Description    string
	PriorityIdx    int // 0: P0, 1: P1, 2: P2, 3: P3
	SPIdx          int // index in []int{1, 2, 3, 5, 8, 13}
	TaskTypeIdx    int // 0: Anchored, 1: Floating, 2: Reminder, 3: Habit, 4: Event
	StartHour      int
	StartMin       int
	DurationMins   int
	ActiveField    int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Type, 5: Start/Due Time, 6: Duration, 7: Location, 8: Commute Buffer, 9: Tags, 10: Submit
	TitleInput     textinput.Model
	DescInput      textinput.Model
	StartTimeInput textinput.Model
	DurationInput  textinput.Model
	TagsInput      textinput.Model
	DueDateInput   textinput.Model
	LocationInput  textinput.Model
	CommuteInput   textinput.Model
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

	return TaskForm{
		PriorityIdx:    2,
		SPIdx:          2,
		TaskTypeIdx:    0,
		StartHour:      now.Hour(),
		StartMin:       now.Minute(),
		DurationMins:   60,
		ActiveField:    0,
		TitleInput:     t,
		DescInput:      d,
		StartTimeInput: st,
		DurationInput:  dur,
		TagsInput:      tags,
		DueDateInput:   dd,
		LocationInput:  loc,
		CommuteInput:   commute,
	}
}

func (f TaskForm) VisibleFields() []int {
	switch f.TaskTypeIdx {
	case 0: // Anchored
		return []int{0, 1, 2, 3, 4, 5, 6, 9, 10}
	case 2: // Reminder
		return []int{0, 1, 2, 4, 5, 6, 9, 10}
	case 4: // Event
		if strings.TrimSpace(f.LocationInput.Value()) == "" {
			return []int{0, 1, 2, 3, 4, 5, 6, 7, 9, 10}
		}
		return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	default: // Floating, Habit
		return []int{0, 1, 2, 3, 4, 9, 10}
	}
}
