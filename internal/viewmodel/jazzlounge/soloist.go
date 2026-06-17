package jazzlounge

import (
	"math"
	"math/rand"
)

// Soloist represents a jazz soloist (sax or trumpet) that improvises
// melodies using MIDI notes directly (not scale indices), enabling
// chromatic approach notes, enclosures, and bebop vocabulary.
type Soloist struct {
	Type         string // "sax" or "trumpet"
	PhraseTicks  int
	PauseTicks   int
	LastMIDINote int
	Motif        []int
	MotifIdx     int

	// Composition Fields
	ImprovState   int     // 0: stepwise, 1: arpeggio, 2: motif, 3: chromatic approach
	MelodyDir     int     // +1 ascending, -1 descending
	MotifNotes    []int   // Actual MIDI notes for a motif pattern
	MotifNoteIdx  int
	LastLeap      int
	PhraseEnergy  float64 // 0.0 to 1.0, builds within a phrase for dynamic arc
	NoteCount     int     // notes played in this phrase
	PhraseLength  int     // total notes planned for this phrase
}

// Rhythmic motifs: durations in ticks
var PhraseMotifs = [][]int{
	{2, 2, 2, 2},             // Quarter notes (lyrical)
	{1, 1, 1, 1, 1, 1, 2},   // Eighth-note run ending on quarter
	{3, 1, 3, 1},             // Dotted swing (long-short)
	{1, 1, 2, 1, 1, 2},      // Bebop phrasing
	{4, 2, 2},               // Long note then two shorts
	{1, 1, 1, 1, 2, 2},      // Run into sustained
	{2, 1, 1, 2, 1, 1},      // Syncopated push
	{6, 2},                   // Very long hold then short
}

func isChordTone(note int, chord JazzChord, keyPitch int) bool {
	noteClass := note % 12
	chordRootClass := (keyPitch + chord.RootOffset) % 12
	for _, interval := range chord.Intervals {
		chordToneClass := (chordRootClass + interval) % 12
		if noteClass == chordToneClass {
			return true
		}
	}
	return false
}

func isGuideTone(note int, chord JazzChord, keyPitch int) bool {
	noteClass := note % 12
	chordRootClass := (keyPitch + chord.RootOffset) % 12
	for _, interval := range chord.Intervals {
		if interval == 3 || interval == 4 || interval == 10 || interval == 11 {
			chordToneClass := (chordRootClass + interval) % 12
			if noteClass == chordToneClass {
				return true
			}
		}
	}
	return false
}

// findNearestChordTone finds the nearest chord tone to the given MIDI note.
func findNearestChordTone(fromNote int, chord JazzChord, keyPitch int, lowBound, highBound int) int {
	best := fromNote
	bestDist := 999
	for n := lowBound; n <= highBound; n++ {
		if isChordTone(n, chord, keyPitch) {
			d := int(math.Abs(float64(n - fromNote)))
			if d < bestDist {
				bestDist = d
				best = n
			}
		}
	}
	return best
}

// findNearestGuideTone finds the nearest guide tone (3rd or 7th) to the given MIDI note.
func findNearestGuideTone(fromNote int, chord JazzChord, keyPitch int, lowBound, highBound int) int {
	best := fromNote
	bestDist := 999
	for n := lowBound; n <= highBound; n++ {
		if isGuideTone(n, chord, keyPitch) {
			d := int(math.Abs(float64(n - fromNote)))
			if d < bestDist {
				bestDist = d
				best = n
			}
		}
	}
	return best
}

// generateMotifNotes creates a short melodic motif rooted at startNote using chord tones.
func generateMotifNotes(startNote int, chord JazzChord, keyPitch int, dir int) []int {
	motifPatterns := [][]int{
		{0, 2, -1, 3},       // step up, step back, leap
		{0, -2, -1, 1, 3},   // dip then climb
		{0, 4, 3, 2, 0},     // leap up, walk back
		{0, -1, -2, 0, 2},   // chromatic dip, return, step up
		{0, 3, 5, 3, 0},     // arch shape
		{0, -3, -5, -3, 0},  // inverse arch
	}
	pattern := motifPatterns[rand.Intn(len(motifPatterns))]

	notes := make([]int, len(pattern))
	for i, offset := range pattern {
		notes[i] = startNote + offset*dir
	}
	return notes
}

func (e *JazzLoungeEngine) processSoloist(s *Soloist, tickCount int) {
	if s.PauseTicks > 0 {
		return
	}

	// Chapter-specific Soloist Constraints
	switch e.narrative.ActiveChapter {
	case "IntimateNocturne", "PianoInterlude", "BassSoliloquy", "StillnessAtmosphere":
		// These chapters aggressively silence horn soloists to focus on piano, bass, or atmosphere
		s.PauseTicks = 8
		return
	case "NocturnalSuspense":
		// Only play extremely sparse, single-note mystery swells occasionally
		if rand.Float64() < 0.85 {
			s.PauseTicks = 16
			return
		}
	}

	keyPitch := keyToPitch(e.key)
	chord := e.progression[e.progress]

	// Instrument range in MIDI
	// Instrument range in MIDI (dynamically scaled by Register.Width)
	var lowBound, highBound int
	if s.Type == "sax" {
		lowBound, highBound = 49, 77
		if e.narrative.Register.Width < 0.4 {
			lowBound, highBound = 54, 70 // Contracted/Intimate
		} else if e.narrative.Register.Width > 0.8 {
			lowBound, highBound = 44, 82 // Expanded/Tense
		}
	} else {
		lowBound, highBound = 55, 84
		if e.narrative.Register.Width < 0.4 {
			lowBound, highBound = 60, 75 // Contracted/Intimate
		} else if e.narrative.Register.Width > 0.8 {
			lowBound, highBound = 50, 89 // Expanded/Tense
		}
	}

	// Chord anticipation: look ahead at next chord 1-2 ticks early
	isAnticipation := (tickCount%16 == 14) || (tickCount%16 == 15)
	targetChord := chord
	if isAnticipation {
		nextIdx := (e.progress + 1) % len(e.progression)
		targetChord = e.progression[nextIdx]
	}

	// === PHRASE MANAGEMENT ===
	if s.PhraseTicks <= 0 {
		if e.soloistPhraseActive {
			// Phrase ended — hand off spotlight
			e.soloistPhraseActive = false
			e.phraseCounter++

			nextSolIdx := (e.activeSoloistIdx + 1) % len(e.soloists)
			e.activeSoloistIdx = nextSolIdx

			// Shorter breathing space for continuous romantic energy
			s.PauseTicks = rand.Intn(8) + 6
			if e.macroEnergy < 0.4 {
				s.PauseTicks += rand.Intn(6) + 4
			}
			s.Motif = nil
			s.MotifNotes = nil

			// Next soloist takes a short moment to respond
			e.soloists[nextSolIdx].PauseTicks = rand.Intn(6) + 4
			if e.macroEnergy < 0.4 {
				e.soloists[nextSolIdx].PauseTicks += rand.Intn(4) + 2
			}
			e.soloists[nextSolIdx].Motif = nil
			e.soloists[nextSolIdx].MotifNotes = nil
			return
		}

		// Decide: play or rest? (Romantic lounge — keep playing, rare complete rest)
		if rand.Float64() < 0.15 {
			s.PauseTicks = rand.Intn(8) + 6
			nextSolIdx := (e.activeSoloistIdx + 1) % len(e.soloists)
			e.activeSoloistIdx = nextSolIdx
			e.soloists[nextSolIdx].PauseTicks = rand.Intn(4) + 4
			return
		}

		// Start a new phrase with lyrical length bias (romantic lounge needs longer melodic statements)
		e.soloistPhraseActive = true

		// Phrase length profiles: strongly bias toward medium and extended lyrical phrases
		lenProfile := rand.Float64()
		if lenProfile < 0.05 {
			// Rare short exclamation (3-5 notes)
			s.PhraseTicks = rand.Intn(6) + 8
			s.PhraseLength = rand.Intn(3) + 3
		} else if lenProfile < 0.65 {
			// Standard medium lyrical phrase — core of the performance
			s.PhraseTicks = rand.Intn(16) + 16
			s.PhraseLength = rand.Intn(6) + 7
		} else {
			// Extended expressive statement — memorable, humable arc (up to 50 ticks, 14-20 notes)
			s.PhraseTicks = rand.Intn(20) + 30
			s.PhraseLength = rand.Intn(7) + 14
		}

		s.Motif = PhraseMotifs[rand.Intn(len(PhraseMotifs))]
		s.MotifIdx = 0
		s.NoteCount = 0
		s.PhraseEnergy = 0.0

		// Choose improvisation strategy
		rVal := rand.Float64()
		if rVal < 0.35 {
			s.ImprovState = 0 // Stepwise (scale runs)
		} else if rVal < 0.60 {
			s.ImprovState = 1 // Chord arpeggio
		} else if rVal < 0.80 {
			s.ImprovState = 2 // Motif development
		} else {
			s.ImprovState = 3 // Chromatic approach
		}

		// Q&A contour: questions ascend, answers descend
		isQuestion := (e.phraseCounter % 2 == 0)
		mid := (lowBound + highBound) / 2
		if isQuestion {
			s.LastMIDINote = lowBound + rand.Intn(mid-lowBound)
			s.MelodyDir = 1
		} else {
			s.LastMIDINote = mid + rand.Intn(highBound-mid)
			s.MelodyDir = -1
		}
		// Snap starting note to nearest chord tone
		s.LastMIDINote = findNearestChordTone(s.LastMIDINote, chord, keyPitch, lowBound, highBound)

		// Pre-generate motif if needed (Long-Term Ensemble Motif Memory & Character Preservation)
		if s.ImprovState == 2 {
			recalled := false
			if len(e.motifInventory) > 0 && rand.Float64() < 0.20 {
				// Recall a motif from the long-term memory inventory
				bestIdx := 0
				bestImportance := -999.0
				for idx, m := range e.motifInventory {
					if m.Importance > bestImportance {
						bestImportance = m.Importance
						bestIdx = idx
					}
				}
				
				// Fetch and increase importance due to reinforcement
				m := &e.motifInventory[bestIdx]
				m.Importance += 0.35
				recalled = true

				if len(m.Contour) > 0 && rand.Float64() < 0.40 {
					// Character-driven recall: preserve contour character, regenerate chord tones
					s.MotifNotes = make([]int, len(m.Contour)+1)
					s.MotifNotes[0] = s.LastMIDINote
					for idx, dir := range m.Contour {
						step := 1
						if rand.Float64() < 0.3 {
							step = 2
						}
						note := s.MotifNotes[idx] + dir*step
						note = findNearestChordTone(note, chord, keyPitch, lowBound, highBound)
						s.MotifNotes[idx+1] = note
					}
				} else {
					// Literal transposition recall
					motifLen := len(m.Notes)
					s.MotifNotes = make([]int, motifLen)
					transposeShift := s.LastMIDINote - m.Notes[0]
					for idx, val := range m.Notes {
						note := val + transposeShift
						if idx > 0 && rand.Float64() < 0.25 {
							note += rand.Intn(3) - 1 // Shift up or down a step
						}
						if note < lowBound {
							note = lowBound
						}
						if note > highBound {
							note = highBound
						}
						s.MotifNotes[idx] = note
					}
				}
				
				// Sync to short-term conversational motif
				e.sharedMotifNotes = make([]int, len(s.MotifNotes))
				copy(e.sharedMotifNotes, s.MotifNotes)
			}
			
			if !recalled {
				// Generate a new motif, save to short-term and long-term memory
				s.MotifNotes = generateMotifNotes(s.LastMIDINote, chord, keyPitch, s.MelodyDir)
				e.sharedMotifNotes = make([]int, len(s.MotifNotes))
				copy(e.sharedMotifNotes, s.MotifNotes)

				// Calculate contour directions
				contour := make([]int, len(s.MotifNotes)-1)
				for j := 0; j < len(s.MotifNotes)-1; j++ {
					diff := s.MotifNotes[j+1] - s.MotifNotes[j]
					if diff > 0 {
						contour[j] = 1
					} else if diff < 0 {
						contour[j] = -1
					} else {
						contour[j] = 0
					}
				}

				// Copy rhythm ticks
				rhythm := append([]int{}, s.Motif...)

				// Register in long-term motif inventory
				newTheme := ThematicMotif{
					Notes:            append([]int{}, s.MotifNotes...),
					Rhythm:           rhythm,
					Contour:          contour,
					EmotionalQuality: e.narrative.Mood,
					Importance:       1.0,
					SourceInstrument: s.Type,
					AgeTicks:         0,
				}
				e.motifInventory = append(e.motifInventory, newTheme)
			}
			s.MotifNoteIdx = 0
		}

		s.LastLeap = 0
	}

	s.PhraseTicks--

	// Swing feel: higher play probability on downbeats; keep ensemble vitally active
	playProb := 0.32
	if tickCount%2 == 0 {
		playProb = 0.70
	}
	if e.macroEnergy < 0.4 {
		playProb *= 0.85 // slightly more laid back at low energy, never truly silent
	}
	if rand.Float64() > playProb {
		return
	}

	// === NOTE GENERATION ===
	isChordChange := (tickCount % 16 == 0)

	// Update phrase energy arc: builds to climax at 50-55%, then resolves gracefully
	if s.PhraseLength > 0 {
		progress := float64(s.NoteCount) / float64(s.PhraseLength)
		if progress < 0.55 {
			s.PhraseEnergy = progress / 0.55 // build up faster
		} else {
			s.PhraseEnergy = 1.0 - (progress-0.55)/0.45 // sustained climax then resolve
		}
		if s.PhraseEnergy < 0 {
			s.PhraseEnergy = 0
		}
	}

	var nextNote int

	if isChordChange || isAnticipation {
		// Target a guide tone of the upcoming/current chord
		nextNote = findNearestGuideTone(s.LastMIDINote, targetChord, keyPitch, lowBound, highBound)
	} else if s.LastLeap != 0 {
		// Resolve leaps by stepping in the opposite direction
		if s.LastLeap > 0 {
			nextNote = s.LastMIDINote - 1
		} else {
			nextNote = s.LastMIDINote + 1
		}
		s.LastLeap = 0
	} else {
		switch s.ImprovState {
		case 1: // Arpeggio: walk through chord tones
			nextNote = s.LastMIDINote + s.MelodyDir*rand.Intn(3) + s.MelodyDir
			nextNote = findNearestChordTone(nextNote, targetChord, keyPitch, lowBound, highBound)

		case 2: // Motif development
			if len(s.MotifNotes) > 0 {
				nextNote = s.MotifNotes[s.MotifNoteIdx%len(s.MotifNotes)]
				s.MotifNoteIdx++
				// Sequence the motif with dynamic transformations when we loop it
				if s.MotifNoteIdx%len(s.MotifNotes) == 0 {
					rVal := rand.Float64()
					if rVal < 0.40 {
						// 1. Transposition
						shift := rand.Intn(5) - 2 // -2 to +2
						for j := range s.MotifNotes {
							s.MotifNotes[j] += shift
						}
					} else if rVal < 0.60 {
						// 2. Octave Register Shift (shift octave up or down, clamped to range)
						shift := 12
						if rand.Float64() < 0.5 {
							shift = -12
						}
						for j := range s.MotifNotes {
							note := s.MotifNotes[j] + shift
							if note >= lowBound && note <= highBound {
								s.MotifNotes[j] = note
							}
						}
					} else if rVal < 0.80 {
						// 3. Melodic Inversion (mirror intervals around the starting note)
						startNote := s.MotifNotes[0]
						for j := 1; j < len(s.MotifNotes); j++ {
							diff := s.MotifNotes[j] - startNote
							inverted := startNote - diff
							if inverted >= lowBound && inverted <= highBound {
								s.MotifNotes[j] = inverted
							}
						}
					} else {
						// 4. Fragmentation: truncate the motif slightly to develop it
						if len(s.MotifNotes) > 2 {
							s.MotifNotes = s.MotifNotes[:len(s.MotifNotes)-1]
						}
					}
				}
			} else {
				nextNote = s.LastMIDINote + s.MelodyDir
			}

		case 3: // Chromatic approach: approach target from a half-step below
			target := findNearestChordTone(s.LastMIDINote+s.MelodyDir*3, targetChord, keyPitch, lowBound, highBound)
			// Play the chromatic approach note (half step below target)
			nextNote = target - 1
			// Next time we'll land on the target
			s.ImprovState = 1 // switch to arpeggio to land

		default: // Stepwise scale run with occasional direction changes
			if rand.Float64() < 0.15 {
				s.MelodyDir = -s.MelodyDir
			}
			// Step by 1 or 2 semitones with bias toward scale tones (modulated by interval obsession)
			stepSize := 1
			if rand.Float64() < 0.4 {
				stepSize = 2
			}
			if e.narrative.Obsession.Strength > 0 && e.narrative.Obsession.Type == "interval" && rand.Float64() < (0.4*e.narrative.Obsession.Strength) {
				stepSize = e.narrative.Obsession.IntervalVal
			}
			nextNote = s.LastMIDINote + s.MelodyDir*stepSize

			// Prefer landing on scale tones on downbeats
			if tickCount%2 == 0 && !isScaleTone(nextNote, e.key, e.isMinor) {
				nextNote += s.MelodyDir // one more step to land on scale tone
			}
		}
	}

	// Register leaps
	leap := nextNote - s.LastMIDINote
	if !isChordChange && !isAnticipation && int(math.Abs(float64(leap))) >= 5 {
		s.LastLeap = leap
	}

	// Boundary enforcement
	if nextNote < lowBound {
		nextNote = lowBound + rand.Intn(3)
		s.MelodyDir = 1
	} else if nextNote > highBound {
		nextNote = highBound - rand.Intn(3)
		s.MelodyDir = -1
	}

	// Resolve to chord tone on strong downbeats and at phrase end
	isEnding := s.PhraseTicks <= 2
	if (tickCount%4 == 0 || isEnding) && !isChordTone(nextNote, targetChord, keyPitch) {
		nextNote = findNearestChordTone(nextNote, targetChord, keyPitch, lowBound, highBound)
	}

	s.LastMIDINote = nextNote
	s.NoteCount++

	// === RHYTHM ===
	var stepTicks int
	if len(s.Motif) > 0 {
		stepTicks = s.Motif[s.MotifIdx%len(s.Motif)]
		s.MotifIdx++
		// Rhythmic displacement variation: shift duration subtly
		if s.ImprovState == 2 && rand.Float64() < 0.25 {
			displacement := rand.Intn(3) - 1 // -1, 0, or 1
			stepTicks += displacement
			if stepTicks < 1 {
				stepTicks = 1
			}
		}
	} else {
		stepTicks = 2
	}

	// Apply rhythmic obsession
	if e.narrative.Obsession.Strength > 0 && e.narrative.Obsession.Type == "rhythmic_gesture" && rand.Float64() < (0.5 * e.narrative.Obsession.Strength) {
		if e.narrative.Obsession.RhythmVal > 0 {
			stepTicks = e.narrative.Obsession.RhythmVal
		}
	}

	// Articulation
	var noteTicks int
	if stepTicks == 1 {
		noteTicks = 1
	} else if rand.Float64() < 0.20 {
		noteTicks = stepTicks - 1
		if noteTicks < 1 {
			noteTicks = 1
		}
	} else {
		noteTicks = stepTicks
	}

	bpm := e.bpm
	quarterNoteDur := 60.0 / bpm
	duration := float64(noteTicks) * 0.5 * quarterNoteDur

	s.PauseTicks = stepTicks

	// Dynamic velocity — confident, warm, projected
	// Base is 0.55 (audible, present) with up to +0.35 at phrase peak
	baseVel := 0.55 + 0.35*s.PhraseEnergy
	// Add subtle randomness for human feel
	vel := baseVel + (rand.Float64()-0.5)*0.10
	if vel < 0.38 {
		vel = 0.38
	}
	if vel > 0.98 {
		vel = 0.98
	}

	e.playSoloistNoteWithVol(s.Type, nextNote, vel, duration)
}
