package viewmodel

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
)

var (
	speakerMu          sync.Mutex
	speakerInitialized = false
	speakerSampleRate  beep.SampleRate
)

// PlaySound plays a sound once using the faiface/beep library.
func PlaySound(soundName string) {
	PlaySoundRepeated(soundName, 1, 0)
}

// PlaySoundRepeated plays a sound multiple times with a delay between each play.
func PlaySoundRepeated(soundName string, count int, delay time.Duration) {
	go func() {
		for i := 0; i < count; i++ {
			if i > 0 {
				time.Sleep(delay)
			}
			playOnce(soundName)
		}
	}()
}

func playOnce(soundName string) {
	filePath := fmt.Sprintf("/usr/share/sounds/freedesktop/stereo/%s.oga", soundName)
	f, err := os.Open(filePath)
	if err != nil {
		// Fallback to terminal bell
		fmt.Print("\a")
		return
	}
	defer f.Close()

	streamer, format, err := vorbis.Decode(f)
	if err != nil {
		// Fallback to terminal bell
		fmt.Print("\a")
		return
	}
	defer streamer.Close()

	speakerMu.Lock()
	if !speakerInitialized {
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err == nil {
			speakerInitialized = true
			speakerSampleRate = format.SampleRate
		}
	}
	initialized := speakerInitialized
	sampleRate := speakerSampleRate
	speakerMu.Unlock()

	if !initialized {
		// Fallback to terminal bell
		fmt.Print("\a")
		return
	}

	var finalStreamer beep.Streamer = streamer
	if format.SampleRate != sampleRate {
		finalStreamer = beep.Resample(3, format.SampleRate, sampleRate, streamer)
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(finalStreamer, beep.Callback(func() {
		done <- true
	})))
	<-done
}

