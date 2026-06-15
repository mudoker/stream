package lofi

import (
	"math"
	"math/rand"

	"github.com/faiface/beep"
)

type SynthVoice struct {
	SampleRate beep.SampleRate
	Frequency  float64
	Amplitude  float64
	Time       float64
	Duration   float64
	VoiceType  string // "sax" or "trumpet"

	// Oscillator Phases
	phase1   float64
	phase2   float64
	phase3   float64
	modPhase float64

	// State-Variable Filter (SVF) memory
	svfLow  float64
	svfBand float64

	// Stereo Delay Line
	delayBufL   []float64
	delayBufR   []float64
	delayWriteL int
	delayWriteR int

	initialized bool
}

func (v *SynthVoice) Stream(samples [][2]float64) (n int, ok bool) {
	fs := float64(v.SampleRate)
	if fs <= 0 {
		fs = 44100
	}

	if !v.initialized {
		// Create stereo delay line buffers
		// We use slightly different delay times for left and right to get a lush stereo spread
		// L = 240ms delay, R = 290ms delay
		sizeL := int(0.24 * fs)
		sizeR := int(0.29 * fs)
		if sizeL < 100 {
			sizeL = 100
		}
		if sizeR < 100 {
			sizeR = 100
		}
		v.delayBufL = make([]float64, sizeL)
		v.delayBufR = make([]float64, sizeR)
		v.initialized = true
	}

	for i := range samples {
		if v.Time >= v.Duration {
			return i, false
		}

		// 1. Dynamic Envelope (ADSR)
		var env float64
		attack := 0.08
		release := 0.15
		if v.VoiceType == "trumpet" {
			attack = 0.04
			release = 0.12
		}

		if v.Duration < attack+release {
			scale := v.Duration / (attack + release)
			attack *= scale
			release *= scale
		}

		if v.Time < attack {
			env = v.Time / attack
		} else if v.Time > v.Duration-release {
			env = (v.Duration - v.Time) / release
		} else {
			env = 1.0
		}
		if env < 0 {
			env = 0
		}

		// 2. Pitch Drift LFO and Slow-Building Vibrato
		// Natural pitch drift to make it sound human (1.1Hz LFO, very subtle)
		drift := 0.0012 * math.Sin(2.0*math.Pi*1.1*v.Time)

		// Vibrato parameters
		var vibratoRate float64 = 5.2
		var maxVibratoDepth float64 = 0.007
		if v.VoiceType == "trumpet" {
			vibratoRate = 6.0
			maxVibratoDepth = 0.005
		}
		// Fade in vibrato slowly so note starts clean
		vibratoDepth := maxVibratoDepth * math.Min(v.Time/0.45, 1.0)
		vibrato := vibratoDepth * math.Sin(2.0*math.Pi*vibratoRate*v.Time)

		// Combine to modify base frequency
		freq := v.Frequency * (1.0 + drift + vibrato)

		// Unison detuned frequencies
		f1 := freq
		f2 := freq * 1.0022 // slightly sharp
		f3 := freq * 0.9978 // slightly flat

		var val float64

		if v.VoiceType == "trumpet" {
			// Brass synthesis (FM synthesis + Unison detuning)
			// Modulator phase step
			v.modPhase += 2.0 * math.Pi * freq / fs
			if v.modPhase > 2.0*math.Pi {
				v.modPhase -= 2.0 * math.Pi
			}
			modVal := math.Sin(v.modPhase)

			// Modulation index decays for a clean attack bite and warm sustain
			modIndex := 0.15 + 0.95*math.Exp(-v.Time*6.0)

			// Modulated carrier frequencies
			fc1 := f1 + modVal*modIndex*f1
			fc2 := f2 + modVal*modIndex*f2
			fc3 := f3 + modVal*modIndex*f3

			v.phase1 += 2.0 * math.Pi * fc1 / fs
			if v.phase1 > 2.0*math.Pi {
				v.phase1 -= 2.0 * math.Pi
			}
			v.phase2 += 2.0 * math.Pi * fc2 / fs
			if v.phase2 > 2.0*math.Pi {
				v.phase2 -= 2.0 * math.Pi
			}
			v.phase3 += 2.0 * math.Pi * fc3 / fs
			if v.phase3 > 2.0*math.Pi {
				v.phase3 -= 2.0 * math.Pi
			}

			// Mix carriers (unison)
			val = 0.5*math.Sin(v.phase1) + 0.25*math.Sin(v.phase2) + 0.25*math.Sin(v.phase3)
		} else {
			// Woodwind/Saxophone synthesis (Rich combination of sine, triangle, and sawtooth waves)
			v.phase1 += 2.0 * math.Pi * f1 / fs
			if v.phase1 > 2.0*math.Pi {
				v.phase1 -= 2.0 * math.Pi
			}
			v.phase2 += 2.0 * math.Pi * f2 / fs
			if v.phase2 > 2.0*math.Pi {
				v.phase2 -= 2.0 * math.Pi
			}
			v.phase3 += 2.0 * math.Pi * f3 / fs
			if v.phase3 > 2.0*math.Pi {
				v.phase3 -= 2.0 * math.Pi
			}

			// Helpers for triangle and sawtooth waves
			getTri := func(p float64) float64 {
				t := p / (2.0 * math.Pi)
				if t < 0.5 {
					return -1.0 + 4.0*t
				}
				return 3.0 - 4.0*t
			}
			getSaw := func(p float64) float64 {
				return 1.0 - (p / math.Pi)
			}

			// Core wave helper: warm mix of sine (fundamental), triangle (even harmonics), and saw (reed buzz)
			getWave := func(p float64) float64 {
				return 0.40*math.Sin(p) + 0.35*getTri(p) + 0.25*getSaw(p)
			}

			// Mix unison detuned oscillators
			val = 0.5*getWave(v.phase1) + 0.25*getWave(v.phase2) + 0.25*getWave(v.phase3)

			// Breath noise modulated by envelope
			noise := (rand.Float64()*2.0 - 1.0) * 0.035 * env
			val += noise
		}

		// 3. Dynamic Resonant Low-Pass State-Variable Filter (SVF)
		var cutoff float64
		var Q float64 = 1.8
		if v.VoiceType == "trumpet" {
			// Very bright on attack, warm/mellow on sustain
			cutoff = 900.0 + 1400.0*env
			Q = 1.4
		} else {
			// Saxophone body formant resonance
			cutoff = 550.0 + 850.0*env
			Q = 2.0
		}

		// SVF coefficients
		f_coeff := 2.0 * math.Sin(math.Pi*cutoff/fs)
		if f_coeff > 1.0 {
			f_coeff = 1.0
		}
		q_coeff := 1.0 / Q

		// SVF equations
		high := val - v.svfLow - q_coeff*v.svfBand
		v.svfBand += f_coeff * high
		v.svfLow += f_coeff * v.svfBand

		filtered := v.svfLow

		// 4. Stereo Delay & Cozy Ambient Reverb
		// Read delayed values
		delayedL := v.delayBufL[v.delayWriteL]
		delayedR := v.delayBufR[v.delayWriteR]

		feedback := 0.35
		if v.VoiceType == "trumpet" {
			feedback = 0.30
		}

		// Cross-feedback delay (Left feeds back into Right delay, and vice-versa)
		outL := filtered + feedback*delayedR
		outR := filtered + feedback*delayedL

		// Write to delay lines
		v.delayBufL[v.delayWriteL] = outL
		v.delayBufR[v.delayWriteR] = outR

		// Increment write indices
		v.delayWriteL = (v.delayWriteL + 1) % len(v.delayBufL)
		v.delayWriteR = (v.delayWriteR + 1) % len(v.delayBufR)

		// Final envelope and gain scaling
		samples[i][0] = outL * env * v.Amplitude
		samples[i][1] = outR * env * v.Amplitude

		v.Time += 1.0 / fs
	}
	return len(samples), true
}

func (v *SynthVoice) Err() error {
	return nil
}
