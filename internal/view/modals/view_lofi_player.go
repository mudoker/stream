package modals

import (
	"fmt"
	"strings"

	"stream/internal/view/theme"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/lofi"

	"github.com/charmbracelet/lipgloss"
)

func RenderLofiPlayerModal(m *viewmodel.Model, t theme.Theme) string {
	engine := lofi.GetLofiEngine()
	key, progression, activeChord, ambientStates, ambientVols, trackStates, trackVols, masterVol, pianoVol, synthVol, drumsVol := engine.GetState()
	isPlaying := engine.IsPlaying()

	const innerW = 54
	var sb strings.Builder

	// Header
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("🎷 j a z z   l o u n g e   e n g i n e") + "\n")
	sb.WriteString(ModalSep(innerW) + "\n\n")

	// Status Line
	var statusStr string
	if !engine.IsInitialized() {
		statusStr = lipgloss.NewStyle().Foreground(t.P1Color).Bold(true).Render("⏳ Loading audio samples...")
	} else if isPlaying {
		statusStr = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("🔊 Chilling in Cozy Bar...")
	} else {
		statusStr = lipgloss.NewStyle().Foreground(t.Muted).Render("🔇 Lounge is Closed")
	}
	sb.WriteString("  Status: " + statusStr + "\n")

	// Chord Progression Line
	var progBuilder strings.Builder
	progBuilder.WriteString("  Key: ")
	progBuilder.WriteString(lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(key))
	progBuilder.WriteString("   Progression: ")
	for _, degree := range progression {
		if degree == activeChord && isPlaying {
			progBuilder.WriteString(lipgloss.NewStyle().
				Background(t.Accent).
				Foreground(lipgloss.Color("#1e1e2e")).
				Bold(true).
				Render(" "+degree+" ") + " ")
		} else {
			progBuilder.WriteString(lipgloss.NewStyle().Foreground(t.Fg).Render(degree) + " ")
		}
	}
	sb.WriteString(progBuilder.String() + "\n\n")
	sb.WriteString(ModalSep(innerW) + "\n\n")

	// List of options helper
	renderItem := func(idx int, label string, valStr string) string {
		selected := m.LofiPlayerSelectedIndex == idx
		var line string
		if selected {
			indicator := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("┃")
			lbl := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(label)
			val := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(valStr)
			line = fmt.Sprintf("%s  %-22s %22s", indicator, lbl, val)
		} else {
			lbl := lipgloss.NewStyle().Foreground(t.Fg).Render(label)
			val := lipgloss.NewStyle().Foreground(t.Muted).Render(valStr)
			line = fmt.Sprintf("   %-22s %22s", lbl, val)
		}
		return line
	}

	renderVolBar := func(level float64) string {
		pct := int(level * 100)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		filled := pct / 10
		empty := 10 - filled
		return fmt.Sprintf("[%s%s] %3d%%", strings.Repeat("█", filled), strings.Repeat("░", empty), pct)
	}

	renderItemWithVol := func(idx int, label string, isPlaying bool, level float64) string {
		selected := m.LofiPlayerSelectedIndex == idx
		pct := int(level * 100)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		filled := pct / 10
		empty := 10 - filled
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

		var valStr string
		if isPlaying {
			valStr = fmt.Sprintf("[ON  %s %3d%%]", bar, pct)
		} else {
			valStr = fmt.Sprintf("[OFF %s %3d%%]", bar, pct)
		}

		var line string
		if selected {
			indicator := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("┃")
			lbl := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(label)
			val := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(valStr)
			line = fmt.Sprintf("%s  %-20s %24s", indicator, lbl, val)
		} else {
			lbl := lipgloss.NewStyle().Foreground(t.Fg).Render(label)
			val := lipgloss.NewStyle().Foreground(t.Muted).Render(valStr)
			line = fmt.Sprintf("   %-20s %24s", lbl, val)
		}
		return line
	}

	// 0. Toggle play
	playLabel := "Start / Stop Lounge Sound"
	playVal := "[OFF]"
	if isPlaying {
		playVal = "[ON]"
	}
	sb.WriteString(renderItem(0, playLabel, playVal) + "\n")

	// Channel Volumes
	sb.WriteString(renderItem(1, "Master Volume (lpf)", renderVolBar(masterVol)) + "\n")
	sb.WriteString(renderItem(2, "Piano (Chords) Vol", renderVolBar(pianoVol)) + "\n")
	sb.WriteString(renderItem(3, "Melody (Sax/Trumpet) Vol", renderVolBar(synthVol)) + "\n")
	sb.WriteString(renderItem(4, "Drums (Swing Beats) Vol", renderVolBar(drumsVol)) + "\n")

	// 5. Regenerate
	sb.WriteString(renderItem(5, "Regenerate Progression", "[GEN]") + "\n\n")

	// Section: Ambient Sounds
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("AMBIENT ENVIRONMENT") + "\n")
	ambientNames := []string{"Rain", "Thunder", "Campfire", "Jungle"}
	for i, name := range ambientNames {
		var state bool
		var vol float64
		if i < len(ambientStates) {
			state = ambientStates[i]
		}
		if i < len(ambientVols) {
			vol = ambientVols[i]
		}
		sb.WriteString(renderItemWithVol(6+i, name, state, vol) + "\n")
	}
	sb.WriteString("\n")

	// Section: Background Tracks
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("BACKGROUND LOOPS") + "\n")
	trackNames := []string{
		"Track 1: Wind",
		"Track 2: Waves",
		"Track 3: Night",
		"Track 4: Seagulls",
		"Track 5: Office",
		"Track 6: City",
		"Track 7: Server",
		"Track 8: Train",
		"Track 9: Underwater",
	}
	for i, name := range trackNames {
		var state bool
		var vol float64
		if i < len(trackStates) {
			state = trackStates[i]
		}
		if i < len(trackVols) {
			vol = trackVols[i]
		}
		sb.WriteString(renderItemWithVol(10+i, name, state, vol) + "\n")
	}

	sb.WriteString("\n" + ModalSep(innerW) + "\n")
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("↑/↓ navigate  ←/→ adjust volume  ↵ toggle/gen  Esc close")
	sb.WriteString("  " + hint)

	return t.ModalStyle.Render(PrepareModalContent(sb.String(), innerW))
}
