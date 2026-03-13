package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/accesscontrol/acimpl"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/live/model"
	"github.com/grafana/grafana/pkg/services/live/pushhttp"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util/testutil"
	"github.com/grafana/grafana/pkg/web/webtest"
)

// TestLegacyLiveAPI_* are characterization tests for /api/live endpoints.
// They document legacy behavior before migration to /apis for parity validation.
func TestLegacyLiveAPI_Publish_BadRequest(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gLive := newTestLive(t)
	cfg := setting.NewCfg()
	gateway := pushhttp.ProvideService(cfg, gLive)

	server := SetupAPITestServer(t, func(hs *HTTPServer) {
		hs.Cfg = cfg
		hs.Live = gLive
		hs.LivePushGateway = gateway
		hs.AccessControl = acimpl.ProvideAccessControl(featuremgmt.WithFeatures())
	})

	t.Run("POST /api/live/publish with empty body returns 400", func(t *testing.T) {
		body := strings.NewReader("")
		req := server.NewRequest(http.MethodPost, "/api/live/publish", body)
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("POST /api/live/publish with invalid JSON returns 400", func(t *testing.T) {
		body := strings.NewReader("{invalid")
		req := server.NewRequest(http.MethodPost, "/api/live/publish", body)
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("POST /api/live/publish with invalid channel ID returns 400", func(t *testing.T) {
		cmd := model.LivePublishCmd{Channel: "invalid-no-slash", Data: json.RawMessage(`{}`)}
		bodyBytes, _ := json.Marshal(cmd)
		req := server.NewRequest(http.MethodPost, "/api/live/publish", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}

func TestLegacyLiveAPI_List(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gLive := newTestLive(t)
	cfg := setting.NewCfg()
	gateway := pushhttp.ProvideService(cfg, gLive)

	server := SetupAPITestServer(t, func(hs *HTTPServer) {
		hs.Cfg = cfg
		hs.Live = gLive
		hs.LivePushGateway = gateway
		hs.AccessControl = acimpl.ProvideAccessControl(featuremgmt.WithFeatures())
	})

	t.Run("GET /api/live/list returns 200 with channels array", func(t *testing.T) {
		req := server.NewGetRequest("/api/live/list")
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		var out struct {
			Channels []any `json:"channels"`
		}
		err = json.NewDecoder(res.Body).Decode(&out)
		require.NoError(t, err)
		require.NotNil(t, out.Channels, "response should have channels key")
	})
}

func TestLegacyLiveAPI_Info(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gLive := newTestLive(t)
	cfg := setting.NewCfg()
	gateway := pushhttp.ProvideService(cfg, gLive)

	server := SetupAPITestServer(t, func(hs *HTTPServer) {
		hs.Cfg = cfg
		hs.Live = gLive
		hs.LivePushGateway = gateway
		hs.AccessControl = acimpl.ProvideAccessControl(featuremgmt.WithFeatures())
	})

	t.Run("GET /api/live/info/any returns 404 with message", func(t *testing.T) {
		req := server.NewGetRequest("/api/live/info/any")
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)

		var out struct {
			Message string `json:"message"`
		}
		err = json.NewDecoder(res.Body).Decode(&out)
		require.NoError(t, err)
		assert.Equal(t, "Info is not supported for this channel", out.Message)
	})
}

func TestLegacyLiveAPI_Push(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gLive := newTestLive(t)
	cfg := setting.NewCfg()
	gateway := pushhttp.ProvideService(cfg, gLive)

	server := SetupAPITestServer(t, func(hs *HTTPServer) {
		hs.Cfg = cfg
		hs.Live = gLive
		hs.LivePushGateway = gateway
		hs.AccessControl = acimpl.ProvideAccessControl(featuremgmt.WithFeatures())
	})

	t.Run("POST /api/live/push/:streamId with valid Influx line returns 200", func(t *testing.T) {
		// Influx line protocol
		body := strings.NewReader("cpu value=1.0")
		req := server.NewRequest(http.MethodPost, "/api/live/push/test-stream", body)
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("POST /api/live/push/:streamId with unsupported format returns 400", func(t *testing.T) {
		// Empty/invalid body
		body := strings.NewReader("")
		req := server.NewRequest(http.MethodPost, "/api/live/push/test-stream", body)
		res, err := server.Send(webtest.RequestWithSignedInUser(req, authedUserWithPermissions(1, 1, nil)))
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}

func TestLegacyLiveAPI_RequiresAuth(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gLive := newTestLive(t)
	cfg := setting.NewCfg()
	gateway := pushhttp.ProvideService(cfg, gLive)

	server := SetupAPITestServer(t, func(hs *HTTPServer) {
		hs.Cfg = cfg
		hs.Live = gLive
		hs.LivePushGateway = gateway
		hs.AccessControl = acimpl.ProvideAccessControl(featuremgmt.WithFeatures())
	})

	t.Run("GET /api/live/list without auth returns 401", func(t *testing.T) {
		req := server.NewGetRequest("/api/live/list")
		res, err := server.Send(req)
		require.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}
