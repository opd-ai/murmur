// Package identity provides macOS/Darwin-specific sharing integration.
//go:build darwin

package identity

import (
	"fmt"
	"os/exec"
)

// openSystemShareImpl opens the system share interface on macOS.
// Uses open command for URIs, email, and images.
func openSystemShareImpl(content string, opts ShareOptions) error {
	switch opts.Method {
	case ShareText:
		// On macOS, copy to clipboard using pbcopy.
		return writeToClipboard(content, "pbcopy")

	case ShareEmail:
		// Use open to open mailto: URL with default email client.
		cmd := exec.Command("open", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening email client: %w", err)
		}
		return nil

	case ShareQR:
		// Use open to open the QR code image with default image viewer.
		cmd := exec.Command("open", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening QR code: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported share method: %d", opts.Method)
	}
}
