// Package identity provides Windows-specific sharing integration.
//go:build windows

package identity

import (
	"fmt"
	"os/exec"
)

// openSystemShareImpl opens the system share interface on Windows.
// Uses cmd.exe commands for clipboard, start for URIs and email.
func openSystemShareImpl(content string, opts ShareOptions) error {
	switch opts.Method {
	case ShareText:
		// On Windows, copy to clipboard using clip.exe.
		return writeToClipboard(content, "cmd", "/c", "clip")

	case ShareEmail:
		// Use start to open mailto: URL with default email client.
		cmd := exec.Command("cmd", "/c", "start", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening email client: %w", err)
		}
		return nil

	case ShareQR:
		// Use start to open the QR code image with default image viewer.
		cmd := exec.Command("cmd", "/c", "start", content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("opening QR code: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported share method: %d", opts.Method)
	}
}
