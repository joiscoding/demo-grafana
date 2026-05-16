package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"
)

func TestFormatKubeConfig(t *testing.T) {
	restConfig := &clientrest.Config{
		Host:        "https://grafana.example:6443",
		BearerToken: "test-bearer-token",
	}

	cfg := FormatKubeConfig(restConfig)

	require.Equal(t, "Config", cfg.Kind)
	require.Equal(t, "v1", cfg.APIVersion)
	require.Equal(t, "default-context", cfg.CurrentContext)

	require.Len(t, cfg.Clusters, 1)
	cluster, ok := cfg.Clusters["default-cluster"]
	require.True(t, ok)
	require.Equal(t, restConfig.Host, cluster.Server)
	require.True(t, cluster.InsecureSkipTLSVerify)

	require.Len(t, cfg.Contexts, 1)
	ctx, ok := cfg.Contexts["default-context"]
	require.True(t, ok)
	require.Equal(t, "default-cluster", ctx.Cluster)
	require.Equal(t, "default", ctx.Namespace)
	require.Equal(t, "default", ctx.AuthInfo)

	require.Len(t, cfg.AuthInfos, 1)
	auth, ok := cfg.AuthInfos["default"]
	require.True(t, ok)
	require.Equal(t, restConfig.BearerToken, auth.Token)
}
