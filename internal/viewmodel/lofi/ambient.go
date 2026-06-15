package lofi

import (
	"io"
	"path/filepath"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
)

type AmbientSound struct {
	Name        string
	Filename    string
	IsPlaying   bool
	Ctrl        *beep.Ctrl
	Volume      *effects.Volume
	Closer      io.Closer
	VolumeLevel float64
}

type Track struct {
	ID          int
	Name        string
	Filename    string
	IsPlaying   bool
	Ctrl        *beep.Ctrl
	Volume      *effects.Volume
	Closer      io.Closer
	VolumeLevel float64
}

func (e *LofiEngine) AdjustAmbientVolume(name string, delta float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.ambientSounds {
		if e.ambientSounds[i].Name == name {
			lvl := e.ambientSounds[i].VolumeLevel + delta
			if lvl < 0 {
				lvl = 0
			}
			if lvl > 1.0 {
				lvl = 1.0
			}
			e.ambientSounds[i].VolumeLevel = lvl
			if e.ambientSounds[i].Volume != nil {
				e.ambientSounds[i].Volume.Volume = linearToVolumeExponent(lvl)
			}
			break
		}
	}
}

func (e *LofiEngine) AdjustTrackVolume(id int, delta float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.tracks {
		if e.tracks[i].ID == id {
			lvl := e.tracks[i].VolumeLevel + delta
			if lvl < 0 {
				lvl = 0
			}
			if lvl > 1.0 {
				lvl = 1.0
			}
			e.tracks[i].VolumeLevel = lvl
			if e.tracks[i].Volume != nil {
				e.tracks[i].Volume.Volume = linearToVolumeExponent(lvl)
			}
			break
		}
	}
}

func (e *LofiEngine) ToggleAmbient(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isInitialized {
		return
	}

	var amb *AmbientSound
	for i := range e.ambientSounds {
		if e.ambientSounds[i].Name == name {
			amb = &e.ambientSounds[i]
			break
		}
	}
	if amb == nil {
		return
	}

	if amb.IsPlaying {
		if amb.Ctrl != nil {
			amb.Ctrl.Paused = true
		}
		if amb.Closer != nil {
			amb.Closer.Close()
		}
		amb.Ctrl = nil
		amb.Volume = nil
		amb.Closer = nil
		amb.IsPlaying = false
	} else {
		path := filepath.Join(e.assetsPath, "effects", amb.Filename)
		ds, format, err := NewDiskStreamer(path)
		if err != nil {
			return
		}

		looped := beep.Loop(-1, ds)
		resampled := beep.Resample(3, format.SampleRate, e.speakerRate, looped)
		ctrl := &beep.Ctrl{Streamer: resampled, Paused: false}

		volStreamer := &effects.Volume{
			Streamer: ctrl,
			Base:     2,
			Volume:   linearToVolumeExponent(amb.VolumeLevel),
		}

		e.mixer.Add(volStreamer)

		amb.Ctrl = ctrl
		amb.Volume = volStreamer
		amb.Closer = ds
		amb.IsPlaying = true
	}
}

func (e *LofiEngine) ToggleTrack(id int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isInitialized {
		return
	}

	var trk *Track
	for i := range e.tracks {
		if e.tracks[i].ID == id {
			trk = &e.tracks[i]
			break
		}
	}
	if trk == nil {
		return
	}

	if trk.IsPlaying {
		if trk.Ctrl != nil {
			trk.Ctrl.Paused = true
		}
		if trk.Closer != nil {
			trk.Closer.Close()
		}
		trk.Ctrl = nil
		trk.Volume = nil
		trk.Closer = nil
		trk.IsPlaying = false
	} else {
		path := filepath.Join(e.assetsPath, "tracks", trk.Filename)
		ds, format, err := NewDiskStreamer(path)
		if err != nil {
			return
		}

		looped := beep.Loop(-1, ds)
		resampled := beep.Resample(3, format.SampleRate, e.speakerRate, looped)
		ctrl := &beep.Ctrl{Streamer: resampled, Paused: false}

		volStreamer := &effects.Volume{
			Streamer: ctrl,
			Base:     2,
			Volume:   linearToVolumeExponent(trk.VolumeLevel),
		}

		e.mixer.Add(volStreamer)

		trk.Ctrl = ctrl
		trk.Volume = volStreamer
		trk.Closer = ds
		trk.IsPlaying = true
	}
}
