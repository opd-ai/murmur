// Package identity provides sharing integration for MURMUR invitations.
// Per VIRAL_GROWTH_AND_ONBOARDING.md and ROADMAP.md line 787, sharing
// enables system share sheets for text, email, and QR code distribution.
package identity

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

// ShareMethod represents different methods of sharing invitations.
type ShareMethod int

const (
	// ShareText shares the invitation as plain text (URI).
	ShareText ShareMethod = iota
	// ShareEmail shares the invitation via email client.
	ShareEmail
	// ShareQR shares the invitation as a QR code image.
	ShareQR
)

// ShareOptions configures invitation sharing behavior.
type ShareOptions struct {
	Method      ShareMethod // Sharing method to use
	Subject     string      // Email subject (for ShareEmail)
	Body        string      // Email body (for ShareEmail)
	QRSize      int         // QR code size in pixels (for ShareQR, default 512)
	TempDirPath string      // Temporary directory for QR images (default: OS temp)
}

// DefaultShareOptions returns default sharing options.
func DefaultShareOptions() ShareOptions {
	return ShareOptions{
		Method:      ShareText,
		Subject:     "Join me on MURMUR",
		Body:        "I'd like to invite you to join MURMUR, a decentralized social network.",
		QRSize:      512,
		TempDirPath: os.TempDir(),
	}
}

// Share shares an invitation using system-native sharing mechanisms.
// Platform support:
// - Desktop (Linux/macOS/Windows): Creates shareable content and returns paths/data
// - Mobile (via Gomobile): Would integrate with native share sheets
// Returns the shared content path or data URL, and any error.
func (inv *Invitation) Share(opts ShareOptions) (string, error) {
	switch opts.Method {
	case ShareText:
		return inv.shareText(opts)
	case ShareEmail:
		return inv.shareEmail(opts)
	case ShareQR:
		return inv.shareQR(opts)
	default:
		return "", fmt.Errorf("unknown share method: %d", opts.Method)
	}
}

// shareText creates a shareable text representation.
// Returns the invitation URI.
func (inv *Invitation) shareText(opts ShareOptions) (string, error) {
	uri, err := inv.EncodeURI()
	if err != nil {
		return "", fmt.Errorf("encoding invitation URI: %w", err)
	}

	// On desktop, we return the URI for clipboard or file writing.
	// On mobile platforms, this would invoke native share sheet.
	return uri, nil
}

// shareEmail creates an email with the invitation.
// Returns a mailto: URL on desktop, invokes native email client on mobile.
func (inv *Invitation) shareEmail(opts ShareOptions) (string, error) {
	uri, err := inv.EncodeURI()
	if err != nil {
		return "", fmt.Errorf("encoding invitation URI: %w", err)
	}

	// Build email body with invitation.
	body := opts.Body
	if inv.WelcomeMessage != "" {
		body += fmt.Sprintf("\n\nPersonal message: %s", inv.WelcomeMessage)
	}
	body += fmt.Sprintf("\n\nClick to join: %s", uri)

	// Create mailto: URL (cross-platform standard).
	// On desktop, applications can open this with xdg-open (Linux),
	// open (macOS), or start (Windows).
	mailtoURL := fmt.Sprintf("mailto:?subject=%s&body=%s",
		urlEncode(opts.Subject),
		urlEncode(body))

	return mailtoURL, nil
}

// shareQR creates and saves a QR code image, returning its path.
// On desktop, writes to temp directory. On mobile, would integrate with
// native share sheet for image sharing.
func (inv *Invitation) shareQR(opts ShareOptions) (string, error) {
	// Generate QR code image.
	qrImg, err := inv.GenerateQRCode(opts.QRSize)
	if err != nil {
		return "", fmt.Errorf("generating QR code: %w", err)
	}

	// Create temporary file for QR code.
	tempDir := opts.TempDirPath
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Ensure temp directory exists.
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	// Create QR code filename with peer ID for uniqueness.
	filename := fmt.Sprintf("murmur-invite-%s.png", inv.PeerID.ShortString())
	filepath := filepath.Join(tempDir, filename)

	// Write QR code to file.
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("creating QR file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, qrImg); err != nil {
		return "", fmt.Errorf("encoding PNG: %w", err)
	}

	return filepath, nil
}

// OpenSystemShare attempts to open the system's native sharing interface.
// This is a platform-specific operation that uses OS commands where available.
// Returns an error if the platform doesn't support native sharing or if
// the share operation fails.
func (inv *Invitation) OpenSystemShare(opts ShareOptions) error {
	content, err := inv.Share(opts)
	if err != nil {
		return fmt.Errorf("preparing share content: %w", err)
	}

	// Platform-specific sharing invocation.
	switch runtime.GOOS {
	case "linux":
		return openSystemShareImpl(content, opts)
	case "darwin":
		return openSystemShareImpl(content, opts)
	case "windows":
		return openSystemShareImpl(content, opts)
	case "android", "ios":
		// On mobile platforms built with Gomobile, this would invoke
		// the native share sheet. For now, we return an error indicating
		// that mobile share sheet integration requires platform bindings.
		return fmt.Errorf("mobile share sheet requires platform bindings (not yet implemented)")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// urlEncode performs simple URL encoding for mailto: URLs.
func urlEncode(s string) string {
	// Simple encoding for common characters. For production use,
	// url.QueryEscape would be more appropriate, but we avoid
	// importing net/url for this basic functionality.
	s = replaceAll(s, " ", "%20")
	s = replaceAll(s, "\n", "%0A")
	s = replaceAll(s, ":", "%3A")
	s = replaceAll(s, "/", "%2F")
	return s
}

// replaceAll is a simple string replacement helper.
func replaceAll(s, old, new string) string {
	result := ""
	for len(s) > 0 {
		idx := findIndex(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// findIndex finds the first occurrence of substr in s.
func findIndex(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
