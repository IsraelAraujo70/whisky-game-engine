//go:build !linux && !windows

package backend

import platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"

// NewDesktop creates the native desktop backend wired to the Vulkan runtime
// stack, returning the platform-specific native window error where
// unsupported.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	return newVulkanDesktopBackend(title, width, height, keyMap)
}
