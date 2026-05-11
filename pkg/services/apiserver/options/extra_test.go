package options

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

func TestNewExtraOptions(t *testing.T) {
	o := NewExtraOptions()
	require.NotNil(t, o)
	assert.False(t, o.DevMode)
	assert.Equal(t, 0, o.Verbosity)
	assert.Equal(t, 10*time.Minute, o.RequestTimeout)
}

func TestExtraOptions_AddFlags(t *testing.T) {
	o := NewExtraOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	o.AddFlags(fs)

	require.NoError(t, fs.Parse([]string{
		"--grafana-apiserver-dev-mode=true",
		"--grafana-apiserver-host=example.com",
		"--grafana-apiserver-api-url=http://api",
		"--verbosity=5",
	}))
	assert.True(t, o.DevMode)
	assert.Equal(t, "example.com", o.ExternalAddress)
	assert.Equal(t, "http://api", o.APIURL)
	assert.Equal(t, 5, o.Verbosity)
}

func TestExtraOptions_Validate(t *testing.T) {
	o := NewExtraOptions()
	assert.Nil(t, o.Validate())
}

func TestExtraOptions_ApplyTo(t *testing.T) {
	o := NewExtraOptions()
	o.ExternalAddress = "ext.example.com"
	o.Verbosity = 9

	cfg := &genericapiserver.RecommendedConfig{}
	// set a placeholder TracerProvider to exercise the nil-out branch
	cfg.Config = genericapiserver.Config{}

	require.NoError(t, o.ApplyTo(cfg))
	assert.Equal(t, "ext.example.com", cfg.ExternalAddress)
	// verbosity should be capped at 7
	assert.Equal(t, 7, o.Verbosity)
}

func TestExtraOptions_ApplyTo_LowVerbosity(t *testing.T) {
	o := NewExtraOptions()
	o.Verbosity = 3
	cfg := &genericapiserver.RecommendedConfig{}
	require.NoError(t, o.ApplyTo(cfg))
	assert.Equal(t, 3, o.Verbosity)
}
