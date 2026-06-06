package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	// Colors (Linear / Arc Palette)
	CanvasBg    lipgloss.Color // Layer 0: Charcoal Black (#121318)
	PanelBg     lipgloss.Color // Layer 1: Elevated Slate (#1c1d24)
	SelectedBg  lipgloss.Color // Layer 2: Selected Focus (#2b2d38)
	ModalBg     lipgloss.Color // Layer 3: Modal Elevated (#252730)
	Fg          lipgloss.Color // Primary Text (#e2e8f0)
	Muted       lipgloss.Color // slate gray helper text (#626875)
	Accent      lipgloss.Color // Linear Signature Indigo (#5e6ad2)
	FocusPurple lipgloss.Color // Focus Highlight (#8b5cf6)

	// Priorities
	P0Color lipgloss.Color // Crimson Rose (#f43f5e)
	P1Color lipgloss.Color // Amber (#f59e0b)
	P2Color lipgloss.Color // Soft Blue (#3b82f6)
	P3Color lipgloss.Color // Gray (#6b7280)

	// Statuses
	SuccessColor lipgloss.Color // Emerald Green (#10b981)

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
	canvasBg := lipgloss.Color("#1e1e2e")
	panelBg := lipgloss.Color("#181825")
	selectedBg := lipgloss.Color("#313244")
	modalBg := lipgloss.Color("#1e1e2e")
	fg := lipgloss.Color("#cdd6f4")
	muted := lipgloss.Color("#6c7086")
	accent := lipgloss.Color("#89b4fa")
	focusPurple := lipgloss.Color("#b4befe")

	p0 := lipgloss.Color("#f38ba8")
	p1 := lipgloss.Color("#fab387")
	p2 := lipgloss.Color("#89b4fa")
	p3 := lipgloss.Color("#45475a")

	success := lipgloss.Color("#a6e3a1")

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
			Foreground(fg),

		PanelStyle: lipgloss.NewStyle().
			Foreground(fg).
			Padding(1, 2),

		SelectedPanel: lipgloss.NewStyle().
			Foreground(fg).
			Padding(1, 2),

		ModalStyle: lipgloss.NewStyle().
			Foreground(fg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2),

		HeaderStyle: lipgloss.NewStyle().
			Foreground(fg).
			Padding(0, 1).
			Bold(true),

		FooterStyle: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),

		TitleHeroStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(fg),

		MetadataStyle: lipgloss.NewStyle().
			Foreground(muted),
	}
}
