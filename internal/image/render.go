package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

// RenderOptions configures image rendering
type RenderOptions struct {
	Width  int    // Terminal width for rendering
	Height int    // Terminal height for rendering
	Data   []byte // Raw image data
	Path   string // Or path to image file (alternative to Data)
}

// Render converts image to terminal-displayable output
func Render(caps Capabilities, opts RenderOptions) (string, error) {
	switch caps.Renderer {
	case RendererChafa:
		return renderWithChafa(caps.ToolPath, opts)
	case RendererViu:
		return renderWithViu(caps.ToolPath, opts)
	case RendererASCII:
		// Use native Go ASCII art converter
		return renderASCIIArt(opts)
	default:
		return "", fmt.Errorf("no image renderer available")
	}
}

// renderWithChafa uses chafa to render image in terminal
func renderWithChafa(toolPath string, opts RenderOptions) (string, error) {
	var args = []string{
		"--size", fmt.Sprintf("%dx%d", opts.Width, opts.Height),
		"--format", "symbols", // Universal fallback, works in all terminals
		"--colors", "full",    // Use full colors
	}

	var cmd *exec.Cmd
	var tempFile string

	if opts.Path != "" {
		args = append(args, opts.Path)
		cmd = exec.Command(toolPath, args...)
	} else if len(opts.Data) > 0 {
		// Write to temp file (chafa works better with files than stdin for some formats)
		var err error
		tempFile, err = writeTempImage(opts.Data)
		if err != nil {
			// Fallback to stdin
			args = append(args, "-")
			cmd = exec.Command(toolPath, args...)
			cmd.Stdin = bytes.NewReader(opts.Data)
		} else {
			defer os.Remove(tempFile)
			args = append(args, tempFile)
			cmd = exec.Command(toolPath, args...)
		}
	} else {
		return "", fmt.Errorf("no image data provided")
	}

	var output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("chafa error: %w", err)
	}

	return string(output), nil
}

// renderWithViu uses viu to render image in terminal
func renderWithViu(toolPath string, opts RenderOptions) (string, error) {
	var args = []string{
		"-w", fmt.Sprintf("%d", opts.Width),
		"-h", fmt.Sprintf("%d", opts.Height),
	}

	var cmd *exec.Cmd
	var tempFile string

	if opts.Path != "" {
		args = append(args, opts.Path)
		cmd = exec.Command(toolPath, args...)
	} else if len(opts.Data) > 0 {
		var err error
		tempFile, err = writeTempImage(opts.Data)
		if err != nil {
			// Fallback to stdin
			args = append(args, "-")
			cmd = exec.Command(toolPath, args...)
			cmd.Stdin = bytes.NewReader(opts.Data)
		} else {
			defer os.Remove(tempFile)
			args = append(args, tempFile)
			cmd = exec.Command(toolPath, args...)
		}
	} else {
		return "", fmt.Errorf("no image data provided")
	}

	var output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("viu error: %w", err)
	}

	return string(output), nil
}

// ASCII art character ramp (from dark to light)
var asciiRamp = []rune(" .:-=+*#%@")

// renderASCIIArt converts image data to ASCII art
func renderASCIIArt(opts RenderOptions) (string, error) {
	if len(opts.Data) == 0 && opts.Path == "" {
		return renderASCIIPlaceholder(opts)
	}

	var reader *bytes.Reader
	if len(opts.Data) > 0 {
		reader = bytes.NewReader(opts.Data)
	} else {
		var data, err = os.ReadFile(opts.Path)
		if err != nil {
			return renderASCIIPlaceholder(opts)
		}
		reader = bytes.NewReader(data)
	}

	// Decode the image
	img, _, err := image.Decode(reader)
	if err != nil {
		return renderASCIIPlaceholder(opts)
	}

	var bounds = img.Bounds()
	var imgWidth = bounds.Dx()
	var imgHeight = bounds.Dy()

	// Calculate aspect ratio (terminal chars are ~2x taller than wide)
	var targetWidth = opts.Width
	var targetHeight = opts.Height
	if targetWidth <= 0 {
		targetWidth = 60
	}
	if targetHeight <= 0 {
		targetHeight = 20
	}

	// Scale to fit
	var scaleX = float64(imgWidth) / float64(targetWidth)
	var scaleY = float64(imgHeight) / float64(targetHeight*2) // *2 for aspect ratio

	var scale = scaleX
	if scaleY > scale {
		scale = scaleY
	}

	var outWidth = int(float64(imgWidth) / scale)
	var outHeight = int(float64(imgHeight) / scale / 2) // /2 for aspect ratio

	if outWidth > targetWidth {
		outWidth = targetWidth
	}
	if outHeight > targetHeight {
		outHeight = targetHeight
	}

	var lines []string
	for y := 0; y < outHeight; y++ {
		var line strings.Builder
		for x := 0; x < outWidth; x++ {
			// Sample the corresponding pixel
			var srcX = int(float64(x) * scale)
			var srcY = int(float64(y) * scale * 2)

			if srcX >= imgWidth {
				srcX = imgWidth - 1
			}
			if srcY >= imgHeight {
				srcY = imgHeight - 1
			}

			var c = img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)
			var gray = colorToGray(c)

			// Map gray value to ASCII character
			var idx = int(float64(gray) / 255.0 * float64(len(asciiRamp)-1))
			if idx >= len(asciiRamp) {
				idx = len(asciiRamp) - 1
			}
			line.WriteRune(asciiRamp[idx])
		}
		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n"), nil
}

// colorToGray converts a color to grayscale value (0-255)
func colorToGray(c color.Color) uint8 {
	var r, g, b, _ = c.RGBA()
	// Use luminance formula
	var gray = (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256
	return uint8(gray)
}

// renderASCIIPlaceholder renders an ASCII placeholder when image can't be decoded
func renderASCIIPlaceholder(opts RenderOptions) (string, error) {
	var width = opts.Width
	if width < 40 {
		width = 40
	}

	var lines []string
	var border = strings.Repeat("─", width-2)

	lines = append(lines, "┌"+border+"┐")
	lines = append(lines, "│"+centerText(" ", width-2)+"│")
	lines = append(lines, "│"+centerText("┌─────────┐", width-2)+"│")
	lines = append(lines, "│"+centerText("│  IMAGE  │", width-2)+"│")
	lines = append(lines, "│"+centerText("│   [?]   │", width-2)+"│")
	lines = append(lines, "│"+centerText("└─────────┘", width-2)+"│")
	lines = append(lines, "│"+centerText(" ", width-2)+"│")
	lines = append(lines, "│"+centerText("Could not decode image", width-2)+"│")
	lines = append(lines, "│"+centerText("Press Enter to open externally", width-2)+"│")
	lines = append(lines, "│"+centerText(" ", width-2)+"│")
	lines = append(lines, "└"+border+"┘")

	return strings.Join(lines, "\n"), nil
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	var textLen = len([]rune(text))
	if textLen >= width {
		return string([]rune(text)[:width])
	}
	var padding = (width - textLen) / 2
	var result = strings.Repeat(" ", padding) + text
	result += strings.Repeat(" ", width-len([]rune(result)))
	return result
}

// writeTempImage writes image data to a temporary file
func writeTempImage(data []byte) (string, error) {
	var tmpDir = os.TempDir()
	var tmpFile = filepath.Join(tmpDir, fmt.Sprintf("miau-image-%d.tmp", os.Getpid()))

	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return "", err
	}

	return tmpFile, nil
}

// FormatSize formats bytes into human-readable size
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	var div, exp int64 = unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
