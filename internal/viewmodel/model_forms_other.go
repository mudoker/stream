package viewmodel

import (
	"strconv"

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
