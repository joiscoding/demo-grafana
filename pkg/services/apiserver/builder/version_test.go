package builder_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/component-base/compatibility"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestGetEffectiveVersion(t *testing.T) {
	t.Parallel()

	const (
		buildTimestamp = int64(1700000000)
		buildVersion   = "11.2.0"
		buildCommit    = "abc123"
		buildBranch    = "main"
	)

	ev := builder.GetEffectiveVersion(buildTimestamp, buildVersion, buildCommit, buildBranch)
	require.NotNil(t, ev)

	var _ compatibility.EffectiveVersion = ev

	info := ev.Info()
	require.NotNil(t, info)
	require.Contains(t, info.GitVersion, "grafana-v"+buildVersion)
	require.Equal(t, buildBranch+"@"+buildCommit, info.GitCommit)
	require.Equal(t, "grafana v"+buildVersion, info.GitTreeState)

	expectedDate := time.Unix(buildTimestamp, 0).UTC().Format(time.RFC3339)
	require.Equal(t, expectedDate, info.BuildDate)

	require.NotEmpty(t, ev.AllowedEmulationVersionRange())
	require.NotEmpty(t, ev.AllowedMinCompatibilityVersionRange())
	require.NotNil(t, ev.BinaryVersion())
	require.NotNil(t, ev.EmulationVersion())
	require.NotNil(t, ev.MinCompatibilityVersion())
	require.NotEmpty(t, ev.String())
	require.Empty(t, ev.Validate())
	require.True(t, ev.EqualTo(ev))
}
