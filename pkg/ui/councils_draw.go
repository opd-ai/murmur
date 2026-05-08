// Package ui — Phantom Council panel rendering functions.
// This file contains all Draw() methods for the CouncilPanel.
//
//go:build !test
// +build !test

package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Draw renders the council panel to the screen.
func (cp *CouncilPanel) Draw(screen *ebiten.Image) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.visible {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	panelW, panelH := 500, 450
	panelX := float32(sw-panelW) / 2
	panelY := float32(sh-panelH) / 2

	// Draw panel background.
	cp.drawPanelBackground(screen, panelX, panelY, float32(panelW), float32(panelH))

	// Draw title.
	cp.drawTitle(screen, panelX, panelY, float32(panelW))

	// Draw content based on mode.
	contentY := panelY + 50
	switch cp.mode {
	case CouncilModeList:
		cp.drawCouncilList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeCreate:
		cp.drawCreateForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeDetail:
		cp.drawCouncilDetail(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeMembers:
		cp.drawMembersList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeProposals:
		cp.drawProposalsList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeInvite:
		cp.drawInviteForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModePropose:
		cp.drawProposeForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeVote:
		cp.drawVoteForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	}

	// Draw messages.
	cp.drawMessages(screen, panelX, panelY+float32(panelH)-30, float32(panelW))
}

// drawPanelBackground draws the panel background with border.
func (cp *CouncilPanel) drawPanelBackground(screen *ebiten.Image, x, y, w, h float32) {
	// Background.
	vector.DrawFilledRect(screen, x, y, w, h, cp.theme.PanelBackground, true)

	// Border.
	vector.StrokeRect(screen, x, y, w, h, 2, cp.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (cp *CouncilPanel) drawTitle(screen *ebiten.Image, x, y, w float32) {
	title := "Phantom Councils"
	switch cp.mode {
	case CouncilModeCreate:
		title = "Create Council"
	case CouncilModeDetail:
		if cp.currentCouncil != nil {
			title = cp.currentCouncil.Name
		}
	case CouncilModeMembers:
		title = "Members"
	case CouncilModeProposals:
		title = "Proposals"
	case CouncilModeInvite:
		title = "Invite Member"
	case CouncilModePropose:
		title = "New Proposal"
	case CouncilModeVote:
		title = "Cast Vote"
	}

	// Draw title background.
	vector.DrawFilledRect(screen, x, y, w, 40, cp.theme.ButtonBackground, true)

	// Draw centered title text.
	drawUICenteredText(screen, title, float64(x)+float64(w)/2, float64(y)+14, cp.theme.TextPrimary)
}

// drawCouncilList draws the list of councils.
func (cp *CouncilPanel) drawCouncilList(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)
	itemHeight := float32(50)

	if len(cp.councils) == 0 {
		cp.drawEmptyState(screen, x, y, w, h)
		return
	}

	for i, council := range cp.councils {
		itemY := y + padding + float32(i)*itemHeight
		cp.drawCouncilItem(screen, x, itemY, w, itemHeight, i, council, padding)
	}

	cp.drawListHelpText(screen, x, y+h-25, w)
}

// drawEmptyState renders the empty council list state.
func (cp *CouncilPanel) drawEmptyState(screen *ebiten.Image, x, y, w, h float32) {
	emptyColor := cp.theme.TextSecondary
	vector.DrawFilledCircle(screen, x+w/2, y+h/3, 30, emptyColor, true)
}

// drawCouncilItem renders a single council list item.
func (cp *CouncilPanel) drawCouncilItem(screen *ebiten.Image, x, itemY, w, itemHeight float32, index int, council *CouncilInfo, padding float32) {
	if index == cp.selectedCouncil {
		vector.DrawFilledRect(screen, x+padding, itemY, w-padding*2, itemHeight-5, cp.theme.Selection, true)
	}

	nameColor := cp.theme.TextPrimary
	if council.State == CouncilStateDormant {
		nameColor = cp.theme.TextSecondary
	}
	vector.DrawFilledRect(screen, x+padding+5, itemY+5, 5, itemHeight-15, nameColor, true)

	cp.drawMemberCountIndicator(screen, x+w-padding-20, itemY+itemHeight/2, council.Members)
}

// drawMemberCountIndicator draws a circle sized by active member count.
func (cp *CouncilPanel) drawMemberCountIndicator(screen *ebiten.Image, x, y float32, members []CouncilMemberInfo) {
	memberCount := 0
	for _, m := range members {
		if m.Status == MemberStatusActive {
			memberCount++
		}
	}
	radius := float32(8 + memberCount)
	if radius > 20 {
		radius = 20
	}
	vector.DrawFilledCircle(screen, x, y, radius, cp.theme.AccentPrimary, true)
}

// drawListHelpText renders the help text at the bottom of the list.
func (cp *CouncilPanel) drawListHelpText(screen *ebiten.Image, x, y, w float32) {
	helpBg := cp.theme.PanelBackground
	helpBg.A = 200
	vector.DrawFilledRect(screen, x, y, w, 25, helpBg, true)
}

// drawCreateForm draws the council creation form.
func (cp *CouncilPanel) drawCreateForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Name field.
	cp.drawTextField(screen, x+padding, y+20, w-padding*2, "Name", cp.createName, true)

	// Purpose field.
	cp.drawTextField(screen, x+padding, y+80, w-padding*2, "Purpose", cp.createPurpose, false)

	// Min Resonance.
	cp.drawNumberField(screen, x+padding, y+140, w/2-padding*2, "Min Resonance", cp.createMinResonance)

	// Max Members.
	cp.drawNumberField(screen, x+w/2+padding, y+140, w/2-padding*2, "Max Members", float64(cp.createMaxMembers))

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+200, 120, 36, "Create", cp.createName != "")
}

// drawCouncilDetail draws the council detail view.
func (cp *CouncilPanel) drawCouncilDetail(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	council := cp.currentCouncil

	// State badge.
	cp.drawStateBadge(screen, x, y, w, padding, council.State)

	// Purpose.
	vector.DrawFilledRect(screen, x+padding, y+40, w-padding*2, 60, cp.theme.InputBackground, true)

	// Stats.
	statsY := y + 120
	cp.drawCouncilStats(screen, x, statsY, padding, council)

	// Action buttons.
	btnY := y + 200
	cp.drawActionButtons(screen, x, btnY, padding, council)
}

// drawStateBadge renders the council state badge in the top-right corner.
func (cp *CouncilPanel) drawStateBadge(screen *ebiten.Image, x, y, w, padding float32, state CouncilState) {
	stateColor := cp.theme.Success
	if state == CouncilStateDormant {
		stateColor = cp.theme.Warning
	} else if state == CouncilStateDisbanded {
		stateColor = cp.theme.TextError
	}
	vector.DrawFilledRect(screen, x+w-padding-80, y+10, 70, 20, stateColor, true)
}

// drawCouncilStats renders the member, proposal, and application count circles.
func (cp *CouncilPanel) drawCouncilStats(screen *ebiten.Image, x, statsY, padding float32, council *CouncilInfo) {
	memberCount := countActiveMembers(council.Members)
	_ = memberCount // For future display
	vector.DrawFilledCircle(screen, x+padding+30, statsY+20, 20, cp.theme.AccentPrimary, true)

	activeProposals := countActiveProposals(council.Proposals)
	_ = activeProposals // For future display
	vector.DrawFilledCircle(screen, x+padding+100, statsY+20, 20, cp.theme.AccentSecondary, true)

	pendingApps := countPendingApplications(council.Applications)
	if pendingApps > 0 {
		vector.DrawFilledCircle(screen, x+padding+170, statsY+20, 20, cp.theme.Warning, true)
	}
}

// drawActionButtons renders the council action buttons.
func (cp *CouncilPanel) drawActionButtons(screen *ebiten.Image, x, btnY, padding float32, council *CouncilInfo) {
	if !council.IsMember {
		return
	}

	btnW := float32(100)
	btnH := float32(32)
	btnSpacing := float32(10)

	cp.drawButton(screen, x+padding, btnY, btnW, btnH, "[M]embers", true)
	cp.drawButton(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, "[P]roposals", true)
	cp.drawButton(screen, x+padding, btnY+btnH+btnSpacing, btnW, btnH, "[I]nvite", true)
	cp.drawButton(screen, x+padding+btnW+btnSpacing, btnY+btnH+btnSpacing, btnW, btnH, "[N]ew Prop", true)

	if !council.IsCreator {
		cp.drawButton(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, "[L]eave", true)
	}
}

// countActiveMembers returns the number of active members.
func countActiveMembers(members []CouncilMemberInfo) int {
	count := 0
	for _, m := range members {
		if m.Status == MemberStatusActive {
			count++
		}
	}
	return count
}

// countActiveProposals returns the number of unresolved proposals.
func countActiveProposals(proposals []CouncilProposalInfo) int {
	count := 0
	for _, p := range proposals {
		if !p.Resolved {
			count++
		}
	}
	return count
}

// countPendingApplications returns the number of unresolved applications.
func countPendingApplications(applications []CouncilApplicationInfo) int {
	count := 0
	for _, a := range applications {
		if !a.Resolved {
			count++
		}
	}
	return count
}

// drawMembersList draws the members list.
func (cp *CouncilPanel) drawMembersList(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	itemHeight := float32(40)
	activeMembers := make([]CouncilMemberInfo, 0, len(cp.currentCouncil.Members))
	for _, member := range cp.currentCouncil.Members {
		if member.Status == MemberStatusActive {
			activeMembers = append(activeMembers, member)
		}
	}

	if len(activeMembers) == 0 {
		return
	}
	start := cp.scrollOffset
	if start < 0 {
		start = 0
	}
	maxVisible := cp.visibleRows(40)
	end := start + maxVisible
	if end > len(activeMembers) {
		end = len(activeMembers)
	}

	for i := start; i < end; i++ {
		member := activeMembers[i]
		itemY := y + padding + float32(i-start)*itemHeight

		// Status indicator.
		statusColor := cp.theme.Success
		vector.DrawFilledCircle(screen, x+padding+10, itemY+itemHeight/2, 6, statusColor, true)

		// Member name (rune-safe truncation to avoid splitting multi-byte characters).
		name := member.Name
		if name == "" {
			name = "(anonymous)"
		}
		name = truncateRunes(name, 20)
		drawUIText(screen, name, float64(x+padding)+25, float64(itemY)+float64(itemHeight)/2-4, cp.theme.TextPrimary)
	}
}

// drawProposalsList draws the proposals list.
func (cp *CouncilPanel) drawProposalsList(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	itemHeight := float32(60)

	for i, prop := range cp.currentCouncil.Proposals {
		itemY := y + padding + float32(i-cp.scrollOffset)*itemHeight

		if cp.isProposalOutOfView(itemY, y, h, itemHeight) {
			continue
		}

		cp.drawProposalItem(screen, i, &prop, x, itemY, w, padding, itemHeight)
	}
}

// isProposalOutOfView checks if a proposal is scrolled out of view.
func (cp *CouncilPanel) isProposalOutOfView(itemY, y, h, itemHeight float32) bool {
	return itemY < y || itemY > y+h-itemHeight
}

// drawProposalItem renders a single proposal item.
func (cp *CouncilPanel) drawProposalItem(screen *ebiten.Image, index int, prop *CouncilProposalInfo, x, itemY, w, padding, itemHeight float32) {
	if index == cp.selectedProposal {
		vector.DrawFilledRect(screen, x+padding, itemY, w-padding*2, itemHeight-5, cp.theme.Selection, true)
	}

	cp.drawProposalStatusIndicator(screen, prop, x, itemY, padding, itemHeight)
	cp.drawProposalText(screen, prop, x, itemY, w, padding, itemHeight)
	cp.drawProposalVoteCounts(screen, prop, x, itemY, w, padding, itemHeight)
}

// drawProposalStatusIndicator draws the proposal status circle.
func (cp *CouncilPanel) drawProposalStatusIndicator(screen *ebiten.Image, prop *CouncilProposalInfo, x, itemY, padding, itemHeight float32) {
	statusColor := cp.theme.AccentPrimary
	if prop.Resolved {
		if prop.Passed {
			statusColor = cp.theme.Success
		} else {
			statusColor = cp.theme.TextError
		}
	}
	vector.DrawFilledCircle(screen, x+padding+10, itemY+itemHeight/2, 8, statusColor, true)
}

// drawProposalText draws the proposal text area with the truncated proposal content.
func (cp *CouncilPanel) drawProposalText(screen *ebiten.Image, prop *CouncilProposalInfo, x, itemY, w, padding, itemHeight float32) {
	vector.DrawFilledRect(screen, x+padding+30, itemY+10, w-padding*2-40, itemHeight-30, cp.theme.InputBackground, true)

	// Render truncated proposal text over the background.
	// Use rune-safe truncation so multi-byte Unicode characters are never split.
	content := truncateRunes(prop.Text, 38)
	drawUIText(screen, content, float64(x+padding)+36, float64(itemY)+14, cp.theme.TextPrimary)
}

// drawProposalVoteCounts draws vote count bars.
func (cp *CouncilPanel) drawProposalVoteCounts(screen *ebiten.Image, prop *CouncilProposalInfo, x, itemY, w, padding, itemHeight float32) {
	forCount, againstCount := cp.countVotes(prop)
	voteY := itemY + itemHeight - 15
	vector.DrawFilledRect(screen, x+w-padding-80, voteY, float32(forCount*10), 10, cp.theme.Success, true)
	vector.DrawFilledRect(screen, x+w-padding-40, voteY, float32(againstCount*10), 10, cp.theme.TextError, true)
}

// countVotes tallies for and against votes for a proposal.
func (cp *CouncilPanel) countVotes(prop *CouncilProposalInfo) (int, int) {
	forCount := 0
	againstCount := 0
	for _, v := range prop.Votes {
		if v == VoteValueFor {
			forCount++
		} else if v == VoteValueAgainst {
			againstCount++
		}
	}
	return forCount, againstCount
}

// drawInviteForm draws the invite member form.
func (cp *CouncilPanel) drawInviteForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Specter key field.
	cp.drawTextField(screen, x+padding, y+40, w-padding*2, "Specter Key", cp.inviteSpecterKey, true)

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+120, 120, 36, "Invite", cp.inviteSpecterKey != "")
}

// drawProposeForm draws the proposal creation form.
func (cp *CouncilPanel) drawProposeForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Proposal text field (multiline).
	vector.DrawFilledRect(screen, x+padding, y+20, w-padding*2, 150, cp.theme.InputBackground, true)
	vector.StrokeRect(screen, x+padding, y+20, w-padding*2, 150, 1, cp.theme.PanelBorder, true)

	// Character count.
	countText := fmt.Sprintf("%d/256", len(cp.proposeText))
	drawUIText(screen, countText, float64(x+padding), float64(y)+175, cp.theme.TextSecondary)

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+190, 120, 36, "Ctrl+Enter", cp.proposeText != "")
}

// drawVoteForm draws the voting form.
func (cp *CouncilPanel) drawVoteForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Vote target description.
	vector.DrawFilledRect(screen, x+padding, y+20, w-padding*2, 60, cp.theme.InputBackground, true)

	// Vote buttons.
	btnW := float32(80)
	btnH := float32(40)
	btnY := y + 100
	btnSpacing := (w - padding*2 - btnW*3) / 2

	// For button.
	forColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueFor {
		forColor = cp.theme.Success
	}
	vector.DrawFilledRect(screen, x+padding, btnY, btnW, btnH, forColor, true)
	vector.StrokeRect(screen, x+padding, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Against button.
	againstColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueAgainst {
		againstColor = cp.theme.TextError
	}
	vector.DrawFilledRect(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, againstColor, true)
	vector.StrokeRect(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Abstain button.
	abstainColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueAbstain {
		abstainColor = cp.theme.TextSecondary
	}
	vector.DrawFilledRect(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, abstainColor, true)
	vector.StrokeRect(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Submit.
	cp.drawButton(screen, x+w/2-60, y+170, 120, 36, "Submit", true)
}

// drawTextField draws a text input field with label and value.
func (cp *CouncilPanel) drawTextField(screen *ebiten.Image, x, y, w float32, label, value string, focused bool) {
	// Label above the field.
	drawUIText(screen, label, float64(x), float64(y)+2, cp.theme.TextSecondary)

	// Background.
	vector.DrawFilledRect(screen, x, y+15, w, float32(cp.theme.InputHeight), cp.theme.InputBackground, true)

	// Border.
	borderColor := cp.theme.PanelBorder
	if focused {
		borderColor = cp.theme.AccentPrimary
	}
	vector.StrokeRect(screen, x, y+15, w, float32(cp.theme.InputHeight), 1, borderColor, true)

	// Value text.
	if value != "" {
		drawUIText(screen, value, float64(x)+8, float64(y)+15+4, cp.theme.TextPrimary)
	}

	// Cursor if focused.
	// Measure the rendered prefix width so cursor placement is accurate for
	// any Unicode input rather than relying on a fixed bytes-per-character estimate.
	if focused {
		prefixW, _ := measureUIText(value)
		cursorX := x + 8 + float32(prefixW)
		cursorAlpha := uint8(128 + 127*sin32(cp.animPhase*4))
		cursorColor := cp.theme.TextPrimary
		cursorColor.A = cursorAlpha
		vector.DrawFilledRect(screen, cursorX, y+20, 2, float32(cp.theme.InputHeight-10), cursorColor, true)
	}
}

// drawNumberField draws a number input field with label and value.
func (cp *CouncilPanel) drawNumberField(screen *ebiten.Image, x, y, w float32, label string, value float64) {
	// Label above the field.
	drawUIText(screen, label, float64(x), float64(y)+2, cp.theme.TextSecondary)

	// Background.
	vector.DrawFilledRect(screen, x, y+15, w, float32(cp.theme.InputHeight), cp.theme.InputBackground, true)
	vector.StrokeRect(screen, x, y+15, w, float32(cp.theme.InputHeight), 1, cp.theme.PanelBorder, true)

	// Value indicator bar.
	indicatorW := float32(value / 500 * float64(w-20))
	if indicatorW > w-20 {
		indicatorW = w - 20
	}
	vector.DrawFilledRect(screen, x+10, y+20+float32(cp.theme.InputHeight-20)/2, indicatorW, 4, cp.theme.AccentPrimary, true)

	// Numeric value text.
	drawUIText(screen, fmt.Sprintf("%.0f", value), float64(x)+8, float64(y)+15+4, cp.theme.TextPrimary)
}

// drawButton draws a button with a text label.
func (cp *CouncilPanel) drawButton(screen *ebiten.Image, x, y, w, h float32, label string, enabled bool) {
	bgColor := cp.theme.ButtonBackground
	if !enabled {
		bgColor.A = 100
	}

	vector.DrawFilledRect(screen, x, y, w, h, bgColor, true)
	vector.StrokeRect(screen, x, y, w, h, 1, cp.theme.PanelBorder, true)

	// Render centered label; dim when disabled.
	labelColor := cp.theme.TextPrimary
	if !enabled {
		labelColor = cp.theme.TextSecondary
	}
	drawUICenteredText(screen, label, float64(x)+float64(w)/2, float64(y)+float64(h)/2-4, labelColor)
}

// drawMessages draws error/success messages.
func (cp *CouncilPanel) drawMessages(screen *ebiten.Image, x, y, w float32) {
	if cp.errorMessage != "" {
		vector.DrawFilledRect(screen, x, y, w, 25, cp.theme.TextError, true)
	} else if cp.successMessage != "" {
		vector.DrawFilledRect(screen, x, y, w, 25, cp.theme.Success, true)
	}
}

// sin32 returns the sine of x as float32.
func sin32(x float64) float32 {
	// Use standard library approximation.
	x = x - float64(int(x/(2*3.14159)))*(2*3.14159)
	if x < 0 {
		x += 2 * 3.14159
	}
	return float32(sinApprox(x))
}

// sinApprox is a basic sine approximation.
func sinApprox(x float64) float64 {
	// Taylor series approximation.
	x = x - float64(int(x/6.28318))*6.28318
	if x > 3.14159 {
		x -= 6.28318
	}
	x2 := x * x
	return x * (1 - x2/6*(1-x2/20*(1-x2/42)))
}
