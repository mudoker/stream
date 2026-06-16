package tests

import (
	"math"
	"testing"
	"time"

	"stream/internal/viewmodel/jazzlounge"
)

func TestJazzLoungeEngineVolumeAdjustmentsAndState(t *testing.T) {
	engine := jazzlounge.GetJazzLoungeEngine()

	// Wait for the engine to initialize in the background
	var initSuccess bool
	for i := 0; i < 200; i++ {
		if engine.IsInitialized() {
			initSuccess = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !initSuccess {
		t.Fatal("Jazz Lounge engine failed to initialize in time")
	}

	// Test default volumes
	_, _, _, _, _, _, _, masterVol, pianoVol, synthVol, drumsVol := engine.GetState()
	if masterVol != 0.8 {
		t.Errorf("Expected default master volume 0.8, got %f", masterVol)
	}
	if pianoVol != 0.55 {
		t.Errorf("Expected default piano volume 0.55, got %f", pianoVol)
	}
	if synthVol != 0.72 {
		t.Errorf("Expected default synth volume 0.72, got %f", synthVol)
	}
	if drumsVol != 0.52 {
		t.Errorf("Expected default drums volume 0.52, got %f", drumsVol)
	}

	// Test volume setting
	engine.SetMasterVolume(0.95)
	engine.SetPianoVolume(0.3)
	engine.SetSynthVolume(0.85)
	engine.SetDrumsVolume(0.1)

	_, _, _, _, _, _, _, masterVol2, pianoVol2, synthVol2, drumsVol2 := engine.GetState()
	if masterVol2 != 0.95 {
		t.Errorf("Expected master volume 0.95, got %f", masterVol2)
	}
	if pianoVol2 != 0.3 {
		t.Errorf("Expected piano volume 0.3, got %f", pianoVol2)
	}
	if synthVol2 != 0.85 {
		t.Errorf("Expected synth volume 0.85, got %f", synthVol2)
	}
	if drumsVol2 != 0.1 {
		t.Errorf("Expected drums volume 0.1, got %f", drumsVol2)
	}

	// Test clamping
	engine.SetMasterVolume(1.5)
	engine.SetPianoVolume(-0.5)

	_, _, _, _, _, _, _, masterVol3, pianoVol3, _, _ := engine.GetState()
	if masterVol3 != 1.0 {
		t.Errorf("Expected clamped master volume 1.0, got %f", masterVol3)
	}
	if pianoVol3 != 0.0 {
		t.Errorf("Expected clamped piano volume 0.0, got %f", pianoVol3)
	}

	// Test adjusting ambient and track volumes with tolerance for floating-point precision
	engine.AdjustAmbientVolume("Rain", 0.1) // 0.5 -> 0.6
	engine.AdjustAmbientVolume("Rain", -0.2) // 0.6 -> 0.4
	engine.AdjustTrackVolume(1, 0.2) // 0.5 -> 0.7

	_, _, _, _, ambientVols, _, trackVols, _, _, _, _ := engine.GetState()
	if math.Abs(ambientVols[0]-0.4) > 1e-9 {
		t.Errorf("Expected Rain ambient volume 0.4, got %f", ambientVols[0])
	}
	if math.Abs(trackVols[0]-0.7) > 1e-9 {
		t.Errorf("Expected Track 1 volume 0.7, got %f", trackVols[0])
	}
}
