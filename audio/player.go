package audio

import (
	"math"
)

// GenerateSineWave creates a stereo int16 PCM buffer containing a sine wave
// at the given frequency, duration, and sample rate. Useful for procedural
// sound effects (beeps, test tones).
func GenerateSineWave(freq float64, durationSec float64, sampleRate int) []int16 {
	frames := int(durationSec * float64(sampleRate))
	samples := make([]int16, frames*2)
	for i := 0; i < frames; i++ {
		t := float64(i) / float64(sampleRate)
		val := math.Sin(2 * math.Pi * freq * t)
		// Apply a short fade-in/fade-out envelope to avoid clicks.
		env := 1.0
		fadeFrames := sampleRate / 200 // ~5ms fade
		if fadeFrames < 1 {
			fadeFrames = 1
		}
		if i < fadeFrames {
			env = float64(i) / float64(fadeFrames)
		} else if i > frames-fadeFrames {
			env = float64(frames-i) / float64(fadeFrames)
		}
		s := int16(val * env * 32767 * 0.5) // 50% amplitude to avoid clipping when mixed
		samples[i*2] = s
		samples[i*2+1] = s
	}
	return samples
}
