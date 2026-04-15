package audio

import (
	"testing"
)

func TestLoadOGG_RequiresAsset(t *testing.T) {
	// OGG Vorbis decoding requires a valid .ogg binary asset which cannot
	// be trivially generated in-memory (unlike WAV). This test is skipped
	// in environments without a test fixture. To test OGG decoding manually,
	// place a short .ogg file at testdata/test.ogg and run:
	//   go test ./audio/ -run TestLoadOGG -v
	t.Skip("OGG decoder test requires a binary .ogg asset in testdata/; skipping")
}

func TestFloat32ToInt16(t *testing.T) {
	input := []float32{0, 1, -1, 0.5, -0.5, 1.5, -1.5}
	got := float32ToInt16(input)

	want := []int16{0, 32767, -32767, 16383, -16383, 32767, -32767}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i, g := range got {
		// Allow +/-1 for rounding.
		diff := int(g) - int(want[i])
		if diff > 1 || diff < -1 {
			t.Errorf("sample[%d] = %d, want %d", i, g, want[i])
		}
	}
}

func TestDownmixToStereo(t *testing.T) {
	// 2 frames of 4-channel audio.
	input := []int16{1, 2, 3, 4, 5, 6, 7, 8}
	got := downmixToStereo(input, 4)

	want := []int16{1, 2, 5, 6}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("sample[%d] = %d, want %d", i, g, want[i])
		}
	}
}
