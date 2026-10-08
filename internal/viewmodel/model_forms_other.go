package viewmodel

import (
	"strconv"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
)

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

type SprintForm struct {
	Name                string
	StartDate           string
	EndDate             string
	RecurringCount      int
	ActiveField         int // 0: Name, 1: Start Date, 2: End Date, 3: Recurring Sprints, 4: Submit
	NameInput           textinput.Model
	StartDateInput      textinput.Model
	EndDateInput        textinput.Model
	RecurringCountInput textinput.Model
	IsEditing           bool
	EditingSprintUUID   string
}

func NewSprintForm(defaultName string) SprintForm {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 13)

	name := textinput.New()
	name.Placeholder = "Sprint 1"
	if defaultName == "" {
		defaultName = "Sprint 1"
	}
	name.SetValue(defaultName)
	name.Focus()

	startDate := textinput.New()
	startDate.Placeholder = start.Format("2006-01-02")
	startDate.SetValue(start.Format("2006-01-02"))

	endDate := textinput.New()
	endDate.Placeholder = end.Format("2006-01-02")
	endDate.SetValue(end.Format("2006-01-02"))

	recCount := textinput.New()
	recCount.Placeholder = "0 (no recurring) or e.g. 4"
	recCount.SetValue("0")

	return SprintForm{
		Name:                defaultName,
		StartDate:           start.Format("2006-01-02"),
		EndDate:             end.Format("2006-01-02"),
		RecurringCount:      0,
		ActiveField:         0,
		NameInput:           name,
		StartDateInput:      startDate,
		EndDateInput:        endDate,
		RecurringCountInput: recCount,
		IsEditing:           false,
	}
}

func NewSprintFormFromSprint(s model.Sprint) SprintForm {
	name := textinput.New()
	name.Placeholder = s.Name
	name.SetValue(s.Name)
	name.Focus()

	startDate := textinput.New()
	startDate.Placeholder = s.StartDate.Format("2006-01-02")
	startDate.SetValue(s.StartDate.Format("2006-01-02"))

	endDate := textinput.New()
	endDate.Placeholder = s.EndDate.Format("2006-01-02")
	endDate.SetValue(s.EndDate.Format("2006-01-02"))

	recCount := textinput.New()
	recCount.Placeholder = "0"
	recCount.SetValue("0")

	return SprintForm{
		Name:                s.Name,
		StartDate:           s.StartDate.Format("2006-01-02"),
		EndDate:             s.EndDate.Format("2006-01-02"),
		RecurringCount:      0,
		ActiveField:         0,
		NameInput:           name,
		StartDateInput:      startDate,
		EndDateInput:        endDate,
		RecurringCountInput: recCount,
		IsEditing:           true,
		EditingSprintUUID:   s.UUID,
	}
}

