package jazzlounge

import (
	"math"
	"math/rand"

	"github.com/faiface/beep"
)

// SynthVoice is a per-note synthesizer with realistic timbre, dynamics, and spatial depth.
type SynthVoice struct {
	SampleRate beep.SampleRate
	Frequency  float64
	Amplitude  float64
	Time       float64
	Duration   float64
	VoiceType  string  // "sax" or "trumpet"
	Velocity   float64 // 0.0-1.0 dynamic velocity for this note

	// Oscillator Phases
	phase1   float64
	phase2   float64
	phase3   float64
	modPhase float64

	// Two-pole biquad filter state
	z1 float64
	z2 float64

	// Stereo Delay Line
	delayBufL   []float64
	delayBufR   []float64
	delayWriteL int
	delayWriteR int

	initialized bool
	noiseSeed   float64
}

func (v *SynthVoice) Stream(samples [][2]float64) (n int, ok bool) {
	fs := float64(v.SampleRate)
	if fs <= 0 {
		fs = 44100
	}

	if !v.initialized {
		sizeL := int(0.18 * fs) // 180ms
		sizeR := int(0.23 * fs) // 230ms
		if sizeL < 64 {
			sizeL = 64
		}
		if sizeR < 64 {
			sizeR = 64
		}
		v.delayBufL = make([]float64, sizeL)
		v.delayBufR = make([]float64, sizeR)
		v.noiseSeed = rand.Float64() * 1000.0
		if v.Velocity <= 0 {
			v.Velocity = 0.7
		}
		v.initialized = true
	}

	vel := v.Velocity

	for i := range samples {
		if v.Time >= v.Duration {
			return i, false
		}

		// -- 1. ADSR Envelope with exponential curves --
		var env float64
		attack := 0.06
		decay := 0.08
		sustainLevel := 0.75
		release := 0.20
		if v.VoiceType == "trumpet" {
			attack = 0.025
			decay = 0.06
			sustainLevel = 0.8
			release = 0.15
		}

		totalMinEnv := attack + decay + release
		if v.Duration < totalMinEnv {
			scale := v.Duration / totalMinEnv
			attack *= scale
			decay *= scale
			release *= scale
		}

		if v.Time < attack {
			// Exponential attack
			t := v.Time / attack
			env = t * t
		} else if v.Time < attack+decay {
			// Exponential decay to sustain
			t := (v.Time - attack) / decay
			env = 1.0 - (1.0-sustainLevel)*(t*t)
		} else if v.Time > v.Duration-release {
			// Exponential release
			t := (v.Duration - v.Time) / release
			env = sustainLevel * t * t
		} else {
			env = sustainLevel
		}
		if env < 0 {
			env = 0
		}

		// -- 2. Pitch: drift + vibrato --
		drift := 0.001 * math.Sin(2.0*math.Pi*0.9*v.Time+v.noiseSeed)

		var vibRate, maxVibDepth float64
		if v.VoiceType == "trumpet" {
			vibRate = 5.5
			maxVibDepth = 0.004
		} else {
			vibRate = 4.8
			maxVibDepth = 0.006
		}
		vibDepth := maxVibDepth * math.Min(v.Time/0.5, 1.0)
		vibrato := vibDepth * math.Sin(2.0*math.Pi*vibRate*v.Time)

		freq := v.Frequency * (1.0 + drift + vibrato)

		// Detuned unison
		f1, f2, f3 := freq, freq*1.002, freq*0.998

		var raw float64

		if v.VoiceType == "trumpet" {
			// FM brass: 2 modulators for richer harmonics
			v.modPhase += 2.0 * math.Pi * freq * 2.0 / fs
			for v.modPhase > 2.0*math.Pi {
				v.modPhase -= 2.0 * math.Pi
			}
			mod1 := math.Sin(v.modPhase)
			mod2 := math.Sin(v.modPhase * 3.01)

			// Modulation index: bright attack, warm sustain
			modIdx := 0.12 + 0.6*math.Exp(-v.Time*4.0)
			modIdx2 := 0.05 + 0.25*math.Exp(-v.Time*6.0)

			fc1 := f1 * (1.0 + mod1*modIdx + mod2*modIdx2)
			fc2 := f2 * (1.0 + mod1*modIdx + mod2*modIdx2)
			fc3 := f3 * (1.0 + mod1*modIdx + mod2*modIdx2)

			v.phase1 += 2.0 * math.Pi * fc1 / fs
			v.phase2 += 2.0 * math.Pi * fc2 / fs
			v.phase3 += 2.0 * math.Pi * fc3 / fs
			for v.phase1 > 2.0*math.Pi { v.phase1 -= 2.0 * math.Pi }
			for v.phase2 > 2.0*math.Pi { v.phase2 -= 2.0 * math.Pi }
			for v.phase3 > 2.0*math.Pi { v.phase3 -= 2.0 * math.Pi }

			raw = 0.50*math.Sin(v.phase1) + 0.28*math.Sin(v.phase2) + 0.22*math.Sin(v.phase3)
		} else {
			// Sax: additive harmonics (fundamental + octave + 5th)
			v.phase1 += 2.0 * math.Pi * f1 / fs
			v.phase2 += 2.0 * math.Pi * f2 / fs
			v.phase3 += 2.0 * math.Pi * f3 / fs
			for v.phase1 > 2.0*math.Pi { v.phase1 -= 2.0 * math.Pi }
			for v.phase2 > 2.0*math.Pi { v.phase2 -= 2.0 * math.Pi }
			for v.phase3 > 2.0*math.Pi { v.phase3 -= 2.0 * math.Pi }

			// Warm sax: fundamental + soft octave + faint 5th harmonic
			waveAt := func(p float64) float64 {
				sin1 := math.Sin(p)
				sin2 := math.Sin(p * 2.0) // octave
				sin3 := math.Sin(p * 3.0) // 5th above octave
				// Soft sawtooth character via additive synthesis
				saw := math.Sin(p) - 0.5*math.Sin(2*p) + 0.33*math.Sin(3*p) - 0.25*math.Sin(4*p)
				return 0.35*sin1 + 0.20*sin2 + 0.10*sin3 + 0.35*saw*0.4
			}

			raw = 0.50*waveAt(v.phase1) + 0.28*waveAt(v.phase2) + 0.22*waveAt(v.phase3)

			// Breath noise
			noise := (rand.Float64()*2.0 - 1.0) * 0.025 * env
			raw += noise
		}

		// -- 3. Biquad Low-Pass Filter (2-pole, 12dB/oct) --
		var cutoff, Q float64
		if v.VoiceType == "trumpet" {
			cutoff = 1200.0 + 2400.0*env*vel
			Q = 0.8
		} else {
			cutoff = 800.0 + 1800.0*env*vel
			Q = 1.1
		}

		omega := 2.0 * math.Pi * cutoff / fs
		sn := math.Sin(omega)
		cs := math.Cos(omega)
		alpha := sn / (2.0 * Q)

		b0 := (1.0 - cs) / 2.0
		b1 := 1.0 - cs
		b2 := (1.0 - cs) / 2.0
		a0 := 1.0 + alpha
		a1 := -2.0 * cs
		a2 := 1.0 - alpha

		// Normalize
		nb0 := b0 / a0
		nb1 := b1 / a0
		nb2 := b2 / a0
		na1 := a1 / a0
		na2 := a2 / a0

		// Direct Form II Transposed
		filtered := nb0*raw + v.z1
		v.z1 = nb1*raw - na1*filtered + v.z2
		v.z2 = nb2*raw - na2*filtered

		// -- 4. Stereo delay with cross-feedback --
		delayedL := v.delayBufL[v.delayWriteL]
		delayedR := v.delayBufR[v.delayWriteR]

		feedback := 0.28
		outL := filtered + feedback*delayedR
		outR := filtered + feedback*delayedL

		v.delayBufL[v.delayWriteL] = outL * 0.7 // damping
		v.delayBufR[v.delayWriteR] = outR * 0.7

		v.delayWriteL = (v.delayWriteL + 1) % len(v.delayBufL)
		v.delayWriteR = (v.delayWriteR + 1) % len(v.delayBufR)

		// Final output with velocity-scaled amplitude
		gain := env * v.Amplitude * vel
		samples[i][0] = outL * gain
		samples[i][1] = outR * gain

		v.Time += 1.0 / fs
	}
	return len(samples), true
}

func (v *SynthVoice) Err() error {
	return nil
}

// BassVoice simulates an acoustic upright double bass using physical modeling with wood body resonance, attack slap noise, and pitch drift.
type BassVoice struct {
	SampleRate beep.SampleRate
	Frequency  float64
	Amplitude  float64
	Time       float64
	Duration   float64
	Velocity   float64

	// Resonator filter state
	z1 float64
	z2 float64

	// Body bandpass filter state
	bpZ1 float64
	bpZ2 float64

	initialized bool
	noiseSeed   float64
}

func (v *BassVoice) Stream(samples [][2]float64) (n int, ok bool) {
	fs := float64(v.SampleRate)
	if fs <= 0 {
		fs = 44100
	}

	if !v.initialized {
		v.noiseSeed = rand.Float64() * 500.0
		if v.Velocity <= 0 {
			v.Velocity = 0.75
		}
		v.initialized = true
	}

	for i := range samples {
		if v.Time >= v.Duration {
			return i, false
		}

		// 1. ADSR Envelope: Fast attack, natural decay
		var env float64
		attack := 0.015
		release := 0.25
		decay := v.Duration - attack - release
		if decay < 0.1 {
			decay = 0.1
		}

		if v.Time < attack {
			env = v.Time / attack
		} else if v.Time < attack+decay {
			t := (v.Time - attack) / decay
			env = 1.0 - 0.7*t // decays to 30% sustain level
		} else if v.Time > v.Duration-release {
			t := (v.Duration - v.Time) / release
			env = 0.3 * t * t
		} else {
			env = 0.3
		}

		// 2. Pitch Drift (subtle pitch instability)
		drift := 0.002 * math.Sin(2.0*math.Pi*2.8*v.Time+v.noiseSeed)
		freq := v.Frequency * (1.0 + drift)

		// 3. Excitation: Pluck Slap (brief burst of high-passed noise at the start)
		var pluck float64
		if v.Time < 0.04 {
			// Noise burst
			rawNoise := rand.Float64()*2.0 - 1.0
			pluck = rawNoise * 0.18 * (1.0 - v.Time/0.04)
		}

		// Harmonics generation (string texture)
		phase := 2.0 * math.Pi * freq * v.Time
		fundamental := math.Sin(phase)
		harm2 := 0.28 * math.Sin(phase*2.0)
		harm3 := 0.12 * math.Sin(phase*3.0)

		rawString := fundamental + harm2 + harm3 + pluck

		// 4. Wood Body Resonance Bandpass Filter (tuned to 110Hz with Q = 2.0)
		bpFreq := 110.0
		bpQ := 2.0
		omega := 2.0 * math.Pi * bpFreq / fs
		sn := math.Sin(omega)
		cs := math.Cos(omega)
		alpha := sn / (2.0 * bpQ)

		b0 := alpha
		b1 := 0.0
		b2 := -alpha
		a0 := 1.0 + alpha
		a1 := -2.0 * cs
		a2 := 1.0 - alpha

		// Normalize coefficients
		nb0 := b0 / a0
		nb1 := b1 / a0
		nb2 := b2 / a0
		na1 := a1 / a0
		na2 := a2 / a0

		// Biquad bandpass filter step
		filteredBody := nb0*rawString + v.bpZ1
		v.bpZ1 = nb1*rawString - na1*filteredBody + v.bpZ2
		v.bpZ2 = nb2*rawString - na2*filteredBody

		// Combine clean and resonance-filtered signals for dynamic woodiness
		output := 0.35*rawString + 0.65*filteredBody

		// 5. Envelope gain scaling
		gain := env * v.Amplitude * v.Velocity
		samples[i][0] = output * gain
		samples[i][1] = output * gain

		v.Time += 1.0 / fs
	}
	return len(samples), true
}

func (v *BassVoice) Err() error {
	return nil
}
