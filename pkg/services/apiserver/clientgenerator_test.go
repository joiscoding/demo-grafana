package apiserver

import (
	"context"
	"testing"

	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"
)

func TestProvideClientGenerator(t *testing.T) {
	t.Parallel()

	gen := ProvideClientGenerator(WithoutRestConfig)
	require.NotNil(t, gen)
}

func Test_lazyClientGenerator_ClientFor(t *testing.T) {
	t.Parallel()

	t.Run("returns rest config provider error", func(t *testing.T) {
		gen := ProvideClientGenerator(WithoutRestConfig)

		_, err := gen.ClientFor(resource.Kind{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rest config will not be available")

		// initOnce ensures the error is cached
		_, err = gen.ClientFor(resource.Kind{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rest config will not be available")
	})

	t.Run("initializes client registry from rest config", func(t *testing.T) {
		provider := RestConfigProviderFunc(func(context.Context) (*clientrest.Config, error) {
			return &clientrest.Config{Host: "http://127.0.0.1:0"}, nil
		})
		gen := ProvideClientGenerator(provider)

		_, err := gen.ClientFor(resource.Kind{})
		// ClientFor may fail for an empty kind, but initialization should succeed.
		if err != nil {
			assert.NotContains(t, err.Error(), "rest config will not be available")
		}
	})
}
