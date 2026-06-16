package jazzlounge

import (
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sort"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type Sample struct {
	Name   string
	Buffer *beep.Buffer
	Format beep.Format
}

type JazzLoungeEngine struct {
	assetsPath           string
	speakerRate          beep.SampleRate
	pianoSamples         map[int]*Sample
	pianoSamplesV1       map[int]*Sample
	pianoSamplesV3       map[int]*Sample
	drumSamples          map[string]*Sample
	ambientSounds        []AmbientSound
	tracks               []Track
	isPlaying            bool
	key                  string
	progression          []JazzChord
	scale                []int
	progress             int
	scalePos             int
	kickOff              bool
	snareOff             bool
	hatOff               bool
	melodyDensity        float64
	melodyOff            bool
	barCount             int
	sectionBarLength     int
	isTransitioning      bool
	stopChan             chan struct{}
	mu                   sync.Mutex
	activeCompingPattern []int

	isInitialized  bool
	isMinor        bool
	bpm            float64
	masterVolLevel float64
	pianoVolLevel  float64
	synthVolLevel  float64
	drumsVolLevel  float64
	soloists       []*Soloist
	activeSoloistIdx     int
	soloistPhraseActive  bool
	phraseCounter        int

	mixer     *beep.Mixer
	lpf       *LowPassFilter
	masterVol *effects.Volume
	noiseCtrl *beep.Ctrl

	lastPianoVoicing       []int
	lastBassNote           int
	chordDurationRemaining int
	chordTickCount         int
	macroEnergy            float64
	sharedMotifNotes       []int
	narrative              JazzNarrative
	personalities          MusicianPersonalities
	motifInventory         []ThematicMotif
}

var (
	jazzLoungeEngineInstance *JazzLoungeEngine
	jazzLoungeEngineOnce     sync.Once
)

func GetJazzLoungeEngine() *JazzLoungeEngine {
	jazzLoungeEngineOnce.Do(func() {
		jazzLoungeEngineInstance = &JazzLoungeEngine{
			pianoSamples:     make(map[int]*Sample),
			pianoSamplesV1:   make(map[int]*Sample),
			pianoSamplesV3:   make(map[int]*Sample),
			drumSamples:      make(map[string]*Sample),
			stopChan:         make(chan struct{}),
			key:              "C",
			melodyDensity:    0.33,
			sectionBarLength: 32,
		}
		// Run initialization asynchronously in the background so it doesn't block the UI thread on boot
		go jazzLoungeEngineInstance.init()
	})
	return jazzLoungeEngineInstance
}

func (e *JazzLoungeEngine) init() {
	// Seed the random number generator so each run produces unique melodies and rhythm variations
	rand.Seed(time.Now().UnixNano())

	sr, err := initSpeakerShared()
	if err != nil {
		// Bypassing audio hardware error for headless / test environments
		sr = beep.SampleRate(44100)
	}
	e.speakerRate = sr
	e.assetsPath = getAssetsPathShared()

	// Load Piano Samples
	if !isTestRun() {
		for note, midi := range SampleMIDIMap {
			name := strings.ReplaceAll(note, "#", "sharp")
			
			// Load v1 (Soft)
			fn1 := name + "v1.mp3"
			path1 := filepath.Join(e.assetsPath, "PianoSamples", fn1)
			buf1, format1, err1 := loadSampleShared(path1)
			if err1 == nil {
				e.pianoSamplesV1[midi] = &Sample{
					Name:   note + "v1",
					Buffer: buf1,
					Format: format1,
				}
				// Populate main map for fallback compatibility
				e.pianoSamples[midi] = e.pianoSamplesV1[midi]
			}

			// Load v3 (Loud)
			fn3 := name + "v3.mp3"
			path3 := filepath.Join(e.assetsPath, "PianoSamples", fn3)
			buf3, format3, err3 := loadSampleShared(path3)
			if err3 == nil {
				e.pianoSamplesV3[midi] = &Sample{
					Name:   note + "v3",
					Buffer: buf3,
					Format: format3,
				}
			}

			time.Sleep(15 * time.Millisecond) // Yield CPU to keep TUI responsive
		}

		// Load Drum Samples
		drumFiles := map[string]string{
			"kick":  "kick.mp3",
			"snare": "snare.mp3",
			"hat":   "hat.mp3",
		}
		for name, file := range drumFiles {
			path := filepath.Join(e.assetsPath, "DrumSamples", file)
			buf, format, err := loadSampleShared(path)
			if err == nil {
				e.drumSamples[name] = &Sample{
					Name:   name,
					Buffer: buf,
					Format: format,
				}
			}
			time.Sleep(15 * time.Millisecond) // Yield CPU to keep TUI responsive
		}
	}

	// Default Slow Jazz Settings
	e.bpm = 65.0
	e.macroEnergy = 0.5
	e.personalities = InitDefaultPersonalities()
	e.narrative = JazzNarrative{
		Mood:             "Romantic",
		ActiveChapter:    "RomanticRendezvous",
		ChapterTicksLeft: 200,
		ActiveLeader:     "trumpet",
		LeaderTicksLeft:  32,
		PerceivedTempo:   0.5,
		Forces: EmotionalForces{
			Intimacy:     0.70,
			Melancholy:   0.20,
			Tension:      0.15,
			Confidence:   0.75,
			Nostalgia:    0.35,
			Mystery:      0.25,
			Anticipation: 0.50,
			Warmth:       0.85,
			Momentum:     0.55,
		},
		Taste: RandomHarmonicTaste(),
		Obsession: EnsembleObsession{
			Type: "none",
		},
		Register: RegisterArchitecture{
			Width:  0.60,
			Center: 0.55,
		},
		History: MetaMemory{
			SoloistLeadTicks: 0,
			PianoLeadTicks:   0,
			SilenceTicks:     0,
			TotalPhrases:     0,
			LastRecalledAge:  0,
		},
	}
	e.motifInventory = []ThematicMotif{}
	e.masterVolLevel = 0.8
	e.pianoVolLevel = 0.55
	e.synthVolLevel = 0.72
	e.drumsVolLevel = 0.52

	e.soloists = []*Soloist{
		{Type: "sax", LastMIDINote: 69},
		{Type: "trumpet", LastMIDINote: 76},
	}
	e.activeSoloistIdx = 0
	e.soloistPhraseActive = false
	e.phraseCounter = 0

	e.mixer = &beep.Mixer{}
	e.lpf = &LowPassFilter{Streamer: e.mixer, Cutoff: 2200, Fs: float64(e.speakerRate)}
	e.masterVol = &effects.Volume{Streamer: e.lpf, Base: 2, Volume: linearToVolumeExponent(e.masterVolLevel * 3.5)}

	// Soft tape hiss / white noise
	noiseStr := WhiteNoiseStreamer{}
	noiseLPF := &LowPassFilter{Streamer: noiseStr, Cutoff: 1500, Fs: float64(e.speakerRate)}
	noiseVol := &effects.Volume{Streamer: noiseLPF, Base: 2, Volume: linearToVolumeExponent(0.015)}
	e.noiseCtrl = &beep.Ctrl{Streamer: noiseVol, Paused: true}
	e.mixer.Add(e.noiseCtrl)

	// Only play if speaker is actually initialized successfully
	SpeakerMu.Lock()
	spInit := SpeakerInitialized
	SpeakerMu.Unlock()

	if spInit {
		speaker.Play(e.masterVol)
	}

	e.ambientSounds = []AmbientSound{
		{Name: "Rain", Filename: "rain.mp3", VolumeLevel: 0.5},
		{Name: "Thunder", Filename: "thunder.mp3", VolumeLevel: 0.4},
		{Name: "Campfire", Filename: "fire.mp3", VolumeLevel: 0.5},
		{Name: "Jungle", Filename: "jungle.mp3", VolumeLevel: 0.3},
	}

	e.tracks = []Track{
		{ID: 1, Name: "Wind", Filename: "Wind-Mark_DiAngelo-1940285615.mp3", VolumeLevel: 0.5},
		{ID: 2, Name: "Waves", Filename: "small-waves-onto-the-sand-143040.mp3", VolumeLevel: 0.5},
		{ID: 3, Name: "Night", Filename: "night-ambience-17064.mp3", VolumeLevel: 0.5},
		{ID: 4, Name: "Seagulls", Filename: "urban-seagulls-30068.mp3", VolumeLevel: 0.4},
		{ID: 5, Name: "Office", Filename: "office-ambience-6322.mp3", VolumeLevel: 0.3},
		{ID: 6, Name: "City", Filename: "city-ambience-9272.mp3", VolumeLevel: 0.3},
		{ID: 7, Name: "Server", Filename: "old-server-turning-on-and-off-24540.mp3", VolumeLevel: 0.2},
		{ID: 8, Name: "Train", Filename: "train-to-munich-germany.mp3", VolumeLevel: 0.3},
		{ID: 9, Name: "Underwater", Filename: "underwater-white-noise-46423.mp3", VolumeLevel: 0.4},
	}

	e.generateProgression()

	e.mu.Lock()
	e.isInitialized = true
	e.mu.Unlock()

	go e.run()
}

func (e *JazzLoungeEngine) run() {
	var tickCount int
	for {
		select {
		case <-e.stopChan:
			return
		default:
			e.mu.Lock()
			playing := e.isPlaying
			e.mu.Unlock()

			if !playing {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			e.mu.Lock()

			// 1. Play Chords (comping style)
			// Choose comping patterns dynamically based on soloist activity and chord duration
			if e.chordDurationRemaining <= 0 {
				e.nextChord()
				chord := e.progression[e.progress]
				duration := chord.Duration
				if e.narrative.ActiveChapter == "IntimateNocturne" || e.narrative.ActiveChapter == "BassSoliloquy" {
					duration *= 2
				} else if e.narrative.ActiveChapter == "StillnessAtmosphere" {
					duration *= 4
				} else if e.macroEnergy < 0.4 {
					duration *= 2
				}
				e.chordDurationRemaining = duration
				e.chordTickCount = 0

				if e.isTransitioning {
					e.activeCompingPattern = []int{0}
				} else if e.narrative.ActiveChapter == "StillnessAtmosphere" || e.narrative.ActiveChapter == "BassSoliloquy" {
					e.activeCompingPattern = []int{0}
				} else if e.narrative.ActiveLeader == "none" {
					// Extremely sparse comping when no one leads (restraint)
					e.activeCompingPattern = []int{0}
				} else if e.narrative.ActiveLeader == "piano" {
					// Pianist is leading: active comping storytelling
					if chord.Duration >= 32 {
						activePatterns := [][]int{
							{0, 6, 14, 22},
							{0, 12, 24},
							{2, 8, 16, 26},
							{0, 8, 16, 24},
						}
						e.activeCompingPattern = activePatterns[rand.Intn(len(activePatterns))]
					} else if chord.Duration == 16 {
						activePatterns := [][]int{{0, 3}, {0, 6, 10}, {2, 6, 12}, {3, 11}}
						e.activeCompingPattern = activePatterns[rand.Intn(len(activePatterns))]
					} else if chord.Duration == 8 {
						activePatterns := [][]int{{0, 4}, {2, 6}}
						e.activeCompingPattern = activePatterns[rand.Intn(len(activePatterns))]
					} else {
						e.activeCompingPattern = []int{0}
					}
				} else {
					// A soloist is leading: play warm, supportive comping — lush but not intrusive
					if chord.Duration >= 32 {
						supportPatterns := [][]int{{0, 8, 16}, {0, 12, 20}, {0, 8, 24}, {4, 12, 24}}
						e.activeCompingPattern = supportPatterns[rand.Intn(len(supportPatterns))]
					} else if chord.Duration == 16 {
						supportPatterns := [][]int{{0, 6}, {0, 8}, {2, 8}, {0, 4, 10}}
						e.activeCompingPattern = supportPatterns[rand.Intn(len(supportPatterns))]
					} else {
						e.activeCompingPattern = []int{0, 4}
					}
				}
			}

			for _, hitTick := range e.activeCompingPattern {
				if hitTick == e.chordTickCount {
					e.playChordHit(e.chordTickCount == 0)
					break
				}
			}

			// 2. Play Soloists (Call & Response arrangement)
			// Decrement PauseTicks for both soloists on every tick so time flows naturally
			for _, sol := range e.soloists {
				if sol.PauseTicks > 0 {
					sol.PauseTicks--
				}
			}

			// Run the active soloist if we are not transitioning and a soloist is the leader
			if !e.isTransitioning && (e.narrative.ActiveLeader == "sax" || e.narrative.ActiveLeader == "trumpet") {
				e.processSoloist(e.soloists[e.activeSoloistIdx], tickCount)
			}

			// 2b. Walking Bass Line (plays on every other tick for quarter-note pulse)
			if tickCount%2 == 0 {
				isStillness := e.narrative.ActiveChapter == "StillnessAtmosphere"
				if !isStillness || tickCount%4 == 0 {
					bassNote := e.walkBassLine(tickCount)
					vol := 0.78
					if isStillness {
						vol = 0.52
					}
					e.playBassNoteWithVol(bassNote, vol)
				}
			}

			// 3. Jazz Drums Sequence (Swung Ride, Soft feathering Kick, Soft Snare rimshot & ghost notes)
			if e.isTransitioning {
				step := tickCount % 8
				// Play only soft ride cymbal (hat) on beat 1 and 3 during lowpass DJ sweep transitions
				if (step == 0 || step == 4) && !e.hatOff {
					e.playDrumWithVol("hat", 0.7*e.drumsVolLevel)
				}
			} else if e.narrative.ActiveChapter == "StillnessAtmosphere" {
				// Ethereal stillness: drummer only plays soft brush sweeps (hat) on beat 1 (tickCount % 8 == 0)
				if tickCount%8 == 0 && !e.hatOff {
					e.playDrumWithVol("hat", 0.35*e.drumsVolLevel)
				}
			} else if e.narrative.ActiveChapter == "RomanticRendezvous" {
				// Romantic Rendezvous: warm flowing brush sweeps on every step with energetic swing
				step := tickCount % 8
				if !e.hatOff {
					// Flowing ride pattern: 1 da 2 da 3 da 4 da (all steps with accent on beats)
					volFactor := 0.55 + 0.20*rand.Float64()
					if step == 0 || step == 4 {
						volFactor = 0.80 + 0.10*rand.Float64() // Strong downbeats
					}
					e.playDrumWithVol("hat", volFactor*e.drumsVolLevel)
				}
				if !e.kickOff && (step == 0 || step == 4) && rand.Float64() < 0.70 {
					e.playDrumWithVol("kick", 0.62*e.drumsVolLevel)
				}
				if !e.snareOff {
					if (step == 2 || step == 6) && rand.Float64() < 0.65 {
						e.playDrumWithVol("snare", 0.75*e.drumsVolLevel)
					} else if rand.Float64() < 0.30 {
						e.playDrumWithVol("snare", 0.38*e.drumsVolLevel)
					}
				}
			} else if e.narrative.ActiveChapter == "BassSoliloquy" {
				// Bass spotlight: drummer only plays soft brush sweeps (hat) on beats 2 and 4 (tickCount % 4 == 2)
				step := tickCount % 8
				if (step == 2 || step == 6) && !e.hatOff {
					e.playDrumWithVol("hat", 0.40*e.drumsVolLevel)
				}
			} else {
				// Drummer initiative: occasionally drive tension early
				if e.narrative.Forces.Tension > 0.3 && e.narrative.Forces.Tension < 0.7 && tickCount%16 == 0 && rand.Float64() < 0.20 {
					e.narrative.Forces.Tension += 0.10
					if !e.snareOff {
						e.playDrumWithVol("snare", 0.85*e.drumsVolLevel)
					}
				}

				// Get soloist energy
				energy := 0.0
				hasActiveSoloist := e.soloistPhraseActive
				if hasActiveSoloist {
					energy = e.soloists[e.activeSoloistIdx].PhraseEnergy
				}

				step32 := tickCount % 32
				
				// Only play roll fills if energy is high at the end of a section
				if step32 >= 28 && hasActiveSoloist && energy > 0.6 {
					// Snare roll drum fill on last bar of 4-bar section when energy is high
					e.playDrumWithVol("snare", (0.50+0.40*energy)*e.drumsVolLevel)
					if step32 == 31 {
						// Anticipate downbeat with a kick hit
						e.playDrumWithVol("kick", (0.80+0.20*energy)*e.drumsVolLevel)
					}
				} else if step32 == 0 && tickCount > 0 && hasActiveSoloist && energy > 0.5 {
					// Landing crash accent at the start of next section
					e.playDrumWithVol("snare", (0.88+0.12*energy)*e.drumsVolLevel)
					e.playDrumWithVol("kick", (0.92+0.08*energy)*e.drumsVolLevel)
					if !e.hatOff {
						e.playDrumWithVol("hat", (0.88+0.12*energy)*e.drumsVolLevel)
					}
				} else {
					// Standard drum patterns based on soloist presence and energy
					step := tickCount % 8
					
					if !hasActiveSoloist || energy < 0.25 {
						// Laid back but not dead: warm timekeeping
						// Hat: hi-hat on beats 2 and 4 plus soft taps on 1 and 3
						if !e.hatOff {
							if step == 2 || step == 6 {
								e.playDrumWithVol("hat", 0.72*e.drumsVolLevel)
							} else if (step == 0 || step == 4) && rand.Float64() < 0.45 {
								// Warmer tap on 1 and 3
								e.playDrumWithVol("hat", 0.52*e.drumsVolLevel)
							} else if (step == 1 || step == 3 || step == 5 || step == 7) && rand.Float64() < 0.20 {
								// Subtle off-beat presence
								e.playDrumWithVol("hat", 0.35*e.drumsVolLevel)
							}
						}

						// Kick: soft feathering on downbeats, more present than before
						if !e.kickOff {
							if step == 0 && rand.Float64() < 0.30 {
								e.playDrumWithVol("kick", 0.62*e.drumsVolLevel)
							} else if step == 4 && rand.Float64() < 0.15 {
								e.playDrumWithVol("kick", 0.50*e.drumsVolLevel)
							}
						}

						// Snare: gentle ghost note presence
						if !e.snareOff {
							if (step == 2 || step == 6) && rand.Float64() < 0.22 {
								e.playDrumWithVol("snare", 0.58*e.drumsVolLevel)
							}
						}
					} else {
						// Active soloist, drums support and dynamic matching
						// Ride swing: 1  .  2  da 3  .  4  da
						if !e.hatOff {
							if step == 0 || step == 2 || step == 3 || step == 4 || step == 6 || step == 7 {
								volFactor := 0.65 + 0.25*energy
								if step == 0 || step == 3 || step == 4 || step == 7 {
									volFactor += 0.2 // Accent on downbeats
								}
								e.playDrumWithVol("hat", volFactor*e.drumsVolLevel)
							}
						}

						// Kick: feathering kick on beats 1 and 3 (ticks 0 and 4)
						if !e.kickOff {
							if (step == 0 || step == 4) && rand.Float64() < (0.3+0.6*energy) {
								e.playDrumWithVol("kick", (0.55+0.25*energy)*e.drumsVolLevel)
							}
						}

						// Snare: rimshot on beats 2 and 4, ghost notes on offbeats
						if !e.snareOff {
							if (step == 2 || step == 6) && rand.Float64() < (0.2+0.6*energy) {
								e.playDrumWithVol("snare", (0.70+0.20*energy)*e.drumsVolLevel)
							}
							// Ghost notes
							if (step == 1 || step == 3 || step == 5 || step == 7) && rand.Float64() < (0.05+0.2*energy) {
								e.playDrumWithVol("snare", 0.35*e.drumsVolLevel)
							}
						}
					}
				}
			}

			tickCount++
			e.chordTickCount++
			e.chordDurationRemaining--
			e.updateNarrative(tickCount)
			bpm := e.bpm
			e.mu.Unlock()

			// Calculate swing-adjusted sleep duration
			quarterNoteDur := 60.0 / bpm
			var sleepFactor float64
			if tickCount%2 == 0 {
				sleepFactor = 0.38
			} else {
				sleepFactor = 0.62
			}
			sleepDur := time.Duration(quarterNoteDur * sleepFactor * float64(time.Second))
			time.Sleep(sleepDur)
		}
	}
}

func (e *JazzLoungeEngine) generateProgression() {
	e.key = KeysList[rand.Intn(len(KeysList))]

	prog, isMinor := GenerateDynamicProgression()
	e.progression = prog
	e.isMinor = isMinor
	e.scale = getJazzScale(e.key, e.isMinor)

	e.progress = 0
	e.sharedMotifNotes = nil
	if len(e.progression) > 0 {
		e.chordDurationRemaining = e.progression[0].Duration
	} else {
		e.chordDurationRemaining = 16
	}
	e.chordTickCount = 0
	e.scalePos = len(e.scale) / 2
	keyPitch := keyToPitch(e.key)
	for i := range e.soloists {
		var lowBound int
		if e.soloists[i].Type == "sax" {
			lowBound = 49
		} else {
			lowBound = 55
		}
		e.soloists[i].LastMIDINote = findNearestChordTone(lowBound+12, e.progression[0], keyPitch, lowBound, lowBound+28)
		e.soloists[i].Motif = nil
		e.soloists[i].MotifIdx = 0
		e.soloists[i].MotifNotes = nil
		e.soloists[i].MotifNoteIdx = 0
		e.soloists[i].ImprovState = 0
		e.soloists[i].MelodyDir = 1
		e.soloists[i].LastLeap = 0
		e.soloists[i].PhraseEnergy = 0
		e.soloists[i].NoteCount = 0
		e.soloists[i].PhraseLength = 0
	}
}

func (e *JazzLoungeEngine) nextChord() {
	nextProgress := e.progress + 1
	if nextProgress >= len(e.progression) {
		nextProgress = 0
	}

	nextKickOff := rand.Float64() < 0.07  // rare kick dropout
	nextSnareOff := rand.Float64() < 0.08 // rare snare dropout
	nextHatOff := rand.Float64() < 0.08   // rare hat dropout
	nextMelodyDensity := rand.Float64()*0.25 + 0.50 // melody density: 0.50–0.75 (rich floor)
	nextMelodyOff := rand.Float64() < 0.08 // rarely silence melody

	if e.progress == len(e.progression)/2 {
		e.progress = nextProgress
		e.kickOff = nextKickOff
		e.snareOff = nextSnareOff
		e.hatOff = nextHatOff
	} else if e.progress == 0 {
		e.progress = nextProgress
		e.kickOff = nextKickOff
		e.snareOff = nextSnareOff
		e.hatOff = nextHatOff
		e.melodyDensity = nextMelodyDensity
		e.melodyOff = nextMelodyOff
	} else {
		e.progress = nextProgress
	}

	e.barCount++
	if e.barCount >= e.sectionBarLength {
		e.barCount = 0
		e.autoDJTransition()
		barLengthOptions := []int{16, 20, 24, 28, 32}
		e.sectionBarLength = barLengthOptions[rand.Intn(len(barLengthOptions))]
	}
}

func (e *JazzLoungeEngine) autoDJTransition() {
	if e.isTransitioning {
		return
	}
	e.isTransitioning = true

	e.generateProgression()

	e.melodyDensity = 0.45 + rand.Float64()*0.30 // 0.45-0.75 density after transition
	e.kickOff = rand.Float64() < 0.07
	e.snareOff = rand.Float64() < 0.08
	e.hatOff = rand.Float64() < 0.08
	e.melodyOff = rand.Float64() < 0.08

	// Cozy lowpass filter sweep transition
	go func() {
		steps := 40
		sleepDur := 40 * time.Millisecond
		startCutoff := e.lpf.Cutoff
		targetCutoff := 400.0
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			e.lpf.Cutoff = startCutoff + t*(targetCutoff-startCutoff)
			time.Sleep(sleepDur)
		}
		startCutoff = 400.0
		targetCutoff = 2200.0
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			e.lpf.Cutoff = startCutoff + t*(targetCutoff-startCutoff)
			time.Sleep(sleepDur)
		}
		e.lpf.Cutoff = 2200.0
		e.isTransitioning = false
	}()
}

func (e *JazzLoungeEngine) GenerateVoiceLedVoicing(chord JazzChord, keyPitch int, size int) []int {
	rootMIDI := 48 + keyPitch + chord.RootOffset
	var candidates []int
	lowBound := 52
	highBound := 76
	if e.narrative.Register.Width < 0.4 {
		lowBound = 56
		highBound = 72
	} else if e.narrative.Register.Width > 0.8 {
		lowBound = 48
		highBound = 80
	}

	for _, interval := range chord.Intervals {
		note := rootMIDI + interval
		for note < lowBound {
			note += 12
		}
		for note > highBound {
			note -= 12
		}
		candidates = append(candidates, note)
		if note-12 >= lowBound {
			candidates = append(candidates, note-12)
		}
		if note+12 <= highBound {
			candidates = append(candidates, note+12)
		}
	}

	uniqueMap := make(map[int]bool)
	var uniqueCandidates []int
	for _, c := range candidates {
		if !uniqueMap[c] {
			uniqueMap[c] = true
			uniqueCandidates = append(uniqueCandidates, c)
		}
	}
	candidates = uniqueCandidates

	if len(e.lastPianoVoicing) == 0 {
		sort.Ints(candidates)
		var voicing []int
		if len(candidates) >= size {
			voicing = candidates[:size]
		} else {
			voicing = candidates
		}
		e.lastPianoVoicing = voicing
		return voicing
	}

	var bestVoicing []int
	bestDist := 999999

	var search func(start int, current []int)
	search = func(start int, current []int) {
		if len(current) == size {
			dist := 0
			sortedCurrent := make([]int, size)
			copy(sortedCurrent, current)
			sort.Ints(sortedCurrent)
			
			for i := 0; i < size; i++ {
				d := int(math.Abs(float64(sortedCurrent[i] - e.lastPianoVoicing[i])))
				dist += d
				if i > 0 && sortedCurrent[i] == sortedCurrent[i-1] {
					dist += 1000
				}
			}
			if dist < bestDist {
				bestDist = dist
				bestVoicing = sortedCurrent
			}
			return
		}
		for i := start; i < len(candidates); i++ {
			search(i+1, append(current, candidates[i]))
		}
	}

	search(0, []int{})

	if len(bestVoicing) == 0 {
		sort.Ints(candidates)
		if len(candidates) >= size {
			bestVoicing = candidates[:size]
		} else {
			bestVoicing = candidates
		}
	}

	e.lastPianoVoicing = bestVoicing
	return bestVoicing
}

func (e *JazzLoungeEngine) playChordHit(isDownbeat bool) {
	chord := e.progression[e.progress]
	keyPitch := keyToPitch(e.key)

	// 20% chance to play a thematic motif echo instead of block chord on offbeats
	// This creates melodic commentary from the piano even during accompaniment
	if !isDownbeat && len(e.motifInventory) > 0 && rand.Float64() < 0.25 {
		// Find highest importance motif
		bestIdx := 0
		bestImportance := -999.0
		for idx, m := range e.motifInventory {
			if m.Importance > bestImportance {
				bestImportance = m.Importance
				bestIdx = idx
			}
		}
		motif := e.motifInventory[bestIdx]
		if len(motif.Notes) > 0 {
			// Echo the first note of the motif in the upper piano register (decorative fill)
			note := motif.Notes[0]
			for note < 56 {
				note += 12
			}
			for note > 80 {
				note -= 12
			}
			e.playPianoNoteWithVol(note, e.pianoVolLevel*0.42)
			return
		}
	}

	voicing := e.GenerateVoiceLedVoicing(chord, keyPitch, 4)
	vol := e.pianoVolLevel * 0.88
	if !isDownbeat {
		vol *= 0.78
	}

	for _, noteMIDI := range voicing {
		e.playPianoNoteWithVol(noteMIDI, vol)
	}

	// After chord hit, occasionally add a decorative upper-register fill or passing tone
	// This makes the piano feel alive and elegant even during accompaniment
	if rand.Float64() < 0.28 {
		// Pick a chord extension or passing tone in the upper register
		kp := keyToPitch(e.key)
		chordRoot := 60 + kp + chord.RootOffset
		ext := []int{14, 16, 21, 9}[rand.Intn(4)] // 9th, major 9th, 13th, 6th
		fillNote := chordRoot + ext
		for fillNote > 84 {
			fillNote -= 12
		}
		for fillNote < 64 {
			fillNote += 12
		}
		e.playPianoNoteWithVol(fillNote, e.pianoVolLevel*0.38)
	}
}

func (e *JazzLoungeEngine) walkBassLine(tickCount int) int {
	kp := keyToPitch(e.key)
	chord := e.progression[e.progress]
	
	// Default base octave is C2 (36 + kp)
	bassRoot := 36 + kp + chord.RootOffset
	
	// Keep bass note in a reasonable range (dynamically scaled)
	lowLimit := 31
	highLimit := 57
	if e.narrative.ActiveChapter == "BassSoliloquy" {
		lowLimit = 41
		highLimit = 67 // Tenor register for bass melodic spotlight
	} else if e.narrative.Register.Width < 0.4 {
		lowLimit = 36
		highLimit = 48
	} else if e.narrative.Register.Width > 0.8 {
		lowLimit = 28
		highLimit = 60
	}

	clampNote := func(note int) int {
		for note < lowLimit {
			note += 12
		}
		for note > highLimit {
			note -= 12
		}
		return note
	}

	// Dynamic melodic step movement during BassSoliloquy
	if e.narrative.ActiveChapter == "BassSoliloquy" && e.lastBassNote > 0 && rand.Float64() < 0.50 {
		step := []int{-2, -1, 1, 2, 3}[rand.Intn(5)]
		return clampNote(e.lastBassNote + step)
	}

	// 1. Pedal Point during Transitions or very quiet moments
	isPedalPoint := e.isTransitioning || e.narrative.ActiveChapter == "StillnessAtmosphere"
	if !isPedalPoint && !e.soloistPhraseActive && rand.Float64() < 0.12 {
		isPedalPoint = true // Bass walks 88% of the time when no soloist is active
	}

	if isPedalPoint {
		// Drone on tonic (36 + kp) or dominant (43 + kp)
		pedalTonic := clampNote(36 + kp)
		pedalDominant := clampNote(43 + kp)
		beatIdx := e.chordTickCount / 2
		if beatIdx%4 == 0 || beatIdx%4 == 2 {
			e.lastBassNote = pedalTonic
			return pedalTonic
		} else {
			// On beats 2 & 4, either repeat tonic, drop down an octave, or play dominant
			if rand.Float64() < 0.5 {
				e.lastBassNote = pedalDominant
				return pedalDominant
			}
			e.lastBassNote = pedalTonic
			return pedalTonic
		}
	}

	beatIdx := e.chordTickCount / 2
	totalBeats := chord.Duration / 2
	if totalBeats <= 0 {
		totalBeats = 8
	}

	// 2. Downbeat of a chord change: play root or inversion
	if beatIdx == 0 {
		if e.lastBassNote == 0 || rand.Float64() < 0.75 {
			note := clampNote(bassRoot)
			e.lastBassNote = note
			return note
		} else {
			// Play the 3rd or 5th of the chord (smooth step-wise transition)
			intervals := []int{3, 4, 7} // minor/major 3rd, 5th
			var bestNote int
			bestDist := 999
			for _, inv := range intervals {
				candidate := clampNote(bassRoot + inv)
				dist := int(math.Abs(float64(candidate - e.lastBassNote)))
				if dist < bestDist {
					bestDist = dist
					bestNote = candidate
				}
			}
			e.lastBassNote = bestNote
			return bestNote
		}
	}

	nextIdx := (e.progress + 1) % len(e.progression)
	nextChord := e.progression[nextIdx]
	nextRoot := clampNote(36 + kp + nextChord.RootOffset)

	// 3. Last beat: Chromatic or dominant approach note to the next chord root
	if beatIdx == totalBeats-1 {
		var approachNote int
		if rand.Float64() < 0.6 {
			// Chromatic approach (half step above or below next root)
			if e.lastBassNote < nextRoot {
				approachNote = nextRoot - 1
			} else {
				approachNote = nextRoot + 1
			}
		} else {
			// Dominant approach (fifth above next root)
			approachNote = nextRoot + 7
		}
		note := clampNote(approachNote)
		e.lastBassNote = note
		return note
	}

	// 4. Intermediate beats: walk step-wise with contrary motion bias
	// Find active soloist's direction to apply contrary motion
	soloistDir := 0
	if e.soloistPhraseActive {
		sol := e.soloists[e.activeSoloistIdx]
		soloistDir = sol.MelodyDir // +1 or -1
	}

	// Generate step candidates
	candidates := []int{
		e.lastBassNote + 1, e.lastBassNote - 1, // half step
		e.lastBassNote + 2, e.lastBassNote - 2, // whole step
		e.lastBassNote + 3, e.lastBassNote - 3, // minor third
		e.lastBassNote + 4, e.lastBassNote - 4, // major third
		e.lastBassNote + 5, e.lastBassNote - 5, // perfect fourth
	}

	var bestNote int = clampNote(bassRoot + 7) // fallback
	bestScore := -9999.0

	for _, cand := range candidates {
		candClamped := clampNote(cand)
		score := 0.0

		// Avoid repeating the last note unless we want dynamic pedal rhythm (handled elsewhere)
		if candClamped == e.lastBassNote {
			score -= 5.0
		}

		// Keep steps small (prefer 1 or 2 semitones)
		stepDist := int(math.Abs(float64(candClamped - e.lastBassNote)))
		if stepDist <= 2 {
			score += 3.0
		} else if stepDist <= 4 {
			score += 1.0
		} else if stepDist > 5 {
			score -= 3.0
		}

		// Prefer chord tones
		if isChordTone(candClamped, chord, kp) {
			score += 2.0
		}

		// Contrary motion: if soloist is ascending, prefer descending bass
		movementDir := 1
		if candClamped < e.lastBassNote {
			movementDir = -1
		}
		if soloistDir != 0 && movementDir == -soloistDir {
			score += 1.5
		}

		// Add subtle variation
		score += rand.Float64() * 0.5

		if score > bestScore {
			bestScore = score
			bestNote = candClamped
		}
	}

	e.lastBassNote = bestNote
	return bestNote
}

func (e *JazzLoungeEngine) updateNarrative(tickCount int) {
	// Chapter Ticks Management
	e.narrative.ChapterTicksLeft--
	if e.narrative.ChapterTicksLeft <= 0 {
		var next string
		r := rand.Float64()
		if r < 0.48 {
			next = "RomanticRendezvous"
		} else if r < 0.76 {
			next = "SoloSpotlight"
		} else if r < 0.88 {
			next = "PianoInterlude"
		} else if r < 0.94 {
			next = "IntimateNocturne"
		} else if r < 0.97 {
			next = "BassSoliloquy"
		} else if r < 0.99 {
			next = "StillnessAtmosphere"
		} else {
			next = "NocturnalSuspense"
		}

		if next == e.narrative.ActiveChapter {
			if rand.Float64() < 0.5 {
				next = "RomanticRendezvous"
			} else {
				next = "SoloSpotlight"
			}
		}

		e.narrative.ActiveChapter = next
		e.narrative.ChapterTicksLeft = 160 + rand.Intn(160) // 160-320 ticks per chapter
	}

	// 1. Update Meta Memory Statistics
	if e.isTransitioning {
		e.narrative.History.SilenceTicks++
	} else if e.narrative.ActiveLeader == "none" {
		e.narrative.History.SilenceTicks++
	} else if e.narrative.ActiveLeader == "piano" {
		e.narrative.History.PianoLeadTicks++
	} else {
		e.narrative.History.SoloistLeadTicks++
	}

	// 2. Modulate Persistent Ensemble Mood (Very Slow Shift)
	if tickCount%400 == 0 {
		// Romantic lounge mood palette: warmth-forward, mystery-reduced
		moods := []string{"Romantic", "Romantic", "Elegant", "Playful", "Nostalgic", "Introspective", "Elegant", "Mysterious"}
		e.narrative.Mood = moods[rand.Intn(len(moods))]
	}

	// 3. Update Interacting Emotional Forces
	forces := &e.narrative.Forces
	chord := e.progression[e.progress]

	// Melancholy: minor tonality raises it slightly, but warmth actively counters it
	if e.isMinor {
		forces.Melancholy += 0.012
	} else {
		forces.Melancholy -= 0.025 // Major key actively dissolves melancholy
	}
	if e.narrative.Mood == "Melancholic" || e.narrative.Mood == "Weary" {
		forces.Melancholy += 0.015
	}
	// Warmth actively reduces melancholy — romance over isolation
	if forces.Warmth > 0.6 {
		forces.Melancholy -= 0.020
	}

	// Tension
	if chord.Name == "7alt" || chord.Name == "7" {
		forces.Tension += 0.04
	} else if chord.Name == "maj9" || chord.Name == "maj7" {
		forces.Tension -= 0.035
	}
	if e.soloistPhraseActive {
		sol := e.soloists[e.activeSoloistIdx]
		forces.Tension += 0.02 * sol.PhraseEnergy
	} else {
		forces.Tension -= 0.015
	}

	// Intimacy & Warmth: social engagement sustains warmth even during active playing
	if e.narrative.ActiveLeader == "none" || e.isTransitioning {
		forces.Intimacy += 0.02
		forces.Warmth += 0.025 // Warmth rises during quiet moments
	} else {
		forces.Intimacy -= 0.015
		forces.Warmth += 0.012 // Warmth also rises during active playing in romantic context
	}
	if forces.Tension > 0.6 {
		forces.Intimacy -= 0.018
	}

	// Momentum
	if e.soloistPhraseActive {
		forces.Momentum += 0.025
	} else {
		forces.Momentum -= 0.02
	}

	// Mystery
	if e.narrative.Taste.Style == "AmbiguousChromatic" {
		forces.Mystery += 0.02
	} else {
		forces.Mystery -= 0.01
	}
	if e.narrative.Mood == "Mysterious" {
		forces.Mystery += 0.025
	}

	// Clamp forces [0, 1]
	clamp := func(v float64) float64 {
		if v < 0.0 {
			return 0.0
		}
		if v > 1.0 {
			return 1.0
		}
		return v
	}
	forces.Melancholy = clamp(forces.Melancholy)
	forces.Tension = clamp(forces.Tension)
	forces.Intimacy = clamp(forces.Intimacy)
	forces.Warmth = clamp(forces.Warmth)
	forces.Momentum = clamp(forces.Momentum)
	forces.Mystery = clamp(forces.Mystery)

	// 4. Update Register Architecture (Center and Width)
	// Center migrates slowly over long range
	e.narrative.Register.Center = 0.5 + 0.2*math.Sin(float64(tickCount)/500.0)
	if e.narrative.Mood == "Introspective" || e.narrative.Mood == "Weary" {
		e.narrative.Register.Center -= 0.15 // Sinks lower
	}
	if e.narrative.Register.Center < 0.1 {
		e.narrative.Register.Center = 0.1
	}
	if e.narrative.Register.Center > 0.9 {
		e.narrative.Register.Center = 0.9
	}

	// Width responds to tension and intimacy
	e.narrative.Register.Width = 0.3 + 0.65*forces.Tension - 0.25*forces.Intimacy
	if e.narrative.Register.Width < 0.15 {
		e.narrative.Register.Width = 0.15
	}
	if e.narrative.Register.Width > 1.0 {
		e.narrative.Register.Width = 1.0
	}

	// 5. Manage Ensemble Politics (Leadership & Meta-Memory Pressures)
	e.narrative.LeaderTicksLeft--
	if e.narrative.LeaderTicksLeft <= 0 {
		// Calculate pressures to balance leadership
		pianoPressure := 0.22
		soloistPressure := 0.55
		silencePressure := 0.15 // Romantic lounge: ensemble almost always actively plays
		warmthBonus := forces.Warmth * 0.08
		soloistPressure += warmthBonus
		silencePressure -= warmthBonus * 0.5

		// Avoid over-dominant soloist
		if e.narrative.History.SoloistLeadTicks > e.narrative.History.PianoLeadTicks*2 {
			pianoPressure += 0.15
			soloistPressure -= 0.15
		}
		// Avoid perpetual clutter
		if forces.Tension > 0.7 {
			silencePressure += 0.20
			soloistPressure -= 0.15
		}

		r := rand.Float64()
		var chosen string
		if r < pianoPressure {
			chosen = "piano"
		} else if r < pianoPressure+soloistPressure {
			// Choose sax or trumpet
			if rand.Float64() < 0.5 {
				chosen = "sax"
				e.activeSoloistIdx = 0
			} else {
				chosen = "trumpet"
				e.activeSoloistIdx = 1
			}
		} else {
			chosen = "none"
		}

		e.narrative.ActiveLeader = chosen
		e.narrative.LeaderTicksLeft = 60 + rand.Intn(60) // 60-120 ticks per leader slot
	}

	// Chapter Overrides of Leader, Register, and Emotional Forces
	switch e.narrative.ActiveChapter {
	case "IntimateNocturne":
		e.narrative.ActiveLeader = "piano"
		e.narrative.Register.Center = 0.30
		e.narrative.Register.Width = 0.25
		forces.Intimacy = 0.85
		forces.Tension = 0.15
		forces.Momentum = 0.10
	case "StillnessAtmosphere":
		e.narrative.ActiveLeader = "none"
		e.narrative.Register.Center = 0.45
		e.narrative.Register.Width = 0.18
		forces.Intimacy = 0.95
		forces.Tension = 0.05
		forces.Momentum = 0.05
	case "BassSoliloquy":
		e.narrative.ActiveLeader = "bass"
		e.narrative.Register.Center = 0.38
		e.narrative.Register.Width = 0.30
		forces.Intimacy = 0.75
		forces.Tension = 0.20
		forces.Momentum = 0.30
	case "PianoInterlude":
		e.narrative.ActiveLeader = "piano"
		e.narrative.Register.Center = 0.65
		e.narrative.Register.Width = 0.70
		forces.Intimacy = 0.50
	case "NocturnalSuspense":
		e.narrative.ActiveLeader = "none"
		e.narrative.Register.Center = 0.68
		e.narrative.Register.Width = 0.35
		forces.Tension = 0.75
		forces.Intimacy = 0.30
		forces.Mystery = 0.85
	case "RomanticRendezvous":
		// Highlight warm, lyrical soloist melody with conversational piano and continuous brush sweeps
		if e.narrative.LeaderTicksLeft <= 0 || e.narrative.ActiveLeader == "none" || e.narrative.ActiveLeader == "bass" {
			leaders := []string{"sax", "trumpet", "piano"}
			e.narrative.ActiveLeader = leaders[rand.Intn(len(leaders))]
		}
		e.narrative.Register.Center = 0.45
		e.narrative.Register.Width = 0.65
		forces.Intimacy = 0.80
		forces.Warmth = 0.90
		forces.Confidence = 0.75
		forces.Tension = 0.15
		forces.Momentum = 0.50
	}

	// 6. Obsessions & Fixations
	if e.narrative.Obsession.Strength > 0 {
		e.narrative.Obsession.Strength -= 0.003
	} else {
		if rand.Float64() < 0.02 {
			types := []string{"interval", "rhythmic_gesture", "register_area"}
			t := types[rand.Intn(len(types))]
			obs := EnsembleObsession{
				Type:     t,
				Strength: 0.7 + 0.3*rand.Float64(),
			}
			if t == "interval" {
				obs.IntervalVal = []int{3, 5, 7, 8, 9}[rand.Intn(5)] // minor 3rd, 4th, 5th, minor 6th, major 6th
			} else if t == "rhythmic_gesture" {
				obs.RhythmVal = []int{1, 3, 4}[rand.Intn(3)] // short swing, dotted swing, quarter notes
			} else {
				obs.RegisterCenter = 40.0 + 35.0*rand.Float64() // low, middle, high
			}
			e.narrative.Obsession = obs
		}
	}

	// 7. Perceived Tempo & Macro Energy
	// Warmer, more romantic energy floor — ensemble always has vitality
	e.narrative.PerceivedTempo = 0.35 + 0.35*forces.Tension + 0.20*forces.Momentum
	if forces.Warmth > 0.6 {
		e.narrative.PerceivedTempo += 0.08 // Warmth adds buoyancy, not heaviness
	}
	if forces.Intimacy > 0.6 {
		e.narrative.PerceivedTempo -= 0.10 // Intimacy still mellows tempo
	}
	if e.narrative.PerceivedTempo < 0.18 {
		e.narrative.PerceivedTempo = 0.18
	}
	if e.narrative.PerceivedTempo > 0.90 {
		e.narrative.PerceivedTempo = 0.90
	}
	e.macroEnergy = e.narrative.PerceivedTempo

	// 8. Decay Motif Inventory Memories
	for i := len(e.motifInventory) - 1; i >= 0; i-- {
		e.motifInventory[i].AgeTicks++
		e.motifInventory[i].Importance -= 0.002
		if e.motifInventory[i].Importance <= 0 || e.motifInventory[i].AgeTicks > 1200 {
			e.motifInventory = append(e.motifInventory[:i], e.motifInventory[i+1:]...)
		}
	}
}

func (e *JazzLoungeEngine) playMelody() {
	// Replaced by processSoloist()
}

func (e *JazzLoungeEngine) playNote(midiNote int) {
	// Replaced by processSoloist()
}

func (e *JazzLoungeEngine) playPianoNoteWithVol(midiNote int, volume float64) {
	sample, dist := e.FindClosestSample(midiNote, volume)
	if sample == nil {
		return
	}
	ratio := math.Pow(2.0, float64(dist)/12.0)
	streamer := sample.Buffer.Streamer(0, sample.Buffer.Len())
	effSourceRate := beep.SampleRate(float64(sample.Format.SampleRate) * ratio)
	resampled := beep.Resample(3, effSourceRate, e.speakerRate, streamer)

	volStreamer := &effects.Volume{
		Streamer: resampled,
		Base:     2,
		Volume:   linearToVolumeExponent(volume),
	}

	// Humanization delay (0 - 10ms)
	delayMs := 1.0 + rand.Float64()*9.0
	delaySamples := int((delayMs / 1000.0) * float64(e.speakerRate))

	// Spatial: Piano is placed wide-left (PanL = 0.85, PanR = 0.35)
	panL := 0.85
	panR := 0.35

	// Dynamic mix adjustment if pianist is leading
	if e.narrative.ActiveLeader == "piano" {
		panL = 0.90
		panR = 0.45
	}

	// Early reflection (reflection delay = 22ms, gain = 0.38)
	reflectDelaySamples := int(0.022 * float64(e.speakerRate))
	reflectGain := 0.38 - 0.15*e.narrative.Forces.Intimacy // dry-up if highly intimate

	spatial := NewSpatialHumanizedStreamer(volStreamer, delaySamples, panL, panR, reflectDelaySamples, reflectGain)
	e.mixer.Add(spatial)
}

func (e *JazzLoungeEngine) playBassNoteWithVol(midiNote int, volume float64) {
	freq := midiToFreq(midiNote)
	duration := 1.6 + rand.Float64()*0.8 // long natural decay for double bass

	voice := &BassVoice{
		SampleRate: e.speakerRate,
		Frequency:  freq,
		Amplitude:  e.pianoVolLevel * 0.95, // relative balance
		Duration:   duration,
		Velocity:   volume,
	}

	// Humanization delay (0 - 9ms)
	delayMs := rand.Float64() * 9.0
	delaySamples := int((delayMs / 1000.0) * float64(e.speakerRate))

	// Spatial positioning: Bass is center-low, slightly left (PanL = 0.72, PanR = 0.68)
	panL := 0.72
	panR := 0.68

	// Early reflection (reflection delay = 15ms, gain = 0.28)
	reflectDelaySamples := int(0.015 * float64(e.speakerRate))
	reflectGain := 0.28 - 0.10*e.narrative.Forces.Intimacy

	spatial := NewSpatialHumanizedStreamer(voice, delaySamples, panL, panR, reflectDelaySamples, reflectGain)
	e.mixer.Add(spatial)
}

func (e *JazzLoungeEngine) playDrum(name string) {
	// Replaced by playDrumWithVol
}

func (e *JazzLoungeEngine) playDrumWithVol(name string, volume float64) {
	sample, ok := e.drumSamples[name]
	if !ok {
		return
	}
	streamer := sample.Buffer.Streamer(0, sample.Buffer.Len())
	resampled := beep.Resample(3, sample.Format.SampleRate, e.speakerRate, streamer)

	volStreamer := &effects.Volume{
		Streamer: resampled,
		Base:     2,
		Volume:   linearToVolumeExponent(volume),
	}

	// Humanization delay (0 - 8ms)
	delayMs := rand.Float64() * 8.0
	delaySamples := int((delayMs / 1000.0) * float64(e.speakerRate))

	// Spatial positioning:
	// Kick: center (PanL = 0.70, PanR = 0.70)
	// Snare: slightly left (PanL = 0.65, PanR = 0.55)
	// Hat/Cymbal: slightly right (PanL = 0.40, PanR = 0.80)
	panL := 0.70
	panR := 0.70
	if name == "snare" {
		panL = 0.65
		panR = 0.55
	} else if name == "hat" {
		panL = 0.40
		panR = 0.80
	}

	// Early reflection (reflection delay = 25ms, gain = 0.32)
	reflectDelaySamples := int(0.025 * float64(e.speakerRate))
	reflectGain := 0.32 - 0.12*e.narrative.Forces.Intimacy

	spatial := NewSpatialHumanizedStreamer(volStreamer, delaySamples, panL, panR, reflectDelaySamples, reflectGain)
	e.mixer.Add(spatial)
}

func (e *JazzLoungeEngine) FindClosestSample(midiNote int, volume float64) (*Sample, int) {
	minDiff := 999
	var bestSample *Sample
	bestMIDI := 0

	samples := e.pianoSamplesV1
	if volume >= 0.35 { // v3 (rich) samples used broadly for warm lounge piano tone
		if len(e.pianoSamplesV3) > 0 {
			samples = e.pianoSamplesV3
		}
	}
	if len(samples) == 0 {
		// Fallback to pianoSamples
		samples = e.pianoSamples
	}

	for midi, sample := range samples {
		diff := int(math.Abs(float64(midiNote - midi)))
		if diff < minDiff {
			minDiff = diff
			bestSample = sample
			bestMIDI = midi
		}
	}
	return bestSample, midiNote - bestMIDI
}

func (e *JazzLoungeEngine) SetPlaying(playing bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isInitialized {
		return
	}

	if e.isPlaying == playing {
		return
	}

	e.isPlaying = playing
	if e.noiseCtrl != nil {
		e.noiseCtrl.Paused = !playing
	}

	if !playing {
		// Stop ambient sounds and tracks
		for i := range e.ambientSounds {
			if e.ambientSounds[i].IsPlaying {
				if e.ambientSounds[i].Ctrl != nil {
					e.ambientSounds[i].Ctrl.Paused = true
				}
				if e.ambientSounds[i].Closer != nil {
					e.ambientSounds[i].Closer.Close()
				}
				e.ambientSounds[i].Ctrl = nil
				e.ambientSounds[i].Volume = nil
				e.ambientSounds[i].Closer = nil
				e.ambientSounds[i].IsPlaying = false
			}
		}
		for i := range e.tracks {
			if e.tracks[i].IsPlaying {
				if e.tracks[i].Ctrl != nil {
					e.tracks[i].Ctrl.Paused = true
				}
				if e.tracks[i].Closer != nil {
					e.tracks[i].Closer.Close()
				}
				e.tracks[i].Ctrl = nil
				e.tracks[i].Volume = nil
				e.tracks[i].Closer = nil
				e.tracks[i].IsPlaying = false
			}
		}
	}
}

func (e *JazzLoungeEngine) IsPlaying() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isPlaying
}

func (e *JazzLoungeEngine) IsInitialized() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isInitialized
}

func (e *JazzLoungeEngine) SetMasterVolume(level float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.isInitialized {
		return
	}
	if level < 0 {
		level = 0
	}
	if level > 1.0 {
		level = 1.0
	}
	e.masterVolLevel = level
	if e.masterVol != nil {
		e.masterVol.Volume = linearToVolumeExponent(level * 3.5)
	}
}

func (e *JazzLoungeEngine) SetPianoVolume(level float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if level < 0 {
		level = 0
	}
	if level > 1.0 {
		level = 1.0
	}
	e.pianoVolLevel = level
}

func (e *JazzLoungeEngine) SetSynthVolume(level float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if level < 0 {
		level = 0
	}
	if level > 1.0 {
		level = 1.0
	}
	e.synthVolLevel = level
}

func (e *JazzLoungeEngine) SetDrumsVolume(level float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if level < 0 {
		level = 0
	}
	if level > 1.0 {
		level = 1.0
	}
	e.drumsVolLevel = level
}

func (e *JazzLoungeEngine) RegenerateProgression() {
	e.mu.Lock()
	if !e.isInitialized || e.isTransitioning {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()
	e.autoDJTransition()
}

func (e *JazzLoungeEngine) GetState() (
	key string,
	chordDegrees []string,
	activeChord string,
	ambientStates []bool,
	ambientVols []float64,
	trackStates []bool,
	trackVols []float64,
	masterVol float64,
	pianoVol float64,
	synthVol float64,
	drumsVol float64,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isInitialized {
		return "C", []string{}, "", []bool{}, []float64{}, []bool{}, []float64{}, 0.8, 0.5, 0.6, 0.4
	}

	chordDegrees = make([]string, len(e.progression))
	for i, c := range e.progression {
		chordDegrees[i] = getNoteName(e.key, c.RootOffset) + c.Name
	}

	ambientStates = make([]bool, len(e.ambientSounds))
	ambientVols = make([]float64, len(e.ambientSounds))
	for i, amb := range e.ambientSounds {
		ambientStates[i] = amb.IsPlaying
		ambientVols[i] = amb.VolumeLevel
	}

	trackStates = make([]bool, len(e.tracks))
	trackVols = make([]float64, len(e.tracks))
	for i, trk := range e.tracks {
		trackStates[i] = trk.IsPlaying
		trackVols[i] = trk.VolumeLevel
	}

	if len(e.progression) > 0 {
		activeChordIdx := (e.progress + len(e.progression) - 1) % len(e.progression)
		c := e.progression[activeChordIdx]
		activeChord = getNoteName(e.key, c.RootOffset) + c.Name
	}

	return e.key, chordDegrees, activeChord, ambientStates, ambientVols, trackStates, trackVols, e.masterVolLevel, e.pianoVolLevel, e.synthVolLevel, e.drumsVolLevel
}
