// Package audio provides a cross-platform audio engine for the Whisky game
// engine. It supports WAV and OGG Vorbis file formats, a multi-channel mixer
// with volume, pan, loop, and pitch controls, and streaming playback for large
// files.
//
// The engine uses oto/v3 as the audio output backend (Linux, Windows, macOS)
// and falls back to a null device when WHISKY_HEADLESS=1 or OTO_NULL=1 is set.
package audio

import (
	"errors"
	"io"
	"math"
	"os"
	"sync"
)

// Config holds audio engine configuration.
type Config struct {
	// Enabled controls whether the engine produces output. Default true.
	Enabled bool
	// Channels is the maximum number of simultaneous mixer channels.
	// Zero means 32.
	Channels int
	// SampleRate is the output sample rate in Hz. Zero means 48000.
	SampleRate int
}

// Engine manages audio playback. It owns a mixer with N channels and an
// output device (oto or null).
type Engine struct {
	mu       sync.Mutex
	cfg      Config
	channels []*channel
	nextID   Handle
	output   io.Closer // oto.Player or nullPlayer
	running  bool

	// Output format (always stereo int16).
	sampleRate int
}

// Handle identifies a playing sound instance. Zero is never valid.
type Handle uint64

// PlayOpts configures playback of a sound.
type PlayOpts struct {
	// Volume in range [0, 1]. Default 1.
	Volume float64
	// Pan in range [-1 (left), 1 (right)]. Default 0 (center).
	Pan float64
	// Loop: 0 = play once, -1 = loop forever, N>0 = play N+1 total times.
	Loop int
	// Pitch multiplier. 1.0 = normal, 2.0 = octave up, 0.5 = octave down.
	// Clamped to [0.1, 4.0]. Default 1.
	Pitch float64
}

func (o PlayOpts) withDefaults() PlayOpts {
	if o.Volume == 0 {
		o.Volume = 1
	}
	if o.Pitch == 0 {
		o.Pitch = 1
	}
	o.Pitch = math.Max(0.1, math.Min(4.0, o.Pitch))
	o.Volume = math.Max(0, math.Min(1, o.Volume))
	o.Pan = math.Max(-1, math.Min(1, o.Pan))
	return o
}

type channel struct {
	handle     Handle
	sound      *Sound
	opts       PlayOpts
	pos        float64 // fractional sample position (for pitch)
	loopsLeft  int     // -1 = infinite
	paused     bool
	done       bool
	// For streaming sounds
	stream     Decoder
	streamBuf  []int16
}

// Init creates the audio engine and starts the output device.
func Init(cfg Config) (*Engine, error) {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 48000
	}
	if cfg.Channels == 0 {
		cfg.Channels = 32
	}

	e := &Engine{
		cfg:        cfg,
		channels:   make([]*channel, 0, cfg.Channels),
		sampleRate: cfg.SampleRate,
	}

	if !cfg.Enabled || isHeadless() {
		e.output = &nullCloser{}
		e.running = true
		return e, nil
	}

	player, err := newOtoOutput(cfg.SampleRate, e)
	if err != nil {
		return nil, err
	}
	e.output = player
	e.running = true
	return e, nil
}

// Shutdown stops the output device and releases resources.
func (e *Engine) Shutdown() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}
	e.running = false
	e.channels = nil
	if e.output != nil {
		return e.output.Close()
	}
	return nil
}

// Play starts a sound. Returns a handle for controlling playback.
func (e *Engine) Play(snd *Sound, opts PlayOpts) Handle {
	if snd == nil {
		return 0
	}
	opts = opts.withDefaults()

	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return 0
	}

	e.nextID++
	h := e.nextID

	ch := &channel{
		handle:    h,
		sound:     snd,
		opts:      opts,
		loopsLeft: opts.Loop,
	}

	// For streaming sounds, create a new decoder instance.
	if snd.Stream && snd.newDecoder != nil {
		dec, err := snd.newDecoder()
		if err == nil {
			ch.stream = dec
			ch.streamBuf = make([]int16, 4096)
		}
	}

	// Reuse a finished channel slot if available.
	for i, existing := range e.channels {
		if existing.done {
			e.channels[i] = ch
			return h
		}
	}

	if len(e.channels) < e.cfg.Channels {
		e.channels = append(e.channels, ch)
	}
	// If all channels are full, the sound is silently dropped.

	return h
}

// Stop stops a playing sound by handle.
func (e *Engine) Stop(h Handle) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch := e.findChannel(h); ch != nil {
		ch.done = true
	}
}

// Pause pauses a playing sound.
func (e *Engine) Pause(h Handle) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch := e.findChannel(h); ch != nil {
		ch.paused = true
	}
}

// Resume resumes a paused sound.
func (e *Engine) Resume(h Handle) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch := e.findChannel(h); ch != nil {
		ch.paused = false
	}
}

// StopAll stops all playing sounds.
func (e *Engine) StopAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ch := range e.channels {
		ch.done = true
	}
}

func (e *Engine) findChannel(h Handle) *channel {
	for _, ch := range e.channels {
		if ch.handle == h && !ch.done {
			return ch
		}
	}
	return nil
}

// Read implements io.Reader for oto. It fills buf with mixed stereo int16
// samples in little-endian format.
func (e *Engine) Read(buf []byte) (int, error) {
	// Each sample frame is 4 bytes: 2 bytes left + 2 bytes right (int16 LE).
	frames := len(buf) / 4
	if frames == 0 {
		return 0, nil
	}

	// Clear buffer.
	for i := range buf {
		buf[i] = 0
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, ch := range e.channels {
		if ch.done || ch.paused {
			continue
		}
		e.mixChannel(ch, buf, frames)
	}

	return frames * 4, nil
}

func (e *Engine) mixChannel(ch *channel, buf []byte, frames int) {
	opts := ch.opts

	// Pan -> left/right gain.
	leftGain := opts.Volume * math.Min(1, 1-opts.Pan)
	rightGain := opts.Volume * math.Min(1, 1+opts.Pan)

	snd := ch.sound
	totalSamples := len(snd.Samples)

	for i := 0; i < frames; i++ {
		intPos := int(ch.pos) * 2 // stereo: 2 samples per frame

		if intPos >= totalSamples {
			if ch.loopsLeft == 0 {
				ch.done = true
				return
			}
			if ch.loopsLeft > 0 {
				ch.loopsLeft--
			}
			ch.pos = 0
			intPos = 0
		}

		var left, right int16
		if intPos+1 < totalSamples {
			left = snd.Samples[intPos]
			right = snd.Samples[intPos+1]
		}

		// Apply gain.
		lf := float64(left) * leftGain
		rf := float64(right) * rightGain

		// Mix into buffer (additive, with clamping).
		off := i * 4
		existL := int16(buf[off]) | int16(buf[off+1])<<8
		existR := int16(buf[off+2]) | int16(buf[off+3])<<8

		mixL := clampInt16(int32(existL) + int32(lf))
		mixR := clampInt16(int32(existR) + int32(rf))

		buf[off] = byte(mixL)
		buf[off+1] = byte(mixL >> 8)
		buf[off+2] = byte(mixR)
		buf[off+3] = byte(mixR >> 8)

		ch.pos += opts.Pitch
	}
}

func clampInt16(v int32) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

func isHeadless() bool {
	return os.Getenv("WHISKY_HEADLESS") == "1" || os.Getenv("OTO_NULL") == "1"
}

type nullCloser struct{}

func (nullCloser) Close() error { return nil }

var errShutdown = errors.New("audio: engine shut down")
