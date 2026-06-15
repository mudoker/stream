package jazzlounge

import (
	"math/rand"
)

type JazzChord struct {
	RootOffset int
	Intervals  []int // offsets from root
	Name       string
}

func (c JazzChord) GenerateVoicing(size int) []int {
	if size < 3 {
		return append([]int{}, c.Intervals[:3]...)
	}
	voicing := append([]int{}, c.Intervals[1:size]...)
	rand.Shuffle(len(voicing), func(i, j int) {
		voicing[i], voicing[j] = voicing[j], voicing[i]
	})
	for i := 1; i < len(voicing); i++ {
		for voicing[i] < voicing[i-1] {
			voicing[i] += 12
		}
	}
	voicing = append([]int{0}, voicing...)
	return voicing
}

var CompingPatterns = [][]int{
	{0, 3},       // Charleston: beat 1, offbeat of 2
	{0, 6, 10},   // Downbeat sync: beat 1, beat 4, beat 2 of next measure
	{2, 6, 12},   // Laid back: beat 2, beat 4, beat 3 of next measure
	{3, 11},      // Offbeats only: offbeat of 2, offbeat of 2 of next measure
	{0},          // Single sustained chord on beat 1
}

// Rich Jazz Progressions (Major Standards, Minor Autumn/Noir, 12-Bar Blues, Modal Jazz)
func GenerateDynamicProgression() ([]JazzChord, bool) {
	style := rand.Intn(4)
	var progression []JazzChord
	var isMinorProg bool

	switch style {
	case 0: // Major ii-V-I standard (8 chords)
		isMinorProg = false
		progression = []JazzChord{
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13"}, // V (G13)
			{RootOffset: 0, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9"},   // I (Cmaj9)
			{RootOffset: 9, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},   // VI (A7alt)
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13"}, // V (G13)
			{RootOffset: 4, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},     // iii (Em9)
			{RootOffset: 9, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},   // VI (A7alt)
		}
	case 1: // Minor Autumn/Noir standard (8 chords)
		isMinorProg = true
		progression = []JazzChord{
			{RootOffset: 5, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},      // iv (Fm9)
			{RootOffset: 10, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13"}, // bVII (Bb13)
			{RootOffset: 3, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9"},    // bIII (Ebmaj9)
			{RootOffset: 8, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9"},    // bVI (Abmaj9)
			{RootOffset: 2, Intervals: []int{0, 3, 6, 10, 13}, Name: "m7b5"},    // iiø (Dø7)
			{RootOffset: 7, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},    // V7alt (G7alt)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},      // i (Cm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},      // i (Cm9)
		}
	case 2: // 12-Bar Jazz Blues (12 chords)
		isMinorProg = rand.Float64() < 0.5
		if isMinorProg {
			progression = []JazzChord{
				{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // i (Cm9)
				{RootOffset: 5, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // iv (Fm9)
				{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // i (Cm9)
				{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // i (Cm9)
				{RootOffset: 5, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // iv (Fm9)
				{RootOffset: 5, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // iv (Fm9)
				{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // i (Cm9)
				{RootOffset: 8, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9"},  // bVI (Abmaj9)
				{RootOffset: 2, Intervals: []int{0, 3, 6, 10, 13}, Name: "m7b5"},  // iiø (Dø7)
				{RootOffset: 7, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},  // V7alt (G7alt)
				{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},    // i (Cm9)
				{RootOffset: 7, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},  // V7alt (G7alt)
			}
		} else {
			progression = []JazzChord{
				{RootOffset: 0, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // I9 (C9)
				{RootOffset: 5, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // IV9 (F9)
				{RootOffset: 0, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // I9 (C9)
				{RootOffset: 0, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // I9 (C9)
				{RootOffset: 5, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // IV9 (F9)
				{RootOffset: 5, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // IV9 (F9)
				{RootOffset: 0, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // I9 (C9)
				{RootOffset: 9, Intervals: []int{0, 4, 8, 10, 13}, Name: "7alt"},  // VI7alt (A7alt)
				{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"},     // ii (Dm9)
				{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13"}, // V (G13)
				{RootOffset: 0, Intervals: []int{0, 4, 7, 10, 14}, Name: "9"},     // I9 (C9)
				{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13"}, // V (G13)
			}
		}
	case 3: // Modal So What / Impressions (8 chords)
		isMinorProg = true
		progression = []JazzChord{
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
			{RootOffset: 1, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // bii (Dbm9)
			{RootOffset: 1, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // bii (Dbm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
			{RootOffset: 0, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9"}, // i (Cm9)
		}
	}
	return progression, isMinorProg
}
