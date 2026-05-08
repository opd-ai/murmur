package views

import (
"fmt"
"image/color"
"strings"

"github.com/opd-ai/murmur/pkg/identity/keys"
"github.com/opd-ai/murmur/pkg/identity/modes"
"github.com/opd-ai/murmur/pkg/identity/sigils"
"github.com/tyler-smith/go-bip39"
tea "github.com/charmbracelet/bubbletea"
)

// IdentityModel handles identity and privacy mode UI.
type IdentityModel struct {
Session      *SessionState
Mnemonic     string
RecoveryText string
Status       string
}

// NewIdentityModel creates the identity view model.
func NewIdentityModel(session *SessionState) IdentityModel {
return IdentityModel{Session: session, Status: "g: generate keypair, 1-4: mode, r: validate recovery mnemonic"}
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
case "backspace":
if len(m.RecoveryText) > 0 {
m.RecoveryText = m.RecoveryText[:len(m.RecoveryText)-1]
}
default:
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
if m.Session.KeyPair != nil {
pub = fmt.Sprintf("%x", m.Session.KeyPair.PublicKey)
if len(pub) > 16 {
pub = pub[:16] + "..."
}
sigilPreview = renderANSISigil(m.Session.KeyPair.PublicKey)
}
mnemonic := m.Mnemonic
if mnemonic == "" {
mnemonic = "<not-generated>"
}
return fmt.Sprintf("Mode: %s\nPublic key: %s\nMnemonic: %s\n\nSigil:\n%s\nRecovery input: %s\nStatus: %s", mode, pub, mnemonic, sigilPreview, m.RecoveryText, m.Status)
}

func renderANSISigil(pub []byte) string {
s := sigils.Generate(pub)
if s == nil || s.Image == nil {
return "<no sigil>"
}
var b strings.Builder
bounds := s.Image.Bounds()
stepX := max(1, bounds.Dx()/16)
stepY := max(1, bounds.Dy()/8)
for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
c := color.RGBAModel.Convert(s.Image.At(x, y)).(color.RGBA)
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
