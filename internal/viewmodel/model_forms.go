package viewmodel

import (
	"strconv"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
)

var SyncModeOptions = []string{"No Sync", "Push Only (Local → GCal)", "Two-Way Sync"}

type TaskForm struct {
	Title          string
	Description    string
	PriorityIdx    int // 0: P0, 1: P1, 2: P2, 3: P3
	SPIdx          int // index in []int{1, 2, 3, 5, 8, 13}
	TaskTypeIdx    int // 0: Anchored, 1: Floating, 2: Reminder, 3: Habit
	StartHour      int
	StartMin       int
	DurationMins   int
	ActiveField    int // 0: Title, 1: Description, 2: Priority, 3: Story Points, 4: Type, 5: Start/Due Time, 6: Duration, 7: Tags, 8: Submit
	TitleInput     textinput.Model
	DescInput      textinput.Model
	StartTimeInput textinput.Model
	DurationInput  textinput.Model
	TagsInput      textinput.Model
	DueDateInput   textinput.Model
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
	}
}

func (f TaskForm) VisibleFields() []int {
	switch f.TaskTypeIdx {
	case 0: // Anchored
		return []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	case 2: // Reminder
		return []int{0, 1, 2, 4, 5, 6, 7, 8}
	default: // Floating
		return []int{0, 1, 2, 3, 4, 7, 8}
	}
}

type WorkspaceForm struct {
	Name        string
	Icon        string
	Badge       string
	ActiveField int // 0: Name, 1: Icon, 2: Badge, 3: Submit
	NameInput   textinput.Model
	IconInput   textinput.Model
	BadgeInput  textinput.Model
}

func NewWorkspaceForm() WorkspaceForm {
	name := textinput.New()
	name.Placeholder = "Aether Workspace"
	name.Focus()

	icon := textinput.New()
	icon.Placeholder = "🚀"
	icon.SetValue("🚀")

	badge := textinput.New()
	badge.Placeholder = "[Dev]"

	return WorkspaceForm{
		Name:        "",
		Icon:        "🚀",
		Badge:       "",
		ActiveField: 0,
		NameInput:   name,
		IconInput:   icon,
		BadgeInput:  badge,
	}
}

type ProfileForm struct {
	Username         string
	Password         string
	LockTimeoutMins  int
	ActiveField      int // 0: Username, 1: Password, 2: Lock Timeout (Mins), 3: Submit
	UsernameInput    textinput.Model
	PasswordInput    textinput.Model
	LockTimeoutInput textinput.Model
}

func NewProfileForm(username string, timeoutMins int) ProfileForm {
	u := textinput.New()
	u.Placeholder = "Doan Huu Quoc"
	u.SetValue(username)
	u.Focus()

	p := textinput.New()
	p.Placeholder = "(leave empty to keep, 'none' to disable)"
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '•'

	t := textinput.New()
	t.Placeholder = "5"
	t.SetValue(strconv.Itoa(timeoutMins))

	return ProfileForm{
		Username:         username,
		Password:         "",
		LockTimeoutMins:  timeoutMins,
		ActiveField:      0,
		UsernameInput:    u,
		PasswordInput:    p,
		LockTimeoutInput: t,
	}
}

type SyncForm struct {
	ModeIdx        int
	IntervalSecs   int
	ActiveField    int // 0: Mode, 1: Interval, 2: Submit
	IntervalInput  textinput.Model
}

func NewSyncForm(settings model.UserSettings) SyncForm {
	settings = settings.NormalizedGCalSync()

	modeIdx := 2
	switch settings.GCalSyncMode {
	case model.GCalSyncNone:
		modeIdx = 0
	case model.GCalSyncPush:
		modeIdx = 1
	case model.GCalSyncTwoWay:
		modeIdx = 2
	}

	interval := textinput.New()
	interval.Placeholder = "5"
	interval.SetValue(strconv.Itoa(settings.GCalSyncIntervalSeconds))
	interval.Focus()

	return SyncForm{
		ModeIdx:       modeIdx,
		IntervalSecs:  settings.GCalSyncIntervalSeconds,
		ActiveField:   0,
		IntervalInput: interval,
	}
}

func (f SyncForm) ModeValue() model.GCalSyncMode {
	switch f.ModeIdx {
	case 0:
		return model.GCalSyncNone
	case 1:
		return model.GCalSyncPush
	default:
		return model.GCalSyncTwoWay
	}
}
