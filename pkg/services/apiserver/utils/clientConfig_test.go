package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"
)

func TestFormatKubeConfig(t *testing.T) {
	rest := &clientrest.Config{
		Host:        "https://example.test:6443",
		BearerToken: "secret-token",
	}

	cfg := FormatKubeConfig(rest)

	require.Equal(t, "Config", cfg.Kind)
	require.Equal(t, "v1", cfg.APIVersion)
	require.Equal(t, "default-context", cfg.CurrentContext)

	require.Contains(t, cfg.Clusters, "default-cluster")
	cluster := cfg.Clusters["default-cluster"]
	require.Equal(t, "https://example.test:6443", cluster.Server)
	require.True(t, cluster.InsecureSkipTLSVerify)

	require.Contains(t, cfg.Contexts, "default-context")
	c := cfg.Contexts["default-context"]
	require.Equal(t, "default-cluster", c.Cluster)
	require.Equal(t, "default", c.Namespace)
	require.Equal(t, "default", c.AuthInfo)

	require.Contains(t, cfg.AuthInfos, "default")
	require.Equal(t, "secret-token", cfg.AuthInfos["default"].Token)
}

func TestFormatKubeConfig_Empty(t *testing.T) {
	cfg := FormatKubeConfig(&clientrest.Config{})
	require.Empty(t, cfg.Clusters["default-cluster"].Server)
	require.Empty(t, cfg.AuthInfos["default"].Token)
	require.True(t, cfg.Clusters["default-cluster"].InsecureSkipTLSVerify)
}
