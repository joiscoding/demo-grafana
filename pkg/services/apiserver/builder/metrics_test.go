package builder_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestProvideBuilderMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := builder.ProvideBuilderMetrics(reg)
	require.NotNil(t, m)

	m.RecordDualWriterModes("dashboards", "dashboard.grafana.app", grafanarest.DualWriterMode(2))

	expected := `
# HELP unified_storage_dual_writer_current_mode Unified storage dual writer current mode
# TYPE unified_storage_dual_writer_current_mode gauge
unified_storage_dual_writer_current_mode{group="dashboard.grafana.app",resource="dashboards"} 2
# HELP unified_storage_dual_writer_target_mode Unified Storage dual writer target mode
# TYPE unified_storage_dual_writer_target_mode gauge
unified_storage_dual_writer_target_mode{group="dashboard.grafana.app",resource="dashboards"} 2
`
	err := testutil.GatherAndCompare(reg, strings.NewReader(expected),
		"unified_storage_dual_writer_current_mode",
		"unified_storage_dual_writer_target_mode",
	)
	require.NoError(t, err)
}

func TestProvideBuilderMetrics_NilRegistererIsAllowed(t *testing.T) {
	m := builder.ProvideBuilderMetrics(nil)
	require.NotNil(t, m)
	require.NotPanics(t, func() {
		m.RecordDualWriterModes("foo", "g", grafanarest.DualWriterMode(1))
	})
}
