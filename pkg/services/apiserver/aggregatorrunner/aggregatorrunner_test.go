package aggregatorrunner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/kube-aggregator/pkg/controllers/autoregister"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
)

var (
	_ AggregatorRunner             = (*NoopAggregatorConfigurator)(nil)
	_ AutoRegistrationController   = (*fakeAutoRegistrationController)(nil)
	_ autoregister.AutoAPIServiceRegistration = (*fakeAutoAPIServiceRegistration)(nil)
)

func TestProvideNoopAggregatorConfigurator(t *testing.T) {
	runner := ProvideNoopAggregatorConfigurator()
	require.NotNil(t, runner)

	_, ok := runner.(*NoopAggregatorConfigurator)
	require.True(t, ok, "expected ProvideNoopAggregatorConfigurator to return *NoopAggregatorConfigurator")
}

func TestNoopAggregatorConfigurator_Configure(t *testing.T) {
	t.Parallel()

	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	config := genericapiserver.NewRecommendedConfig(codecs)
	opts := options.NewOptions(codecs.LegacyCodec())
	noop := &NoopAggregatorConfigurator{}

	server, err := noop.Configure(
		opts,
		config,
		&ExtraConfig{},
		genericapiserver.NewEmptyDelegate(),
		scheme,
		[]builder.APIGroupBuilder{},
	)
	require.NoError(t, err)
	assert.Nil(t, server)
}

func TestNoopAggregatorConfigurator_Configure_withExtraConfigProvider(t *testing.T) {
	t.Parallel()

	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	config := genericapiserver.NewRecommendedConfig(codecs)
	opts := options.NewOptions(codecs.LegacyCodec())
	noop := &NoopAggregatorConfigurator{}

	fakeController := &fakeAutoRegistrationController{}
	extra := &ExtraConfig{
		AutoRegistrationControllerProvider: func(autoregister.AutoAPIServiceRegistration) AutoRegistrationController {
			return fakeController
		},
	}

	server, err := noop.Configure(
		opts,
		config,
		extra,
		genericapiserver.NewEmptyDelegate(),
		scheme,
		nil,
	)
	require.NoError(t, err)
	assert.Nil(t, server)
}

func TestNoopAggregatorConfigurator_Run(t *testing.T) {
	t.Parallel()

	noop := &NoopAggregatorConfigurator{}
	ctx := context.Background()
	transport := &options.RoundTripperFunc{Ready: make(chan struct{})}
	stoppedCh := make(chan error, 1)

	server, err := noop.Run(ctx, transport, stoppedCh)
	require.NoError(t, err)
	assert.Nil(t, server)
}

func TestExtraConfig_AutoRegistrationControllerProvider(t *testing.T) {
	t.Parallel()

	fakeController := &fakeAutoRegistrationController{}
	provider := func(reg autoregister.AutoAPIServiceRegistration) AutoRegistrationController {
		require.NotNil(t, reg)
		return fakeController
	}

	extra := &ExtraConfig{AutoRegistrationControllerProvider: provider}
	require.NotNil(t, extra.AutoRegistrationControllerProvider)

	controller := extra.AutoRegistrationControllerProvider(&fakeAutoAPIServiceRegistration{})
	require.Same(t, fakeController, controller)

	controller.Run(1, nil)
	controller.WaitForInitialSync()

	assert.True(t, fakeController.runCalled)
	assert.Equal(t, 1, fakeController.workers)
	assert.True(t, fakeController.waitCalled)
}

func TestAggregatorRunner_interfaceViaNoop(t *testing.T) {
	t.Parallel()

	var runner AggregatorRunner = ProvideNoopAggregatorConfigurator()
	require.NotNil(t, runner)

	scheme := runtime.NewScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	config := genericapiserver.NewRecommendedConfig(codecs)

	_, err := runner.Configure(
		options.NewOptions(codecs.LegacyCodec()),
		config,
		&ExtraConfig{},
		genericapiserver.NewEmptyDelegate(),
		scheme,
		nil,
	)
	require.NoError(t, err)

	_, err = runner.Run(context.Background(), &options.RoundTripperFunc{}, make(chan error))
	require.NoError(t, err)
}

type fakeAutoRegistrationController struct {
	runCalled  bool
	waitCalled bool
	workers    int
}

func (f *fakeAutoRegistrationController) Run(workers int, _ <-chan struct{}) {
	f.runCalled = true
	f.workers = workers
}

func (f *fakeAutoRegistrationController) WaitForInitialSync() {
	f.waitCalled = true
}

type fakeAutoAPIServiceRegistration struct{}

func (f *fakeAutoAPIServiceRegistration) AddAPIServiceToSyncOnStart(_ *v1.APIService) {}
func (f *fakeAutoAPIServiceRegistration) AddAPIServiceToSync(_ *v1.APIService)       {}
func (f *fakeAutoAPIServiceRegistration) RemoveAPIServiceToSync(_ string)            {}
