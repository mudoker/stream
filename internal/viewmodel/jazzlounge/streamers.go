package jazzlounge

import (
	"math"
	"math/rand"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
)

type LowPassFilter struct {
	Streamer beep.Streamer
	Cutoff   float64
	Fs       float64
	prevL    float64
	prevR    float64
}

func (l *LowPassFilter) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = l.Streamer.Stream(samples)
	alpha := 2.0 * math.Pi * l.Cutoff / l.Fs
	if alpha > 1.0 {
		alpha = 1.0
	}
	for i := 0; i < n; i++ {
		l.prevL = l.prevL + alpha*(samples[i][0]-l.prevL)
		l.prevR = l.prevR + alpha*(samples[i][1]-l.prevR)
		samples[i][0] = l.prevL
		samples[i][1] = l.prevR
	}
	return n, ok
}

func (l *LowPassFilter) Err() error {
	return l.Streamer.Err()
}

type WhiteNoiseStreamer struct{}

func (w WhiteNoiseStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		val := rand.Float64()*2.0 - 1.0
		samples[i][0] = val
		samples[i][1] = val
	}
	return len(samples), true
}

func (w WhiteNoiseStreamer) Err() error {
	return nil
}

type DiskStreamer struct {
	file     *os.File
	streamer beep.StreamSeekCloser
}

func NewDiskStreamer(path string) (*DiskStreamer, beep.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	s, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return nil, beep.Format{}, err
	}
	return &DiskStreamer{file: f, streamer: s}, format, nil
}

func (ds *DiskStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	return ds.streamer.Stream(samples)
}

func (ds *DiskStreamer) Err() error {
	return ds.streamer.Err()
}

func (ds *DiskStreamer) Close() error {
	ds.streamer.Close()
	return ds.file.Close()
}

func (ds *DiskStreamer) Len() int {
	return ds.streamer.Len()
}

func (ds *DiskStreamer) Position() int {
	return ds.streamer.Position()
}

func (ds *DiskStreamer) Seek(p int) error {
	return ds.streamer.Seek(p)
}

// SpatialHumanizedStreamer wraps any beep.Streamer to add micro-timing delay, stereo panning, and early reflection simulation.
type SpatialHumanizedStreamer struct {
	Streamer     beep.Streamer
	DelaySamples int
	PanL         float64
	PanR         float64
	ReflectDelay int
	ReflectGain  float64

	playedSamples int
	history       [][2]float64
	writeIdx      int
}

func NewSpatialHumanizedStreamer(s beep.Streamer, delaySamples int, panL, panR float64, reflectDelay int, reflectGain float64) *SpatialHumanizedStreamer {
	var history [][2]float64
	if reflectDelay > 0 {
		history = make([][2]float64, reflectDelay)
	}
	return &SpatialHumanizedStreamer{
		Streamer:     s,
		DelaySamples: delaySamples,
		PanL:         panL,
		PanR:         panR,
		ReflectDelay: reflectDelay,
		ReflectGain:  reflectGain,
		history:      history,
	}
}

func (s *SpatialHumanizedStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	// Initialize with silence if we are still in the delay phase
	for i := range samples {
		if s.playedSamples < s.DelaySamples {
			samples[i] = [2]float64{0, 0}
			s.playedSamples++
			continue
		}

		// Stream single sample from the source
		var srcBuf [1][2]float64
		sn, sok := s.Streamer.Stream(srcBuf[:])
		if sn == 0 {
			if i == 0 {
				return 0, sok
			}
			return i, true
		}

		inL := srcBuf[0][0]
		inR := srcBuf[0][1]

		// Apply stereo panning
		outL := inL * s.PanL
		outR := inR * s.PanR

		// Early reflections cross-talk
		if s.ReflectDelay > 0 {
			delayed := s.history[s.writeIdx]
			
			// Cross-mix reflected signal
			outL += delayed[1] * s.ReflectGain
			outR += delayed[0] * s.ReflectGain

			// Save to reflection memory
			s.history[s.writeIdx] = [2]float64{outL, outR}
			s.writeIdx = (s.writeIdx + 1) % s.ReflectDelay
		}

		samples[i] = [2]float64{outL, outR}
		s.playedSamples++
	}
	return len(samples), true
}

func (s *SpatialHumanizedStreamer) Err() error {
	return s.Streamer.Err()
}

// EnvelopeStreamer wraps a beep.Streamer and fades out after a given duration.
type EnvelopeStreamer struct {
	Streamer   beep.Streamer
	SampleRate beep.SampleRate
	Duration   float64
	Release    float64
	time       float64
}

func NewEnvelopeStreamer(s beep.Streamer, sr beep.SampleRate, duration, release float64) *EnvelopeStreamer {
	return &EnvelopeStreamer{
		Streamer:   s,
		SampleRate: sr,
		Duration:   duration,
		Release:    release,
	}
}

func (env *EnvelopeStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	fs := float64(env.SampleRate)
	n, ok = env.Streamer.Stream(samples)
	if n == 0 {
		return 0, ok
	}

	for i := 0; i < n; i++ {
		t := env.time
		var gain float64
		if t >= env.Duration+env.Release {
			// Done, terminate early
			return i, false
		} else if t > env.Duration {
			// Release/Fade-out phase
			rem := (env.Duration + env.Release) - t
			gain = rem / env.Release
			if gain < 0 {
				gain = 0
			}
		} else {
			// Sustain phase
			gain = 1.0
		}

		samples[i][0] *= gain
		samples[i][1] *= gain
		env.time += 1.0 / fs
	}
	return n, ok
}

func (env *EnvelopeStreamer) Err() error {
	return env.Streamer.Err()
}
