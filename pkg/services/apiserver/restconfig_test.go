package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"

	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/web"
)

// fakeInnerProvider implements both RestConfigProvider and DirectRestConfigProvider.
type fakeInnerProvider struct {
	cfg          *clientrest.Config
	err          error
	servedReq    *http.Request
	directCalled bool
}

func (f *fakeInnerProvider) GetRestConfig(ctx context.Context) (*clientrest.Config, error) {
	return f.cfg, f.err
}

func (f *fakeInnerProvider) GetDirectRestConfig(c *contextmodel.ReqContext) *clientrest.Config {
	f.directCalled = true
	return f.cfg
}

func (f *fakeInnerProvider) DirectlyServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.servedReq = r
	w.WriteHeader(http.StatusTeapot)
}

func TestWithoutRestConfig_ReturnsError(t *testing.T) {
	cfg, err := WithoutRestConfig.GetRestConfig(context.Background())
	require.Nil(t, cfg)
	require.Error(t, err)
}

func TestRestConfigProviderFunc_Delegates(t *testing.T) {
	expected := &clientrest.Config{Host: "https://example"}
	f := RestConfigProviderFunc(func(ctx context.Context) (*clientrest.Config, error) {
		return expected, nil
	})
	got, err := f.GetRestConfig(context.Background())
	require.NoError(t, err)
	require.Same(t, expected, got)
}

func TestProvideEventualRestConfigProvider_NotNil(t *testing.T) {
	e := ProvideEventualRestConfigProvider()
	require.NotNil(t, e)
	require.NotNil(t, e.ready)
}

func TestEventualRestConfigProvider_GetRestConfig_ContextCancelled(t *testing.T) {
	e := ProvideEventualRestConfigProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg, err := e.GetRestConfig(ctx)
	require.Nil(t, cfg)
	require.ErrorIs(t, err, context.Canceled)
}

func TestEventualRestConfigProvider_GetRestConfig_Ready(t *testing.T) {
	expected := &clientrest.Config{Host: "https://example"}
	inner := &fakeInnerProvider{cfg: expected}
	e := ProvideEventualRestConfigProvider()
	e.cfg = inner
	close(e.ready)

	got, err := e.GetRestConfig(context.Background())
	require.NoError(t, err)
	require.Same(t, expected, got)
}

func TestEventualRestConfigProvider_GetRestConfig_ReadyPropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	inner := &fakeInnerProvider{err: wantErr}
	e := ProvideEventualRestConfigProvider()
	e.cfg = inner
	close(e.ready)

	got, err := e.GetRestConfig(context.Background())
	require.Nil(t, got)
	require.ErrorIs(t, err, wantErr)
}

func newReqContext(ctx context.Context) *contextmodel.ReqContext {
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	return &contextmodel.ReqContext{
		Context: &web.Context{Req: req, Resp: web.NewResponseWriter(req.Method, httptest.NewRecorder())},
	}
}

func TestEventualRestConfigProvider_GetDirectRestConfig_ContextCancelled(t *testing.T) {
	e := ProvideEventualRestConfigProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rc := newReqContext(ctx)

	got := e.GetDirectRestConfig(rc)
	require.Nil(t, got)
}

func TestEventualRestConfigProvider_GetDirectRestConfig_Ready(t *testing.T) {
	expected := &clientrest.Config{Host: "https://example"}
	inner := &fakeInnerProvider{cfg: expected}
	e := ProvideEventualRestConfigProvider()
	e.cfg = inner
	close(e.ready)

	rc := newReqContext(context.Background())
	got := e.GetDirectRestConfig(rc)
	require.Same(t, expected, got)
	require.True(t, inner.directCalled)
}

func TestEventualRestConfigProvider_DirectlyServeHTTP_ContextCancelled(t *testing.T) {
	e := ProvideEventualRestConfigProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	e.DirectlyServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code) // unchanged default
}

func TestEventualRestConfigProvider_DirectlyServeHTTP_Ready(t *testing.T) {
	inner := &fakeInnerProvider{}
	e := ProvideEventualRestConfigProvider()
	e.cfg = inner
	close(e.ready)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	e.DirectlyServeHTTP(rec, req)
	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Same(t, req, inner.servedReq)
}

// Sanity check that the blocking select doesn't return prematurely before ready.
func TestEventualRestConfigProvider_GetRestConfig_BlocksUntilReady(t *testing.T) {
	expected := &clientrest.Config{Host: "https://example"}
	inner := &fakeInnerProvider{cfg: expected}
	e := ProvideEventualRestConfigProvider()
	e.cfg = inner

	done := make(chan struct{})
	go func() {
		_, err := e.GetRestConfig(context.Background())
		require.NoError(t, err)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("GetRestConfig returned before ready was closed")
	case <-time.After(50 * time.Millisecond):
	}

	close(e.ready)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GetRestConfig did not return after ready was closed")
	}
}
