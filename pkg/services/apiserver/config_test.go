package apiserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"github.com/grafana/grafana/pkg/aggregator/apis/aggregation/v0alpha1"
	aggregatorscheme "github.com/grafana/grafana/pkg/aggregator/apiserver/scheme"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
)

func newTestOptions() *options.Options {
	return options.NewOptions(aggregatorscheme.Codecs.LegacyCodec(v0alpha1.SchemeGroupVersion))
}

func newTestCfgFromINI(t *testing.T, body string) *setting.Cfg {
	t.Helper()
	cfg := setting.NewCfg()
	cfg.HTTPAddr = "127.0.0.1"
	cfg.HTTPPort = "3000"
	cfg.AppURL = "http://localhost:3000/"
	cfg.Env = setting.Prod
	cfg.DataPath = t.TempDir()
	if body != "" {
		f, err := ini.Load([]byte(body))
		require.NoError(t, err)
		cfg.Raw = f
	}
	return cfg
}

func TestApplyGrafanaConfig_InvalidIP(t *testing.T) {
	cfg := setting.NewCfg()
	cfg.HTTPAddr = "not-an-ip"
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid IP")
}

func TestApplyGrafanaConfig_DefaultsAndProd(t *testing.T) {
	cfg := newTestCfgFromINI(t, "")
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)

	require.Equal(t, 3000, o.RecommendedOptions.SecureServing.BindPort)
	require.Equal(t, "127.0.0.1", o.RecommendedOptions.SecureServing.BindAddress.String())
	require.Equal(t, "127.0.0.1:3000", o.ExtraOptions.ExternalAddress)
	require.Equal(t, "http://localhost:3000/", o.ExtraOptions.APIURL)
	require.True(t, o.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional)
	require.True(t, o.RecommendedOptions.Authorization.RemoteKubeConfigFileOptional)
	require.Nil(t, o.RecommendedOptions.Admission)
	require.Nil(t, o.RecommendedOptions.CoreAPI)
	// default storage type
	require.Equal(t, options.StorageTypeUnified, o.StorageOptions.StorageType)
	require.Contains(t, o.StorageOptions.DataPath, "grafana-apiserver")
	require.False(t, o.ExtraOptions.DevMode)
	require.Equal(t, 0, o.ExtraOptions.Verbosity)
}

func TestApplyGrafanaConfig_InvalidPortFallsBackTo3000(t *testing.T) {
	cfg := newTestCfgFromINI(t, "")
	cfg.HTTPPort = "not-a-port"
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
	require.Equal(t, 3000, o.RecommendedOptions.SecureServing.BindPort)
}

func TestApplyGrafanaConfig_DevModeOverridesAddressAndPort(t *testing.T) {
	cfg := newTestCfgFromINI(t, "")
	cfg.Env = setting.Dev
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
	require.Equal(t, 6443, o.RecommendedOptions.SecureServing.BindPort)
	require.Equal(t, "0.0.0.0", o.RecommendedOptions.SecureServing.BindAddress.String())
	require.Equal(t, "https://0.0.0.0:6443", o.ExtraOptions.APIURL)
}

func TestApplyGrafanaConfig_DebugLogIncreasesVerbosity(t *testing.T) {
	body := `
[log]
level = debug
`
	cfg := newTestCfgFromINI(t, body)
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
	require.Equal(t, 7, o.ExtraOptions.Verbosity)
}

func TestApplyGrafanaConfig_ExplicitLogLevelWins(t *testing.T) {
	body := `
[log]
level = debug
[grafana-apiserver]
log_level = 4
`
	cfg := newTestCfgFromINI(t, body)
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
	require.Equal(t, 4, o.ExtraOptions.Verbosity)
}

func TestApplyGrafanaConfig_RuntimeConfigSuccess(t *testing.T) {
	body := `
[grafana-apiserver]
runtime_config = api/all=true
`
	cfg := newTestCfgFromINI(t, body)
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
}

func TestApplyGrafanaConfig_RequestTimeoutAndStorageAndEtcd(t *testing.T) {
	body := `
[grafana-apiserver]
request_timeout = 30s
etcd_servers = http://etcd-a:2379,http://etcd-b:2379
storage_type = etcd
storage_path = /custom/path
address = mem://test
blob_url = file:///tmp/blobs
blob_threshold_bytes = 12345
`
	cfg := newTestCfgFromINI(t, body)
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)

	require.Equal(t, 30*time.Second, o.ExtraOptions.RequestTimeout)
	require.Equal(t, []string{"http://etcd-a:2379", "http://etcd-b:2379"}, o.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList)
	require.Equal(t, options.StorageType("etcd"), o.StorageOptions.StorageType)
	require.Equal(t, "/custom/path", o.StorageOptions.DataPath)
	require.Equal(t, "mem://test", o.StorageOptions.Address)
	require.Equal(t, "file:///tmp/blobs", o.StorageOptions.BlobStoreURL)
	require.Equal(t, 12345, o.StorageOptions.BlobThresholdBytes)
}

func TestApplyGrafanaConfig_UnifiedStorageConfigPassed(t *testing.T) {
	cfg := newTestCfgFromINI(t, "")
	cfg.UnifiedStorage = map[string]setting.UnifiedStorageConfig{
		"folders.folder.grafana.app": {DualWriterMode: 2},
	}
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
	require.NoError(t, err)
	require.Equal(t, cfg.UnifiedStorage, o.StorageOptions.UnifiedStorageConfig)
}

func TestApplyGrafanaConfig_DevModeFeatureFlagEnablesKubectlAccess(t *testing.T) {
	cfg := newTestCfgFromINI(t, "")
	o := newTestOptions()
	err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerEnsureKubectlAccess), o)
	require.NoError(t, err)
	require.True(t, o.ExtraOptions.DevMode)
}
