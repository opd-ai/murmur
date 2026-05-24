package views

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/identity/declarations"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/identity/sigils"
	"github.com/tyler-smith/go-bip39"
)

// IdentityModel handles identity and privacy mode UI.
type IdentityModel struct {
	Session      *SessionState
	Mnemonic     string
	RecoveryText string
	RecoverMode  bool
	DisplayName  string
	Bio          string
	Declaration  *declarations.Declaration
	ProfileMode  bool
	EditTarget   string
	Status       string
}

// NewIdentityModel creates the identity view model.
func NewIdentityModel(session *SessionState) IdentityModel {
	return IdentityModel{
		Session:     session,
		Status:      "g: generate keypair, 1-4: mode, R: recovery mode, d: declaration mode, u: publish declaration",
		DisplayName: "murmur-user",
		EditTarget:  "name",
	}
}

// Update handles identity interactions.
func (m IdentityModel) Update(msg tea.Msg) (IdentityModel, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "g":
		kp, err := keys.GenerateKeyPair()
		if err != nil {
			m.Status = "key generation failed: " + err.Error()
			return m, nil
		}
		m.Session.KeyPair = kp
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			m.Status = "entropy generation failed: " + err.Error()
			return m, nil
		}
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			m.Status = "mnemonic generation failed: " + err.Error()
			return m, nil
		}
		m.Mnemonic = mnemonic
		m.Status = "identity generated"
	case "1":
		m.transitionMode(modes.Open)
	case "2":
		m.Session.ModeManager.SetSpecterAvailable(true)
		m.transitionMode(modes.Hybrid)
	case "3":
		m.Session.ModeManager.SetSpecterAvailable(true)
		m.transitionMode(modes.Guarded)
	case "4":
		m.Session.ModeManager.SetSpecterAvailable(true)
		m.Session.ModeManager.SetShroudAvailable(true)
		m.transitionMode(modes.Fortress)
	case "r":
		if bip39.IsMnemonicValid(m.RecoveryText) {
			m.Status = "recovery mnemonic valid"
		} else {
			m.Status = "recovery mnemonic invalid"
		}
	case "d":
		m.ProfileMode = !m.ProfileMode
		if m.ProfileMode {
			m.Status = "declaration mode enabled (n=append name, b=append bio, u=publish)"
			m.EditTarget = "name"
		} else {
			m.Status = "declaration mode disabled"
		}
	case "u":
		if m.Session.KeyPair == nil {
			m.Status = "generate keypair before publishing declaration"
			return m, nil
		}
		decl, err := declarations.New(m.Session.KeyPair, strings.TrimSpace(m.DisplayName))
		if err != nil {
			m.Status = "declaration create failed: " + err.Error()
			return m, nil
		}
		if err := decl.SetBio(strings.TrimSpace(m.Bio)); err != nil {
			m.Status = "Error: " + err.Error()
			return m, nil
		}
		decl.SetPrivacyMode(m.Session.ModeManager.Current())
		if err := decl.Sign(m.Session.KeyPair); err != nil {
			m.Status = "declaration sign failed: " + err.Error()
			return m, nil
		}
		m.Declaration = decl
		m.Status = fmt.Sprintf("declaration published v%d", decl.Version)
	case "R":
		m.RecoverMode = !m.RecoverMode
		if m.RecoverMode {
			m.Status = "recovery mode enabled (enter mnemonic and press r)"
		} else {
			m.Status = "recovery mode disabled"
		}
	case "backspace":
		if m.ProfileMode {
			if m.EditTarget == "bio" && len(m.Bio) > 0 {
				m.Bio = m.Bio[:len(m.Bio)-1]
			} else if m.EditTarget != "bio" && len(m.DisplayName) > 0 {
				m.DisplayName = m.DisplayName[:len(m.DisplayName)-1]
			}
		} else if len(m.RecoveryText) > 0 {
			m.RecoveryText = m.RecoveryText[:len(m.RecoveryText)-1]
		}
	default:
		if m.ProfileMode && k.String() == "n" {
			m.EditTarget = "name"
			m.Status = "editing display name"
			return m, nil
		}
		if m.ProfileMode && k.String() == "b" {
			m.EditTarget = "bio"
			m.Status = "editing bio"
			return m, nil
		}
		if m.ProfileMode && len(k.Runes) > 0 {
			if m.EditTarget == "bio" {
				m.Bio += string(k.Runes)
			} else {
				m.DisplayName += string(k.Runes)
			}
			return m, nil
		}
		if len(k.Runes) > 0 {
			m.RecoveryText += string(k.Runes)
		}
	}
	return m, nil
}

func (m *IdentityModel) transitionMode(target modes.Mode) {
	if err := m.Session.ModeManager.Transition(target); err != nil {
		m.Status = "mode transition blocked: " + err.Error()
		return
	}
	m.Status = "mode set to " + target.String()
}

// View renders identity details.
func (m IdentityModel) View(width int) string {
	mode := m.Session.ModeManager.Current().String()
	pub := "<not-generated>"
	sigilPreview := ""
	specterSigilPreview := ""
	maskedSigilPreview := ""
	fingerprint := "<none>"
	inviteCode := "<none>"
	declState := "<unpublished>"
	if m.Session.KeyPair != nil {
		pub = fmt.Sprintf("%x", m.Session.KeyPair.PublicKey)
		if len(pub) > 16 {
			pub = pub[:16] + "..."
		}
		full := fmt.Sprintf("%x", m.Session.KeyPair.PublicKey)
		if len(full) >= 8 {
			fingerprint = full[:8]
		}
		if len(full) >= 12 {
			inviteCode = "MURMUR-" + full[:6] + "-" + full[6:12]
		}
		sigilPreview = renderANSISigil(m.Session.KeyPair.PublicKey)
		specterSigilPreview = renderANSISigilFromSigil(sigils.GenerateSpecter(m.Session.KeyPair.PublicKey))
		maskedSigilPreview = renderANSISigilFromSigil(sigils.GenerateMaskedEvent(m.Session.KeyPair.PublicKey))
	}
	if m.Declaration != nil {
		declState = fmt.Sprintf("v%d signed=%t mode=%s", m.Declaration.Version, len(m.Declaration.Signature) > 0, m.Declaration.PrivacyMode.String())
	}
	mnemonic := m.Mnemonic
	if mnemonic == "" {
		mnemonic = "<not-generated>"
	}
	return fmt.Sprintf(
		"Mode: %s\nCooldown remaining: %s\nTraffic padding: %t\nPublic key: %s\nFingerprint: %s\nInvite code: %s\nMnemonic: %s\n\nSigil:\n%s\nRecovery mode: %t\nRecovery input: %s\nStatus: %s",
		mode,
		m.Session.ModeManager.CooldownRemaining().Round(time.Second),
		m.Session.ModeManager.IsTrafficPaddingEnabled(),
		pub,
		fingerprint,
		inviteCode,
		mnemonic,
		sigilPreview,
		m.RecoverMode,
		m.RecoveryText,
		m.Status,
	) + fmt.Sprintf("\n\nSpecter Sigil:\n%s\nMasked Sigil:\n%s\n\nDeclaration mode: %t (target=%s)\nDisplay name: %s\nBio: %s\nDeclaration: %s",
		specterSigilPreview,
		maskedSigilPreview,
		m.ProfileMode,
		m.EditTarget,
		m.DisplayName,
		m.Bio,
		declState,
	)
}

func renderANSISigil(pub []byte) string {
	return renderANSISigilFromSigil(sigils.Generate(pub))
}

func renderANSISigilFromSigil(s *sigils.Sigil) string {
	if s == nil || s.Image == nil {
		return "<no sigil>"
	}
	var b strings.Builder
	bounds := s.Image.Bounds()
	stepX := max(1, bounds.Dx()/16)
	stepY := max(1, bounds.Dy()/8)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			// F-NIL-1 fix: Use two-value type assertion to prevent panic.
			colorVal := color.RGBAModel.Convert(s.Image.At(x, y))
			c, ok := colorVal.(color.RGBA)
			if !ok {
				// Fallback: use gray value if not RGBA.
				c = color.RGBA{R: 128, G: 128, B: 128, A: 255}
			}
			l := int(c.R) + int(c.G) + int(c.B)
			switch {
			case l > 550:
				b.WriteRune('█')
			case l > 420:
				b.WriteRune('▓')
			case l > 300:
				b.WriteRune('▒')
			default:
				b.WriteRune('░')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
