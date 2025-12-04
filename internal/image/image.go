package image

import (
	"os/exec"
)

// Renderer represents available image rendering backends
type Renderer string

const (
	RendererChafa Renderer = "chafa"
	RendererViu   Renderer = "viu"
	RendererASCII Renderer = "ascii"
	RendererNone  Renderer = "none"
)

// Capabilities holds detected terminal image capabilities
type Capabilities struct {
	Renderer     Renderer
	ToolPath     string
	SupportsSize bool
}

// DetectCapabilities checks available image rendering options
// Priority: chafa > viu > ASCII fallback
func DetectCapabilities() Capabilities {
	// Check chafa first (preferred - best terminal protocol support)
	if path, err := exec.LookPath("chafa"); err == nil {
		return Capabilities{
			Renderer:     RendererChafa,
			ToolPath:     path,
			SupportsSize: true,
		}
	}

	// Check viu as fallback
	if path, err := exec.LookPath("viu"); err == nil {
		return Capabilities{
			Renderer:     RendererViu,
			ToolPath:     path,
			SupportsSize: true,
		}
	}

	// ASCII fallback always available
	return Capabilities{
		Renderer:     RendererASCII,
		SupportsSize: false,
	}
}

// HasGraphicsSupport returns true if terminal can render actual images
func (c Capabilities) HasGraphicsSupport() bool {
	return c.Renderer == RendererChafa || c.Renderer == RendererViu
}

// String returns a human-readable description of the renderer
func (c Capabilities) String() string {
	switch c.Renderer {
	case RendererChafa:
		return "chafa (terminal graphics)"
	case RendererViu:
		return "viu (terminal graphics)"
	case RendererASCII:
		return "ASCII art (native Go)"
	default:
		return "none"
	}
}
