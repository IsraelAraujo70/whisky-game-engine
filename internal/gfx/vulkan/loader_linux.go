//go:build linux

package vulkan

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	defaultAPIOnce sync.Once
	defaultAPI     *vulkanAPI
	defaultAPIErr  error
)

func loadDefaultAPI() (*vulkanAPI, error) {
	defaultAPIOnce.Do(func() {
		handle, err := loadUnixVulkanLibrary()
		if err != nil {
			defaultAPIErr = err
			return
		}

		api := &vulkanAPI{}
		purego.RegisterLibFunc(&api.enumerateInstanceExtensionProperties, handle, "vkEnumerateInstanceExtensionProperties")
		purego.RegisterLibFunc(&api.enumerateInstanceLayerProperties, handle, "vkEnumerateInstanceLayerProperties")
		purego.RegisterLibFunc(&api.createInstance, handle, "vkCreateInstance")
		purego.RegisterLibFunc(&api.destroyInstance, handle, "vkDestroyInstance")
		tryRegisterLibFunc(handle, "vkEnumeratePhysicalDevices", &api.enumeratePhysicalDevices)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceProperties", &api.getPhysicalDeviceProperties)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceQueueFamilyProperties", &api.getPhysicalDeviceQueueFamilyProperties)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceMemoryProperties", &api.getPhysicalDeviceMemoryProperties)
		tryRegisterLibFunc(handle, "vkEnumerateDeviceExtensionProperties", &api.enumerateDeviceExtensionProperties)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceSurfaceSupportKHR", &api.getPhysicalDeviceSurfaceSupportKHR)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceSurfaceCapabilitiesKHR", &api.getPhysicalDeviceSurfaceCapabilitiesKHR)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceSurfaceFormatsKHR", &api.getPhysicalDeviceSurfaceFormatsKHR)
		tryRegisterLibFunc(handle, "vkGetPhysicalDeviceSurfacePresentModesKHR", &api.getPhysicalDeviceSurfacePresentModesKHR)
		tryRegisterLibFunc(handle, "vkCreateDevice", &api.createDevice)
		tryRegisterLibFunc(handle, "vkDestroyDevice", &api.destroyDevice)
		tryRegisterLibFunc(handle, "vkGetDeviceQueue", &api.getDeviceQueue)
		tryRegisterLibFunc(handle, "vkDeviceWaitIdle", &api.deviceWaitIdle)
		tryRegisterLibFunc(handle, "vkCreateXlibSurfaceKHR", &api.createXlibSurfaceKHR)
		tryRegisterLibFunc(handle, "vkCreateWaylandSurfaceKHR", &api.createWaylandSurfaceKHR)
		tryRegisterLibFunc(handle, "vkDestroySurfaceKHR", &api.destroySurfaceKHR)
		tryRegisterLibFunc(handle, "vkCreateSwapchainKHR", &api.createSwapchainKHR)
		tryRegisterLibFunc(handle, "vkDestroySwapchainKHR", &api.destroySwapchainKHR)
		tryRegisterLibFunc(handle, "vkGetSwapchainImagesKHR", &api.getSwapchainImagesKHR)
		tryRegisterLibFunc(handle, "vkAcquireNextImageKHR", &api.acquireNextImageKHR)
		tryRegisterLibFunc(handle, "vkQueuePresentKHR", &api.queuePresentKHR)
		tryRegisterLibFunc(handle, "vkQueueSubmit", &api.queueSubmit)
		tryRegisterLibFunc(handle, "vkQueueWaitIdle", &api.queueWaitIdle)
		tryRegisterLibFunc(handle, "vkCreateImageView", &api.createImageView)
		tryRegisterLibFunc(handle, "vkDestroyImageView", &api.destroyImageView)
		tryRegisterLibFunc(handle, "vkCreateRenderPass", &api.createRenderPass)
		tryRegisterLibFunc(handle, "vkDestroyRenderPass", &api.destroyRenderPass)
		tryRegisterLibFunc(handle, "vkCreateFramebuffer", &api.createFramebuffer)
		tryRegisterLibFunc(handle, "vkDestroyFramebuffer", &api.destroyFramebuffer)
		tryRegisterLibFunc(handle, "vkCreateCommandPool", &api.createCommandPool)
		tryRegisterLibFunc(handle, "vkDestroyCommandPool", &api.destroyCommandPool)
		tryRegisterLibFunc(handle, "vkAllocateCommandBuffers", &api.allocateCommandBuffers)
		tryRegisterLibFunc(handle, "vkFreeCommandBuffers", &api.freeCommandBuffers)
		tryRegisterLibFunc(handle, "vkBeginCommandBuffer", &api.beginCommandBuffer)
		tryRegisterLibFunc(handle, "vkEndCommandBuffer", &api.endCommandBuffer)
		tryRegisterLibFunc(handle, "vkResetCommandBuffer", &api.resetCommandBuffer)
		tryRegisterLibFunc(handle, "vkCmdBeginRenderPass", &api.cmdBeginRenderPass)
		tryRegisterLibFunc(handle, "vkCmdEndRenderPass", &api.cmdEndRenderPass)
		tryRegisterLibFunc(handle, "vkCmdBindPipeline", &api.cmdBindPipeline)
		tryRegisterLibFunc(handle, "vkCmdSetViewport", &api.cmdSetViewport)
		tryRegisterLibFunc(handle, "vkCmdSetScissor", &api.cmdSetScissor)
		tryRegisterLibFunc(handle, "vkCmdBindVertexBuffers", &api.cmdBindVertexBuffers)
		tryRegisterLibFunc(handle, "vkCmdBindDescriptorSets", &api.cmdBindDescriptorSets)
		tryRegisterLibFunc(handle, "vkCmdDraw", &api.cmdDraw)
		tryRegisterLibFunc(handle, "vkCreateShaderModule", &api.createShaderModule)
		tryRegisterLibFunc(handle, "vkDestroyShaderModule", &api.destroyShaderModule)
		tryRegisterLibFunc(handle, "vkCreatePipelineLayout", &api.createPipelineLayout)
		tryRegisterLibFunc(handle, "vkDestroyPipelineLayout", &api.destroyPipelineLayout)
		tryRegisterLibFunc(handle, "vkCreateGraphicsPipelines", &api.createGraphicsPipelines)
		tryRegisterLibFunc(handle, "vkDestroyPipeline", &api.destroyPipeline)
		tryRegisterLibFunc(handle, "vkCreateDescriptorSetLayout", &api.createDescriptorSetLayout)
		tryRegisterLibFunc(handle, "vkDestroyDescriptorSetLayout", &api.destroyDescriptorSetLayout)
		tryRegisterLibFunc(handle, "vkCreateDescriptorPool", &api.createDescriptorPool)
		tryRegisterLibFunc(handle, "vkDestroyDescriptorPool", &api.destroyDescriptorPool)
		tryRegisterLibFunc(handle, "vkAllocateDescriptorSets", &api.allocateDescriptorSets)
		tryRegisterLibFunc(handle, "vkFreeDescriptorSets", &api.freeDescriptorSets)
		tryRegisterLibFunc(handle, "vkUpdateDescriptorSets", &api.updateDescriptorSets)
		tryRegisterLibFunc(handle, "vkCreateSampler", &api.createSampler)
		tryRegisterLibFunc(handle, "vkDestroySampler", &api.destroySampler)
		tryRegisterLibFunc(handle, "vkCreateBuffer", &api.createBuffer)
		tryRegisterLibFunc(handle, "vkDestroyBuffer", &api.destroyBuffer)
		tryRegisterLibFunc(handle, "vkGetBufferMemoryRequirements", &api.getBufferMemoryRequirements)
		tryRegisterLibFunc(handle, "vkAllocateMemory", &api.allocateMemory)
		tryRegisterLibFunc(handle, "vkFreeMemory", &api.freeMemory)
		tryRegisterLibFunc(handle, "vkBindBufferMemory", &api.bindBufferMemory)
		tryRegisterLibFunc(handle, "vkMapMemory", &api.mapMemory)
		tryRegisterLibFunc(handle, "vkUnmapMemory", &api.unmapMemory)
		tryRegisterLibFunc(handle, "vkCreateImage", &api.createImage)
		tryRegisterLibFunc(handle, "vkDestroyImage", &api.destroyImage)
		tryRegisterLibFunc(handle, "vkGetImageMemoryRequirements", &api.getImageMemoryRequirements)
		tryRegisterLibFunc(handle, "vkBindImageMemory", &api.bindImageMemory)
		tryRegisterLibFunc(handle, "vkCreateSemaphore", &api.createSemaphore)
		tryRegisterLibFunc(handle, "vkDestroySemaphore", &api.destroySemaphore)
		tryRegisterLibFunc(handle, "vkCreateFence", &api.createFence)
		tryRegisterLibFunc(handle, "vkDestroyFence", &api.destroyFence)
		tryRegisterLibFunc(handle, "vkWaitForFences", &api.waitForFences)
		tryRegisterLibFunc(handle, "vkResetFences", &api.resetFences)
		tryRegisterLibFunc(handle, "vkCmdPipelineBarrier", &api.cmdPipelineBarrier)
		tryRegisterLibFunc(handle, "vkCmdCopyBufferToImage", &api.cmdCopyBufferToImage)
		defaultAPI = api
	})
	return defaultAPI, defaultAPIErr
}

func tryRegisterLibFunc(handle uintptr, name string, target any) {
	addr, err := purego.Dlsym(handle, name)
	if err != nil {
		return
	}
	purego.RegisterFunc(target, addr)
}

func loadUnixVulkanLibrary() (uintptr, error) {
	candidates := []string{"libvulkan.so.1", "libvulkan.so"}
	for _, name := range candidates {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err == nil {
			return handle, nil
		}
	}
	return 0, fmt.Errorf("%w: unable to load libvulkan.so", ErrUnavailable)
}
