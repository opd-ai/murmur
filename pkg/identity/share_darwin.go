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
		cmd := exec.Command("pbcopy")
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("creating pipe to pbcopy: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("starting pbcopy: %w", err)
		}

		if _, err := pipe.Write([]byte(content)); err != nil {
			return fmt.Errorf("writing to pbcopy: %w", err)
		}

		if err := pipe.Close(); err != nil {
			return fmt.Errorf("closing pipe to pbcopy: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("waiting for pbcopy: %w", err)
		}

		return nil

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
