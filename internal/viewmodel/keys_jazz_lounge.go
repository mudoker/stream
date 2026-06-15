package viewmodel

import (
	tea "github.com/charmbracelet/bubbletea"
	"stream/internal/viewmodel/jazzlounge"
)

func (m *Model) handleJazzLoungeKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()
	engine := jazzlounge.GetJazzLoungeEngine()

	// 19 Options:
	// 0: Start / Stop Sound
	// 1: Master Volume
	// 2: Chords (Piano) Volume
	// 3: Melody (Solo Synth) Volume
	// 4: Drums (Beats) Volume
	// 5: Regenerate Chord Progression
	// 6: Ambient: Rain
	// 7: Ambient: Thunder
	// 8: Ambient: Campfire
	// 9: Ambient: Jungle
	// 10 to 18: Loop 1 to 9 (Background loops)

	switch key {
	case "up", "k":
		m.JazzLoungeSelectedIndex--
		if m.JazzLoungeSelectedIndex < 0 {
			m.JazzLoungeSelectedIndex = 18
		}
		return true, nil

	case "down", "j":
		m.JazzLoungeSelectedIndex++
		if m.JazzLoungeSelectedIndex > 18 {
			m.JazzLoungeSelectedIndex = 0
		}
		return true, nil

	case "right", "l":
		idx := m.JazzLoungeSelectedIndex
		switch idx {
		case 1:
			_, _, _, _, _, _, _, masterVol, _, _, _ := engine.GetState()
			engine.SetMasterVolume(masterVol + 0.05)
		case 2:
			_, _, _, _, _, _, _, _, pianoVol, _, _ := engine.GetState()
			engine.SetPianoVolume(pianoVol + 0.05)
		case 3:
			_, _, _, _, _, _, _, _, _, synthVol, _ := engine.GetState()
			engine.SetSynthVolume(synthVol + 0.05)
		case 4:
			_, _, _, _, _, _, _, _, _, _, drumsVol := engine.GetState()
			engine.SetDrumsVolume(drumsVol + 0.05)
		case 6, 7, 8, 9:
			ambName := []string{"Rain", "Thunder", "Campfire", "Jungle"}[idx-6]
			engine.AdjustAmbientVolume(ambName, 0.05)
		default:
			if idx >= 10 && idx <= 18 {
				trackID := idx - 9
				engine.AdjustTrackVolume(trackID, 0.05)
			}
		}
		return true, nil

	case "left", "h":
		idx := m.JazzLoungeSelectedIndex
		switch idx {
		case 1:
			_, _, _, _, _, _, _, masterVol, _, _, _ := engine.GetState()
			engine.SetMasterVolume(masterVol - 0.05)
		case 2:
			_, _, _, _, _, _, _, _, pianoVol, _, _ := engine.GetState()
			engine.SetPianoVolume(pianoVol - 0.05)
		case 3:
			_, _, _, _, _, _, _, _, _, synthVol, _ := engine.GetState()
			engine.SetSynthVolume(synthVol - 0.05)
		case 4:
			_, _, _, _, _, _, _, _, _, _, drumsVol := engine.GetState()
			engine.SetDrumsVolume(drumsVol - 0.05)
		case 6, 7, 8, 9:
			ambName := []string{"Rain", "Thunder", "Campfire", "Jungle"}[idx-6]
			engine.AdjustAmbientVolume(ambName, -0.05)
		default:
			if idx >= 10 && idx <= 18 {
				trackID := idx - 9
				engine.AdjustTrackVolume(trackID, -0.05)
			}
		}
		return true, nil

	case "enter", " ":
		idx := m.JazzLoungeSelectedIndex
		switch idx {
		case 0:
			engine.SetPlaying(!engine.IsPlaying())
			if engine.IsPlaying() {
				m.StatusMsg = "🔊 Jazz Lounge Engine started"
			} else {
				m.StatusMsg = "🔇 Jazz Lounge Engine stopped"
			}
		case 1, 2, 3, 4:
			// Volume options do nothing on Enter, adjust with Left/Right
		case 5:
			engine.RegenerateProgression()
			m.StatusMsg = "Generated new jazz progression"
		case 6:
			engine.ToggleAmbient("Rain")
		case 7:
			engine.ToggleAmbient("Thunder")
		case 8:
			engine.ToggleAmbient("Campfire")
		case 9:
			engine.ToggleAmbient("Jungle")
		default:
			// idx is 10 to 18 -> Track ID is idx - 9 (1 to 9)
			trackID := idx - 9
			engine.ToggleTrack(trackID)
		}
		return true, nil

	case "q", "esc":
		m.JazzLoungeOpen = false
		m.StatusMsg = ""
		return true, nil
	}

	return false, nil
}
