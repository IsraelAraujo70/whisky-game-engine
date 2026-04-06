//go:build windows

package vulkan

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

var (
	defaultAPIOnce sync.Once
	defaultAPI     *vulkanAPI
	defaultAPIErr  error
)

func loadDefaultAPI() (*vulkanAPI, error) {
	defaultAPIOnce.Do(func() {
		dll := windows.NewLazyDLL("vulkan-1.dll")
		api := &vulkanAPI{}

		if err := registerProc(dll, "vkEnumerateInstanceExtensionProperties", &api.enumerateInstanceExtensionProperties); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkEnumerateInstanceLayerProperties", &api.enumerateInstanceLayerProperties); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkCreateInstance", &api.createInstance); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkDestroyInstance", &api.destroyInstance); err != nil {
			defaultAPIErr = err
			return
		}
		tryRegisterProc(dll, "vkEnumeratePhysicalDevices", &api.enumeratePhysicalDevices)
		tryRegisterProc(dll, "vkGetPhysicalDeviceProperties", &api.getPhysicalDeviceProperties)
		tryRegisterProc(dll, "vkGetPhysicalDeviceQueueFamilyProperties", &api.getPhysicalDeviceQueueFamilyProperties)
		tryRegisterProc(dll, "vkGetPhysicalDeviceMemoryProperties", &api.getPhysicalDeviceMemoryProperties)
		tryRegisterProc(dll, "vkEnumerateDeviceExtensionProperties", &api.enumerateDeviceExtensionProperties)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceSupportKHR", &api.getPhysicalDeviceSurfaceSupportKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceCapabilitiesKHR", &api.getPhysicalDeviceSurfaceCapabilitiesKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceFormatsKHR", &api.getPhysicalDeviceSurfaceFormatsKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfacePresentModesKHR", &api.getPhysicalDeviceSurfacePresentModesKHR)
		tryRegisterProc(dll, "vkCreateDevice", &api.createDevice)
		tryRegisterProc(dll, "vkDestroyDevice", &api.destroyDevice)
		tryRegisterProc(dll, "vkGetDeviceQueue", &api.getDeviceQueue)
		tryRegisterProc(dll, "vkDeviceWaitIdle", &api.deviceWaitIdle)
		tryRegisterProc(dll, "vkCreateWin32SurfaceKHR", &api.createWin32SurfaceKHR)
		tryRegisterProc(dll, "vkDestroySurfaceKHR", &api.destroySurfaceKHR)
		tryRegisterProc(dll, "vkCreateSwapchainKHR", &api.createSwapchainKHR)
		tryRegisterProc(dll, "vkDestroySwapchainKHR", &api.destroySwapchainKHR)
		tryRegisterProc(dll, "vkGetSwapchainImagesKHR", &api.getSwapchainImagesKHR)
		tryRegisterProc(dll, "vkAcquireNextImageKHR", &api.acquireNextImageKHR)
		tryRegisterProc(dll, "vkQueuePresentKHR", &api.queuePresentKHR)
		tryRegisterProc(dll, "vkQueueSubmit", &api.queueSubmit)
		tryRegisterProc(dll, "vkQueueWaitIdle", &api.queueWaitIdle)
		tryRegisterProc(dll, "vkCreateImageView", &api.createImageView)
		tryRegisterProc(dll, "vkDestroyImageView", &api.destroyImageView)
		tryRegisterProc(dll, "vkCreateRenderPass", &api.createRenderPass)
		tryRegisterProc(dll, "vkDestroyRenderPass", &api.destroyRenderPass)
		tryRegisterProc(dll, "vkCreateFramebuffer", &api.createFramebuffer)
		tryRegisterProc(dll, "vkDestroyFramebuffer", &api.destroyFramebuffer)
		tryRegisterProc(dll, "vkCreateCommandPool", &api.createCommandPool)
		tryRegisterProc(dll, "vkDestroyCommandPool", &api.destroyCommandPool)
		tryRegisterProc(dll, "vkAllocateCommandBuffers", &api.allocateCommandBuffers)
		tryRegisterProc(dll, "vkFreeCommandBuffers", &api.freeCommandBuffers)
		tryRegisterProc(dll, "vkBeginCommandBuffer", &api.beginCommandBuffer)
		tryRegisterProc(dll, "vkEndCommandBuffer", &api.endCommandBuffer)
		tryRegisterProc(dll, "vkResetCommandBuffer", &api.resetCommandBuffer)
		tryRegisterProc(dll, "vkCmdBeginRenderPass", &api.cmdBeginRenderPass)
		tryRegisterProc(dll, "vkCmdEndRenderPass", &api.cmdEndRenderPass)
		tryRegisterProc(dll, "vkCmdBindPipeline", &api.cmdBindPipeline)
		tryRegisterProc(dll, "vkCmdSetViewport", &api.cmdSetViewport)
		tryRegisterProc(dll, "vkCmdSetScissor", &api.cmdSetScissor)
		tryRegisterProc(dll, "vkCmdBindVertexBuffers", &api.cmdBindVertexBuffers)
		tryRegisterProc(dll, "vkCmdBindDescriptorSets", &api.cmdBindDescriptorSets)
		tryRegisterProc(dll, "vkCmdDraw", &api.cmdDraw)
		tryRegisterProc(dll, "vkCreateShaderModule", &api.createShaderModule)
		tryRegisterProc(dll, "vkDestroyShaderModule", &api.destroyShaderModule)
		tryRegisterProc(dll, "vkCreatePipelineLayout", &api.createPipelineLayout)
		tryRegisterProc(dll, "vkDestroyPipelineLayout", &api.destroyPipelineLayout)
		tryRegisterProc(dll, "vkCreateGraphicsPipelines", &api.createGraphicsPipelines)
		tryRegisterProc(dll, "vkDestroyPipeline", &api.destroyPipeline)
		tryRegisterProc(dll, "vkCreateDescriptorSetLayout", &api.createDescriptorSetLayout)
		tryRegisterProc(dll, "vkDestroyDescriptorSetLayout", &api.destroyDescriptorSetLayout)
		tryRegisterProc(dll, "vkCreateDescriptorPool", &api.createDescriptorPool)
		tryRegisterProc(dll, "vkDestroyDescriptorPool", &api.destroyDescriptorPool)
		tryRegisterProc(dll, "vkAllocateDescriptorSets", &api.allocateDescriptorSets)
		tryRegisterProc(dll, "vkUpdateDescriptorSets", &api.updateDescriptorSets)
		tryRegisterProc(dll, "vkCreateSampler", &api.createSampler)
		tryRegisterProc(dll, "vkDestroySampler", &api.destroySampler)
		tryRegisterProc(dll, "vkCreateBuffer", &api.createBuffer)
		tryRegisterProc(dll, "vkDestroyBuffer", &api.destroyBuffer)
		tryRegisterProc(dll, "vkGetBufferMemoryRequirements", &api.getBufferMemoryRequirements)
		tryRegisterProc(dll, "vkAllocateMemory", &api.allocateMemory)
		tryRegisterProc(dll, "vkFreeMemory", &api.freeMemory)
		tryRegisterProc(dll, "vkBindBufferMemory", &api.bindBufferMemory)
		tryRegisterProc(dll, "vkMapMemory", &api.mapMemory)
		tryRegisterProc(dll, "vkUnmapMemory", &api.unmapMemory)
		tryRegisterProc(dll, "vkCreateImage", &api.createImage)
		tryRegisterProc(dll, "vkDestroyImage", &api.destroyImage)
		tryRegisterProc(dll, "vkGetImageMemoryRequirements", &api.getImageMemoryRequirements)
		tryRegisterProc(dll, "vkBindImageMemory", &api.bindImageMemory)
		tryRegisterProc(dll, "vkCreateSemaphore", &api.createSemaphore)
		tryRegisterProc(dll, "vkDestroySemaphore", &api.destroySemaphore)
		tryRegisterProc(dll, "vkCreateFence", &api.createFence)
		tryRegisterProc(dll, "vkDestroyFence", &api.destroyFence)
		tryRegisterProc(dll, "vkWaitForFences", &api.waitForFences)
		tryRegisterProc(dll, "vkResetFences", &api.resetFences)
		tryRegisterProc(dll, "vkCmdPipelineBarrier", &api.cmdPipelineBarrier)
		tryRegisterProc(dll, "vkCmdCopyBufferToImage", &api.cmdCopyBufferToImage)
		defaultAPI = api
	})
	return defaultAPI, defaultAPIErr
}

func registerProc(dll *windows.LazyDLL, name string, target any) error {
	proc := dll.NewProc(name)
	if err := proc.Find(); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	purego.RegisterFunc(target, proc.Addr())
	return nil
}

func tryRegisterProc(dll *windows.LazyDLL, name string, target any) {
	proc := dll.NewProc(name)
	if err := proc.Find(); err != nil {
		return
	}
	purego.RegisterFunc(target, proc.Addr())
}
