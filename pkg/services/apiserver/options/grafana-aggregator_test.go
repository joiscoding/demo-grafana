package options

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/aggregator/apis/aggregation/v0alpha1"
	aggregatorapiserver "github.com/grafana/grafana/pkg/aggregator/apiserver"
	aggregatorscheme "github.com/grafana/grafana/pkg/aggregator/apiserver/scheme"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

func TestNewGrafanaAggregatorOptions(t *testing.T) {
	opts := NewGrafanaAggregatorOptions()
	require.NotNil(t, opts)
}

func TestGrafanaAggregatorOptions_AddFlags(t *testing.T) {
	t.Run("nil receiver is a no-op", func(t *testing.T) {
		var opts *GrafanaAggregatorOptions
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		assert.NotPanics(t, func() { opts.AddFlags(fs) })
	})

	t.Run("non-nil receiver", func(t *testing.T) {
		opts := NewGrafanaAggregatorOptions()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		assert.NotPanics(t, func() { opts.AddFlags(fs) })
	})
}

func TestGrafanaAggregatorOptions_Validate(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var opts *GrafanaAggregatorOptions
		assert.Nil(t, opts.Validate())
	})

	t.Run("non-nil receiver", func(t *testing.T) {
		opts := NewGrafanaAggregatorOptions()
		assert.Empty(t, opts.Validate())
	})
}

func TestGrafanaAggregatorOptions_ApplyTo(t *testing.T) {
	opts := NewGrafanaAggregatorOptions()
	codec := aggregatorscheme.Codecs.LegacyCodec(v0alpha1.SchemeGroupVersion)

	genericConfig := genericapiserver.NewRecommendedConfig(aggregatorscheme.Codecs)
	aggregatorConfig := &aggregatorapiserver.Config{
		GenericConfig: genericConfig,
	}

	recommendedOpts := NewRecommendedOptions(codec)
	etcdOpts := recommendedOpts.Etcd

	require.NoError(t, opts.ApplyTo(aggregatorConfig, etcdOpts))

	assert.True(t, genericConfig.SkipOpenAPIInstallation)
	assert.NotNil(t, genericConfig.RESTOptionsGetter)
	assert.NotNil(t, genericConfig.OpenAPIConfig)
	assert.NotNil(t, genericConfig.OpenAPIV3Config)
	assert.Equal(t, "Grafana Aggregator", genericConfig.OpenAPIConfig.Info.Title)
	assert.Equal(t, "0.1", genericConfig.OpenAPIConfig.Info.Version)
	assert.Equal(t, "Grafana Aggregator", genericConfig.OpenAPIV3Config.Info.Title)
	assert.NotNil(t, genericConfig.MergedResourceConfig)
	assert.Empty(t, genericConfig.PostStartHooks)
}
