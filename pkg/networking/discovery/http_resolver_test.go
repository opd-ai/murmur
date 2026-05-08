// Package discovery provides tests for HTTP bootstrap resolver utilities.
package discovery

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveBody starts a test server that responds with the given body and status.
func serveBody(status int, body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
}

func TestFetchPeerListBody_OversizedResponse(t *testing.T) {
	// Serve a body larger than maxBootstrapResponseBytes.
	oversized := bytes.Repeat([]byte("x"), int(maxBootstrapResponseBytes)+2)
	srv := serveBody(http.StatusOK, oversized)
	defer srv.Close()

	_, err := fetchPeerListBody(context.Background(), http.DefaultClient, srv.URL, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestFetchPeerListBody_NonOKStatus(t *testing.T) {
	srv := serveBody(http.StatusNotFound, []byte("not found"))
	defer srv.Close()

	_, err := fetchPeerListBody(context.Background(), http.DefaultClient, srv.URL, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchPeerListBody_ValidResponse(t *testing.T) {
	payload := []byte(`{"data":"ok"}`)
	srv := serveBody(http.StatusOK, payload)
	defer srv.Close()

	got, err := fetchPeerListBody(context.Background(), http.DefaultClient, srv.URL, "test")
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestIPFSGatewayResolver_OversizedPeerList(t *testing.T) {
	// Serve an oversized peers.json body.
	oversized := bytes.Repeat([]byte("x"), int(maxBootstrapResponseBytes)+2)
	peersSrv := serveBody(http.StatusOK, oversized)
	defer peersSrv.Close()

	// Serve a valid CID.
	cidSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bafybeifakecidfortesting")
	}))
	defer cidSrv.Close()

	// Point the gateway at the oversized server by crafting the URL directly.
	// We override gatewayURL so that /ipfs/<cid>/peers.json resolves to our server.
	// Use the peersSrv URL as a fake gateway base and a CID that won't appear in path.
	resolver := &IPFSGatewayResolver{
		cidURL:      cidSrv.URL,
		gatewayURL:  strings.TrimSuffix(peersSrv.URL, "/"),
		verifyKey:   make([]byte, 32), // dummy — will fail before verification
		client:      &http.Client{},
		pagesClient: &http.Client{},
	}

	_, err := resolver.Resolve(context.Background())
	require.Error(t, err)
}

func TestIPFSGatewayResolver_OversizedCID(t *testing.T) {
	// Serve an oversized CID body.
	oversized := bytes.Repeat([]byte("x"), int(maxBootstrapResponseBytes)+2)
	cidSrv := serveBody(http.StatusOK, oversized)
	defer cidSrv.Close()

	resolver := &IPFSGatewayResolver{
		cidURL:      cidSrv.URL,
		gatewayURL:  "https://dweb.link",
		verifyKey:   make([]byte, 32),
		client:      &http.Client{},
		pagesClient: &http.Client{},
	}

	_, err := resolver.Resolve(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}
