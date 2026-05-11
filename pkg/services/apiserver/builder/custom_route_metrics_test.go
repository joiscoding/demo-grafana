package builder_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestNewCustomRouteMetrics(t *testing.T) {
	require.NotNil(t, builder.NewCustomRouteMetrics(nil))
	require.NotNil(t, builder.NewCustomRouteMetrics(prometheus.NewRegistry()))
}

func TestInstrumentHandler_StatusCodeFromWriteHeader(t *testing.T) {
	m := builder.NewCustomRouteMetrics(nil)
	handler := m.InstrumentHandler("grafana.app", "v1", "stats", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
		w.WriteHeader(http.StatusInternalServerError) // ignored (already written)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler(rr, req)

	require.Equal(t, http.StatusTeapot, rr.Code)
	require.Equal(t, "hi", rr.Body.String())
}

func TestInstrumentHandler_DefaultStatusOK(t *testing.T) {
	m := builder.NewCustomRouteMetrics(nil)
	handler := m.InstrumentHandler("g", "v1", "r", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	handler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "ok", rr.Body.String())
}

func TestInstrumentHandler_ScopeFromRequestInfo(t *testing.T) {
	m := builder.NewCustomRouteMetrics(nil)
	called := false
	handler := m.InstrumentHandler("g", "v1", "r", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := request.WithRequestInfo(req.Context(), &request.RequestInfo{
		IsResourceRequest: true,
		Namespace:         "default",
		Resource:          "r",
		APIGroup:          "g",
		APIVersion:        "v1",
		Verb:              "get",
	})
	req = req.WithContext(ctx)

	handler(rr, req)
	require.True(t, called)
}
