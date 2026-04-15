package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// buildWAV creates a minimal WAV file in memory with the given parameters.
func buildWAV(t *testing.T, sampleRate uint32, bitsPerSample uint16, numChannels uint16, samples []byte) []byte {
	t.Helper()

	dataSize := uint32(len(samples))
	fmtSize := uint32(16)
	fileSize := 4 + (8 + fmtSize) + (8 + dataSize)

	var buf bytes.Buffer

	// RIFF header.
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt chunk.
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, fmtSize)
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, numChannels)
	binary.Write(&buf, binary.LittleEndian, sampleRate)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample/8)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	blockAlign := numChannels * (bitsPerSample / 8)
	binary.Write(&buf, binary.LittleEndian, blockAlign)
	binary.Write(&buf, binary.LittleEndian, bitsPerSample)

	// data chunk.
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(samples)

	return buf.Bytes()
}

func TestDecodeWAV_16bit_Stereo(t *testing.T) {
	// Create 4 frames of stereo 16-bit audio at 48000 Hz.
	raw := make([]byte, 4*2*2) // 4 frames * 2 channels * 2 bytes

	// Helper to put signed int16 as little-endian bytes.
	putInt16LE := func(b []byte, v int16) {
		binary.LittleEndian.PutUint16(b, uint16(v))
	}

	// Frame 0: L=1000, R=-1000
	putInt16LE(raw[0:], 1000)
	putInt16LE(raw[2:], -1000)
	// Frame 1: L=2000, R=-2000
	putInt16LE(raw[4:], 2000)
	putInt16LE(raw[6:], -2000)
	// Frame 2: L=0, R=0
	// Frame 3: L=32767, R=-32768
	putInt16LE(raw[12:], 32767)
	putInt16LE(raw[14:], -32768)

	wavData := buildWAV(t, 48000, 16, 2, raw)
	snd, err := decodeWAV(bytes.NewReader(wavData), 48000)
	if err != nil {
		t.Fatalf("decodeWAV error: %v", err)
	}

	if len(snd.Samples) != 8 {
		t.Fatalf("expected 8 samples, got %d", len(snd.Samples))
	}
	if snd.Samples[0] != 1000 {
		t.Errorf("sample[0] = %d, want 1000", snd.Samples[0])
	}
	if snd.Samples[1] != -1000 {
		t.Errorf("sample[1] = %d, want -1000", snd.Samples[1])
	}
	if snd.Samples[6] != 32767 {
		t.Errorf("sample[6] = %d, want 32767", snd.Samples[6])
	}
	if snd.Samples[7] != -32768 {
		t.Errorf("sample[7] = %d, want -32768", snd.Samples[7])
	}
}

func TestDecodeWAV_8bit_Mono(t *testing.T) {
	// 4 samples of 8-bit mono.
	raw := []byte{128, 255, 0, 192} // silence, max, min, 3/4

	wavData := buildWAV(t, 44100, 8, 1, raw)
	snd, err := decodeWAV(bytes.NewReader(wavData), 44100)
	if err != nil {
		t.Fatalf("decodeWAV error: %v", err)
	}

	// 4 mono samples -> 4 stereo frames = 8 int16 samples.
	if len(snd.Samples) != 8 {
		t.Fatalf("expected 8 samples (stereo), got %d", len(snd.Samples))
	}

	// 128 -> 0 -> L=0, R=0.
	if snd.Samples[0] != 0 {
		t.Errorf("silence sample = %d, want 0", snd.Samples[0])
	}

	// 255 -> (255-128)<<8 = 127*256 = 32512.
	if snd.Samples[2] != 32512 {
		t.Errorf("max sample = %d, want 32512", snd.Samples[2])
	}

	// 0 -> (0-128)<<8 = -128*256 = -32768.
	if snd.Samples[4] != -32768 {
		t.Errorf("min sample = %d, want -32768", snd.Samples[4])
	}
}

func TestDecodeWAV_24bit_Stereo(t *testing.T) {
	// 2 stereo frames of 24-bit audio.
	raw := make([]byte, 2*2*3) // 2 frames * 2 channels * 3 bytes

	// Frame 0 L: 0x7FFFFF (max positive 24-bit) -> >> 8 = 32767
	raw[0] = 0xFF
	raw[1] = 0xFF
	raw[2] = 0x7F
	// Frame 0 R: 0x800000 (min negative 24-bit) -> -8388608 >> 8 = -32768
	raw[3] = 0x00
	raw[4] = 0x00
	raw[5] = 0x80

	// Frame 1 L: 0x000000 -> 0
	// Frame 1 R: 0x000000 -> 0

	wavData := buildWAV(t, 48000, 24, 2, raw)
	snd, err := decodeWAV(bytes.NewReader(wavData), 48000)
	if err != nil {
		t.Fatalf("decodeWAV error: %v", err)
	}

	if len(snd.Samples) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(snd.Samples))
	}
	if snd.Samples[0] != 32767 {
		t.Errorf("24-bit max = %d, want 32767", snd.Samples[0])
	}
	if snd.Samples[1] != -32768 {
		t.Errorf("24-bit min = %d, want -32768", snd.Samples[1])
	}
}

func TestDecodeWAV_Resample(t *testing.T) {
	// Create a 440 Hz sine at 22050 Hz, resample to 44100 Hz.
	srcRate := uint32(22050)
	frames := 100
	raw := make([]byte, frames*2*2) // stereo 16-bit

	for i := 0; i < frames; i++ {
		val := math.Sin(2 * math.Pi * 440 * float64(i) / float64(srcRate))
		s := int16(val * 32767 * 0.5)
		off := i * 4
		binary.LittleEndian.PutUint16(raw[off:], uint16(s))
		binary.LittleEndian.PutUint16(raw[off+2:], uint16(s))
	}

	wavData := buildWAV(t, srcRate, 16, 2, raw)
	snd, err := decodeWAV(bytes.NewReader(wavData), 44100)
	if err != nil {
		t.Fatalf("decodeWAV error: %v", err)
	}

	// Resampled from 22050 to 44100 should roughly double the frame count.
	expectedFrames := frames * 2
	gotFrames := len(snd.Samples) / 2
	if gotFrames < expectedFrames-5 || gotFrames > expectedFrames+5 {
		t.Errorf("resampled frames = %d, expected ~%d", gotFrames, expectedFrames)
	}
}

func TestDecodeWAV_InvalidFormat(t *testing.T) {
	_, err := decodeWAV(bytes.NewReader([]byte("not a wav file")), 48000)
	if err == nil {
		t.Error("expected error for invalid WAV data")
	}
}
