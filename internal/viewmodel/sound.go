package viewmodel

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
)

var (
	speakerInitialized = false
	speakerSampleRate  beep.SampleRate
)

// PlaySound plays a sound using the faiface/beep library.
func PlaySound(soundName string) {
	go func() {
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

		if !speakerInitialized {
			err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			if err == nil {
				speakerInitialized = true
				speakerSampleRate = format.SampleRate
			} else {
				// Fallback to terminal bell
				fmt.Print("\a")
				return
			}
		}

		var finalStreamer beep.Streamer = streamer
		if speakerInitialized && format.SampleRate != speakerSampleRate {
			finalStreamer = beep.Resample(3, format.SampleRate, speakerSampleRate, streamer)
		}

		done := make(chan bool)
		speaker.Play(beep.Seq(finalStreamer, beep.Callback(func() {
			done <- true
		})))
		<-done
	}()
}
