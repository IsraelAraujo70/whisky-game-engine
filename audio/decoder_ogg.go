package audio

import (
	"fmt"
	"os"

	"github.com/jfreymuth/oggvorbis"
)

// loadOGG loads an OGG Vorbis file, fully decodes it, and converts to
// stereo int16 at the target sample rate.
func loadOGG(path string, targetSampleRate int) (*Sound, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("audio/ogg: %w", err)
	}
	defer f.Close()

	reader, err := oggvorbis.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("audio/ogg: failed to create reader: %w", err)
	}

	srcRate := reader.SampleRate()
	channels := reader.Channels()

	// Read all float32 samples.
	buf := make([]float32, 4096)
	var allSamples []float32
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			allSamples = append(allSamples, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	// Convert float32 to int16.
	int16Samples := float32ToInt16(allSamples)

	// Convert to stereo if mono.
	if channels == 1 {
		int16Samples = resampleMono(int16Samples)
	} else if channels > 2 {
		// Downmix to stereo by taking first two channels.
		int16Samples = downmixToStereo(int16Samples, channels)
	}

	// Resample to target rate.
	int16Samples = resampleRate(int16Samples, srcRate, targetSampleRate)

	frames := len(int16Samples) / 2
	return &Sound{
		Samples:    int16Samples,
		SampleRate: targetSampleRate,
		Duration:   float64(frames) / float64(targetSampleRate),
	}, nil
}

// loadOGGStream creates a streaming Sound from an OGG Vorbis file.
func loadOGGStream(path string, targetSampleRate int) (*Sound, error) {
	// Validate by opening once.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader, err := oggvorbis.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("audio/ogg: %w", err)
	}
	duration := reader.Length()
	srcRate := reader.SampleRate()
	f.Close()

	return &Sound{
		SampleRate: targetSampleRate,
		Stream:     true,
		Duration:   float64(duration) / float64(srcRate),
		newDecoder: func() (Decoder, error) {
			sf, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			sr, err := oggvorbis.NewReader(sf)
			if err != nil {
				sf.Close()
				return nil, err
			}
			return &oggStreamDecoder{
				file:     sf,
				reader:   sr,
				channels: sr.Channels(),
				rate:     sr.SampleRate(),
			}, nil
		},
	}, nil
}

type oggStreamDecoder struct {
	file     *os.File
	reader   *oggvorbis.Reader
	channels int
	rate     int
}

func (d *oggStreamDecoder) SampleRate() int { return d.rate }
func (d *oggStreamDecoder) Channels() int   { return d.channels }

func (d *oggStreamDecoder) DecodeSamples(dst []int16) (int, error) {
	buf := make([]float32, len(dst))
	n, err := d.reader.Read(buf)
	if n > 0 {
		for i := 0; i < n; i++ {
			s := buf[i]
			if s > 1 {
				s = 1
			} else if s < -1 {
				s = -1
			}
			dst[i] = int16(s * 32767)
		}
	}
	return n, err
}

// float32ToInt16 converts float32 samples [-1, 1] to int16.
func float32ToInt16(f32 []float32) []int16 {
	out := make([]int16, len(f32))
	for i, s := range f32 {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		out[i] = int16(s * 32767)
	}
	return out
}

// downmixToStereo takes interleaved multi-channel audio and picks the
// first two channels.
func downmixToStereo(samples []int16, channels int) []int16 {
	frames := len(samples) / channels
	out := make([]int16, frames*2)
	for i := 0; i < frames; i++ {
		out[i*2] = samples[i*channels]
		out[i*2+1] = samples[i*channels+1]
	}
	return out
}
