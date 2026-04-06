package vulkan

import (
	"errors"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

var ErrNotImplemented = errors.New("vulkan backend is not implemented yet")

type Options struct {
	EnableValidation bool
}

func NewInstance(opts Options) (rhi.Instance, error) {
	return nil, ErrNotImplemented
}
