package options

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	genericapiserver "k8s.io/apiserver/pkg/server"
	k8stracing "k8s.io/component-base/tracing"
)

func TestNewExtraOptions(t *testing.T) {
	opts := NewExtraOptions()
	require.NotNil(t, opts)
	assert.False(t, opts.DevMode)
	assert.Equal(t, 0, opts.Verbosity)
	assert.Equal(t, 10*time.Minute, opts.RequestTimeout)
	assert.Empty(t, opts.ExternalAddress)
	assert.Empty(t, opts.APIURL)
}

func TestExtraOptions_Validate(t *testing.T) {
	opts := &ExtraOptions{Verbosity: 99}
	assert.Empty(t, opts.Validate())
}

func TestExtraOptions_AddFlags(t *testing.T) {
	opts := NewExtraOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	opts.AddFlags(fs)

	require.NoError(t, fs.Parse([]string{
		"--grafana-apiserver-dev-mode=true",
		"--grafana-apiserver-host=localhost:3000",
		"--grafana-apiserver-api-url=https://localhost:3000",
		"--verbosity=3",
	}))

	assert.True(t, opts.DevMode)
	assert.Equal(t, "localhost:3000", opts.ExternalAddress)
	assert.Equal(t, "https://localhost:3000", opts.APIURL)
	assert.Equal(t, 3, opts.Verbosity)
}

func TestExtraOptions_ApplyTo(t *testing.T) {
	t.Run("sets external address and clears tracer provider", func(t *testing.T) {
		opts := &ExtraOptions{
			ExternalAddress: "grafana.example:443",
			Verbosity:       2,
		}
		cfg := genericapiserver.NewRecommendedConfig(testCodecs())
		cfg.TracerProvider = k8stracing.NewNoopTracerProvider()

		require.NoError(t, opts.ApplyTo(cfg))

		assert.Equal(t, "grafana.example:443", cfg.ExternalAddress)
		assert.Nil(t, cfg.TracerProvider)
		assert.Equal(t, 2, opts.Verbosity)
	})

	t.Run("caps verbosity above 7", func(t *testing.T) {
		opts := &ExtraOptions{Verbosity: 10}
		cfg := genericapiserver.NewRecommendedConfig(testCodecs())

		require.NoError(t, opts.ApplyTo(cfg))

		assert.Equal(t, 7, opts.Verbosity)
	})
}
