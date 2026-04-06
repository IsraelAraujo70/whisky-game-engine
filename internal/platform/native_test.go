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
}
