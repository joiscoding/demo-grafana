package options

import (
	"errors"
	"net"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

func newTestCodecs() serializer.CodecFactory {
	return serializer.NewCodecFactory(runtime.NewScheme())
}

// fakeAPIOptions implements OptionsProvider for testing.
type fakeAPIOptions struct {
	addFlagsCalled bool
	validateErrs   []error
}

func (f *fakeAPIOptions) AddFlags(fs *pflag.FlagSet) {
	f.addFlagsCalled = true
}

func (f *fakeAPIOptions) ApplyTo(config *genericapiserver.RecommendedConfig) error {
	return nil
}

func (f *fakeAPIOptions) ValidateOptions() []error {
	return f.validateErrs
}

func TestNewOptions(t *testing.T) {
	o := NewOptions(nil)
	require.NotNil(t, o)
	assert.NotNil(t, o.RecommendedOptions)
	assert.NotNil(t, o.APIEnablementOptions)
	assert.NotNil(t, o.GrafanaAggregatorOptions)
	assert.NotNil(t, o.StorageOptions)
	assert.NotNil(t, o.ExtraOptions)
}

func TestOptions_AddFlags(t *testing.T) {
	o := NewOptions(nil)
	fake := &fakeAPIOptions{}
	o.APIOptions = []OptionsProvider{fake}

	fs := pflag.NewFlagSet("opts", pflag.ContinueOnError)
	o.AddFlags(fs)

	assert.True(t, fake.addFlagsCalled)
	// well-known flags should be registered
	assert.NotNil(t, fs.Lookup("grafana-apiserver-storage-type"))
	assert.NotNil(t, fs.Lookup("grafana-apiserver-dev-mode"))
	assert.NotNil(t, fs.Lookup("verbosity"))
}

func TestOptions_Validate_Default(t *testing.T) {
	o := NewOptions(nil)
	// default StorageType is unified; address is valid; secure serving should pass with defaults
	errs := o.Validate()
	assert.Empty(t, errs)
}

func TestOptions_Validate_BadStorage(t *testing.T) {
	o := NewOptions(nil)
	o.StorageOptions.StorageType = "bogus"
	errs := o.Validate()
	assert.NotEmpty(t, errs)
}

func TestOptions_Validate_ExtraOptionsErrPath(t *testing.T) {
	// ExtraOptions.Validate always returns nil today, so we exercise the
	// other early-return branches by injecting an APIOption that fails.
	o := NewOptions(nil)
	o.APIOptions = []OptionsProvider{&fakeAPIOptions{
		validateErrs: []error{errors.New("boom")},
	}}
	errs := o.Validate()
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "boom")
}

func TestOptions_Validate_DevModeAuthn(t *testing.T) {
	o := NewOptions(nil)
	o.ExtraOptions.DevMode = true
	errs := o.Validate()
	// authn validation with default options should succeed
	assert.Empty(t, errs)
}

func TestOptions_Validate_EtcdInvalid(t *testing.T) {
	o := NewOptions(nil)
	o.StorageOptions.StorageType = StorageTypeEtcd
	errs := o.Validate()
	assert.NotEmpty(t, errs)
}

func TestOptions_Validate_AggregatorErr(t *testing.T) {
	// nil GrafanaAggregatorOptions short-circuits to nil errs but exercises the if-branch.
	o := NewOptions(nil)
	o.GrafanaAggregatorOptions = nil
	assert.Empty(t, o.Validate())
}

func TestOptions_Validate_SecureServingInvalid(t *testing.T) {
	o := NewOptions(nil)
	o.RecommendedOptions.SecureServing.BindPort = -1
	errs := o.Validate()
	assert.NotEmpty(t, errs)
}

func TestNewRecommendedOptions(t *testing.T) {
	r := NewRecommendedOptions(nil)
	require.NotNil(t, r)
	assert.NotNil(t, r.SecureServing)
}

func prepareOptionsForApplyTo(t *testing.T, o *Options) {
	t.Helper()
	dir := t.TempDir()
	o.RecommendedOptions.SecureServing.ServerCert.CertDirectory = dir
	o.RecommendedOptions.SecureServing.ServerCert.PairName = "test"
	o.RecommendedOptions.SecureServing.BindAddress = net.IPv4(127, 0, 0, 1)
	o.RecommendedOptions.SecureServing.BindPort = 0
	o.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional = true
	require.NoError(t, o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.IPv4(127, 0, 0, 1)}))
}

func TestOptions_ApplyTo_NonDevMode(t *testing.T) {
	o := NewOptions(nil)
	prepareOptionsForApplyTo(t, o)

	cfg := genericapiserver.NewRecommendedConfig(newTestCodecs())
	require.NoError(t, o.ApplyTo(cfg))
	assert.Nil(t, cfg.SecureServing)
	assert.NotNil(t, cfg.AggregatedDiscoveryGroupManager)
	assert.True(t, o.RecommendedOptions.Authentication.SkipInClusterLookup)
}

func TestOptions_ApplyTo_DevMode(t *testing.T) {
	o := NewOptions(nil)
	o.ExtraOptions.DevMode = true
	o.ExtraOptions.RequestTimeout = 0
	prepareOptionsForApplyTo(t, o)

	cfg := genericapiserver.NewRecommendedConfig(newTestCodecs())
	require.NoError(t, o.ApplyTo(cfg))
	if cfg.SecureServing != nil && cfg.SecureServing.Listener != nil {
		_ = cfg.SecureServing.Listener.Close()
	}
}

func TestFakeListener(t *testing.T) {
	l := newFakeListener()
	require.NotNil(t, l)

	addr := l.Addr()
	tcpAddr, ok := addr.(*net.TCPAddr)
	require.True(t, ok)
	assert.Equal(t, 3000, tcpAddr.Port)
	assert.True(t, tcpAddr.IP.Equal(net.IPv4(127, 0, 0, 1)))

	// Accept returns the server side of the pipe immediately.
	conn, err := l.Accept()
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.NoError(t, l.Close())
}
