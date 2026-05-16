package options

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTripperFunc_RoundTrip_withFn(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rt := &RoundTripperFunc{
		Fn: func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, req, r)
			return &http.Response{StatusCode: http.StatusTeapot}, nil
		},
	}

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
}

func TestRoundTripperFunc_RoundTrip_waitsForReady(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	ready := make(chan struct{})
	rt := &RoundTripperFunc{Ready: ready}

	go func() {
		time.Sleep(50 * time.Millisecond)
		rt.Fn = func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}
		close(ready)
	}()

	start := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
}
