//go:build !js

package network

import (
	"context"
	"testing"
	"time"
)

// TestDesktopBrowserInteropContractDesktop validates the transport-neutral
// Adapter message contract on the desktop runtime so browser and desktop
// clients can share the same publish/subscribe semantics.
func TestDesktopBrowserInteropContractDesktop(t *testing.T) {
	t.Parallel()

	publisherRaw, err := newDesktopAdapter(Config{Platform: PlatformDesktop})
	if err != nil {
		t.Fatalf("newDesktopAdapter(publisher) error = %v", err)
	}
	publisher := publisherRaw.(*desktopAdapter)
	if err := publisher.Start(context.Background()); err != nil {
		t.Fatalf("publisher Start() error = %v", err)
	}
	t.Cleanup(func() { _ = publisher.Stop(context.Background()) })

	subscriberRaw, err := newDesktopAdapter(Config{Platform: PlatformDesktop})
	if err != nil {
		t.Fatalf("newDesktopAdapter(subscriber) error = %v", err)
	}
	subscriber := subscriberRaw.(*desktopAdapter)
	if err := subscriber.Start(context.Background()); err != nil {
		t.Fatalf("subscriber Start() error = %v", err)
	}
	t.Cleanup(func() { _ = subscriber.Stop(context.Background()) })

	peerAddr, err := firstPeerAddr(publisher)
	if err != nil {
		t.Fatalf("firstPeerAddr() error = %v", err)
	}
	if err := subscriber.DialPeer(context.Background(), peerAddr); err != nil {
		t.Fatalf("DialPeer() error = %v", err)
	}

	topic := "/murmur/test/interop-contract"
	messageCh, err := subscriber.Subscribe(topic)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	payload := []byte("interop-contract")
	deadline := time.After(8 * time.Second)
	publishTicker := time.NewTicker(250 * time.Millisecond)
	defer publishTicker.Stop()

	for {
		select {
		case msg := <-messageCh:
			if msg.Topic != topic {
				continue
			}
			if string(msg.Payload) != string(payload) {
				continue
			}
			if msg.From == "" {
				t.Fatal("expected non-empty sender ID in adapter message")
			}
			return
		case <-publishTicker.C:
			if err := publisher.Publish(context.Background(), topic, payload); err != nil {
				t.Fatalf("Publish() error = %v", err)
			}
		case <-deadline:
			t.Fatal("timed out waiting for interop contract message")
		}
	}
}
