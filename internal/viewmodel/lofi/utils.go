package lofi

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

func getJazzScale(key string, isMinor bool) []int {
	keyPitch := keyToPitch(key)
	centerMIDI := 60 + keyPitch // C4 is 60 (comfortable middle range)

	// Jazz scale intervals
	var intervals []int
	if isMinor {
		// Minor Blues Scale: [0, 3, 5, 6, 7, 10]
		intervals = []int{0, 3, 5, 6, 7, 10, 12, 15, 17, 18, 19, 22, 24}
	} else {
		// Major Pentatonic Scale: [0, 2, 4, 7, 9]
		intervals = []int{0, 2, 4, 7, 9, 12, 14, 16, 19, 21, 24}
	}

	scale := make([]int, len(intervals))
	for i, offset := range intervals {
		scale[i] = centerMIDI + offset
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
