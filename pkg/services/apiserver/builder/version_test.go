package builder_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestGetEffectiveVersion(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).Unix()
	v := builder.GetEffectiveVersion(ts, "10.1.0", "abc123", "main")
	require.NotNil(t, v)

	info := v.Info()
	require.NotNil(t, info)
	require.Equal(t, "2024-01-02T03:04:05Z", info.BuildDate)
	require.True(t, strings.Contains(info.GitVersion, "+grafana-v10.1.0"), "got %q", info.GitVersion)
	require.Equal(t, "main@abc123", info.GitCommit)
	require.Equal(t, "grafana v10.1.0", info.GitTreeState)

	require.NotNil(t, v.BinaryVersion())
	require.NotNil(t, v.EmulationVersion())
	require.NotNil(t, v.MinCompatibilityVersion())

	require.NotPanics(t, func() { _ = v.AllowedEmulationVersionRange() })
	require.NotPanics(t, func() { _ = v.AllowedMinCompatibilityVersionRange() })

	require.NotEmpty(t, v.String())

	require.True(t, v.EqualTo(v))

	other := builder.GetEffectiveVersion(ts, "10.1.0", "abc123", "main")
	require.NotNil(t, other)

	errs := v.Validate()
	_ = errs
}
