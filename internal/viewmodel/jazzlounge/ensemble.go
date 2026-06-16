package jazzlounge

import (
	"math/rand"
)

// ThematicMotif represents a long-term musical memory.
type ThematicMotif struct {
	Notes            []int
	Rhythm           []int
	Importance       float64
	SourceInstrument string
	AgeTicks         int
}

// JazzNarrative manages emotional velocity, accumulated tension, and register space.
type JazzNarrative struct {
	TicksSinceLastClimax int
	TicksSinceLastSparse int
	AccumulatedTension   float64
	NarrativeState       string  // "exposition", "development", "climax", "resolution", "stillness"
	RegisterRange        float64 // 0.0 (narrow) to 1.0 (wide)
	ActiveLeader         string  // "sax", "trumpet", "piano", "bass", "none"
	LeaderTicksLeft      int
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
			UpperExtensionBias: rand.Float64() < 0.75,
			SparsityBias:       0.4 + rand.Float64()*0.4,
		},
		Bass: BassistPersonality{
			StepwiseBias: 0.5 + rand.Float64()*0.4,
		},
		Drums: DrummerPersonality{
			BrushActivity: 0.3 + rand.Float64()*0.4,
		},
	}
}
