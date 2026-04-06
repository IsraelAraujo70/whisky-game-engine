//go:build linux

package nativewindow

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/wayland"
)

func TestSelectedLinuxWindowSystemExplicitOverride(t *testing.T) {
	t.Setenv("WHISKY_WINDOW_SYSTEM", "wayland")
	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("DISPLAY", ":0")
	t.Setenv("WAYLAND_DISPLAY", "")

	if got := selectedLinuxWindowSystem(); got != windowSystemWayland {
		t.Fatalf("expected %q, got %q", windowSystemWayland, got)
	}
}

func TestSelectedLinuxWindowSystemPrefersWaylandSession(t *testing.T) {
	t.Setenv("WHISKY_WINDOW_SYSTEM", "")
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", ":0")

	if got := selectedLinuxWindowSystem(); got != windowSystemWayland {
		t.Fatalf("expected %q, got %q", windowSystemWayland, got)
	}
}

func TestSelectedLinuxWindowSystemFallsBackToX11WhenDisplayPresent(t *testing.T) {
	t.Setenv("WHISKY_WINDOW_SYSTEM", "")
	t.Setenv("XDG_SESSION_TYPE", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", ":0")

	if got := selectedLinuxWindowSystem(); got != windowSystemX11 {
		t.Fatalf("expected %q, got %q", windowSystemX11, got)
	}
}

func TestShouldFallbackToX11(t *testing.T) {
	t.Setenv("DISPLAY", ":0")

	if !shouldFallbackToX11(wayland.ErrNotImplemented) {
		t.Fatal("expected fallback to X11 when wayland is unavailable and DISPLAY exists")
	}
}

func TestShouldNotFallbackToX11WithoutDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")

	if shouldFallbackToX11(wayland.ErrNotImplemented) {
		t.Fatal("did not expect fallback to X11 without DISPLAY")
	}
}
