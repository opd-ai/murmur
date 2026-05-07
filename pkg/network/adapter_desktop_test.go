//go:build !js

package network

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestDesktopAdapterStartTwiceFails(t *testing.T) {
	t.Parallel()

	adapter, err := newDesktopAdapter(Config{Platform: PlatformDesktop})
	if err != nil {
		t.Fatalf("newDesktopAdapter() error = %v", err)
	}

	if err := adapter.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = adapter.Stop(context.Background())
	})

	err = adapter.Start(context.Background())
	if !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("second Start() error = %v, want %v", err, ErrAlreadyStarted)
	}
}

func TestDesktopAdapterPublishSubscribe(t *testing.T) {
	t.Parallel()

	publisherRaw, err := newDesktopAdapter(Config{Platform: PlatformDesktop})
	if err != nil {
		t.Fatalf("newDesktopAdapter(publisher) error = %v", err)
	}
	publisher := publisherRaw.(*desktopAdapter)
	if err := publisher.Start(context.Background()); err != nil {
		t.Fatalf("publisher Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = publisher.Stop(context.Background())
	})

	subscriberRaw, err := newDesktopAdapter(Config{Platform: PlatformDesktop})
	if err != nil {
		t.Fatalf("newDesktopAdapter(subscriber) error = %v", err)
	}
	subscriber := subscriberRaw.(*desktopAdapter)
	if err := subscriber.Start(context.Background()); err != nil {
		t.Fatalf("subscriber Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = subscriber.Stop(context.Background())
	})

	peerAddr, err := firstPeerAddr(publisher)
	if err != nil {
		t.Fatalf("firstPeerAddr() error = %v", err)
	}
	if err := subscriber.DialPeer(context.Background(), peerAddr); err != nil {
		t.Fatalf("DialPeer() error = %v", err)
	}

	topic := "/murmur/test/desktop-adapter"
	messageCh, err := subscriber.Subscribe(topic)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	payload := []byte("hello-wave")
	deadline := time.After(8 * time.Second)
	publishTicker := time.NewTicker(250 * time.Millisecond)
	defer publishTicker.Stop()

	for {
		select {
		case msg := <-messageCh:
			if string(msg.Payload) != string(payload) {
				continue
			}
			return
		case <-publishTicker.C:
			if err := publisher.Publish(context.Background(), topic, payload); err != nil {
				t.Fatalf("Publish() error = %v", err)
			}
		case <-deadline:
			t.Fatal("timed out waiting for subscribed message")
		}
	}
}

func firstPeerAddr(a *desktopAdapter) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.host == nil {
		return "", ErrNotStarted
	}

	addrs := a.host.Addrs()
	if len(addrs) == 0 {
		return "", errors.New("host has no listen addresses")
	}

	return fmt.Sprintf("%s/p2p/%s", addrs[0].String(), a.host.ID().String()), nil
}
