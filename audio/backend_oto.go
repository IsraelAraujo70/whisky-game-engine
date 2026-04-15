package audio

import (
	"io"

	"github.com/ebitengine/oto/v3"
)

// otoOutput wraps oto.Context and oto.Player to implement io.Closer.
type otoOutput struct {
	ctx    *oto.Context
	player *oto.Player
}

func (o *otoOutput) Close() error {
	// oto.Player does not have a Close — we just discard it.
	// oto.Context.Close is not exposed either; the context lives for the
	// process lifetime.  We stop feeding it samples by returning from Read.
	return nil
}

// newOtoOutput creates an oto context and starts a player that pulls audio
// from the engine's Read method. The engine implements io.Reader.
func newOtoOutput(sampleRate int, src io.Reader) (*otoOutput, error) {
	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		return nil, err
	}
	<-ready

	player := ctx.NewPlayer(src)
	// Buffer size: ~50ms worth of audio for low latency.
	bufSize := sampleRate * 4 / 20 // 4 bytes per frame, 20 = 1000/50
	player.SetBufferSize(bufSize)
	player.Play()

	return &otoOutput{
		ctx:    ctx,
		player: player,
	}, nil
}
