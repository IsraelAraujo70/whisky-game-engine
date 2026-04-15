package audio

import (
	"math"
	"os"
	"testing"
)

func init() {
	// Force headless mode for all tests so oto is never initialized.
	os.Setenv("WHISKY_HEADLESS", "1")
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := Init(Config{Enabled: true, Channels: 8, SampleRate: 48000})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { e.Shutdown() })
	return e
}

func TestEngine_InitShutdown(t *testing.T) {
	e := newTestEngine(t)
	if !e.running {
		t.Error("engine should be running after Init")
	}
	if err := e.Shutdown(); err != nil {
		t.Errorf("Shutdown error: %v", err)
	}
	if e.running {
		t.Error("engine should not be running after Shutdown")
	}
}

func TestEngine_PlayAndMix(t *testing.T) {
	e := newTestEngine(t)

	// Create a simple sound: 100 frames of stereo silence.
	samples := make([]int16, 200)
	// Set frame 0: L=1000, R=2000.
	samples[0] = 1000
	samples[1] = 2000
	snd := NewSoundFromSamples(samples, 48000)

	h := e.Play(snd, PlayOpts{Volume: 1, Pan: 0, Pitch: 1})
	if h == 0 {
		t.Fatal("Play returned zero handle")
	}

	// Read one frame of audio (4 bytes).
	buf := make([]byte, 4)
	n, err := e.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 4 {
		t.Fatalf("Read returned %d bytes, want 4", n)
	}

	gotL := int16(buf[0]) | int16(buf[1])<<8
	gotR := int16(buf[2]) | int16(buf[3])<<8

	if gotL != 1000 {
		t.Errorf("left sample = %d, want 1000", gotL)
	}
	if gotR != 2000 {
		t.Errorf("right sample = %d, want 2000", gotR)
	}
}

func TestEngine_Volume(t *testing.T) {
	e := newTestEngine(t)

	samples := make([]int16, 200)
	samples[0] = 10000
	samples[1] = 10000
	snd := NewSoundFromSamples(samples, 48000)

	e.Play(snd, PlayOpts{Volume: 0.5, Pan: 0, Pitch: 1})

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	// Volume 0.5 of 10000 = 5000.
	if gotL != 5000 {
		t.Errorf("volume 0.5: left = %d, want 5000", gotL)
	}
}

func TestEngine_Pan(t *testing.T) {
	e := newTestEngine(t)

	samples := make([]int16, 200)
	samples[0] = 10000
	samples[1] = 10000
	snd := NewSoundFromSamples(samples, 48000)

	// Pan fully right: left should be 0, right should be 10000.
	e.Play(snd, PlayOpts{Volume: 1, Pan: 1, Pitch: 1})

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	gotR := int16(buf[2]) | int16(buf[3])<<8

	if gotL != 0 {
		t.Errorf("pan right: left = %d, want 0", gotL)
	}
	if gotR != 10000 {
		t.Errorf("pan right: right = %d, want 10000", gotR)
	}
}

func TestEngine_PanLeft(t *testing.T) {
	e := newTestEngine(t)

	samples := make([]int16, 200)
	samples[0] = 10000
	samples[1] = 10000
	snd := NewSoundFromSamples(samples, 48000)

	e.Play(snd, PlayOpts{Volume: 1, Pan: -1, Pitch: 1})

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	gotR := int16(buf[2]) | int16(buf[3])<<8

	if gotL != 10000 {
		t.Errorf("pan left: left = %d, want 10000", gotL)
	}
	if gotR != 0 {
		t.Errorf("pan left: right = %d, want 0", gotR)
	}
}

func TestEngine_Loop(t *testing.T) {
	e := newTestEngine(t)

	// 2-frame sound with distinct values.
	samples := []int16{100, 100, 200, 200}
	snd := NewSoundFromSamples(samples, 48000)

	// Loop=1 means play 2 times total (original + 1 repeat).
	e.Play(snd, PlayOpts{Volume: 1, Pan: 0, Pitch: 1, Loop: 1})

	// Read 4 frames (8 bytes per frame).
	buf := make([]byte, 16)
	e.Read(buf)

	// Frame 0: 100, frame 1: 200, frame 2: 100 (looped), frame 3: 200 (looped).
	for frame := 0; frame < 4; frame++ {
		off := frame * 4
		gotL := int16(buf[off]) | int16(buf[off+1])<<8
		want := int16(100)
		if frame%2 == 1 {
			want = 200
		}
		if gotL != want {
			t.Errorf("frame %d: left = %d, want %d", frame, gotL, want)
		}
	}
}

func TestEngine_Stop(t *testing.T) {
	e := newTestEngine(t)

	samples := make([]int16, 200)
	for i := range samples {
		samples[i] = 5000
	}
	snd := NewSoundFromSamples(samples, 48000)

	h := e.Play(snd, PlayOpts{Volume: 1, Pitch: 1})
	e.Stop(h)

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	if gotL != 0 {
		t.Errorf("after Stop: left = %d, want 0 (silence)", gotL)
	}
}

func TestEngine_Pause_Resume(t *testing.T) {
	e := newTestEngine(t)

	samples := make([]int16, 400)
	for i := range samples {
		samples[i] = 3000
	}
	snd := NewSoundFromSamples(samples, 48000)

	h := e.Play(snd, PlayOpts{Volume: 1, Pitch: 1})
	e.Pause(h)

	buf := make([]byte, 4)
	e.Read(buf)
	gotL := int16(buf[0]) | int16(buf[1])<<8
	if gotL != 0 {
		t.Errorf("after Pause: left = %d, want 0", gotL)
	}

	e.Resume(h)
	e.Read(buf)
	gotL = int16(buf[0]) | int16(buf[1])<<8
	if gotL != 3000 {
		t.Errorf("after Resume: left = %d, want 3000", gotL)
	}
}

func TestEngine_MixMultipleSounds(t *testing.T) {
	e := newTestEngine(t)

	s1 := make([]int16, 200)
	s1[0] = 5000
	s1[1] = 5000
	s2 := make([]int16, 200)
	s2[0] = 3000
	s2[1] = 3000

	snd1 := NewSoundFromSamples(s1, 48000)
	snd2 := NewSoundFromSamples(s2, 48000)

	e.Play(snd1, PlayOpts{Volume: 1, Pitch: 1})
	e.Play(snd2, PlayOpts{Volume: 1, Pitch: 1})

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	if gotL != 8000 {
		t.Errorf("mixed left = %d, want 8000", gotL)
	}
}

func TestEngine_Clipping(t *testing.T) {
	e := newTestEngine(t)

	s1 := make([]int16, 200)
	s1[0] = 32000
	s1[1] = 32000
	s2 := make([]int16, 200)
	s2[0] = 32000
	s2[1] = 32000

	snd1 := NewSoundFromSamples(s1, 48000)
	snd2 := NewSoundFromSamples(s2, 48000)

	e.Play(snd1, PlayOpts{Volume: 1, Pitch: 1})
	e.Play(snd2, PlayOpts{Volume: 1, Pitch: 1})

	buf := make([]byte, 4)
	e.Read(buf)

	gotL := int16(buf[0]) | int16(buf[1])<<8
	if gotL != 32767 {
		t.Errorf("clipped left = %d, want 32767", gotL)
	}
}

func TestGenerateSineWave(t *testing.T) {
	samples := GenerateSineWave(440, 0.1, 48000)
	expectedFrames := int(0.1 * 48000)
	if len(samples) != expectedFrames*2 {
		t.Fatalf("sine wave length = %d, want %d", len(samples), expectedFrames*2)
	}

	// Verify it's not all zeros.
	hasNonZero := false
	for _, s := range samples {
		if s != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("sine wave is all zeros")
	}

	// Verify amplitude is reasonable (50% of 32767 ~ 16383).
	var maxAbs int16
	for _, s := range samples {
		if s < 0 {
			s = -s
		}
		if s > maxAbs {
			maxAbs = s
		}
	}
	expected := int16(math.Round(32767.0 * 0.5))
	if math.Abs(float64(maxAbs)-float64(expected)) > 500 {
		t.Errorf("max amplitude = %d, expected ~%d", maxAbs, expected)
	}
}

func TestPlayOpts_Defaults(t *testing.T) {
	opts := PlayOpts{}.withDefaults()
	if opts.Volume != 1 {
		t.Errorf("default volume = %f, want 1", opts.Volume)
	}
	if opts.Pitch != 1 {
		t.Errorf("default pitch = %f, want 1", opts.Pitch)
	}
	if opts.Pan != 0 {
		t.Errorf("default pan = %f, want 0", opts.Pan)
	}
}

func TestPlayOpts_Clamping(t *testing.T) {
	opts := PlayOpts{Volume: 5, Pan: -3, Pitch: 10}.withDefaults()
	if opts.Volume != 1 {
		t.Errorf("clamped volume = %f, want 1", opts.Volume)
	}
	if opts.Pan != -1 {
		t.Errorf("clamped pan = %f, want -1", opts.Pan)
	}
	if opts.Pitch != 4 {
		t.Errorf("clamped pitch = %f, want 4", opts.Pitch)
	}
}
