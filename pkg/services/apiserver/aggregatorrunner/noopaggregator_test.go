package aggregatorrunner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
)

func TestProvideNoopAggregatorConfigurator(t *testing.T) {
	r := ProvideNoopAggregatorConfigurator()
	require.NotNil(t, r)
	_, ok := r.(*NoopAggregatorConfigurator)
	assert.True(t, ok)
}

func TestNoopAggregatorConfigurator_Configure(t *testing.T) {
	n := NoopAggregatorConfigurator{}
	server, err := n.Configure(
		&options.Options{},
		&genericapiserver.RecommendedConfig{},
		&ExtraConfig{},
		genericapiserver.NewEmptyDelegate(),
		runtime.NewScheme(),
		[]builder.APIGroupBuilder{},
	)
	assert.NoError(t, err)
	assert.Nil(t, server)
}

func TestNoopAggregatorConfigurator_ConfigureWithNilArgs(t *testing.T) {
	n := NoopAggregatorConfigurator{}
	server, err := n.Configure(nil, nil, nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, server)
}

func TestNoopAggregatorConfigurator_Run(t *testing.T) {
	n := NoopAggregatorConfigurator{}
	stoppedCh := make(chan error, 1)
	server, err := n.Run(context.Background(), nil, stoppedCh)
	assert.NoError(t, err)
	assert.Nil(t, server)
}

func TestExtraConfig_ZeroValue(t *testing.T) {
	cfg := ExtraConfig{}
	assert.Nil(t, cfg.AutoRegistrationControllerProvider)
}
