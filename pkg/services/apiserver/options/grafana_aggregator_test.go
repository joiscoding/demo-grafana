package options

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"

	"github.com/grafana/grafana/pkg/aggregator/apis/aggregation/v0alpha1"
	aggregatorapiserver "github.com/grafana/grafana/pkg/aggregator/apiserver"
	aggregatorscheme "github.com/grafana/grafana/pkg/aggregator/apiserver/scheme"
)

func TestNewGrafanaAggregatorOptions(t *testing.T) {
	o := NewGrafanaAggregatorOptions()
	require.NotNil(t, o)
}

func TestGrafanaAggregatorOptions_AddFlags(t *testing.T) {
	o := NewGrafanaAggregatorOptions()
	fs := pflag.NewFlagSet("agg", pflag.ContinueOnError)
	o.AddFlags(fs)

	// also exercise the nil-safe branch
	var nilOpts *GrafanaAggregatorOptions
	nilOpts.AddFlags(fs)
}

func TestGrafanaAggregatorOptions_Validate(t *testing.T) {
	o := NewGrafanaAggregatorOptions()
	assert.Nil(t, o.Validate())

	var nilOpts *GrafanaAggregatorOptions
	assert.Nil(t, nilOpts.Validate())
}

func TestGrafanaAggregatorOptions_ApplyTo(t *testing.T) {
	o := NewGrafanaAggregatorOptions()

	recommended := genericapiserver.NewRecommendedConfig(aggregatorscheme.Codecs)
	aggCfg := &aggregatorapiserver.Config{
		GenericConfig: recommended,
	}

	backendCfg := storagebackend.NewDefaultConfig("/registry/test", aggregatorscheme.Codecs.LegacyCodec(v0alpha1.SchemeGroupVersion))
	etcdOpts := options.NewEtcdOptions(backendCfg)
	etcdOpts.StorageConfig.Transport.ServerList = []string{"http://localhost:0"}

	err := o.ApplyTo(aggCfg, etcdOpts)
	require.NoError(t, err)

	assert.NotNil(t, recommended.OpenAPIConfig)
	assert.Equal(t, "Grafana Aggregator", recommended.OpenAPIConfig.Info.Title)
	assert.NotNil(t, recommended.OpenAPIV3Config)
	assert.Equal(t, "Grafana Aggregator", recommended.OpenAPIV3Config.Info.Title)
	assert.True(t, recommended.SkipOpenAPIInstallation)
	assert.NotNil(t, recommended.RESTOptionsGetter)
	assert.NotNil(t, recommended.MergedResourceConfig)
}
