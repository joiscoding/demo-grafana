package options

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTripperFunc_RoundTrip_Calls(t *testing.T) {
	called := false
	rt := &RoundTripperFunc{
		Ready: make(chan struct{}),
		Fn: func(req *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{StatusCode: http.StatusTeapot}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
}

func TestRoundTripperFunc_RoundTrip_WaitsForReady(t *testing.T) {
	ready := make(chan struct{})
	rt := &RoundTripperFunc{Ready: ready}

	done := make(chan struct{})
	var panicked bool
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
			close(done)
		}()
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		_, _ = rt.RoundTrip(req)
	}()

	// release the gate; Fn is nil, so the call panics after waking up.
	close(ready)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("round trip did not return after ready was closed")
	}
	assert.True(t, panicked)
}
