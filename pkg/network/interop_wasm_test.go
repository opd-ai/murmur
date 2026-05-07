//go:build js && wasm

package network

import (
	"context"
	"testing"
	"time"
)

// TestDesktopBrowserInteropContractWASM validates that the browser runtime
// produces the same Adapter-level message semantics as desktop.
func TestDesktopBrowserInteropContractWASM(t *testing.T) {
	adapterRaw, err := newWASMAdapter(Config{Platform: PlatformWASM})
	if err != nil {
		t.Fatalf("newWASMAdapter() error = %v", err)
	}

	if err := adapterRaw.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = adapterRaw.Stop(context.Background()) })

	topic := "/murmur/test/interop-contract"
	messageCh, err := adapterRaw.Subscribe(topic)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	payload := []byte("interop-contract")
	if err := adapterRaw.Publish(context.Background(), topic, payload); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case msg := <-messageCh:
		if msg.Topic != topic {
			t.Fatalf("message topic = %q, want %q", msg.Topic, topic)
		}
		if string(msg.Payload) != string(payload) {
			t.Fatalf("message payload = %q, want %q", string(msg.Payload), string(payload))
		}
		if msg.From == "" {
			t.Fatal("expected non-empty sender ID in adapter message")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for interop contract message")
	}
}
