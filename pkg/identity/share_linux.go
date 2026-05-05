// Package identity provides Linux-specific sharing integration.
//go:build linux

package identity

import (
	"fmt"
	"os/exec"
)

// openSystemShareImpl opens the system share interface on Linux.
// Uses xdg-open for URIs and email, and attempts to use xdg-open
// for image files as well.
func openSystemShareImpl(content string, opts ShareOptions) error {
	switch opts.Method {
	case ShareText:
		// On Linux, copy to clipboard using xclip or xsel if available.
		// First try xclip.
		cmd := exec.Command("sh", "-c", "echo "+content+" | xclip -selection clipboard")
		if err := cmd.Run(); err == nil {
			return nil
		}
		// Try xsel as fallback.
		cmd = exec.Command("sh", "-c", "echo "+content+" | xsel --clipboard --input")
		if err := cmd.Run(); err == nil {
			return nil
		}
		// If neither is available, just return an error.
		return fmt.Errorf("clipboard tools not available (install xclip or xsel)")

	case ShareEmail:
		// Use xdg-open to open mailto: URL with default email client.
		cmd := exec.Command("xdg-open", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening email client: %w", err)
		}
		return nil

	case ShareQR:
		// Use xdg-open to open the QR code image with default image viewer.
		cmd := exec.Command("xdg-open", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening QR code: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported share method: %d", opts.Method)
	}
}
