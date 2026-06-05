package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Bg          lipgloss.TerminalColor
	PanelBg     lipgloss.TerminalColor
	Fg          lipgloss.TerminalColor
	Accent      lipgloss.TerminalColor
	Focus       lipgloss.TerminalColor
	Success     lipgloss.TerminalColor
	Warning     lipgloss.TerminalColor
	Critical    lipgloss.TerminalColor
	Muted       lipgloss.TerminalColor
	BorderColor lipgloss.TerminalColor

	NormalBorder   lipgloss.Border
	ActiveBorder   lipgloss.Border
	HeaderStyle    lipgloss.Style
	FooterStyle    lipgloss.Style
	PanelStyle     lipgloss.Style
	SelectedStyle  lipgloss.Style
	ActiveCard     lipgloss.Style
	PausedCard     lipgloss.Style
	CompletedCard  lipgloss.Style
	OverdueCard    lipgloss.Style
}

func NewTheme() Theme {
	bg := lipgloss.Color("#1a1b26")      // Dark Tokyo Night canvas
	panelBg := lipgloss.Color("#1f2335") // Tokyo Night panel
	fg := lipgloss.Color("#c0caf5")
	accent := lipgloss.Color("#7aa2f7")   // Tokyo Night blue
	focus := lipgloss.Color("#bb9af7")    // Tokyo Night purple
	success := lipgloss.Color("#9ece6a")  // Tokyo Night green (Sage)
	warning := lipgloss.Color("#e0af68")  // Tokyo Night yellow/amber
	critical := lipgloss.Color("#f7768e") // Tokyo Night red/crimson
	muted := lipgloss.Color("#565f89")

	return Theme{
		Bg:          bg,
		PanelBg:     panelBg,
		Fg:          fg,
		Accent:      accent,
		Focus:       focus,
		Success:     success,
		Warning:     warning,
		Critical:    critical,
		Muted:       muted,
		BorderColor: muted,

		NormalBorder: lipgloss.RoundedBorder(),
		ActiveBorder: lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "┃", // Thick left border
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		},

		HeaderStyle: lipgloss.NewStyle().
			Foreground(fg).
			Background(panelBg).
			Padding(0, 1).
			Bold(true),

		FooterStyle: lipgloss.NewStyle().
			Foreground(fg).
			Background(panelBg).
			Padding(0, 1),

		PanelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(muted).
			Padding(0, 1),

		SelectedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(focus).
			Padding(0, 1),

		ActiveCard: lipgloss.NewStyle().
			Border(lipgloss.Border{
				Top:         "─",
				Bottom:      "─",
				Left:        "▌", // Bold bar indicator on the left
				Right:       "│",
				TopLeft:     "╭",
				TopRight:    "╮",
				BottomLeft:  "╰",
				BottomRight: "╯",
			}).
			BorderForeground(accent).
			Foreground(accent).
			Padding(0, 1).
			Bold(true),

		PausedCard: lipgloss.NewStyle().
			Border(lipgloss.Border{
				Top:         "─",
				Bottom:      "─",
				Left:        "▌",
				Right:       "│",
				TopLeft:     "╭",
				TopRight:    "╮",
				BottomLeft:  "╰",
				BottomRight: "╯",
			}).
			BorderForeground(warning).
			Foreground(warning).
			Padding(0, 1),

		CompletedCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(success).
			Foreground(muted).
			Padding(0, 1),

		OverdueCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(critical).
			Foreground(critical).
			Padding(0, 1),
	}
}
