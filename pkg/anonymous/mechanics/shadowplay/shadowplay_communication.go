// Package mechanics implements anonymous layer game mechanics for MURMUR.
// Shadow Play Communication: In-game discussion between voting rounds
// via encrypted group channel.
// Per ROADMAP.md line 493: "Communication phase — in-game discussion between
// rounds via encrypted group channel".
// Per ANONYMOUS_GAME_MECHANICS.md: "Between voting rounds, participants can
// discuss (via Shadow Play's in-game discussion channel)."
package shadowplay

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// Communication phase constants.
const (
	// DiscussionDuration is the discussion phase duration between rounds.
	DiscussionDuration = 2 * time.Minute

	// MaxMessageLength is the maximum discussion message length.
	MaxMessageLength = 500

	// MessageRateLimit is the minimum interval between messages from same player.
	MessageRateLimit = 3 * time.Second

	// MaxMessagesPerPhase limits messages per player per discussion phase.
	MaxMessagesPerPhase = 10
)

// DiscussionPhaseState represents the state of discussion phase.
type DiscussionPhaseState uint8

const (
	// DiscussionInactive means no discussion is happening.
	DiscussionInactive DiscussionPhaseState = iota

	// DiscussionActive means discussion is open for messages.
	DiscussionActive

	// DiscussionEnding means discussion is about to end (30 second warning).
	DiscussionEnding

	// DiscussionEnded means discussion phase has concluded.
	DiscussionEnded
)

// DiscussionPhaseStateString returns a human-readable string.
func DiscussionPhaseStateString(s DiscussionPhaseState) string {
	switch s {
	case DiscussionInactive:
		return "Inactive"
	case DiscussionActive:
		return "Active"
	case DiscussionEnding:
		return "Ending"
	case DiscussionEnded:
		return "Ended"
	default:
		return "Unknown"
	}
}

// Discussion errors.
var (
	ErrDiscussionNotActive    = errors.New("discussion phase not active")
	ErrDiscussionNotPlayer    = errors.New("not a player in this game")
	ErrDiscussionEliminated   = errors.New("eliminated players cannot send messages")
	ErrDiscussionMessageEmpty = errors.New("message cannot be empty")
	ErrDiscussionMessageLong  = errors.New("message exceeds maximum length")
	ErrDiscussionRateLimit    = errors.New("sending messages too quickly")
	ErrDiscussionLimitReached = errors.New("message limit reached for this phase")
)

// DiscussionMessage represents a message in the discussion phase.
type DiscussionMessage struct {
	// MessageID uniquely identifies this message.
	MessageID [32]byte

	// GameID links to the Shadow Play game.
	GameID [32]byte

	// SenderKey is the sender's Specter key.
	SenderKey [32]byte

	// Round is the round this message was sent during.
	Round int

	// Content is the message text (max 500 bytes).
	Content string

	// SentAt is when the message was sent.
	SentAt time.Time

	// SequenceNum is the message sequence within the phase.
	SequenceNum int
}

// DiscussionPhase manages the communication phase between voting rounds.
type DiscussionPhase struct {
	mu sync.RWMutex

	// GameID links to the parent Shadow Play game.
	GameID [32]byte

	// Round is the round number for this discussion.
	Round int

	// State is the current phase state.
	State DiscussionPhaseState

	// StartTime is when the discussion phase started.
	StartTime time.Time

	// EndTime is when the discussion phase ends.
	EndTime time.Time

	// Messages is the list of messages sent during this phase.
	Messages []*DiscussionMessage

	// PlayerMessages tracks message count per player.
	playerMessages map[string]int

	// LastMessage tracks last message time per player for rate limiting.
	lastMessage map[string]time.Time

	// Participants are the players who can send messages.
	participants map[string]bool

	// sequenceCounter tracks message ordering.
	sequenceCounter int

	// onMessage callback for new messages.
	onMessage []func(*DiscussionMessage)

	// onPhaseChange callback for state changes.
	onPhaseChange []func(DiscussionPhaseState, DiscussionPhaseState)
}

// NewDiscussionPhase creates a new discussion phase for a round.
func NewDiscussionPhase(gameID [32]byte, round int, players [][32]byte) *DiscussionPhase {
	participants := make(map[string]bool)
	for _, p := range players {
		participants[mechanics.KeyToHex(p[:])] = true
	}

	now := time.Now()

	return &DiscussionPhase{
		GameID:         gameID,
		Round:          round,
		State:          DiscussionInactive,
		StartTime:      now,
		EndTime:        now.Add(DiscussionDuration),
		Messages:       make([]*DiscussionMessage, 0),
		playerMessages: make(map[string]int),
		lastMessage:    make(map[string]time.Time),
		participants:   participants,
	}
}

// Start begins the discussion phase.
func (dp *DiscussionPhase) Start() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	oldState := dp.State
	dp.State = DiscussionActive
	dp.StartTime = time.Now()
	dp.EndTime = dp.StartTime.Add(DiscussionDuration)

	// Reset message counts.
	dp.playerMessages = make(map[string]int)
	dp.lastMessage = make(map[string]time.Time)
	dp.sequenceCounter = 0

	dp.firePhaseChange(oldState, DiscussionActive)
}

// OnMessage registers a callback for new messages.
func (dp *DiscussionPhase) OnMessage(cb func(*DiscussionMessage)) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.onMessage = append(dp.onMessage, cb)
}

// OnPhaseChange registers a callback for state changes.
func (dp *DiscussionPhase) OnPhaseChange(cb func(DiscussionPhaseState, DiscussionPhaseState)) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.onPhaseChange = append(dp.onPhaseChange, cb)
}

// firePhaseChange notifies state change listeners.
func (dp *DiscussionPhase) firePhaseChange(old, new DiscussionPhaseState) {
	callbacks := dp.onPhaseChange
	// Fire outside lock.
	dp.mu.Unlock()
	for _, cb := range callbacks {
		cb(old, new)
	}
	dp.mu.Lock()
}

// Update checks time and updates phase state.
func (dp *DiscussionPhase) Update() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.State != DiscussionActive && dp.State != DiscussionEnding {
		return
	}

	now := time.Now()

	// Check for ending warning (30 seconds left).
	if dp.State == DiscussionActive && now.Add(30*time.Second).After(dp.EndTime) {
		dp.State = DiscussionEnding
		dp.firePhaseChange(DiscussionActive, DiscussionEnding)
	}

	// Check for phase end.
	if now.After(dp.EndTime) {
		oldState := dp.State
		dp.State = DiscussionEnded
		dp.firePhaseChange(oldState, DiscussionEnded)
	}
}

// SendMessage adds a message to the discussion.
func (dp *DiscussionPhase) SendMessage(sender [32]byte, content string) (*DiscussionMessage, error) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if err := dp.validateSendMessage(sender, content); err != nil {
		return nil, err
	}

	msg := dp.createMessage(sender, content)
	dp.recordMessage(sender, msg)

	// Fire callbacks.
	callbacks := dp.onMessage
	dp.mu.Unlock()
	for _, cb := range callbacks {
		cb(msg)
	}
	dp.mu.Lock()

	return msg, nil
}

// validateSendMessage checks if a message can be sent.
func (dp *DiscussionPhase) validateSendMessage(sender [32]byte, content string) error {
	// Check phase is active.
	if dp.State != DiscussionActive && dp.State != DiscussionEnding {
		return ErrDiscussionNotActive
	}

	// Check sender is participant.
	senderHex := mechanics.KeyToHex(sender[:])
	if !dp.participants[senderHex] {
		return ErrDiscussionNotPlayer
	}

	// Check message content.
	if len(content) == 0 {
		return ErrDiscussionMessageEmpty
	}
	if len(content) > MaxMessageLength {
		return ErrDiscussionMessageLong
	}

	// Check rate limit.
	if last, ok := dp.lastMessage[senderHex]; ok {
		if time.Since(last) < MessageRateLimit {
			return ErrDiscussionRateLimit
		}
	}

	// Check message limit.
	if dp.playerMessages[senderHex] >= MaxMessagesPerPhase {
		return ErrDiscussionLimitReached
	}

	return nil
}

// createMessage constructs a new message.
func (dp *DiscussionPhase) createMessage(sender [32]byte, content string) *DiscussionMessage {
	// Generate message ID.
	var idSeed []byte
	idSeed = append(idSeed, dp.GameID[:]...)
	idSeed = append(idSeed, sender[:]...)
	idSeed = append(idSeed, []byte(content)...)

	var nonce [8]byte
	rand.Read(nonce[:])
	idSeed = append(idSeed, nonce[:]...)

	return &DiscussionMessage{
		MessageID:   sha256.Sum256(idSeed),
		GameID:      dp.GameID,
		SenderKey:   sender,
		Round:       dp.Round,
		Content:     content,
		SentAt:      time.Now(),
		SequenceNum: dp.sequenceCounter,
	}
}

// recordMessage tracks the message.
func (dp *DiscussionPhase) recordMessage(sender [32]byte, msg *DiscussionMessage) {
	senderHex := mechanics.KeyToHex(sender[:])
	dp.Messages = append(dp.Messages, msg)
	dp.playerMessages[senderHex]++
	dp.lastMessage[senderHex] = msg.SentAt
	dp.sequenceCounter++
}

// GetMessages returns all messages in the discussion phase.
func (dp *DiscussionPhase) GetMessages() []*DiscussionMessage {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	result := make([]*DiscussionMessage, len(dp.Messages))
	copy(result, dp.Messages)
	return result
}

// GetMessagesSince returns messages after a given sequence number.
func (dp *DiscussionPhase) GetMessagesSince(seqNum int) []*DiscussionMessage {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	var result []*DiscussionMessage
	for _, msg := range dp.Messages {
		if msg.SequenceNum > seqNum {
			result = append(result, msg)
		}
	}
	return result
}

// MessageCount returns the total number of messages.
func (dp *DiscussionPhase) MessageCount() int {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return len(dp.Messages)
}

// PlayerMessageCount returns message count for a specific player.
func (dp *DiscussionPhase) PlayerMessageCount(player [32]byte) int {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.playerMessages[mechanics.KeyToHex(player[:])]
}

// TimeRemaining returns the time until discussion ends.
func (dp *DiscussionPhase) TimeRemaining() time.Duration {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	remaining := time.Until(dp.EndTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsActive returns true if discussion phase is active.
func (dp *DiscussionPhase) IsActive() bool {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.State == DiscussionActive || dp.State == DiscussionEnding
}

// GetState returns the current phase state.
func (dp *DiscussionPhase) GetState() DiscussionPhaseState {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.State
}

// AddParticipant adds a player who can send messages.
func (dp *DiscussionPhase) AddParticipant(player [32]byte) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.participants[mechanics.KeyToHex(player[:])] = true
}

// RemoveParticipant removes a player (e.g., when eliminated).
func (dp *DiscussionPhase) RemoveParticipant(player [32]byte) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	delete(dp.participants, mechanics.KeyToHex(player[:]))
}

// ParticipantCount returns the number of active participants.
func (dp *DiscussionPhase) ParticipantCount() int {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return len(dp.participants)
}

// End immediately ends the discussion phase.
func (dp *DiscussionPhase) End() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.State == DiscussionEnded || dp.State == DiscussionInactive {
		return
	}

	oldState := dp.State
	dp.State = DiscussionEnded
	dp.EndTime = time.Now()

	dp.firePhaseChange(oldState, DiscussionEnded)
}

// Reset prepares for a new round's discussion.
func (dp *DiscussionPhase) Reset(round int) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.Round = round
	dp.State = DiscussionInactive
	dp.Messages = make([]*DiscussionMessage, 0)
	dp.playerMessages = make(map[string]int)
	dp.lastMessage = make(map[string]time.Time)
	dp.sequenceCounter = 0
}

// ShadowPlayCommunication integrates discussion with Shadow Play game.
type ShadowPlayCommunication struct {
	mu sync.RWMutex

	// game is the parent Shadow Play game.
	game *ShadowPlay

	// currentDiscussion is the active discussion phase.
	currentDiscussion *DiscussionPhase

	// discussionHistory stores past discussions by round.
	discussionHistory map[int]*DiscussionPhase
}

// NewShadowPlayCommunication creates communication management for a game.
func NewShadowPlayCommunication(game *ShadowPlay) *ShadowPlayCommunication {
	return &ShadowPlayCommunication{
		game:              game,
		discussionHistory: make(map[int]*DiscussionPhase),
	}
}

// StartDiscussion begins a new discussion phase for the current round.
func (spc *ShadowPlayCommunication) StartDiscussion() *DiscussionPhase {
	spc.mu.Lock()
	defer spc.mu.Unlock()

	if spc.game == nil {
		return nil
	}

	// Get active players.
	spc.game.mu.RLock()
	round := spc.game.CurrentRound
	var activePlayers [][32]byte
	for _, p := range spc.game.Players {
		if !p.IsEliminated {
			activePlayers = append(activePlayers, p.SpecterKey)
		}
	}
	spc.game.mu.RUnlock()

	// Create and start discussion.
	dp := NewDiscussionPhase(spc.game.ID, round, activePlayers)
	dp.Start()

	spc.currentDiscussion = dp
	spc.discussionHistory[round] = dp

	return dp
}

// CurrentDiscussion returns the active discussion phase.
func (spc *ShadowPlayCommunication) CurrentDiscussion() *DiscussionPhase {
	spc.mu.RLock()
	defer spc.mu.RUnlock()
	return spc.currentDiscussion
}

// GetDiscussionForRound returns the discussion for a specific round.
func (spc *ShadowPlayCommunication) GetDiscussionForRound(round int) *DiscussionPhase {
	spc.mu.RLock()
	defer spc.mu.RUnlock()
	return spc.discussionHistory[round]
}

// EndDiscussion ends the current discussion phase.
func (spc *ShadowPlayCommunication) EndDiscussion() {
	spc.mu.Lock()
	defer spc.mu.Unlock()

	if spc.currentDiscussion != nil {
		spc.currentDiscussion.End()
	}
}

// OnElimination removes an eliminated player from active discussion.
func (spc *ShadowPlayCommunication) OnElimination(player [32]byte) {
	spc.mu.Lock()
	defer spc.mu.Unlock()

	if spc.currentDiscussion != nil {
		spc.currentDiscussion.RemoveParticipant(player)
	}
}

// Update checks and updates discussion phase state.
func (spc *ShadowPlayCommunication) Update() {
	spc.mu.RLock()
	dp := spc.currentDiscussion
	spc.mu.RUnlock()

	if dp != nil {
		dp.Update()
	}
}

// TotalMessageCount returns total messages across all rounds.
func (spc *ShadowPlayCommunication) TotalMessageCount() int {
	spc.mu.RLock()
	defer spc.mu.RUnlock()

	total := 0
	for _, dp := range spc.discussionHistory {
		total += dp.MessageCount()
	}
	return total
}

// GetAllMessages returns all messages from all rounds.
func (spc *ShadowPlayCommunication) GetAllMessages() []*DiscussionMessage {
	spc.mu.RLock()
	defer spc.mu.RUnlock()

	var all []*DiscussionMessage
	// Iterate over all rounds in history (rounds are 0-indexed when game hasn't started).
	for _, dp := range spc.discussionHistory {
		all = append(all, dp.GetMessages()...)
	}
	return all
}
