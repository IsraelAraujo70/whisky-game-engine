package backend

import "errors"

var ErrVulkanRendererUnavailable = errors.New("desktop runtime requires the Vulkan renderer path; the SDL3 renderer has been removed and the Vulkan device/swapchain integration is not implemented yet")
