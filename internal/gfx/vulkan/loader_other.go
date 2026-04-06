//go:build !linux && !windows

package vulkan

func loadDefaultAPI() (*vulkanAPI, error) {
	return nil, ErrUnavailable
}
