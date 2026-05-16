package apiserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
)

func testOptions(t *testing.T) *options.Options {
	t.Helper()
	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	gvs := []schema.GroupVersion{{Group: "", Version: "v1"}}
	return options.NewOptions(codecs.LegacyCodec(gvs...))
}

func Test_applyGrafanaConfig(t *testing.T) {
	t.Run("production defaults", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.AppURL = "http://localhost:3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod

		o := testOptions(t)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)

		assert.Equal(t, "127.0.0.1", o.RecommendedOptions.SecureServing.BindAddress.String())
		assert.Equal(t, 3000, o.RecommendedOptions.SecureServing.BindPort)
		assert.Equal(t, "127.0.0.1:3000", o.ExtraOptions.ExternalAddress)
		assert.Equal(t, cfg.AppURL, o.ExtraOptions.APIURL)
		assert.False(t, o.ExtraOptions.DevMode)
		assert.Equal(t, 0, o.ExtraOptions.Verbosity)
		assert.Equal(t, options.StorageTypeUnified, o.StorageOptions.StorageType)
	})

	t.Run("development overrides bind address and port", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.AppURL = "http://localhost:3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Dev

		o := testOptions(t)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)

		assert.Equal(t, "0.0.0.0", o.RecommendedOptions.SecureServing.BindAddress.String())
		assert.Equal(t, 6443, o.RecommendedOptions.SecureServing.BindPort)
		assert.Equal(t, "https://0.0.0.0:6443", o.ExtraOptions.APIURL)
	})

	t.Run("invalid HTTP address", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "not-an-ip"
		cfg.HTTPPort = "3000"
		cfg.Env = setting.Prod

		o := testOptions(t)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP address")
	})

	t.Run("invalid HTTP port uses default", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "not-a-port"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod

		o := testOptions(t)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)
		assert.Equal(t, 3000, o.RecommendedOptions.SecureServing.BindPort)
	})

	t.Run("debug log level increases verbosity", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod
		_, err := cfg.Raw.NewSection("log")
		require.NoError(t, err)
		cfg.Raw.Section("log").Key("level").SetValue("debug")

		o := testOptions(t)
		err = applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)
		assert.Equal(t, 7, o.ExtraOptions.Verbosity)
	})

	t.Run("grafana-apiserver section options", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod

		sec, err := cfg.Raw.NewSection("grafana-apiserver")
		require.NoError(t, err)
		sec.Key("storage_type").SetValue(string(options.StorageTypeEtcd))
		sec.Key("log_level").SetValue("3")
		sec.Key("request_timeout").SetValue("30s")
		sec.Key("etcd_servers").SetValue("http://localhost:2379,http://localhost:2380")

		o := testOptions(t)
		err = applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)

		assert.Equal(t, options.StorageTypeEtcd, o.StorageOptions.StorageType)
		assert.Equal(t, 3, o.ExtraOptions.Verbosity)
		assert.Equal(t, 30*time.Second, o.ExtraOptions.RequestTimeout)
		assert.Equal(t, []string{"http://localhost:2379", "http://localhost:2380"}, o.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList)
	})

	t.Run("runtime config from settings", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod

		sec, err := cfg.Raw.NewSection("grafana-apiserver")
		require.NoError(t, err)
		sec.Key("runtime_config").SetValue("foo=bar,enabled")

		o := testOptions(t)
		err = applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), o)
		require.NoError(t, err)
		assert.Equal(t, "bar", o.APIEnablementOptions.RuntimeConfig["foo"])
		assert.Equal(t, "", o.APIEnablementOptions.RuntimeConfig["enabled"])
	})

	t.Run("kubectl access feature enables dev mode", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "3000"
		cfg.DataPath = t.TempDir()
		cfg.Env = setting.Prod

		o := testOptions(t)
		features := featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerEnsureKubectlAccess)
		err := applyGrafanaConfig(cfg, features, o)
		require.NoError(t, err)
		assert.True(t, o.ExtraOptions.DevMode)
	})
}
