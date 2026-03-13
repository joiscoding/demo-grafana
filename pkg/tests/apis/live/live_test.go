package live

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

// TestIntegrationLiveAPI_LegacyDocumentsBehavior documents legacy /api/live behavior
// for parity validation when /apis live app is extended with publish, list, info, push routes.
// Enable FlagLiveAPIServer to also exercise /apis routes once implemented.
func TestIntegrationLiveAPI_LegacyDocumentsBehavior(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
		EnableFeatureToggles: []string{
			featuremgmt.FlagLiveAPIServer, // enables live app at /apis when implemented
		},
	})

	// Legacy /api/live/list - returns 200 with channels array
	t.Run("GET /api/live/list returns 200 with channels", func(t *testing.T) {
		var result struct {
			Channels []any `json:"channels"`
		}
		rsp := apis.DoRequest(helper, apis.RequestParams{
			User:   helper.Org1.Admin,
			Method: http.MethodGet,
			Path:   "/api/live/list",
		}, &result)
		require.Equal(t, http.StatusOK, rsp.Response.StatusCode, "legacy list should return 200")
		require.NotNil(t, result.Channels, "response should have channels array")
	})

	// Legacy /api/live/info/* - returns 404 with message
	t.Run("GET /api/live/info/any returns 404", func(t *testing.T) {
		var result struct {
			Message string `json:"message"`
		}
		rsp := apis.DoRequest(helper, apis.RequestParams{
			User:   helper.Org1.Admin,
			Method: http.MethodGet,
			Path:   "/api/live/info/any",
		}, &result)
		require.Equal(t, http.StatusNotFound, rsp.Response.StatusCode)
		require.Equal(t, "Info is not supported for this channel", result.Message)
	})

	// Legacy /api/live/publish - bad request cases
	t.Run("POST /api/live/publish with invalid channel returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"channel": "invalid-no-slash", "data": map[string]any{}})
		rsp := apis.DoRequest(helper, apis.RequestParams{
			User:        helper.Org1.Admin,
			Method:      http.MethodPost,
			Path:        "/api/live/publish",
			Body:        body,
			ContentType: "application/json",
		}, &struct{}{})
		require.Equal(t, http.StatusBadRequest, rsp.Response.StatusCode)
	})

	// Legacy /api/live/push - valid Influx returns 200
	t.Run("POST /api/live/push/:streamId with Influx line returns 200", func(t *testing.T) {
		rsp := apis.DoRequest(helper, apis.RequestParams{
			User:   helper.Org1.Admin,
			Method: http.MethodPost,
			Path:   "/api/live/push/test-stream",
			Body:   []byte("cpu value=1.0"),
		}, &struct{}{})
		require.Equal(t, http.StatusOK, rsp.Response.StatusCode)
	})
}
