package audio

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Decoder is the interface that audio decoders implement. For streaming
// playback the decoder reads incrementally rather than loading the full
// file into memory.
type Decoder interface {
	// DecodeSamples decodes up to len(dst) stereo int16 samples into dst.
	// Returns the number of samples written and any error.
	DecodeSamples(dst []int16) (int, error)
	// SampleRate returns the source sample rate.
	SampleRate() int
	// Channels returns the number of source channels (1=mono, 2=stereo).
	Channels() int
}

// Sound holds decoded PCM audio data. Samples are always stored in stereo
// interleaved int16 format at the engine's native sample rate.
type Sound struct {
	// Samples is the decoded PCM buffer in stereo interleaved int16
	// (left, right, left, right, ...). For streaming sounds this may be nil.
	Samples []int16

	// SampleRate is the sample rate these samples are encoded at.
	SampleRate int

	// Stream indicates this is a streaming sound (large file). When true,
	// newDecoder is used to create fresh decoder instances per playback.
	Stream bool

	// newDecoder creates a fresh streaming decoder for this sound.
	newDecoder func() (Decoder, error)

	// Duration in seconds.
	Duration float64
}

// LoadSound loads a sound file from disk. The format is detected by file
// extension (.wav or .ogg). The audio is decoded and resampled to stereo
// int16 at targetSampleRate.
func LoadSound(path string, targetSampleRate int) (*Sound, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".wav":
		return loadWAV(path, targetSampleRate)
	case ".ogg":
		return loadOGG(path, targetSampleRate)
	default:
		return nil, fmt.Errorf("audio: unsupported format %q (supported: .wav, .ogg)", ext)
	}
}

// LoadSoundStream opens a sound for streaming playback (no full decode
// into memory). Useful for music or ambient tracks. Format detected by
// file extension.
func LoadSoundStream(path string, targetSampleRate int) (*Sound, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".wav":
		return loadWAVStream(path, targetSampleRate)
	case ".ogg":
		return loadOGGStream(path, targetSampleRate)
	default:
		return nil, fmt.Errorf("audio: unsupported stream format %q", ext)
	}
}

// NewSoundFromSamples creates a Sound from raw stereo int16 PCM samples.
// This is useful for procedurally generated audio.
func NewSoundFromSamples(samples []int16, sampleRate int) *Sound {
	frames := len(samples) / 2
	return &Sound{
		Samples:    samples,
		SampleRate: sampleRate,
		Duration:   float64(frames) / float64(sampleRate),
	}
}

// resampleMono converts mono samples to stereo (duplicate L/R).
func resampleMono(mono []int16) []int16 {
	stereo := make([]int16, len(mono)*2)
	for i, s := range mono {
		stereo[i*2] = s
		stereo[i*2+1] = s
	}
	return stereo
}

// resampleRate performs basic linear interpolation resampling.
func resampleRate(samples []int16, srcRate, dstRate int) []int16 {
	if srcRate == dstRate {
		return samples
	}
	ratio := float64(srcRate) / float64(dstRate)
	srcFrames := len(samples) / 2
	dstFrames := int(float64(srcFrames) / ratio)
	out := make([]int16, dstFrames*2)

	for i := 0; i < dstFrames; i++ {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)

		for ch := 0; ch < 2; ch++ {
			s0 := float64(samples[idx*2+ch])
			s1 := s0
			if idx+1 < srcFrames {
				s1 = float64(samples[(idx+1)*2+ch])
			}
			out[i*2+ch] = int16(s0 + (s1-s0)*frac)
		}
	}
	return out
}
