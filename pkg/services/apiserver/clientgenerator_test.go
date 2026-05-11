package apiserver

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"
)

type countingRestConfigProvider struct {
	cfg   *clientrest.Config
	err   error
	calls int32
}

func (c *countingRestConfigProvider) GetRestConfig(context.Context) (*clientrest.Config, error) {
	atomic.AddInt32(&c.calls, 1)
	return c.cfg, c.err
}

func TestProvideClientGenerator_ReturnsLazy(t *testing.T) {
	p := &countingRestConfigProvider{cfg: &clientrest.Config{}}
	g := ProvideClientGenerator(p)
	require.NotNil(t, g)
	lazy, ok := g.(*lazyClientGenerator)
	require.True(t, ok)
	require.Same(t, p, lazy.restConfigProvider)
}

func TestLazyClientGenerator_ClientFor_RestConfigError(t *testing.T) {
	wantErr := errors.New("no config")
	p := &countingRestConfigProvider{err: wantErr}
	g := ProvideClientGenerator(p)

	c, err := g.ClientFor(resource.Kind{})
	require.Nil(t, c)
	require.ErrorIs(t, err, wantErr)

	c, err = g.ClientFor(resource.Kind{})
	require.Nil(t, c)
	require.ErrorIs(t, err, wantErr)

	// initOnce should ensure provider was only invoked once.
	require.Equal(t, int32(1), atomic.LoadInt32(&p.calls))
}

func TestLazyClientGenerator_ClientFor_InitializesRegistry(t *testing.T) {
	p := &countingRestConfigProvider{cfg: &clientrest.Config{Host: "https://example.test"}}
	g := ProvideClientGenerator(p)
	lazy := g.(*lazyClientGenerator)

	// We don't care whether ClientFor succeeds with an empty Kind; we care that
	// initialization runs and sets APIPath + creates the underlying generator.
	_, _ = g.ClientFor(resource.Kind{})

	require.Equal(t, "apis", p.cfg.APIPath)
	require.NotNil(t, lazy.clientGenerator)
	require.NoError(t, lazy.initError)

	// Subsequent calls must reuse the same provider invocation (sync.Once).
	_, _ = g.ClientFor(resource.Kind{})
	require.Equal(t, int32(1), atomic.LoadInt32(&p.calls))
}
