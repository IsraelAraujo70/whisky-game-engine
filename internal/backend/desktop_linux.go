//go:build linux

package backend

import platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"

// NewDesktop no longer falls back to SDL3. Desktop rendering is expected to go
// through the Vulkan path once device/swapchain integration is finished.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	return nil, ErrVulkanRendererUnavailable
}
