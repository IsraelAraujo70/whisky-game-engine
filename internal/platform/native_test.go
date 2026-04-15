package platform

import "testing"

func TestNativeWindowHandleWin32Shape(t *testing.T) {
	handle := NativeWindowHandle{
		Kind:     NativeWindowKindWin32,
		Window:   0x1234,
		Instance: 0x5678,
	}

	if handle.Kind != NativeWindowKindWin32 {
		t.Fatalf("expected kind %q, got %q", NativeWindowKindWin32, handle.Kind)
	}
	if handle.Window != 0x1234 {
		t.Fatalf("expected window handle 0x1234, got %#x", handle.Window)
	}
	if handle.Instance != 0x5678 {
		t.Fatalf("expected instance handle 0x5678, got %#x", handle.Instance)
	}
	if handle.Display != 0 {
		t.Fatalf("expected display handle 0 for Win32, got %#x", handle.Display)
	}
	if handle.View != 0 {
		t.Fatalf("expected view handle 0 for Win32, got %#x", handle.View)
	}
	if handle.Layer != 0 {
		t.Fatalf("expected layer handle 0 for Win32, got %#x", handle.Layer)
	}
}

func TestNativeWindowHandleCocoaShape(t *testing.T) {
	handle := NativeWindowHandle{
		Kind:   NativeWindowKindCocoa,
		Window: 0x1000,
		View:   0x2000,
		Layer:  0x3000,
	}

	if handle.Kind != NativeWindowKindCocoa {
		t.Fatalf("expected kind %q, got %q", NativeWindowKindCocoa, handle.Kind)
	}
	if handle.Window != 0x1000 {
		t.Fatalf("expected window handle 0x1000, got %#x", handle.Window)
	}
	if handle.View != 0x2000 {
		t.Fatalf("expected view handle 0x2000, got %#x", handle.View)
	}
	if handle.Layer != 0x3000 {
		t.Fatalf("expected layer handle 0x3000, got %#x", handle.Layer)
	}
	if handle.Display != 0 {
		t.Fatalf("expected display handle 0 for Cocoa, got %#x", handle.Display)
	}
	if handle.Instance != 0 {
		t.Fatalf("expected instance handle 0 for Cocoa, got %#x", handle.Instance)
	}
}
