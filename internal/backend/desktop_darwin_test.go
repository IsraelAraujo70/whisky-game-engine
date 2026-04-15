//go:build darwin

package backend

import "testing"

func TestNewDesktopUsesMetalConstructor(t *testing.T) {
	constructorCalled := false
	original := metalDesktopBackendFactory
	metalDesktopBackendFactory = func(title string, width, height int, keyMap map[string]string) (desktopBackend, error) {
		constructorCalled = true
		return nil, nil
	}
	defer func() {
		metalDesktopBackendFactory = original
	}()

	_, _ = NewDesktop("whisky", 320, 180, map[string]string{"left": "move_left"})
	if !constructorCalled {
		t.Fatalf("expected darwin NewDesktop to delegate to metal backend factory")
	}
}
