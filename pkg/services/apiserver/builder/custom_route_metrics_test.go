package builder_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestNewCustomRouteMetrics(t *testing.T) {
	t.Parallel()

	metrics := builder.NewCustomRouteMetrics(prometheus.NewRegistry())
	require.NotNil(t, metrics)
}

func TestCustomRouteMetrics_InstrumentHandler(t *testing.T) {
	t.Parallel()

	metrics := builder.NewCustomRouteMetrics(nil)
	called := false
	handler := metrics.InstrumentHandler("example.grafana.app", "v1", "stats", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/apis/example.grafana.app/v1/stats", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.True(t, called)
	require.Equal(t, http.StatusCreated, rec.Code)
}
