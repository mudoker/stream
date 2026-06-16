package jazzlounge

import (
	"math/rand"
)

// ThematicMotif represents a long-term musical memory with character details.
type ThematicMotif struct {
	Notes            []int
	Rhythm           []int
	Contour          []int   // Direction profile: +1 (up), -1 (down), 0 (same)
	EmotionalQuality string  // "melancholic", "warm", "tense", "mysterious"
	Importance       float64
	SourceInstrument string
	AgeTicks         int
}

// EmotionalForces represents interacting narrative values rather than a linear track.
type EmotionalForces struct {
	Intimacy     float64
	Melancholy   float64
	Tension      float64
	Confidence   float64
	Nostalgia    float64
	Mystery      float64
	Anticipation float64
	Warmth       float64
	Momentum     float64
}

// HarmonicTaste defines the session's persistent harmonic character.
type HarmonicTaste struct {
	Style              string  // "ConsonantClassic", "DarkModal", "AmbiguousChromatic", "BackdoorDominant"
	NoirChordsBias     float64
	PedalToneBias      float64
	ChromaticBias      float64
	SubstitutionChance float64
}

// EnsembleObsession defines temporary fixation topics.
type EnsembleObsession struct {
	Type           string  // "none", "interval", "rhythmic_gesture", "register_area", "harmonic_color"
	IntervalVal    int     // e.g. 8 for minor 6th
	RhythmVal      int     // e.g. 1 for staccato eighths
	RegisterCenter float64 // MIDI pitch target
	Strength       float64 // 0.0 to 1.0 (decays over time)
}

// RegisterArchitecture determines how vertical space expands, contracts, and migrates.
type RegisterArchitecture struct {
	Width  float64 // Spread: 0.0 (narrow/intimate) to 1.0 (wide/climax)
	Center float64 // Height: 0.0 (deep bass) to 1.0 (crystalline high)
}

// MetaMemory collects execution statistics to generate stylistic adjustments (pressures).
type MetaMemory struct {
	SoloistLeadTicks int
	PianoLeadTicks   int
	SilenceTicks     int
	TotalPhrases     int
	LastRecalledAge  int
}

// JazzNarrative combines the emotional system, active mood, politics, and register configs.
type JazzNarrative struct {
	Mood                 string // "Introspective", "Romantic", "Weary", "Melancholic", "Nostalgic", "Elegant", "Playful", "Mysterious"
	ActiveChapter        string // "IntimateNocturne", "SoloSpotlight", "PianoInterlude", "BassSoliloquy", "StillnessAtmosphere", "NocturnalSuspense"
	ChapterTicksLeft     int
	ActiveLeader         string // "sax", "trumpet", "piano", "bass", "none"
	LeaderTicksLeft      int
	PerceivedTempo       float64 // perceived speed: 0.0 (very slow) to 1.0 (faster/urgent)
	
	Forces               EmotionalForces
	Taste                HarmonicTaste
	Obsession            EnsembleObsession
	Register             RegisterArchitecture
	History              MetaMemory
}

// MusicianPersonalities holds persistent behavioral preferences for the ensemble.
type MusicianPersonalities struct {
	Ensemble MinorNoirBias
	Piano    PianistPersonality
	Bass     BassistPersonality
	Drums    DrummerPersonality
}

type MinorNoirBias struct {
	NoirChordsBias float64
	PedalToneBias  float64
	ChromaticBias  float64
	MelodicDirBias int // -1 descending, +1 ascending, 0 neutral
}

type PianistPersonality struct {
	UpperExtensionBias bool
	SparsityBias       float64
}

type BassistPersonality struct {
	StepwiseBias float64
}

type DrummerPersonality struct {
	BrushActivity float64
}

// InitDefaultPersonalities initializes dynamic states with lounge-appropriate defaults.
func InitDefaultPersonalities() MusicianPersonalities {
	return MusicianPersonalities{
		Ensemble: MinorNoirBias{
			NoirChordsBias: 0.6 + rand.Float64()*0.4,
			PedalToneBias:  0.3 + rand.Float64()*0.4,
			ChromaticBias:  0.4 + rand.Float64()*0.4,
			MelodicDirBias: []int{-1, 0, 1}[rand.Intn(3)],
		},
		Piano: PianistPersonality{
			UpperExtensionBias: rand.Float64() < 0.85, // almost always use extensions for richness
			SparsityBias:       0.20 + rand.Float64()*0.25, // lower sparsity = more activity
		},
		Bass: BassistPersonality{
			StepwiseBias: 0.5 + rand.Float64()*0.4,
		},
		Drums: DrummerPersonality{
			BrushActivity: 0.55 + rand.Float64()*0.25, // warm, engaged drummer floor
		},
	}
}

// RandomHarmonicTaste selects a dynamic harmonic preference for the session.
func RandomHarmonicTaste() HarmonicTaste {
	var style string
	r := rand.Float64()
	if r < 0.45 {
		style = "VelvetSophistication"
	} else if r < 0.75 {
		style = "ConsonantClassic"
	} else {
		styles := []string{"DarkModal", "AmbiguousChromatic", "BackdoorDominant"}
		style = styles[rand.Intn(len(styles))]
	}
	
	taste := HarmonicTaste{Style: style}
	switch style {
	case "VelvetSophistication":
		taste.NoirChordsBias = 0.15
		taste.PedalToneBias = 0.20
		taste.ChromaticBias = 0.15
		taste.SubstitutionChance = 0.30
	case "ConsonantClassic":
		taste.NoirChordsBias = 0.25
		taste.PedalToneBias = 0.3
		taste.ChromaticBias = 0.2
		taste.SubstitutionChance = 0.15
	case "DarkModal":
		taste.NoirChordsBias = 0.85
		taste.PedalToneBias = 0.65
		taste.ChromaticBias = 0.4
		taste.SubstitutionChance = 0.35
	case "AmbiguousChromatic":
		taste.NoirChordsBias = 0.5
		taste.PedalToneBias = 0.2
		taste.ChromaticBias = 0.9
		taste.SubstitutionChance = 0.6
	case "BackdoorDominant":
		taste.NoirChordsBias = 0.4
		taste.PedalToneBias = 0.4
		taste.ChromaticBias = 0.5
		taste.SubstitutionChance = 0.7
	}
	return taste
}
