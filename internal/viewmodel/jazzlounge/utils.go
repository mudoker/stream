package jazzlounge

import (
	"errors"
	"flag"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var (
	SpeakerMu          sync.Mutex
	SpeakerInitialized = false
	SpeakerSampleRate  beep.SampleRate
)

func isTestRun() bool {
	return flag.Lookup("test.v") != nil
}

var KeysList = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func keyToPitch(key string) int {
	for i, k := range KeysList {
		if k == key {
			return i
		}
	}
	return 0
}

// getJazzScale returns a chromatic-complete bebop/dorian scale spanning 3 octaves.
// This gives soloists access to ALL 12 notes per octave so they can use
// chromatic approach notes, enclosures, and passing tones — not just
// pentatonic "toy" notes.
func getJazzScale(key string, isMinor bool) []int {
	keyPitch := keyToPitch(key)
	startMIDI := 48 + keyPitch // C3 range

	var intervals []int
	if isMinor {
		// Dorian mode (jazz minor workhorse): 0 2 3 5 7 9 10
		// Plus chromatic passing tones for bebop vocabulary: 1 4 6 8 11
		// = full chromatic but with dorian "gravity" via chord-tone resolution
		for oct := 0; oct < 3; oct++ {
			base := oct * 12
			for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
				intervals = append(intervals, base+n)
			}
		}
		intervals = append(intervals, 36) // top note
	} else {
		// Mixolydian/bebop dominant: 0 2 4 5 7 9 10 11
		// Plus chromatic passing tones
		for oct := 0; oct < 3; oct++ {
			base := oct * 12
			for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
				intervals = append(intervals, base+n)
			}
		}
		intervals = append(intervals, 36) // top note
	}

	scale := make([]int, len(intervals))
	for i, offset := range intervals {
		scale[i] = startMIDI + offset
	}
	return scale
}

var SampleMIDIMap = map[string]int{
	"C1": 24, "D#1": 27, "F#1": 30, "A1": 33,
	"C2": 36, "D#2": 39, "F#2": 42, "A2": 45,
	"C3": 48, "D#3": 51, "F#3": 54, "A3": 57,
	"C4": 60, "D#4": 63, "F#4": 66, "A4": 69,
	"C5": 72, "D#5": 75, "F#5": 78, "A5": 81,
	"C6": 84, "D#6": 87, "F#6": 90, "A6": 93,
}

var fiveToFive = []int{-5, -3, -1, 0, 2, 4, 5, 7, 9, 11, 12, 14, 16, 17, 19}

func getNoteName(key string, offset int) string {
	keyPitch := keyToPitch(key)
	pitch := (keyPitch + offset) % 12
	return KeysList[pitch]
}

// isScaleTone returns true if the MIDI note belongs to the "strong" scale
// (dorian for minor, mixolydian for major) — used to weight note choices.
func isScaleTone(midiNote int, key string, isMinor bool) bool {
	keyPitch := keyToPitch(key)
	noteClass := ((midiNote - keyPitch) % 12 + 12) % 12
	if isMinor {
		// Dorian: 0 2 3 5 7 9 10
		switch noteClass {
		case 0, 2, 3, 5, 7, 9, 10:
			return true
		}
	} else {
		// Mixolydian: 0 2 4 5 7 9 10
		switch noteClass {
		case 0, 2, 4, 5, 7, 9, 10:
			return true
		}
	}
	return false
}

func initSpeakerShared() (beep.SampleRate, error) {
	SpeakerMu.Lock()
	defer SpeakerMu.Unlock()
	if !SpeakerInitialized {
		sr := beep.SampleRate(44100)

		type initResult struct {
			err error
		}
		resChan := make(chan initResult, 1)
		go func() {
			err := speaker.Init(sr, sr.N(time.Second/10))
			resChan <- initResult{err: err}
		}()

		select {
		case res := <-resChan:
			if res.err == nil {
				SpeakerInitialized = true
				SpeakerSampleRate = sr
			} else {
				return 0, res.err
			}
		case <-time.After(500 * time.Millisecond):
			return 0, errors.New("speaker initialization timed out")
		}
	}
	return SpeakerSampleRate, nil
}

func getAssetsPathShared() string {
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".config", "stream", "assets", "engine")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	cwd, err := os.Getwd()
	if err == nil {
		path := filepath.Join(cwd, "assets", "engine")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "/home/mudoker/mudoker/projects/stream/assets/engine"
}

func getPianoFilenameShared(note string) string {
	name := strings.ReplaceAll(note, "#", "sharp")
	return name + "v1.mp3"
}

func loadSampleShared(path string) (*beep.Buffer, beep.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	defer f.Close()

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return nil, beep.Format{}, err
	}
	defer streamer.Close()

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	return buffer, format, nil
}

func midiToFreq(note int) float64 {
	return 440.0 * math.Pow(2.0, float64(note-69)/12.0)
}

func linearToVolumeExponent(v float64) float64 {
	if v <= 0 {
		return -100.0
	}
	return math.Log2(v)
}

// noteNameToMIDI maps a pitch string like "Cs4" (C#4), "As3" (A#3), or "A2" to its MIDI value.
func noteNameToMIDI(note string) int {
	if len(note) < 2 {
		return 0
	}
	octave := int(note[len(note)-1] - '0')
	pitch := note[:len(note)-1]

	pitchClassMap := map[string]int{
		"C": 0, "Cs": 1, "D": 2, "Ds": 3, "E": 4, "F": 5,
		"Fs": 6, "G": 7, "Gs": 8, "A": 9, "As": 10, "B": 11,
	}

	pc, ok := pitchClassMap[pitch]
	if !ok {
		return 0
	}
	return (octave + 1) * 12 + pc
}
