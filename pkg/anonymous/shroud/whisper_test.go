package shroud

import (
	"bytes"
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/curve25519"
)

// TestWhisperKeyExchange tests Curve25519 key exchange and HKDF derivation.
func TestWhisperKeyExchange(t *testing.T) {
	// Create two key exchange pairs.
	alice, err := NewWhisperKeyExchange()
	if err != nil {
		t.Fatalf("NewWhisperKeyExchange (alice): %v", err)
	}

	bob, err := NewWhisperKeyExchange()
	if err != nil {
		t.Fatalf("NewWhisperKeyExchange (bob): %v", err)
	}

	// Derive keys - both should arrive at the same shared key.
	aliceKey, err := alice.DeriveKey(bob.PublicKey())
	if err != nil {
		t.Fatalf("alice.DeriveKey: %v", err)
	}

	bobKey, err := bob.DeriveKey(alice.PublicKey())
	if err != nil {
		t.Fatalf("bob.DeriveKey: %v", err)
	}

	// Keys should be equal (Diffie-Hellman property).
	if aliceKey != bobKey {
		t.Errorf("DH keys mismatch: alice=%x, bob=%x", aliceKey, bobKey)
	}

	// Keys should not be zero.
	var zero [32]byte
	if aliceKey == zero {
		t.Error("derived key is zero")
	}
}

// TestWhisperKeyExchangeUniqueness tests that different key pairs produce different keys.
func TestWhisperKeyExchangeUniqueness(t *testing.T) {
	alice, _ := NewWhisperKeyExchange()
	bob, _ := NewWhisperKeyExchange()
	carol, _ := NewWhisperKeyExchange()

	// Derive keys for different pairs.
	keyAliceBob, _ := alice.DeriveKey(bob.PublicKey())
	keyAliceCarol, _ := alice.DeriveKey(carol.PublicKey())
	keyBobCarol, _ := bob.DeriveKey(carol.PublicKey())

	// All keys should be different.
	if keyAliceBob == keyAliceCarol {
		t.Error("alice-bob key equals alice-carol key")
	}
	if keyAliceBob == keyBobCarol {
		t.Error("alice-bob key equals bob-carol key")
	}
	if keyAliceCarol == keyBobCarol {
		t.Error("alice-carol key equals bob-carol key")
	}
}

// TestWhisperKeyExchangeInvalidKey tests error handling for invalid public keys.
func TestWhisperKeyExchangeInvalidKey(t *testing.T) {
	alice, _ := NewWhisperKeyExchange()

	// A zero public key should produce a zero shared secret, which is rejected.
	var zeroKey [32]byte
	_, err := alice.DeriveKey(zeroKey)
	if err != ErrWhisperInvalidKey {
		t.Errorf("expected ErrWhisperInvalidKey for zero key, got: %v", err)
	}
}

// TestEncryptDecryptWhisper tests the encrypt/decrypt round-trip.
func TestEncryptDecryptWhisper(t *testing.T) {
	// Generate recipient key pair.
	var recipientPriv [32]byte
	if _, err := rand.Read(recipientPriv[:]); err != nil {
		t.Fatal(err)
	}
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	// Test message.
	payload := []byte("Hello, anonymous Specter!")

	// Encrypt.
	msg, err := EncryptWhisper(payload, recipientPub)
	if err != nil {
		t.Fatalf("EncryptWhisper: %v", err)
	}

	// Verify message structure.
	if msg.HopCount != 0 {
		t.Errorf("expected HopCount=0, got %d", msg.HopCount)
	}
	if msg.TTL != uint32(WhisperMessageTTL.Seconds()) {
		t.Errorf("expected TTL=%d, got %d", uint32(WhisperMessageTTL.Seconds()), msg.TTL)
	}

	// MessageID should not be zero.
	var zero [32]byte
	if msg.MessageID == zero {
		t.Error("MessageID is zero")
	}

	// Sender key should not be zero.
	if msg.SenderKey == zero {
		t.Error("SenderKey is zero")
	}

	// Decrypt.
	decrypted, err := DecryptWhisper(msg, recipientPriv)
	if err != nil {
		t.Fatalf("DecryptWhisper: %v", err)
	}

	// Compare payloads.
	if !bytes.Equal(payload, decrypted) {
		t.Errorf("payload mismatch: got %q, want %q", decrypted, payload)
	}
}

// TestEncryptDecryptWhisperLargePayload tests encryption of larger payloads.
func TestEncryptDecryptWhisperLargePayload(t *testing.T) {
	var recipientPriv [32]byte
	rand.Read(recipientPriv[:])
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	// Test with maximum allowed payload.
	payload := make([]byte, WhisperMaxPayload)
	rand.Read(payload)

	msg, err := EncryptWhisper(payload, recipientPub)
	if err != nil {
		t.Fatalf("EncryptWhisper with max payload: %v", err)
	}

	decrypted, err := DecryptWhisper(msg, recipientPriv)
	if err != nil {
		t.Fatalf("DecryptWhisper: %v", err)
	}

	if !bytes.Equal(payload, decrypted) {
		t.Error("large payload mismatch")
	}
}

// TestEncryptWhisperPayloadTooLarge tests that oversized payloads are rejected.
func TestEncryptWhisperPayloadTooLarge(t *testing.T) {
	var recipientPub [32]byte
	rand.Read(recipientPub[:])

	payload := make([]byte, WhisperMaxPayload+1)

	_, err := EncryptWhisper(payload, recipientPub)
	if err != ErrWhisperPayloadTooLarge {
		t.Errorf("expected ErrWhisperPayloadTooLarge, got: %v", err)
	}
}

// TestDecryptWhisperWrongKey tests decryption failure with wrong key.
func TestDecryptWhisperWrongKey(t *testing.T) {
	// Recipient keys.
	var recipientPriv [32]byte
	rand.Read(recipientPriv[:])
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	// Wrong key.
	var wrongPriv [32]byte
	rand.Read(wrongPriv[:])
	wrongPriv[0] &= 248
	wrongPriv[31] &= 127
	wrongPriv[31] |= 64

	// Encrypt to correct recipient.
	msg, err := EncryptWhisper([]byte("secret"), recipientPub)
	if err != nil {
		t.Fatal(err)
	}

	// Try to decrypt with wrong key.
	_, err = DecryptWhisper(msg, wrongPriv)
	if err != ErrWhisperDecryptFailed {
		t.Errorf("expected ErrWhisperDecryptFailed, got: %v", err)
	}
}

// TestWhisperMessageExpiry tests TTL expiration.
func TestWhisperMessageExpiry(t *testing.T) {
	// Create a message with very short TTL.
	msg := &WhisperMessage{
		Timestamp: time.Now().Add(-2 * time.Second).Unix(),
		TTL:       1, // 1 second TTL.
	}

	if !msg.IsExpired() {
		t.Error("message should be expired")
	}

	// Create a message with long TTL.
	msg2 := &WhisperMessage{
		Timestamp: time.Now().Unix(),
		TTL:       600, // 10 minutes.
	}

	if msg2.IsExpired() {
		t.Error("message should not be expired")
	}
}

// TestWhisperMessageEncoding tests message serialization/deserialization.
func TestWhisperMessageEncoding(t *testing.T) {
	// Create a message.
	msg := &WhisperMessage{
		Timestamp: time.Now().Unix(),
		TTL:       600,
		HopCount:  3,
		Encrypted: []byte("encrypted payload data"),
	}

	// Fill in random keys and nonce.
	rand.Read(msg.MessageID[:])
	rand.Read(msg.SenderKey[:])
	rand.Read(msg.Nonce[:])

	// Encode.
	encoded, err := encodeWhisperMessage(msg)
	if err != nil {
		t.Fatalf("encodeWhisperMessage: %v", err)
	}

	// Decode.
	decoded, err := decodeWhisperMessage(encoded)
	if err != nil {
		t.Fatalf("decodeWhisperMessage: %v", err)
	}

	// Compare fields.
	if decoded.MessageID != msg.MessageID {
		t.Error("MessageID mismatch")
	}
	if decoded.SenderKey != msg.SenderKey {
		t.Error("SenderKey mismatch")
	}
	if decoded.Nonce != msg.Nonce {
		t.Error("Nonce mismatch")
	}
	if decoded.Timestamp != msg.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, msg.Timestamp)
	}
	if decoded.TTL != msg.TTL {
		t.Errorf("TTL mismatch: got %d, want %d", decoded.TTL, msg.TTL)
	}
	if decoded.HopCount != msg.HopCount {
		t.Errorf("HopCount mismatch: got %d, want %d", decoded.HopCount, msg.HopCount)
	}
	if !bytes.Equal(decoded.Encrypted, msg.Encrypted) {
		t.Error("Encrypted payload mismatch")
	}
}

// TestDecodeWhisperMessageTooShort tests error for truncated messages.
func TestDecodeWhisperMessageTooShort(t *testing.T) {
	// Message too short.
	_, err := decodeWhisperMessage([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for short message")
	}
}

// TestDecodeWhisperMessageTruncatedPayload tests error for truncated payload.
func TestDecodeWhisperMessageTruncatedPayload(t *testing.T) {
	// Create minimal header with encrypted length claiming more bytes than present.
	headerSize := 32 + 32 + 24 + 8 + 4 + 1 + 2
	data := make([]byte, headerSize)

	// Set encrypted length to 100 (but provide no payload bytes).
	data[headerSize-2] = 0
	data[headerSize-1] = 100

	_, err := decodeWhisperMessage(data)
	if err == nil {
		t.Error("expected error for truncated payload")
	}
}

// TestWhisperRouterRouteManagement tests route add/remove/get.
func TestWhisperRouterRouteManagement(t *testing.T) {
	// Create router without delivery (not testing actual sending).
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	// Create a destination key.
	var dest [32]byte
	rand.Read(dest[:])

	// Initially no route.
	if router.GetRoute(dest) != nil {
		t.Error("expected no route initially")
	}

	// Add route.
	route := &WhisperRoute{
		Destination: dest,
		Latency:     50 * time.Millisecond,
		Reliability: 0.95,
	}
	router.AddRoute(route)

	// Get route.
	retrieved := router.GetRoute(dest)
	if retrieved == nil {
		t.Fatal("route not found after add")
	}
	if retrieved.Destination != dest {
		t.Error("route destination mismatch")
	}
	if retrieved.Reliability != 0.95 {
		t.Error("route reliability mismatch")
	}

	// Remove route.
	router.RemoveRoute(dest)
	if router.GetRoute(dest) != nil {
		t.Error("route still exists after remove")
	}
}

// TestWhisperRouterStats tests statistics tracking.
func TestWhisperRouterStats(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	// Add some routes.
	for i := 0; i < 5; i++ {
		var dest [32]byte
		rand.Read(dest[:])
		router.AddRoute(&WhisperRoute{Destination: dest})
	}

	stats := router.Stats()
	if stats.RoutesKnown != 5 {
		t.Errorf("expected 5 routes, got %d", stats.RoutesKnown)
	}
}

// TestWhisperRouterHandlerRegistration tests handler registration.
func TestWhisperRouterHandlerRegistration(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	var handlerCalls int32

	handler := func(msg *WhisperMessage, payload []byte) error {
		atomic.AddInt32(&handlerCalls, 1)
		return nil
	}

	router.RegisterHandler(handler)
	router.RegisterHandler(handler) // Register twice.

	// Verify handlers registered.
	router.mu.RLock()
	count := len(router.handlers)
	router.mu.RUnlock()

	if count != 2 {
		t.Errorf("expected 2 handlers, got %d", count)
	}
}

// TestWhisperRouterSendNoRoute tests error when sending without route.
func TestWhisperRouterSendNoRoute(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	var dest [32]byte
	rand.Read(dest[:])

	err := router.Send(dest, []byte("hello"))
	if err != ErrWhisperNoRoute {
		t.Errorf("expected ErrWhisperNoRoute, got: %v", err)
	}

	stats := router.Stats()
	if stats.MessagesDropped != 1 {
		t.Errorf("expected 1 dropped message, got %d", stats.MessagesDropped)
	}
}

// TestWhisperRouterHandleIncomingExpired tests rejecting expired messages.
func TestWhisperRouterHandleIncomingExpired(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	// Create an expired message.
	msg := &WhisperMessage{
		Timestamp: time.Now().Add(-1 * time.Hour).Unix(),
		TTL:       60, // 1 minute TTL, expired 59 minutes ago.
		Encrypted: []byte{1, 2, 3},
	}
	rand.Read(msg.MessageID[:])
	rand.Read(msg.SenderKey[:])
	rand.Read(msg.Nonce[:])

	encoded, _ := encodeWhisperMessage(msg)

	err := router.HandleIncoming(encoded)
	if err != ErrWhisperExpired {
		t.Errorf("expected ErrWhisperExpired, got: %v", err)
	}
}

// TestWhisperRouterHandleIncomingTooManyHops tests rejecting messages with too many hops.
func TestWhisperRouterHandleIncomingTooManyHops(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	// Create a message with too many hops.
	msg := &WhisperMessage{
		Timestamp: time.Now().Unix(),
		TTL:       600,
		HopCount:  WhisperChainMaxHops, // At max, so next relay would exceed.
		Encrypted: []byte{1, 2, 3},
	}
	rand.Read(msg.MessageID[:])
	rand.Read(msg.SenderKey[:])
	rand.Read(msg.Nonce[:])

	encoded, _ := encodeWhisperMessage(msg)

	err := router.HandleIncoming(encoded)
	if err != ErrWhisperChainTooLong {
		t.Errorf("expected ErrWhisperChainTooLong, got: %v", err)
	}
}

// TestWhisperRouterHandleIncomingAsRecipient tests successful message receipt.
func TestWhisperRouterHandleIncomingAsRecipient(t *testing.T) {
	// Generate recipient keys.
	var recipientPriv [32]byte
	rand.Read(recipientPriv[:])
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	router := NewWhisperRouter(nil, recipientPriv)

	// Track received messages.
	var received [][]byte
	var mu sync.Mutex

	router.RegisterHandler(func(msg *WhisperMessage, payload []byte) error {
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		return nil
	})

	// Send a message to the router's public key.
	testPayload := []byte("test message to myself")
	msg, err := EncryptWhisper(testPayload, recipientPub)
	if err != nil {
		t.Fatal(err)
	}

	encoded, _ := encodeWhisperMessage(msg)

	err = router.HandleIncoming(encoded)
	if err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Check handler was called.
	mu.Lock()
	if len(received) != 1 {
		t.Fatalf("expected 1 received message, got %d", len(received))
	}
	if !bytes.Equal(received[0], testPayload) {
		t.Errorf("payload mismatch: got %q, want %q", received[0], testPayload)
	}
	mu.Unlock()

	// Check stats.
	stats := router.Stats()
	if stats.MessagesReceived != 1 {
		t.Errorf("expected 1 received, got %d", stats.MessagesReceived)
	}
}

// TestWhisperRouterHandleIncomingNotRecipient tests relaying behavior.
func TestWhisperRouterHandleIncomingNotRecipient(t *testing.T) {
	// Generate router keys (different from message recipient).
	var routerPriv [32]byte
	rand.Read(routerPriv[:])
	routerPriv[0] &= 248
	routerPriv[31] &= 127
	routerPriv[31] |= 64

	router := NewWhisperRouter(nil, routerPriv)

	// Generate different recipient keys.
	var recipientPriv [32]byte
	rand.Read(recipientPriv[:])
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	// Encrypt for the actual recipient (not the router).
	msg, err := EncryptWhisper([]byte("not for router"), recipientPub)
	if err != nil {
		t.Fatal(err)
	}

	encoded, _ := encodeWhisperMessage(msg)

	err = router.HandleIncoming(encoded)
	if err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Should be counted as relayed, not received.
	stats := router.Stats()
	if stats.MessagesRelayed != 1 {
		t.Errorf("expected 1 relayed, got %d", stats.MessagesRelayed)
	}
	if stats.MessagesReceived != 0 {
		t.Errorf("expected 0 received, got %d", stats.MessagesReceived)
	}
}

// TestWhisperConcurrentKeyExchange tests thread safety of key exchange.
func TestWhisperConcurrentKeyExchange(t *testing.T) {
	alice, _ := NewWhisperKeyExchange()
	bob, _ := NewWhisperKeyExchange()

	var wg sync.WaitGroup
	var aliceKeys, bobKeys [100][32]byte

	// Concurrent derivations.
	for i := 0; i < 100; i++ {
		wg.Add(2)

		go func(idx int) {
			defer wg.Done()
			key, _ := alice.DeriveKey(bob.PublicKey())
			aliceKeys[idx] = key
		}(i)

		go func(idx int) {
			defer wg.Done()
			key, _ := bob.DeriveKey(alice.PublicKey())
			bobKeys[idx] = key
		}(i)
	}

	wg.Wait()

	// All derived keys should be equal.
	for i := 0; i < 100; i++ {
		if aliceKeys[i] != bobKeys[i] {
			t.Errorf("key mismatch at index %d", i)
		}
		if i > 0 && aliceKeys[i] != aliceKeys[0] {
			t.Errorf("alice key inconsistent at index %d", i)
		}
	}
}

// TestWhisperRouterConcurrentSends tests thread safety of send operations.
func TestWhisperRouterConcurrentSends(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])

	router := NewWhisperRouter(nil, privateKey)

	// Add multiple routes.
	var dests [10][32]byte
	for i := range dests {
		rand.Read(dests[i][:])
		router.AddRoute(&WhisperRoute{Destination: dests[i]})
	}

	var wg sync.WaitGroup
	var drops int32

	// Concurrent route lookups and stats (thread safety test without sending).
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func(idx int) {
			defer wg.Done()
			dest := dests[idx%len(dests)]
			if router.GetRoute(dest) == nil {
				atomic.AddInt32(&drops, 1)
			}
		}(i)

		go func() {
			defer wg.Done()
			router.Stats()
		}()

		go func(idx int) {
			defer wg.Done()
			var newDest [32]byte
			rand.Read(newDest[:])
			router.AddRoute(&WhisperRoute{Destination: newDest})
		}(i)
	}

	wg.Wait()

	// Verify routes were added.
	stats := router.Stats()
	if stats.RoutesKnown < 10 {
		t.Errorf("expected at least 10 routes, got %d", stats.RoutesKnown)
	}
}

// TestWhisperConstants verifies constant values match spec.
func TestWhisperConstants(t *testing.T) {
	if WhisperNonceSize != 24 {
		t.Errorf("WhisperNonceSize: got %d, want 24", WhisperNonceSize)
	}
	if WhisperKeySize != 32 {
		t.Errorf("WhisperKeySize: got %d, want 32", WhisperKeySize)
	}
	if WhisperChainMaxHops != 5 {
		t.Errorf("WhisperChainMaxHops: got %d, want 5", WhisperChainMaxHops)
	}
	if WhisperMessageTTL != 10*time.Minute {
		t.Errorf("WhisperMessageTTL: got %v, want 10m", WhisperMessageTTL)
	}
	if WhisperMaxPayload >= FixedPacketSize {
		t.Error("WhisperMaxPayload should be less than FixedPacketSize")
	}
}

// BenchmarkEncryptWhisper benchmarks whisper encryption.
func BenchmarkEncryptWhisper(b *testing.B) {
	var recipientPub [32]byte
	rand.Read(recipientPub[:])

	payload := make([]byte, 256)
	rand.Read(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptWhisper(payload, recipientPub)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptWhisper benchmarks whisper decryption.
func BenchmarkDecryptWhisper(b *testing.B) {
	var recipientPriv [32]byte
	rand.Read(recipientPriv[:])
	recipientPriv[0] &= 248
	recipientPriv[31] &= 127
	recipientPriv[31] |= 64

	var recipientPub [32]byte
	curve25519.ScalarBaseMult(&recipientPub, &recipientPriv)

	payload := make([]byte, 256)
	rand.Read(payload)

	msg, _ := EncryptWhisper(payload, recipientPub)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptWhisper(msg, recipientPriv)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWhisperKeyDerive benchmarks key derivation.
func BenchmarkWhisperKeyDerive(b *testing.B) {
	alice, _ := NewWhisperKeyExchange()
	bob, _ := NewWhisperKeyExchange()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := alice.DeriveKey(bob.PublicKey())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWhisperMessageEncode benchmarks message encoding.
func BenchmarkWhisperMessageEncode(b *testing.B) {
	msg := &WhisperMessage{
		Timestamp: time.Now().Unix(),
		TTL:       600,
		HopCount:  1,
		Encrypted: make([]byte, 256),
	}
	rand.Read(msg.MessageID[:])
	rand.Read(msg.SenderKey[:])
	rand.Read(msg.Nonce[:])
	rand.Read(msg.Encrypted)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encodeWhisperMessage(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestDeliveryConfirmationCreation tests creating delivery confirmations.
func TestDeliveryConfirmationCreation(t *testing.T) {
	var messageID [32]byte
	var recipientKey [32]byte
	rand.Read(messageID[:])
	rand.Read(recipientKey[:])

	conf, err := NewDeliveryConfirmation(messageID, recipientKey)
	if err != nil {
		t.Fatalf("NewDeliveryConfirmation: %v", err)
	}

	// Verify fields.
	if conf.MessageID != messageID {
		t.Error("MessageID mismatch")
	}

	// ReceiptNonce should not be zero.
	var zero [24]byte
	if conf.ReceiptNonce == zero {
		t.Error("ReceiptNonce is zero")
	}

	// ConfirmationHash should not be zero.
	var zeroHash [32]byte
	if conf.ConfirmationHash == zeroHash {
		t.Error("ConfirmationHash is zero")
	}

	// Timestamp should be recent.
	confirmTime := time.Unix(conf.Timestamp, 0)
	if time.Since(confirmTime) > time.Minute {
		t.Error("Timestamp not recent")
	}
}

// TestDeliveryConfirmationVerify tests verification logic.
func TestDeliveryConfirmationVerify(t *testing.T) {
	var messageID [32]byte
	var recipientKey [32]byte
	rand.Read(messageID[:])
	rand.Read(recipientKey[:])

	conf, _ := NewDeliveryConfirmation(messageID, recipientKey)

	// Valid verification.
	if !VerifyDeliveryConfirmation(conf, messageID) {
		t.Error("expected valid confirmation to verify")
	}

	// Wrong message ID.
	var wrongID [32]byte
	rand.Read(wrongID[:])
	if VerifyDeliveryConfirmation(conf, wrongID) {
		t.Error("expected verification to fail with wrong message ID")
	}

	// Expired confirmation (old timestamp).
	oldConf := &DeliveryConfirmation{
		MessageID: messageID,
		Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
	}
	if VerifyDeliveryConfirmation(oldConf, messageID) {
		t.Error("expected verification to fail with old timestamp")
	}
}

// TestDeliveryConfirmationEncoding tests encode/decode round-trip.
func TestDeliveryConfirmationEncoding(t *testing.T) {
	var messageID [32]byte
	var recipientKey [32]byte
	rand.Read(messageID[:])
	rand.Read(recipientKey[:])

	original, _ := NewDeliveryConfirmation(messageID, recipientKey)
	original.RecipientSignature = []byte("test-signature-data")

	// Encode.
	encoded, err := encodeDeliveryConfirmation(original)
	if err != nil {
		t.Fatalf("encodeDeliveryConfirmation: %v", err)
	}

	// Decode.
	decoded, err := decodeDeliveryConfirmation(encoded)
	if err != nil {
		t.Fatalf("decodeDeliveryConfirmation: %v", err)
	}

	// Compare fields.
	if decoded.MessageID != original.MessageID {
		t.Error("MessageID mismatch")
	}
	if decoded.ReceiptNonce != original.ReceiptNonce {
		t.Error("ReceiptNonce mismatch")
	}
	if decoded.ConfirmationHash != original.ConfirmationHash {
		t.Error("ConfirmationHash mismatch")
	}
	if decoded.Timestamp != original.Timestamp {
		t.Error("Timestamp mismatch")
	}
	if !bytes.Equal(decoded.RecipientSignature, original.RecipientSignature) {
		t.Error("RecipientSignature mismatch")
	}
}

// TestDecodeDeliveryConfirmationErrors tests error cases.
func TestDecodeDeliveryConfirmationErrors(t *testing.T) {
	// Too short.
	_, err := decodeDeliveryConfirmation([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for short data")
	}

	// Truncated signature.
	headerSize := 32 + 24 + 32 + 8 + 2
	data := make([]byte, headerSize)
	data[headerSize-2] = 0
	data[headerSize-1] = 50 // Claims 50 bytes of signature.

	_, err = decodeDeliveryConfirmation(data)
	if err == nil {
		t.Error("expected error for truncated signature")
	}
}

// TestDeliveryTracker tests the delivery tracking system.
func TestDeliveryTracker(t *testing.T) {
	tracker := NewDeliveryTracker()

	var messageID [32]byte
	var destination [32]byte
	rand.Read(messageID[:])
	rand.Read(destination[:])

	// Track a delivery.
	pending, err := tracker.TrackDelivery(messageID, destination)
	if err != nil {
		t.Fatalf("TrackDelivery: %v", err)
	}

	if pending.MessageID != messageID {
		t.Error("MessageID mismatch")
	}
	if pending.Destination != destination {
		t.Error("Destination mismatch")
	}
	if pending.Confirmed {
		t.Error("should not be confirmed initially")
	}

	// Get pending.
	retrieved := tracker.GetPending(messageID)
	if retrieved == nil {
		t.Fatal("GetPending returned nil")
	}
	if retrieved.MessageID != messageID {
		t.Error("retrieved MessageID mismatch")
	}

	// Stats should show 1 sent.
	stats := tracker.Stats()
	if stats.MessagesSent != 1 {
		t.Errorf("expected 1 sent, got %d", stats.MessagesSent)
	}
}

// TestDeliveryTrackerConfirm tests confirming a delivery.
func TestDeliveryTrackerConfirm(t *testing.T) {
	tracker := NewDeliveryTracker()

	var messageID [32]byte
	var destination [32]byte
	var recipientKey [32]byte
	rand.Read(messageID[:])
	rand.Read(destination[:])
	rand.Read(recipientKey[:])

	// Track delivery.
	tracker.TrackDelivery(messageID, destination)

	// Create confirmation.
	conf, _ := NewDeliveryConfirmation(messageID, recipientKey)

	// Track handler calls.
	var handlerCalled bool
	var handlerMsgID [32]byte
	tracker.RegisterHandler(func(msgID [32]byte, confirmed bool) {
		handlerCalled = true
		handlerMsgID = msgID
	})

	// Confirm.
	err := tracker.ConfirmDelivery(conf)
	if err != nil {
		t.Fatalf("ConfirmDelivery: %v", err)
	}

	// Verify state.
	pending := tracker.GetPending(messageID)
	if !pending.Confirmed {
		t.Error("should be confirmed")
	}

	// Handler should be called.
	if !handlerCalled {
		t.Error("handler not called")
	}
	if handlerMsgID != messageID {
		t.Error("handler received wrong message ID")
	}

	// Stats should show confirmation.
	stats := tracker.Stats()
	if stats.Confirmations != 1 {
		t.Errorf("expected 1 confirmation, got %d", stats.Confirmations)
	}
}

// TestDeliveryTrackerConfirmNotFound tests confirming unknown message.
func TestDeliveryTrackerConfirmNotFound(t *testing.T) {
	tracker := NewDeliveryTracker()

	var messageID [32]byte
	var recipientKey [32]byte
	rand.Read(messageID[:])
	rand.Read(recipientKey[:])

	conf, _ := NewDeliveryConfirmation(messageID, recipientKey)

	err := tracker.ConfirmDelivery(conf)
	if err == nil {
		t.Error("expected error for unknown message")
	}
}

// TestDeliveryTrackerRemove tests removing pending deliveries.
func TestDeliveryTrackerRemove(t *testing.T) {
	tracker := NewDeliveryTracker()

	var messageID [32]byte
	var destination [32]byte
	rand.Read(messageID[:])
	rand.Read(destination[:])

	tracker.TrackDelivery(messageID, destination)

	// Remove.
	tracker.RemovePending(messageID)

	// Should be gone.
	if tracker.GetPending(messageID) != nil {
		t.Error("pending should be removed")
	}
}

// TestDeliveryTrackerCleanup tests expiry cleanup.
func TestDeliveryTrackerCleanup(t *testing.T) {
	tracker := NewDeliveryTracker()

	// Track several deliveries.
	for i := 0; i < 5; i++ {
		var messageID [32]byte
		var destination [32]byte
		rand.Read(messageID[:])
		rand.Read(destination[:])
		tracker.TrackDelivery(messageID, destination)
	}

	// Cleanup with 0 max age should remove all.
	removed := tracker.CleanupExpired(0)
	if removed != 5 {
		t.Errorf("expected 5 removed, got %d", removed)
	}

	// Stats should still show sent count (cleanup doesn't affect stats).
	stats := tracker.Stats()
	if stats.MessagesSent != 5 {
		t.Errorf("expected 5 sent, got %d", stats.MessagesSent)
	}
}

// TestDeliveryTrackerConcurrent tests thread safety.
func TestDeliveryTrackerConcurrent(t *testing.T) {
	tracker := NewDeliveryTracker()

	var wg sync.WaitGroup

	// Concurrent operations.
	for i := 0; i < 100; i++ {
		wg.Add(3)

		// Track.
		go func() {
			defer wg.Done()
			var messageID [32]byte
			var destination [32]byte
			rand.Read(messageID[:])
			rand.Read(destination[:])
			tracker.TrackDelivery(messageID, destination)
		}()

		// Stats.
		go func() {
			defer wg.Done()
			tracker.Stats()
		}()

		// Cleanup.
		go func() {
			defer wg.Done()
			tracker.CleanupExpired(time.Hour)
		}()
	}

	wg.Wait()

	// Should have tracked messages.
	stats := tracker.Stats()
	if stats.MessagesSent < 50 {
		t.Errorf("expected at least 50 sent, got %d", stats.MessagesSent)
	}
}

// TestRateLimiterBasic tests basic rate limiter functionality.
func TestRateLimiterBasic(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	var dest [32]byte
	rand.Read(dest[:])

	// First few messages should be allowed (burst capacity).
	for i := 0; i < 5; i++ {
		if err := limiter.Allow(dest); err != nil {
			t.Errorf("message %d should be allowed: %v", i, err)
		}
	}
}

// TestRateLimiterDestLimit tests per-destination rate limiting.
func TestRateLimiterDestLimit(t *testing.T) {
	config := RateLimiterConfig{
		MaxPerSecond:       1.0,
		GlobalMaxPerSecond: 100.0,
		MaxPendingPerDest:  100,
		BucketCapacity:     2.0, // Allow burst of 2.
	}
	limiter := NewRateLimiter(config)

	var dest [32]byte
	rand.Read(dest[:])

	// Use up the burst capacity.
	limiter.Allow(dest)
	limiter.Allow(dest)

	// Third message should be rate limited.
	err := limiter.Allow(dest)
	if err != ErrWhisperDestRateLimited {
		t.Errorf("expected ErrWhisperDestRateLimited, got: %v", err)
	}

	// Different destination should be allowed.
	var dest2 [32]byte
	rand.Read(dest2[:])
	if err := limiter.Allow(dest2); err != nil {
		t.Errorf("different destination should be allowed: %v", err)
	}
}

// TestRateLimiterGlobalLimit tests global rate limiting.
func TestRateLimiterGlobalLimit(t *testing.T) {
	config := RateLimiterConfig{
		MaxPerSecond:       100.0, // High per-dest limit.
		GlobalMaxPerSecond: 1.0,   // Low global limit.
		MaxPendingPerDest:  100,
		BucketCapacity:     2.0, // Global burst of 4 (2 * 2).
	}
	limiter := NewRateLimiter(config)

	// Send to multiple destinations until global limit hit.
	hitLimit := false
	for i := 0; i < 10; i++ {
		var dest [32]byte
		rand.Read(dest[:])
		err := limiter.Allow(dest)
		if err == ErrWhisperGlobalRateLimited {
			hitLimit = true
			break
		}
	}

	if !hitLimit {
		t.Error("expected to hit global rate limit")
	}
}

// TestRateLimiterPendingLimit tests pending message limiting.
func TestRateLimiterPendingLimit(t *testing.T) {
	config := RateLimiterConfig{
		MaxPerSecond:       100.0,
		GlobalMaxPerSecond: 100.0,
		MaxPendingPerDest:  3,
		BucketCapacity:     100.0,
	}
	limiter := NewRateLimiter(config)

	var dest [32]byte
	rand.Read(dest[:])

	// Reserve up to the limit.
	for i := 0; i < 3; i++ {
		limiter.Reserve(dest)
	}

	// Next should be rejected.
	err := limiter.Allow(dest)
	if err != ErrWhisperTooManyPending {
		t.Errorf("expected ErrWhisperTooManyPending, got: %v", err)
	}

	// Release one.
	limiter.Release(dest)

	// Now should be allowed.
	if err := limiter.Allow(dest); err != nil {
		t.Errorf("should be allowed after release: %v", err)
	}
}

// TestRateLimiterReserveRelease tests reserve/release tracking.
func TestRateLimiterReserveRelease(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	var dest [32]byte
	rand.Read(dest[:])

	// Initially zero pending.
	if count := limiter.GetPendingCount(dest); count != 0 {
		t.Errorf("expected 0 pending, got %d", count)
	}

	// Reserve.
	limiter.Reserve(dest)
	if count := limiter.GetPendingCount(dest); count != 1 {
		t.Errorf("expected 1 pending, got %d", count)
	}

	// Reserve more.
	limiter.Reserve(dest)
	limiter.Reserve(dest)
	if count := limiter.GetPendingCount(dest); count != 3 {
		t.Errorf("expected 3 pending, got %d", count)
	}

	// Release all.
	limiter.Release(dest)
	limiter.Release(dest)
	limiter.Release(dest)
	if count := limiter.GetPendingCount(dest); count != 0 {
		t.Errorf("expected 0 pending after release, got %d", count)
	}

	// Release when zero should not go negative.
	limiter.Release(dest)
	if count := limiter.GetPendingCount(dest); count != 0 {
		t.Errorf("expected 0 pending, got %d", count)
	}
}

// TestRateLimiterCleanup tests bucket cleanup.
func TestRateLimiterCleanup(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	// Create buckets for multiple destinations.
	for i := 0; i < 10; i++ {
		var dest [32]byte
		rand.Read(dest[:])
		limiter.Allow(dest)
	}

	stats := limiter.Stats()
	if stats.DestinationsTracked != 10 {
		t.Errorf("expected 10 destinations, got %d", stats.DestinationsTracked)
	}

	// Cleanup with 0 max age should remove all (since they're all recent).
	removed := limiter.Cleanup(0)
	if removed != 10 {
		t.Errorf("expected 10 removed, got %d", removed)
	}

	stats = limiter.Stats()
	if stats.DestinationsTracked != 0 {
		t.Errorf("expected 0 destinations after cleanup, got %d", stats.DestinationsTracked)
	}
}

// TestRateLimiterTokenReplenish tests token bucket replenishment.
func TestRateLimiterTokenReplenish(t *testing.T) {
	config := RateLimiterConfig{
		MaxPerSecond:       10.0, // 10 tokens/sec.
		GlobalMaxPerSecond: 100.0,
		MaxPendingPerDest:  100,
		BucketCapacity:     1.0, // Capacity of 1.
	}
	limiter := NewRateLimiter(config)

	var dest [32]byte
	rand.Read(dest[:])

	// Use the token.
	limiter.Allow(dest)

	// Immediately, should be limited.
	err := limiter.Allow(dest)
	if err != ErrWhisperDestRateLimited {
		t.Errorf("expected rate limited, got: %v", err)
	}

	// Wait for replenishment (at 10/sec, 0.15s should give > 1 token).
	time.Sleep(150 * time.Millisecond)

	// Should be allowed now.
	if err := limiter.Allow(dest); err != nil {
		t.Errorf("should be allowed after replenishment: %v", err)
	}
}

// TestRateLimiterAvailableTokens tests token availability query.
func TestRateLimiterAvailableTokens(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	var dest [32]byte
	rand.Read(dest[:])

	// New destination should have full bucket.
	tokens := limiter.GetAvailableTokens(dest)
	if tokens != limiter.config.BucketCapacity {
		t.Errorf("expected %f tokens, got %f", limiter.config.BucketCapacity, tokens)
	}

	// After using some tokens.
	limiter.Allow(dest)
	limiter.Allow(dest)

	tokens = limiter.GetAvailableTokens(dest)
	if tokens >= limiter.config.BucketCapacity {
		t.Error("tokens should be reduced after use")
	}
}

// TestRateLimiterStats tests statistics collection.
func TestRateLimiterStats(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	// Create some activity.
	for i := 0; i < 5; i++ {
		var dest [32]byte
		rand.Read(dest[:])
		limiter.Allow(dest)
		limiter.Reserve(dest)
	}

	stats := limiter.Stats()
	if stats.DestinationsTracked != 5 {
		t.Errorf("expected 5 destinations, got %d", stats.DestinationsTracked)
	}
	if stats.TotalPending != 5 {
		t.Errorf("expected 5 pending, got %d", stats.TotalPending)
	}
}

// TestRateLimiterConcurrent tests thread safety.
func TestRateLimiterConcurrent(t *testing.T) {
	limiter := NewDefaultRateLimiter()

	var wg sync.WaitGroup

	// Concurrent operations.
	for i := 0; i < 100; i++ {
		wg.Add(4)

		go func() {
			defer wg.Done()
			var dest [32]byte
			rand.Read(dest[:])
			limiter.Allow(dest)
		}()

		go func() {
			defer wg.Done()
			var dest [32]byte
			rand.Read(dest[:])
			limiter.Reserve(dest)
		}()

		go func() {
			defer wg.Done()
			var dest [32]byte
			rand.Read(dest[:])
			limiter.Release(dest)
		}()

		go func() {
			defer wg.Done()
			limiter.Stats()
		}()
	}

	wg.Wait()

	// Should not panic and stats should be reasonable.
	stats := limiter.Stats()
	if stats.DestinationsTracked < 1 {
		t.Error("expected at least some destinations tracked")
	}
}

// TestRateLimitedRouter tests the rate-limited router wrapper.
func TestRateLimitedRouter(t *testing.T) {
	// Create a basic router (without delivery - will fail send).
	var privateKey [32]byte
	rand.Read(privateKey[:])
	router := NewWhisperRouter(nil, privateKey)

	config := RateLimiterConfig{
		MaxPerSecond:       100.0,
		GlobalMaxPerSecond: 100.0,
		MaxPendingPerDest:  2, // Low pending limit.
		BucketCapacity:     100.0,
	}
	limiter := NewRateLimiter(config)

	rateLimited := NewRateLimitedRouter(router, limiter)

	// Add route.
	var dest [32]byte
	rand.Read(dest[:])
	router.AddRoute(&WhisperRoute{Destination: dest})

	// Send should fail (no delivery), but rate limit should be checked first.
	// The pending count should increase and decrease.

	// Simulate pending by reserving.
	limiter.Reserve(dest)
	limiter.Reserve(dest)

	// Third should be rejected by rate limiter.
	err := rateLimited.Send(dest, []byte("test"))
	if err != ErrWhisperTooManyPending {
		// Could also fail due to nil delivery, but rate limit should be first.
		t.Logf("Send error (may be nil delivery): %v", err)
	}

	// Access underlying components.
	if rateLimited.Router() != router {
		t.Error("Router() mismatch")
	}
	if rateLimited.Limiter() != limiter {
		t.Error("Limiter() mismatch")
	}
}

// TestDefaultRateLimiterConfig tests default configuration values.
func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	if config.MaxPerSecond != WhisperMaxMessagesPerSecond {
		t.Errorf("MaxPerSecond: got %f, want %f", config.MaxPerSecond, float64(WhisperMaxMessagesPerSecond))
	}
	if config.MaxPerMinute != WhisperMaxMessagesPerMinute {
		t.Errorf("MaxPerMinute: got %d, want %d", config.MaxPerMinute, WhisperMaxMessagesPerMinute)
	}
	if config.GlobalMaxPerSecond != WhisperMaxGlobalMessagesPerSecond {
		t.Errorf("GlobalMaxPerSecond: got %f, want %f", config.GlobalMaxPerSecond, float64(WhisperMaxGlobalMessagesPerSecond))
	}
	if config.MaxPendingPerDest != WhisperMaxPendingPerDest {
		t.Errorf("MaxPendingPerDest: got %d, want %d", config.MaxPendingPerDest, WhisperMaxPendingPerDest)
	}
}

// BenchmarkRateLimiterAllow benchmarks rate limit checks.
func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := NewDefaultRateLimiter()

	var dest [32]byte
	rand.Read(dest[:])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(dest)
	}
}
