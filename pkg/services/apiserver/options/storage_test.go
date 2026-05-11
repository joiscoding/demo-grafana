package options

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"

	apiserverrest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/setting"
)

func TestStorageOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		Opts    StorageOptions
		wantErr bool
	}{
		{
			name: "with unified storage grpc and no auth token",
			Opts: StorageOptions{
				StorageType: StorageTypeUnifiedGrpc,
			},
			wantErr: true,
		},
		{
			name: "with unified storage grpc and auth info",
			Opts: StorageOptions{
				StorageType:                              StorageTypeUnifiedGrpc,
				Address:                                  "localhost:10000",
				GrpcClientAuthenticationToken:            "1234",
				GrpcClientAuthenticationTokenExchangeURL: "http://localhost:8080",
				GrpcClientAuthenticationTokenNamespace:   "*",
			},
			wantErr: false,
		},
		{
			name: "with secrets manager grpc client and no server address",
			Opts: StorageOptions{
				StorageType:                              StorageTypeUnifiedGrpc,
				Address:                                  "localhost:10000",
				GrpcClientAuthenticationToken:            "1234",
				GrpcClientAuthenticationTokenExchangeURL: "http://localhost:8080",
				GrpcClientAuthenticationTokenNamespace:   "*",
				SecretsManagerGrpcClientEnable:           true,
			},
			wantErr: true,
		},
		{
			name: "with secrets manager grpc client and no server ca file",
			Opts: StorageOptions{
				StorageType:                              StorageTypeUnifiedGrpc,
				Address:                                  "localhost:10000",
				GrpcClientAuthenticationToken:            "1234",
				GrpcClientAuthenticationTokenExchangeURL: "http://localhost:8080",
				GrpcClientAuthenticationTokenNamespace:   "*",
				SecretsManagerGrpcClientEnable:           true,
				SecretsManagerGrpcServerAddress:          "localhost:10000",
				SecretsManagerGrpcServerUseTLS:           true,
			},
			wantErr: true,
		},
		{
			name: "with secrets manager grpc client and server ca file",
			Opts: StorageOptions{
				StorageType:                              StorageTypeUnifiedGrpc,
				Address:                                  "localhost:10000",
				GrpcClientAuthenticationToken:            "1234",
				GrpcClientAuthenticationTokenExchangeURL: "http://localhost:8080",
				GrpcClientAuthenticationTokenNamespace:   "*",
				SecretsManagerGrpcClientEnable:           true,
				SecretsManagerGrpcServerAddress:          "localhost:10000",
				SecretsManagerGrpcServerUseTLS:           true,
				SecretsManagerGrpcServerTLSCAFile:        "ca.crt",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.Opts.Validate()
			if tt.wantErr {
				assert.NotEmpty(t, errs)
				return
			}
			assert.Empty(t, errs)
		})
	}
}

func TestStorageOptions_Validate_AllTypes(t *testing.T) {
	for _, st := range []StorageType{
		StorageTypeFile,
		StorageTypeEtcd,
		StorageTypeUnified,
		StorageTypeUnifiedGrpc,
		StorageTypeUnifiedKVGrpc,
		StorageTypeLegacy,
	} {
		o := NewStorageOptions()
		o.StorageType = st
		assert.Empty(t, o.Validate(), "storage type %s should validate", st)
	}
}

func TestStorageOptions_Validate_BadAddress(t *testing.T) {
	o := NewStorageOptions()
	o.Address = "not-a-valid-host-port"
	errs := o.Validate()
	assert.NotEmpty(t, errs)
}

func TestStorageOptions_Validate_BlobStoreOnlyUnified(t *testing.T) {
	o := NewStorageOptions()
	o.StorageType = StorageTypeFile
	o.BlobStoreURL = "file:///tmp/blob"
	errs := o.Validate()
	assert.NotEmpty(t, errs)
}

func TestNewStorageOptions_Defaults(t *testing.T) {
	o := NewStorageOptions()
	assert.Equal(t, StorageTypeUnified, o.StorageType)
	assert.Equal(t, "localhost:10000", o.Address)
	assert.Equal(t, "*", o.GrpcClientAuthenticationTokenNamespace)
	assert.False(t, o.GrpcClientAuthenticationAllowInsecure)
	assert.Equal(t, BlobThresholdDefault, o.BlobThresholdBytes)
	assert.NotNil(t, o.UnifiedStorageConfig)
}

func TestStorageOptions_AddFlags(t *testing.T) {
	o := NewStorageOptions()
	fs := pflag.NewFlagSet("st", pflag.ContinueOnError)
	o.AddFlags(fs)

	require.NoError(t, fs.Parse([]string{
		"--grafana-apiserver-storage-type=unified-grpc",
		"--grafana-apiserver-storage-path=/tmp/data",
		"--grafana-apiserver-storage-address=localhost:1234",
		"--grafana-apiserver-search-address=localhost:1235",
		"--grpc-client-authentication-token=tok",
		"--grpc-client-authentication-token-exchange-url=http://x",
		"--grpc-client-authentication-token-namespace=ns",
		"--grpc-client-authentication-allow-insecure=true",
		"--grpc-client-keepalive-time=30s",
		"--grafana-apiserver-unified-storage-config=dashboards.grafana.app=2,folders.grafana.app=1",
		"--grafana.secrets-manager.grpc-client-enable=true",
		"--grafana.secrets-manager.grpc-server-address=localhost:9999",
		"--grafana.secrets-manager.grpc-server-use-tls=true",
		"--grafana.secrets-manager.grpc-server-tls-skip-verify=true",
		"--grafana.secrets-manager.grpc-server-tls-server-name=server",
		"--grafana.secrets-manager.grpc-server-tls-ca-file=/tmp/ca",
		"--grafana.secrets-manager.grpc-client-load-balancing=true",
	}))

	assert.Equal(t, StorageTypeUnifiedGrpc, o.StorageType)
	assert.Equal(t, "/tmp/data", o.DataPath)
	assert.Equal(t, "localhost:1234", o.Address)
	assert.Equal(t, "localhost:1235", o.SearchServerAddress)
	assert.Equal(t, "tok", o.GrpcClientAuthenticationToken)
	assert.Equal(t, "http://x", o.GrpcClientAuthenticationTokenExchangeURL)
	assert.Equal(t, "ns", o.GrpcClientAuthenticationTokenNamespace)
	assert.True(t, o.GrpcClientAuthenticationAllowInsecure)
	assert.Equal(t, 30*time.Second, o.GrpcClientKeepaliveTime)
	require.Contains(t, o.UnifiedStorageConfig, "dashboards.grafana.app")
	assert.Equal(t, apiserverrest.DualWriterMode(2), o.UnifiedStorageConfig["dashboards.grafana.app"].DualWriterMode)
	assert.Equal(t, apiserverrest.DualWriterMode(1), o.UnifiedStorageConfig["folders.grafana.app"].DualWriterMode)
	assert.True(t, o.SecretsManagerGrpcClientEnable)
	assert.Equal(t, "localhost:9999", o.SecretsManagerGrpcServerAddress)
	assert.True(t, o.SecretsManagerGrpcServerUseTLS)
	assert.True(t, o.SecretsManagerGrpcServerTLSSkipVerify)
	assert.Equal(t, "server", o.SecretsManagerGrpcServerTLSServerName)
	assert.Equal(t, "/tmp/ca", o.SecretsManagerGrpcServerTLSCAFile)
	assert.True(t, o.SecretsManagerGrpcClientLoadBalancing)
}

func TestUnifiedStorageConfigValue(t *testing.T) {
	cfg := map[string]setting.UnifiedStorageConfig{}
	v := &unifiedStorageConfigValue{config: &cfg}

	assert.Equal(t, "stringToUnifiedStorageConfig", v.Type())
	assert.Equal(t, "", v.String())

	// empty value is a no-op
	require.NoError(t, v.Set(""))
	assert.Empty(t, cfg)

	require.NoError(t, v.Set("res1.grp=1,res2.grp=3"))
	assert.Equal(t, apiserverrest.DualWriterMode(1), cfg["res1.grp"].DualWriterMode)
	assert.Equal(t, apiserverrest.DualWriterMode(3), cfg["res2.grp"].DualWriterMode)

	// String returns a non-empty serialization
	assert.NotEmpty(t, v.String())

	// invalid format
	assert.Error(t, v.Set("badpair"))
	// non-integer mode
	assert.Error(t, v.Set("res.grp=notanumber"))
	// out-of-range mode
	assert.Error(t, v.Set("res.grp=99"))
}

func newTestEtcdOptions() *genericoptions.EtcdOptions {
	return genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig("/registry/test", nil))
}

func TestStorageOptions_ApplyTo_NoopForNonGrpc(t *testing.T) {
	o := NewStorageOptions()
	o.StorageType = StorageTypeUnified
	cfg := &genericapiserver.RecommendedConfig{}
	err := o.ApplyTo(cfg, newTestEtcdOptions(), tracing.NewNoopTracerService(), genericoptions.NewSecureServingOptions())
	require.NoError(t, err)
	assert.Nil(t, cfg.RESTOptionsGetter)
}

func TestStorageOptions_ApplyTo_UnifiedGrpc(t *testing.T) {
	o := NewStorageOptions()
	o.StorageType = StorageTypeUnifiedGrpc
	o.Address = "localhost:10000"
	o.SearchServerAddress = "localhost:10001"
	o.GrpcClientKeepaliveTime = 5 * time.Second
	o.GrpcClientAuthenticationToken = "tok"
	o.GrpcClientAuthenticationTokenExchangeURL = "http://exchange"
	o.GrpcClientAuthenticationTokenNamespace = "*"
	cfg := &genericapiserver.RecommendedConfig{}

	err := o.ApplyTo(cfg, newTestEtcdOptions(), tracing.NewNoopTracerService(), genericoptions.NewSecureServingOptions())
	require.NoError(t, err)
	assert.NotNil(t, cfg.RESTOptionsGetter)
}

func TestStorageOptions_ApplyTo_UnifiedGrpcWithSecretsManager(t *testing.T) {
	o := NewStorageOptions()
	o.StorageType = StorageTypeUnifiedGrpc
	o.Address = "localhost:10000"
	o.GrpcClientAuthenticationToken = "tok"
	o.GrpcClientAuthenticationTokenExchangeURL = "http://exchange"
	o.GrpcClientAuthenticationTokenNamespace = "*"
	o.SecretsManagerGrpcClientEnable = true
	o.SecretsManagerGrpcServerAddress = "localhost:9999"

	cfg := &genericapiserver.RecommendedConfig{}
	require.NoError(t, o.ApplyTo(cfg, newTestEtcdOptions(), tracing.NewNoopTracerService(), genericoptions.NewSecureServingOptions()))
	assert.NotNil(t, o.InlineSecrets)
}

func TestStorageOptions_ApplyTo_UnifiedGrpcWithSecretsManagerError(t *testing.T) {
	o := NewStorageOptions()
	o.StorageType = StorageTypeUnifiedGrpc
	o.Address = "localhost:10000"
	o.GrpcClientAuthenticationToken = "tok"
	o.GrpcClientAuthenticationTokenExchangeURL = "http://exchange"
	o.GrpcClientAuthenticationTokenNamespace = "*"
	o.SecretsManagerGrpcClientEnable = true
	// missing SecretsManagerGrpcServerAddress -> inline secure value service errors
	cfg := &genericapiserver.RecommendedConfig{}
	err := o.ApplyTo(cfg, newTestEtcdOptions(), tracing.NewNoopTracerService(), genericoptions.NewSecureServingOptions())
	require.Error(t, err)
}

func TestStorageOptions_BuildGrpcDialOptions(t *testing.T) {
	o := NewStorageOptions()
	opts := o.buildGrpcDialOptions()
	assert.NotEmpty(t, opts)

	o.GrpcClientKeepaliveTime = 30 * time.Second
	opts2 := o.buildGrpcDialOptions()
	assert.Greater(t, len(opts2), len(opts))
}
