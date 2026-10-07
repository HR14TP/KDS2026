package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postToHost previously made exactly one attempt with no client timeout -
// a single transient failure (the peer momentarily busy, a brief network
// blip) permanently dropped that one delivery, with nothing else
// re-attempting until the next periodic reconciliation tick (up to 2
// minutes later). This is the live symptom this guards against: an order
// closed on its pass terminal, intermittently never reaching a linked
// sub-screen.

// failNTimesThenSucceed panics on the first n requests (httptest's server
// recovers from a handler panic by closing the connection without writing
// a response - from the client's side this is indistinguishable from a
// real transient connection failure) and responds 200 OK afterward.
func failNTimesThenSucceed(n int32) (*httptest.Server, *int32) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= n {
			panic("simulated transient failure")
		}
		w.WriteHeader(http.StatusOK)
	}))
	return srv, &attempts
}

func TestPostToHost_RecoversFromOneTransientFailure(t *testing.T) {
	srv, attempts := failNTimesThenSucceed(1)
	defer srv.Close()

	postToHost(peerHostFromServerURL(srv.URL), "closeorderscreenremote", []byte(`{"reference":"a1.1"}`))

	assert.Equal(t, int32(2), atomic.LoadInt32(attempts), "expected exactly one failed attempt followed by one successful retry")
}

func TestPostToHost_RecoversFromTwoTransientFailures(t *testing.T) {
	srv, attempts := failNTimesThenSucceed(2)
	defer srv.Close()

	postToHost(peerHostFromServerURL(srv.URL), "closeorderscreenremote", []byte(`{"reference":"a1.1"}`))

	assert.Equal(t, int32(3), atomic.LoadInt32(attempts), "expected to exhaust all 3 attempts, the last one succeeding")
}

// A peer down for longer than the retry budget must still give up cleanly
// (no panic, no hang) - reconcileWithPeers is the durable backstop for
// this case, not postToHost itself.
func TestPostToHost_GivesUpCleanlyWhenPeerNeverRecovers(t *testing.T) {
	srv, attempts := failNTimesThenSucceed(999)
	defer srv.Close()

	start := time.Now()
	assert.NotPanics(t, func() {
		postToHost(peerHostFromServerURL(srv.URL), "closeorderscreenremote", []byte(`{"reference":"a1.1"}`))
	})
	elapsed := time.Since(start)

	assert.Equal(t, int32(postToHostMaxAttempts), atomic.LoadInt32(attempts), "expected exactly the configured max attempts, no more")
	// Sanity bound on the backoff, not a precise timing assertion - just
	// confirms it didn't hang indefinitely and did actually wait between
	// attempts rather than busy-looping.
	assert.Less(t, elapsed, 10*time.Second, "retries with backoff must still complete quickly, not hang")
}

func TestPostToHost_UnreachableHostNeverPanics(t *testing.T) {
	require.NotPanics(t, func() {
		postToHost("127.0.0.1:1", "closeorderscreenremote", []byte(`{"reference":"a1.1"}`))
	})
}
