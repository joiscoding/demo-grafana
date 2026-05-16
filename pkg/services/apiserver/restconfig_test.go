package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"

	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/web"
)

func TestWithoutRestConfig(t *testing.T) {
	t.Parallel()

	_, err := WithoutRestConfig.GetRestConfig(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rest config will not be available")
}

func TestRestConfigProviderFunc(t *testing.T) {
	t.Parallel()

	expected := &clientrest.Config{Host: "https://example.test"}
	provider := RestConfigProviderFunc(func(ctx context.Context) (*clientrest.Config, error) {
		return expected, nil
	})

	cfg, err := provider.GetRestConfig(context.Background())
	require.NoError(t, err)
	assert.Same(t, expected, cfg)
}

func TestProvideEventualRestConfigProvider(t *testing.T) {
	t.Parallel()

	provider := ProvideEventualRestConfigProvider()
	require.NotNil(t, provider)
	require.NotNil(t, provider.ready)
}

func Test_eventualRestConfigProvider_GetRestConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns error when context is cancelled before ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := provider.GetRestConfig(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("delegates after ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()
		expected := &clientrest.Config{Host: "https://loopback.test"}
		provider.cfg = &stubDirectRestConfigProvider{
			restConfig: expected,
		}
		close(provider.ready)

		cfg, err := provider.GetRestConfig(context.Background())
		require.NoError(t, err)
		assert.Same(t, expected, cfg)
	})
}

func Test_eventualRestConfigProvider_GetDirectRestConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when request context is cancelled before ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)

		rc := &contextmodel.ReqContext{Context: &web.Context{Req: req}}
		assert.Nil(t, provider.GetDirectRestConfig(rc))
	})

	t.Run("delegates after ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()
		expected := &clientrest.Config{Host: "https://direct.test"}
		provider.cfg = &stubDirectRestConfigProvider{
			directConfig: expected,
		}
		close(provider.ready)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rc := &contextmodel.ReqContext{Context: &web.Context{Req: req}}

		cfg := provider.GetDirectRestConfig(rc)
		assert.Same(t, expected, cfg)
	})
}

func Test_eventualRestConfigProvider_DirectlyServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when request is cancelled before ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		provider.DirectlyServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("delegates after ready", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()
		delegate := &stubDirectRestConfigProvider{}
		provider.cfg = delegate
		close(provider.ready)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		provider.DirectlyServeHTTP(rec, req)

		assert.True(t, delegate.served)
	})
}

type stubDirectRestConfigProvider struct {
	restConfig   *clientrest.Config
	directConfig *clientrest.Config
	served       bool
}

func (s *stubDirectRestConfigProvider) GetRestConfig(context.Context) (*clientrest.Config, error) {
	if s.restConfig != nil {
		return s.restConfig, nil
	}
	return &clientrest.Config{}, nil
}

func (s *stubDirectRestConfigProvider) GetDirectRestConfig(*contextmodel.ReqContext) *clientrest.Config {
	return s.directConfig
}

func (s *stubDirectRestConfigProvider) DirectlyServeHTTP(http.ResponseWriter, *http.Request) {
	s.served = true
}
