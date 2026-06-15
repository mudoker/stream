package jazzlounge

import (
	"math"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

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
}

var (
	jazzLoungeEngineInstance *JazzLoungeEngine
	jazzLoungeEngineOnce     sync.Once
)

func GetJazzLoungeEngine() *JazzLoungeEngine {
	jazzLoungeEngineOnce.Do(func() {
		jazzLoungeEngineInstance = &JazzLoungeEngine{
			pianoSamples:     make(map[int]*Sample),
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
			fn := getPianoFilenameShared(note)
			path := filepath.Join(e.assetsPath, "PianoSamples", fn)
			buf, format, err := loadSampleShared(path)
			if err == nil {
				e.pianoSamples[midi] = &Sample{
					Name:   note,
					Buffer: buf,
					Format: format,
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
	e.masterVolLevel = 0.8
	e.pianoVolLevel = 0.5
	e.synthVolLevel = 0.6
	e.drumsVolLevel = 0.4

	e.soloists = []*Soloist{
		{Type: "sax", LastMIDINote: 69},
		{Type: "trumpet", LastMIDINote: 76},
	}
	e.activeSoloistIdx = 0
	e.soloistPhraseActive = false
	e.phraseCounter = 0

	e.mixer = &beep.Mixer{}
	e.lpf = &LowPassFilter{Streamer: e.mixer, Cutoff: 2200, Fs: float64(e.speakerRate)}
	e.masterVol = &effects.Volume{Streamer: e.lpf, Base: 2, Volume: linearToVolumeExponent(e.masterVolLevel)}

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
			// Choose comping patterns dynamically based on soloist activity
			if tickCount%16 == 0 {
				if e.isTransitioning {
					// Single sustained chord on transition
					e.activeCompingPattern = []int{0}
				} else if e.soloistPhraseActive {
					// Sparse comping when soloist is active
					sparsePatterns := [][]int{{0}, {0, 8}}
					e.activeCompingPattern = sparsePatterns[rand.Intn(len(sparsePatterns))]
				} else {
					// Active comping fills when soloist is breathing/pausing
					activePatterns := [][]int{{0, 3}, {0, 6, 10}, {2, 6, 12}, {3, 11}}
					e.activeCompingPattern = activePatterns[rand.Intn(len(activePatterns))]
				}
				e.nextChord()
			}

			chordTick := tickCount % 16
			for _, hitTick := range e.activeCompingPattern {
				if hitTick == chordTick {
					e.playChordHit(chordTick == 0)
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

			// Run the active soloist if we are not transitioning
			if !e.isTransitioning {
				e.processSoloist(e.soloists[e.activeSoloistIdx], tickCount)
			}

			// 2b. Walking Bass Line (plays on every other tick for quarter-note pulse)
			if tickCount%2 == 0 && !e.isTransitioning {
				chord := e.progression[e.progress]
				kp := keyToPitch(e.key)
				bassRoot := 36 + kp + chord.RootOffset // C2 range
				// Walking bass: root, 5th, chromatic approach to next root
				bassStep := (tickCount / 2) % 4
				var bassNote int
				switch bassStep {
				case 0:
					bassNote = bassRoot // Root
				case 1:
					bassNote = bassRoot + 7 // 5th
				case 2:
					bassNote = bassRoot + 5 // 4th (passing tone)
				case 3:
					// Chromatic approach to next chord root
					nextIdx := (e.progress + 1) % len(e.progression)
					nextRoot := 36 + kp + e.progression[nextIdx].RootOffset
					if nextRoot > bassRoot {
						bassNote = nextRoot - 1 // approach from below
					} else {
						bassNote = nextRoot + 1 // approach from above
					}
				}
				e.playPianoNoteWithVol(bassNote, e.pianoVolLevel*0.38)
			}

			// 3. Jazz Drums Sequence (Swung Ride, Soft feathering Kick, Soft Snare rimshot & ghost notes)
			if e.isTransitioning {
				step := tickCount % 8
				// Play only soft ride cymbal (hat) on beat 1 and 3 during lowpass DJ sweep transitions
				if (step == 0 || step == 4) && !e.hatOff {
					e.playDrumWithVol("hat", 0.3*e.drumsVolLevel)
				}
			} else {
				step32 := tickCount % 32
				if step32 >= 28 {
					// Snare roll drum fill on last bar of 4-bar section
					e.playDrumWithVol("snare", (0.2+0.35*rand.Float64())*e.drumsVolLevel)
					if step32 == 31 {
						// Anticipate downbeat with a kick hit
						e.playDrumWithVol("kick", 0.55*e.drumsVolLevel)
					}
				} else if step32 == 0 && tickCount > 0 {
					// Landing crash accent at the start of next section
					e.playDrumWithVol("snare", 0.7*e.drumsVolLevel)
					e.playDrumWithVol("kick", 0.85*e.drumsVolLevel)
					if !e.hatOff {
						e.playDrumWithVol("hat", 0.8*e.drumsVolLevel)
					}
				} else {
					// Standard drum pattern
					if !e.hatOff {
						step := tickCount % 8
						// Classic jazz ride swing: 1  .  2  da 3  .  4  da
						if step == 0 || step == 2 || step == 3 || step == 4 || step == 6 || step == 7 {
							volFactor := 0.45
							if step == 0 || step == 3 || step == 4 || step == 7 {
								volFactor = 0.65 // Accent on downbeats
							}
							e.playDrumWithVol("hat", volFactor*e.drumsVolLevel)
						}
					}

					if !e.kickOff {
						step := tickCount % 8
						// Soft feathering kick on beats 1 and 3 (ticks 0 and 4)
						if (step == 0 || step == 4) && rand.Float64() < 0.9 {
							e.playDrumWithVol("kick", 0.35*e.drumsVolLevel)
						}
					}

					if !e.snareOff {
						step := tickCount % 8
						// Soft snare rimshot on beats 2 and 4 (ticks 2 and 6)
						if (step == 2 || step == 6) && rand.Float64() < 0.8 {
							e.playDrumWithVol("snare", 0.48*e.drumsVolLevel)
						}
						// Sneak in occasional soft snare ghost notes on swung offbeats
						if (step == 1 || step == 3 || step == 5 || step == 7) && rand.Float64() < 0.15 {
							e.playDrumWithVol("snare", 0.12*e.drumsVolLevel)
						}
					}
				}
			}

			tickCount++
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

	nextKickOff := rand.Float64() < 0.15
	nextSnareOff := rand.Float64() < 0.2
	nextHatOff := rand.Float64() < 0.25
	nextMelodyDensity := rand.Float64()*0.3 + 0.2
	nextMelodyOff := rand.Float64() < 0.25

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

	e.melodyDensity = 0.2 + rand.Float64()*0.4
	e.kickOff = rand.Float64() < 0.15
	e.snareOff = rand.Float64() < 0.2
	e.hatOff = rand.Float64() < 0.25
	e.melodyOff = rand.Float64() < 0.25

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

func (e *JazzLoungeEngine) playChordHit(isDownbeat bool) {
	chord := e.progression[e.progress]
	keyPitch := keyToPitch(e.key)
	rootMIDI := 48 + keyPitch + chord.RootOffset
	voicing := chord.GenerateVoicing(4)

	vol := e.pianoVolLevel * 0.42
	if !isDownbeat {
		vol *= 0.7 // Comping hits are slightly softer
	}

	for _, pitch := range voicing {
		e.playPianoNoteWithVol(rootMIDI+pitch, vol)
	}
}

func (e *JazzLoungeEngine) playMelody() {
	// Replaced by processSoloist()
}

func (e *JazzLoungeEngine) playNote(midiNote int) {
	// Replaced by processSoloist()
}

func (e *JazzLoungeEngine) playPianoNoteWithVol(midiNote int, volume float64) {
	sample, dist := e.FindClosestSample(midiNote)
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
	e.mixer.Add(volStreamer)
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
	e.mixer.Add(volStreamer)
}

func (e *JazzLoungeEngine) FindClosestSample(midiNote int) (*Sample, int) {
	minDiff := 999
	var bestSample *Sample
	bestMIDI := 0

	for midi, sample := range e.pianoSamples {
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
		e.masterVol.Volume = linearToVolumeExponent(level)
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
