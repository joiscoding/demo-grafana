package builder_test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestProvideBuilderMetrics_RecordDualWriterModes(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	metrics := builder.ProvideBuilderMetrics(reg)
	require.NotNil(t, metrics)

	metrics.RecordDualWriterModes("foos", "example.grafana.app", grafanarest.Mode3)

	targetValue := float64(grafanarest.Mode3)
	require.Equal(t, targetValue, metricValue(t, reg, "unified_storage_dual_writer_target_mode", "foos", "example.grafana.app"))
	require.Equal(t, targetValue, metricValue(t, reg, "unified_storage_dual_writer_current_mode", "foos", "example.grafana.app"))
}

func metricValue(t *testing.T, reg *prometheus.Registry, name, resource, group string) float64 {
	t.Helper()

	mfs, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelValue(m, "resource") == resource && labelValue(m, "group") == group {
				return m.GetGauge().GetValue()
			}
		}
	}

	t.Fatalf("metric %q with resource=%q group=%q not found", name, resource, group)
	return 0
}

func labelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}
