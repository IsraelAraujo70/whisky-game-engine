package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WAV RIFF format constants.
const (
	wavRIFF = "RIFF"
	wavWAVE = "WAVE"
	wavFmt  = "fmt "
	wavData = "data"
)

// wavHeader represents the parsed WAV header.
type wavHeader struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	DataSize      uint32
}

// loadWAV loads a WAV file, decodes it fully into memory, and converts to
// stereo int16 at the target sample rate.
func loadWAV(path string, targetSampleRate int) (*Sound, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("audio/wav: %w", err)
	}
	return decodeWAV(bytes.NewReader(data), targetSampleRate)
}

// DecodeWAV decodes WAV data from a reader. Exported for testing.
func DecodeWAV(r io.ReadSeeker, targetSampleRate int) (*Sound, error) {
	return decodeWAV(r, targetSampleRate)
}

func decodeWAV(r io.ReadSeeker, targetSampleRate int) (*Sound, error) {
	hdr, dataReader, err := parseWAVHeader(r)
	if err != nil {
		return nil, err
	}

	samples, err := readWAVSamples(dataReader, hdr)
	if err != nil {
		return nil, err
	}

	// Convert mono to stereo.
	if hdr.NumChannels == 1 {
		samples = resampleMono(samples)
	}

	// Resample to target rate.
	samples = resampleRate(samples, int(hdr.SampleRate), targetSampleRate)

	frames := len(samples) / 2
	return &Sound{
		Samples:    samples,
		SampleRate: targetSampleRate,
		Duration:   float64(frames) / float64(targetSampleRate),
	}, nil
}

// parseWAVHeader reads and validates the RIFF/WAVE header, returning the
// format info and a reader positioned at the start of sample data.
func parseWAVHeader(r io.ReadSeeker) (*wavHeader, io.Reader, error) {
	// RIFF header.
	var riffID [4]byte
	if err := binary.Read(r, binary.LittleEndian, &riffID); err != nil {
		return nil, nil, fmt.Errorf("audio/wav: failed to read RIFF ID: %w", err)
	}
	if string(riffID[:]) != wavRIFF {
		return nil, nil, fmt.Errorf("audio/wav: not a RIFF file")
	}

	var fileSize uint32
	if err := binary.Read(r, binary.LittleEndian, &fileSize); err != nil {
		return nil, nil, fmt.Errorf("audio/wav: failed to read file size: %w", err)
	}

	var waveID [4]byte
	if err := binary.Read(r, binary.LittleEndian, &waveID); err != nil {
		return nil, nil, fmt.Errorf("audio/wav: failed to read WAVE ID: %w", err)
	}
	if string(waveID[:]) != wavWAVE {
		return nil, nil, fmt.Errorf("audio/wav: not a WAVE file")
	}

	var hdr wavHeader
	var dataReader io.Reader

	// Read chunks until we find fmt and data.
	foundFmt := false
	for {
		var chunkID [4]byte
		if err := binary.Read(r, binary.LittleEndian, &chunkID); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("audio/wav: failed to read chunk ID: %w", err)
		}

		var chunkSize uint32
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return nil, nil, fmt.Errorf("audio/wav: failed to read chunk size: %w", err)
		}

		switch string(chunkID[:]) {
		case wavFmt:
			if err := binary.Read(r, binary.LittleEndian, &hdr.AudioFormat); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(r, binary.LittleEndian, &hdr.NumChannels); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(r, binary.LittleEndian, &hdr.SampleRate); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(r, binary.LittleEndian, &hdr.ByteRate); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(r, binary.LittleEndian, &hdr.BlockAlign); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(r, binary.LittleEndian, &hdr.BitsPerSample); err != nil {
				return nil, nil, err
			}

			// Skip any extra format bytes.
			extra := int64(chunkSize) - 16
			if extra > 0 {
				if _, err := r.Seek(extra, io.SeekCurrent); err != nil {
					return nil, nil, err
				}
			}
			foundFmt = true

		case wavData:
			if !foundFmt {
				return nil, nil, fmt.Errorf("audio/wav: data chunk before fmt chunk")
			}
			hdr.DataSize = chunkSize
			dataReader = io.LimitReader(r, int64(chunkSize))
			return &hdr, dataReader, nil

		default:
			// Skip unknown chunk. Pad to even boundary.
			skip := int64(chunkSize)
			if skip%2 != 0 {
				skip++
			}
			if _, err := r.Seek(skip, io.SeekCurrent); err != nil {
				return nil, nil, fmt.Errorf("audio/wav: failed to skip chunk %q: %w", chunkID, err)
			}
		}
	}

	return nil, nil, fmt.Errorf("audio/wav: no data chunk found")
}

// readWAVSamples reads raw PCM samples from a WAV data chunk and converts
// them to int16. Supports 8-bit unsigned, 16-bit signed, and 24-bit signed.
func readWAVSamples(dataReader io.Reader, hdr *wavHeader) ([]int16, error) {
	if hdr.AudioFormat != 1 {
		return nil, fmt.Errorf("audio/wav: unsupported format %d (only PCM=1 supported)", hdr.AudioFormat)
	}

	rawData, err := io.ReadAll(dataReader)
	if err != nil {
		return nil, fmt.Errorf("audio/wav: failed to read sample data: %w", err)
	}

	bytesPerSample := int(hdr.BitsPerSample) / 8
	if bytesPerSample == 0 {
		return nil, fmt.Errorf("audio/wav: invalid bits per sample: %d", hdr.BitsPerSample)
	}
	numSamples := len(rawData) / bytesPerSample

	samples := make([]int16, numSamples)

	switch hdr.BitsPerSample {
	case 8:
		// 8-bit WAV is unsigned: 0-255, with 128 as silence.
		for i := 0; i < numSamples; i++ {
			samples[i] = int16(int(rawData[i])-128) << 8
		}
	case 16:
		for i := 0; i < numSamples; i++ {
			off := i * 2
			samples[i] = int16(rawData[off]) | int16(rawData[off+1])<<8
		}
	case 24:
		for i := 0; i < numSamples; i++ {
			off := i * 3
			// 24-bit signed, take upper 16 bits.
			val := int32(rawData[off]) | int32(rawData[off+1])<<8 | int32(rawData[off+2])<<16
			if val >= 0x800000 {
				val -= 0x1000000
			}
			samples[i] = int16(val >> 8)
		}
	default:
		return nil, fmt.Errorf("audio/wav: unsupported bit depth %d", hdr.BitsPerSample)
	}

	return samples, nil
}

// loadWAVStream creates a streaming Sound from a WAV file.
func loadWAVStream(path string, targetSampleRate int) (*Sound, error) {
	// For WAV streaming, we validate the header first.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hdr, _, err := parseWAVHeader(f)
	if err != nil {
		return nil, err
	}

	bytesPerSample := int(hdr.BitsPerSample) / 8
	totalSamples := int(hdr.DataSize) / bytesPerSample
	totalFrames := totalSamples / int(hdr.NumChannels)
	duration := float64(totalFrames) / float64(hdr.SampleRate)

	return &Sound{
		SampleRate: targetSampleRate,
		Stream:     true,
		Duration:   duration,
		newDecoder: func() (Decoder, error) {
			// Each playback opens a fresh file handle.
			sf, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			shdr, reader, err := parseWAVHeader(sf)
			if err != nil {
				sf.Close()
				return nil, err
			}
			return &wavStreamDecoder{
				file:   sf,
				reader: reader,
				hdr:    shdr,
			}, nil
		},
	}, nil
}

type wavStreamDecoder struct {
	file   *os.File
	reader io.Reader
	hdr    *wavHeader
}

func (d *wavStreamDecoder) SampleRate() int { return int(d.hdr.SampleRate) }
func (d *wavStreamDecoder) Channels() int   { return int(d.hdr.NumChannels) }

func (d *wavStreamDecoder) DecodeSamples(dst []int16) (int, error) {
	bps := int(d.hdr.BitsPerSample) / 8
	buf := make([]byte, len(dst)*bps)
	n, err := d.reader.Read(buf)
	if n == 0 {
		return 0, err
	}
	buf = buf[:n]
	count := n / bps
	if count > len(dst) {
		count = len(dst)
	}

	switch d.hdr.BitsPerSample {
	case 8:
		for i := 0; i < count; i++ {
			dst[i] = int16(int(buf[i])-128) << 8
		}
	case 16:
		for i := 0; i < count; i++ {
			off := i * 2
			dst[i] = int16(buf[off]) | int16(buf[off+1])<<8
		}
	case 24:
		for i := 0; i < count; i++ {
			off := i * 3
			val := int32(buf[off]) | int32(buf[off+1])<<8 | int32(buf[off+2])<<16
			if val >= 0x800000 {
				val -= 0x1000000
			}
			dst[i] = int16(val >> 8)
		}
	}

	return count, err
}
