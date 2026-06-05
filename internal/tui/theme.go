package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	// Colors
	CanvasBg    lipgloss.Color // Layer 0: Dark Canvas (#1a1b26)
	PanelBg     lipgloss.Color // Layer 1: Panel Background (#1f2335)
	SelectedBg  lipgloss.Color // Layer 2: Selected/Active (#24283b)
	ModalBg     lipgloss.Color // Layer 3: Modal/Elevated (#2f354f)
	Fg          lipgloss.Color // Primary Text (#c0caf5)
	Muted       lipgloss.Color // Secondary/Muted (#565f89)
	Accent      lipgloss.Color // Active Accent (#7aa2f7)
	FocusPurple lipgloss.Color // Focused Accent (#bb9af7)

	// Priorities
	P0Color lipgloss.Color // Crimson/Red (#f7768e)
	P1Color lipgloss.Color // Orange/Amber (#e0af68)
	P2Color lipgloss.Color // Blue (#7aa2f7)
	P3Color lipgloss.Color // Gray (#565f89)

	// Statuses
	SuccessColor lipgloss.Color // Muted Green (#9ece6a)

	// Styling templates
	BaseStyle      lipgloss.Style
	PanelStyle     lipgloss.Style
	SelectedPanel  lipgloss.Style
	ModalStyle     lipgloss.Style
	HeaderStyle    lipgloss.Style
	FooterStyle    lipgloss.Style
	TitleHeroStyle lipgloss.Style
	MetadataStyle  lipgloss.Style
}

func NewTheme() Theme {
	canvasBg := lipgloss.Color("#1a1b26")
	panelBg := lipgloss.Color("#1f2335")
	selectedBg := lipgloss.Color("#24283b")
	modalBg := lipgloss.Color("#2f354f")
	fg := lipgloss.Color("#c0caf5")
	muted := lipgloss.Color("#565f89")
	accent := lipgloss.Color("#7aa2f7")
	focusPurple := lipgloss.Color("#bb9af7")

	p0 := lipgloss.Color("#f7768e")
	p1 := lipgloss.Color("#e0af68")
	p2 := lipgloss.Color("#7aa2f7")
	p3 := lipgloss.Color("#565f89")

	success := lipgloss.Color("#9ece6a")

	return Theme{
		CanvasBg:     canvasBg,
		PanelBg:      panelBg,
		SelectedBg:   selectedBg,
		ModalBg:      modalBg,
		Fg:           fg,
		Muted:        muted,
		Accent:       accent,
		FocusPurple:  focusPurple,
		P0Color:      p0,
		P1Color:      p1,
		P2Color:      p2,
		P3Color:      p3,
		SuccessColor: success,

		BaseStyle: lipgloss.NewStyle().
			Background(canvasBg).
			Foreground(fg),

		PanelStyle: lipgloss.NewStyle().
			Background(panelBg).
			Foreground(fg).
			Padding(1, 2),

		SelectedPanel: lipgloss.NewStyle().
			Background(selectedBg).
			Foreground(fg).
			Padding(1, 2),

		ModalStyle: lipgloss.NewStyle().
			Background(modalBg).
			Foreground(fg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2),

		HeaderStyle: lipgloss.NewStyle().
			Background(panelBg).
			Foreground(fg).
			Padding(0, 1).
			Bold(true),

		FooterStyle: lipgloss.NewStyle().
			Background(panelBg).
			Foreground(muted).
			Padding(0, 1),

		TitleHeroStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(fg),

		MetadataStyle: lipgloss.NewStyle().
			Foreground(muted),
	}
}
