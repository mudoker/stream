package jazzlounge

import (
	"math/rand"
)

type JazzChord struct {
	RootOffset int
	Intervals  []int // offsets from root
	Name       string
	Duration   int // duration in ticks
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

// Rich Jazz Progressions (Major Standards, Turnarounds, and Ballads)
func GenerateDynamicProgression() ([]JazzChord, bool) {
	style := rand.Intn(4)
	var progression []JazzChord
	var isMinorProg bool = false // Romance/Warmth favors major tonality

	switch style {
	case 0: // Major ii-V-I Turnaround (8 chords)
		progression = []JazzChord{
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 16},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 16}, // V (G13)
			{RootOffset: 0, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9", Duration: 32},   // I (Cmaj9)
			{RootOffset: 9, Intervals: []int{0, 4, 7, 10, 14}, Name: "9", Duration: 16},      // VI (A9)
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 16},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 16}, // V (G13)
			{RootOffset: 4, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 16},     // iii (Em9)
			{RootOffset: 9, Intervals: []int{0, 4, 7, 10, 14}, Name: "9", Duration: 16},      // VI (A9)
		}
	case 1: // Romantic Turnaround Ballad (8 chords)
		progression = []JazzChord{
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // I (C6/9)
			{RootOffset: 9, Intervals: []int{0, 4, 7, 10, 14}, Name: "9", Duration: 16},      // VI (A9)
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 16},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 32}, // V (G13)
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // I (C6/9)
			{RootOffset: 5, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9", Duration: 16},   // IV (Fmaj9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 16}, // V (G13)
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 16},  // I (C6/9)
		}
	case 2: // Velvet A-B-A Standard (8 chords)
		progression = []JazzChord{
			{RootOffset: 0, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9", Duration: 16},   // I
			{RootOffset: 5, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9", Duration: 16},   // IV
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 16},     // ii
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 16}, // V
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // I
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 32}, // V
		}
	case 3: // Elegant Ballad Circle (8 chords)
		progression = []JazzChord{
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 32},     // ii (Dm9)
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 32}, // V (G13)
			{RootOffset: 0, Intervals: []int{0, 4, 7, 11, 14}, Name: "maj9", Duration: 32},   // I (Cmaj9)
			{RootOffset: 5, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // IV (F6/9)
			{RootOffset: 2, Intervals: []int{0, 3, 7, 10, 14}, Name: "m9", Duration: 32},     // ii
			{RootOffset: 7, Intervals: []int{0, 4, 7, 10, 14, 21}, Name: "13", Duration: 32}, // V
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // I
			{RootOffset: 0, Intervals: []int{0, 4, 7, 9, 14}, Name: "maj6/9", Duration: 32},  // I
		}
	}
	return progression, isMinorProg
}
