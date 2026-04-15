//go:build darwin

package metal

import (
	"strings"
	"testing"
)

func TestShaderSourceExportsEntryPoints(t *testing.T) {
	if !strings.Contains(metalShaderSource, "vertex VertexOut whiskyVertex") {
		t.Fatalf("expected vertex entry point in shader source")
	}
	if !strings.Contains(metalShaderSource, "fragment float4 whiskyFragment") {
		t.Fatalf("expected fragment entry point in shader source")
	}
}

func TestNextVertexCapacityGrowsExponentially(t *testing.T) {
	if got := nextVertexCapacity(0, 1); got < 256 {
		t.Fatalf("expected minimum capacity of 256, got %d", got)
	}
	if got := nextVertexCapacity(256, 300); got != 512 {
		t.Fatalf("expected next capacity 512, got %d", got)
	}
}
