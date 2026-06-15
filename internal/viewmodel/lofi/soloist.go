package lofi

import (
	"math"
	"math/rand"
)

type Soloist struct {
	Type         string // "sax" or "trumpet"
	PhraseTicks  int
	PauseTicks   int
	ScaleIdx     int
	LastMIDINote int
	Motif        []int
	MotifIdx     int

	// Composition Fields
	ImprovState  int
	MelodyDir    int
	MotifSteps   []int
	MotifStepIdx int
	LastLeap     int
}

var PhraseMotifs = [][]int{
	{2, 2, 2, 2},       // Quarter notes (straight)
	{1, 1, 1, 1, 1, 1}, // Swing eighth-note run (continuous and flowing)
	{3, 1, 3, 1},       // Dotted swing (long-short)
	{2, 1, 2, 1},       // Syncopated push
	{4, 4, 4},          // Long breathing notes (lyrical)
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
	// Guide tones are 3rd (3/4) and 7th (10/11)
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

func (e *LofiEngine) processSoloist(s *Soloist, tickCount int) {
	if s.PauseTicks > 0 {
		return
	}

	keyPitch := keyToPitch(e.key)
	chord := e.progression[e.progress]

	// Determine if we are at a chord change or anticipating it
	// Chord changes on tickCount % 16 == 0
	// We anticipate 1 tick early (tickCount % 16 == 15)
	isAnticipation := (tickCount%16 == 15)
	targetChord := chord
	if isAnticipation {
		nextIdx := (e.progress + 1) % len(e.progression)
		targetChord = e.progression[nextIdx]
	}

	if s.PhraseTicks <= 0 {
		// Hand off/turn check
		if e.soloistPhraseActive {
			// Soloist phrase ended!
			e.soloistPhraseActive = false
			e.phraseCounter++
			
			// Hand off spotlight to the other soloist
			nextSolIdx := (e.activeSoloistIdx + 1) % len(e.soloists)
			e.activeSoloistIdx = nextSolIdx
			
			// Set a natural breath pause for BOTH soloists
			// The current soloist pauses to rest
			s.PauseTicks = rand.Intn(16) + 12
			s.Motif = nil
			s.MotifSteps = nil
			
			// The next soloist takes a tiny breath (e.g. 2 to 4 ticks) before responding
			e.soloists[nextSolIdx].PauseTicks = rand.Intn(3) + 3
			e.soloists[nextSolIdx].Motif = nil
			e.soloists[nextSolIdx].MotifSteps = nil
			return
		} else {
			// Decide whether to start a phrase or stay silent
			if rand.Float64() < 0.35 {
				// Take a rest and pass spotlight if we were silent
				s.PauseTicks = rand.Intn(8) + 4
				nextSolIdx := (e.activeSoloistIdx + 1) % len(e.soloists)
				e.activeSoloistIdx = nextSolIdx
				e.soloists[nextSolIdx].PauseTicks = rand.Intn(3) + 2
				return
			}

			// Start playing a melodic phrase (8 to 24 ticks = 1 to 3 measures)
			e.soloistPhraseActive = true
			s.PhraseTicks = rand.Intn(16) + 8
			s.Motif = PhraseMotifs[rand.Intn(len(PhraseMotifs))]
			s.MotifIdx = 0

			// Setup Composition State for this phrase:
			// 0: Stepwise scale run, 1: Chord arpeggio, 2: Repeating motif
			rVal := rand.Float64()
			if rVal < 0.50 {
				s.ImprovState = 0
			} else if rVal < 0.80 {
				s.ImprovState = 1
			} else {
				s.ImprovState = 2
				// Generate a melodic motif of 3 to 4 steps
				motifOptions := [][]int{
					{0, 2, -1},
					{0, 1, 0, -1},
					{0, 3, -2, -1},
					{0, -1, 2, -1},
				}
				s.MotifSteps = motifOptions[rand.Intn(len(motifOptions))]
				s.MotifStepIdx = 0
			}

			// Question & Answer melodic contour
			isQuestion := (e.phraseCounter % 2 == 0)
			if isQuestion {
				// Question phrase starts in lower register, goes up
				s.ScaleIdx = rand.Intn(len(e.scale) / 2)
				s.MelodyDir = 1
			} else {
				// Answer phrase starts in higher register, goes down
				s.ScaleIdx = len(e.scale)/2 + rand.Intn(len(e.scale)/2)
				s.MelodyDir = -1
			}
			s.LastLeap = 0
		}
	}

	s.PhraseTicks--

	// Phrasing: notes play mostly on downbeats (even ticks) or with swing feel
	playProb := 0.35
	if tickCount%2 == 0 {
		playProb = 0.65
	}
	if rand.Float64() > playProb {
		return
	}

	// 1. MELODIC STEP GENERATION (Composer Rules)
	var step int
	isChordChange := (tickCount % 16 == 0)

	if isChordChange || isAnticipation {
		// Guide-tone targeting: snap s.ScaleIdx to the nearest guide/chord tone of the target chord
		bestDiff := 999
		bestIdx := s.ScaleIdx
		for idx := 0; idx < len(e.scale); idx++ {
			testNote := e.scale[idx]
			if s.Type == "sax" {
				testNote -= 12
			}
			matchesTarget := false
			if isChordChange {
				matchesTarget = isGuideTone(testNote, targetChord, keyPitch)
			} else {
				matchesTarget = isChordTone(testNote, targetChord, keyPitch)
			}
			if matchesTarget {
				diff := int(math.Abs(float64(idx - s.ScaleIdx)))
				if diff < bestDiff {
					bestDiff = diff
					bestIdx = idx
				}
			}
		}
		s.ScaleIdx = bestIdx
		step = 0 // land on the targeted tone
	} else if s.LastLeap != 0 {
		// Opposite-direction leap resolution: resolve the leap by step in the opposite direction
		if s.LastLeap > 0 {
			step = -1
		} else {
			step = 1
		}
		s.LastLeap = 0
	} else {
		// Standard walk based on improvisation state
		switch s.ImprovState {
		case 1: // Chord arpeggio: walk through chord tones in range
			found := false
			idx := s.ScaleIdx
			for i := 0; i < len(e.scale); i++ {
				idx += s.MelodyDir
				if idx < 0 || idx >= len(e.scale) {
					s.MelodyDir = -s.MelodyDir
					idx = s.ScaleIdx
					continue
				}
				testNote := e.scale[idx]
				if s.Type == "sax" {
					testNote -= 12
				}
				if isChordTone(testNote, targetChord, keyPitch) {
					step = idx - s.ScaleIdx
					found = true
					break
				}
			}
			if !found {
				step = s.MelodyDir
			}
		case 2: // Repeating motif sequencing
			if len(s.MotifSteps) > 0 {
				step = s.MotifSteps[s.MotifStepIdx%len(s.MotifSteps)]
				s.MotifStepIdx++
				// Melodic Sequence: shift start of motif occasionally
				if s.MotifStepIdx%len(s.MotifSteps) == 0 {
					s.ScaleIdx += rand.Intn(3) - 1
				}
			} else {
				step = s.MelodyDir
			}
		default: // Stepwise scale run
			// 80% chance to continue in the same direction, 20% to turn
			if rand.Float64() < 0.20 {
				s.MelodyDir = -s.MelodyDir
			}
			step = s.MelodyDir
		}
	}

	// 2. APPLY LEAP REGISTRATION
	if !isChordChange && !isAnticipation && int(math.Abs(float64(step))) >= 2 {
		s.LastLeap = step
	}

	// 3. APPLY STEP WITH BOUNDARY SAFETY & DIRECTION BOUNCE
	s.ScaleIdx += step
	if s.ScaleIdx < 0 {
		s.ScaleIdx = 0
		s.MelodyDir = 1
	} else if s.ScaleIdx >= len(e.scale) {
		s.ScaleIdx = len(e.scale) - 1
		s.MelodyDir = -1
	}

	// 4. RESOLVE TO Stable Chord Tone Near the End of an Answer Phrase
	note := e.scale[s.ScaleIdx]
	if s.Type == "sax" {
		note -= 12
	}

	isAnswerEnding := (e.phraseCounter%2 == 1) && (s.PhraseTicks <= 2)

	// Resolve to a chord tone on strong downbeats or during an Answer resolution
	if (tickCount%2 == 0 || isAnswerEnding) && !isChordTone(note, targetChord, keyPitch) {
		leftIdx := s.ScaleIdx - 1
		rightIdx := s.ScaleIdx + 1
		resolved := false
		if leftIdx >= 0 {
			leftNote := e.scale[leftIdx]
			if s.Type == "sax" {
				leftNote -= 12
			}
			if isChordTone(leftNote, targetChord, keyPitch) {
				note = leftNote
				s.ScaleIdx = leftIdx
				resolved = true
			}
		}
		if !resolved && rightIdx < len(e.scale) {
			rightNote := e.scale[rightIdx]
			if s.Type == "sax" {
				rightNote -= 12
			}
			if isChordTone(rightNote, targetChord, keyPitch) {
				note = rightNote
				s.ScaleIdx = rightIdx
				resolved = true
			}
		}
	}

	s.LastMIDINote = note

	// Choose stepTicks from motif or fall back
	var stepTicks int
	if len(s.Motif) > 0 {
		stepTicks = s.Motif[s.MotifIdx%len(s.Motif)]
		s.MotifIdx++
	} else {
		stepTicks = 2
	}

	// Choose noteTicks (articulation: staccato vs legato)
	var noteTicks int
	if stepTicks == 1 {
		noteTicks = 1 // Legato eighth notes
	} else {
		if rand.Float64() < 0.25 {
			noteTicks = stepTicks - 1 // slightly detached / staccato
			if noteTicks < 1 {
				noteTicks = 1
			}
		} else {
			noteTicks = stepTicks // legato
		}
	}

	bpm := e.bpm
	quarterNoteDur := 60.0 / bpm
	duration := float64(noteTicks) * 0.5 * quarterNoteDur

	s.PauseTicks = stepTicks

	freq := midiToFreq(note)
	voice := &SynthVoice{
		SampleRate: e.speakerRate,
		Frequency:  freq,
		Amplitude:  e.synthVolLevel * 0.14,
		VoiceType:  s.Type,
		Duration:   duration,
	}
	e.mixer.Add(voice)
}
