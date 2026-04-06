package vulkan

import (
	"errors"
	"testing"
)

func TestNewInstanceNotImplemented(t *testing.T) {
	_, err := NewInstance(Options{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
