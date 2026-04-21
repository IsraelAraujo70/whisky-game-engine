package render

import "testing"

func TestDrawCmdImplementations(t *testing.T) {
	var _ DrawCmd = FillRect{}
	var _ DrawCmd = SpriteCmd{}
	var _ DrawCmd = TextCmd{}
}
