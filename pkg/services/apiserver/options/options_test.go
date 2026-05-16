package options

import (
	"net"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

func testCodecs() serializer.CodecFactory {
	scheme := runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	return serializer.NewCodecFactory(scheme)
}

func testCodec() runtime.Codec {
	return testCodecs().LegacyCodec(schema.GroupVersion{Version: "v1"})
}

type mockOptionsProvider struct {
	addFlagsCalled      bool
	validateCalled      bool
	validateErr         bool
	applyToCalled       bool
	applyToErr          error
}

func (m *mockOptionsProvider) AddFlags(fs *pflag.FlagSet) {
	m.addFlagsCalled = true
}

func (m *mockOptionsProvider) ApplyTo(config *genericapiserver.RecommendedConfig) error {
	m.applyToCalled = true
	return m.applyToErr
}

func (m *mockOptionsProvider) ValidateOptions() []error {
	m.validateCalled = true
	if m.validateErr {
		return []error{assert.AnError}
	}
	return nil
}

func TestNewOptions(t *testing.T) {
	codec := testCodec()
	opts := NewOptions(codec)

	require.NotNil(t, opts)
	assert.NotNil(t, opts.RecommendedOptions)
	assert.NotNil(t, opts.APIEnablementOptions)
	assert.NotNil(t, opts.GrafanaAggregatorOptions)
	assert.NotNil(t, opts.StorageOptions)
	assert.NotNil(t, opts.ExtraOptions)
	assert.Equal(t, StorageTypeUnified, opts.StorageOptions.StorageType)
}

func TestNewRecommendedOptions(t *testing.T) {
	codec := testCodec()
	ro := NewRecommendedOptions(codec)

	require.NotNil(t, ro)
	assert.Equal(t, defaultEtcdPathPrefix, ro.Etcd.StorageConfig.Prefix)
}

func TestOptions_AddFlags(t *testing.T) {
	opts := NewOptions(testCodec())
	opts.APIOptions = []OptionsProvider{&mockOptionsProvider{}}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs)

	mock := opts.APIOptions[0].(*mockOptionsProvider)
	assert.True(t, mock.addFlagsCalled)
}

func TestOptions_Validate(t *testing.T) {
	t.Run("valid unified storage defaults", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.StorageOptions.StorageType = StorageTypeUnified
		opts.StorageOptions.Address = "localhost:10000"

		assert.Empty(t, opts.Validate())
	})

	t.Run("storage validation errors propagate", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.StorageOptions.StorageType = StorageType("invalid")

		errs := opts.Validate()
		require.NotEmpty(t, errs)
	})

	t.Run("dev mode runs authentication validation", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.ExtraOptions.DevMode = true
		opts.StorageOptions.Address = "localhost:10000"
		opts.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional = true

		assert.Empty(t, opts.Validate())
	})

	t.Run("etcd storage validates etcd options", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.StorageOptions.StorageType = StorageTypeEtcd
		opts.StorageOptions.Address = "localhost:10000"
		opts.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList = []string{}

		errs := opts.Validate()
		require.NotEmpty(t, errs)
	})

	t.Run("api options validation errors propagate", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.StorageOptions.Address = "localhost:10000"
		opts.APIOptions = []OptionsProvider{&mockOptionsProvider{validateErr: true}}

		errs := opts.Validate()
		require.NotEmpty(t, errs)
		assert.True(t, opts.APIOptions[0].(*mockOptionsProvider).validateCalled)
	})

	t.Run("nil grafana aggregator options validate", func(t *testing.T) {
		opts := NewOptions(testCodec())
		opts.GrafanaAggregatorOptions = nil
		opts.StorageOptions.Address = "localhost:10000"

		assert.Empty(t, opts.Validate())
	})
}

func configureRecommendedOptionsForTest(ro *genericoptions.RecommendedOptions) {
	ro.Authentication.RemoteKubeConfigFileOptional = true
	ro.Authorization.RemoteKubeConfigFileOptional = true
}

func TestOptions_ApplyTo(t *testing.T) {
	t.Run("non-dev mode uses fake listener and sets request timeout", func(t *testing.T) {
		opts := NewOptions(testCodec())
		configureRecommendedOptionsForTest(opts.RecommendedOptions)
		opts.ExtraOptions.DevMode = false
		opts.ExtraOptions.RequestTimeout = 15 * time.Minute
		opts.ExtraOptions.ExternalAddress = "127.0.0.1:3000"
		opts.StorageOptions.Address = "localhost:10000"

		opts.RecommendedOptions.SecureServing.BindAddress = net.ParseIP("127.0.0.1")
		opts.RecommendedOptions.SecureServing.BindPort = 0

		serverConfig := genericapiserver.NewRecommendedConfig(testCodecs())
		require.NoError(t, opts.ApplyTo(serverConfig))

		assert.Equal(t, 15*time.Minute, serverConfig.RequestTimeout)
		assert.Nil(t, serverConfig.SecureServing)
		assert.NotNil(t, serverConfig.AggregatedDiscoveryGroupManager)
		assert.True(t, opts.RecommendedOptions.Authentication.SkipInClusterLookup)
	})

	t.Run("dev mode keeps secure serving", func(t *testing.T) {
		opts := NewOptions(testCodec())
		configureRecommendedOptionsForTest(opts.RecommendedOptions)
		opts.ExtraOptions.DevMode = true
		opts.StorageOptions.Address = "localhost:10000"

		opts.RecommendedOptions.SecureServing.BindAddress = net.ParseIP("127.0.0.1")
		opts.RecommendedOptions.SecureServing.BindPort = 6443

		serverConfig := genericapiserver.NewRecommendedConfig(testCodecs())
		require.NoError(t, opts.ApplyTo(serverConfig))

		assert.NotNil(t, serverConfig.SecureServing)
	})

	t.Run("zero request timeout leaves server default", func(t *testing.T) {
		opts := NewOptions(testCodec())
		configureRecommendedOptionsForTest(opts.RecommendedOptions)
		opts.ExtraOptions.RequestTimeout = 0
		opts.StorageOptions.Address = "localhost:10000"
		opts.RecommendedOptions.SecureServing.BindAddress = net.ParseIP("127.0.0.1")
		opts.RecommendedOptions.SecureServing.BindPort = 0

		serverConfig := genericapiserver.NewRecommendedConfig(testCodecs())
		defaultTimeout := serverConfig.RequestTimeout
		require.NoError(t, opts.ApplyTo(serverConfig))

		assert.Equal(t, defaultTimeout, serverConfig.RequestTimeout)
	})
}

func TestFakeListener(t *testing.T) {
	listener := newFakeListener()
	t.Cleanup(func() { _ = listener.Close() })

	addr := listener.Addr()
	require.NotNil(t, addr)
	tcpAddr, ok := addr.(*net.TCPAddr)
	require.True(t, ok)
	assert.Equal(t, 3000, tcpAddr.Port)

	serverConn, err := listener.Accept()
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverConn.Close() })

	require.NoError(t, listener.Close())
}
