package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
)

func newK8sStatusErr(code int, message string) *k8serrors.StatusError {
	return &k8serrors.StatusError{ErrStatus: metav1.Status{Code: int32(code), Message: message}}
}

func TestRerouteAnnotationsEnabled(t *testing.T) {
	t.Run("off when reroute flag disabled", func(t *testing.T) {
		hs := &HTTPServer{
			Cfg:      setting.NewCfg(),
			Features: featuremgmt.WithFeatures(),
		}
		require.False(t, hs.rerouteAnnotationsEnabled())
	})

	t.Run("off when reroute flag on but resource not installed", func(t *testing.T) {
		hs := &HTTPServer{
			Cfg:      setting.NewCfg(),
			Features: featuremgmt.WithFeatures(featuremgmt.FlagAnnotationsRerouteLegacyCRUDAPIs),
		}
		require.False(t, hs.rerouteAnnotationsEnabled())
	})

	t.Run("on when both reroute flag and kubernetesAnnotations flag are on", func(t *testing.T) {
		hs := &HTTPServer{
			Cfg: setting.NewCfg(),
			Features: featuremgmt.WithFeatures(
				featuremgmt.FlagAnnotationsRerouteLegacyCRUDAPIs,
				featuremgmt.FlagKubernetesAnnotations,
			),
		}
		require.True(t, hs.rerouteAnnotationsEnabled())
	})

	t.Run("on when reroute flag and config-based annotation app platform are enabled", func(t *testing.T) {
		cfg := setting.NewCfg()
		cfg.AnnotationAppPlatform.Enabled = true
		hs := &HTTPServer{
			Cfg:      cfg,
			Features: featuremgmt.WithFeatures(featuremgmt.FlagAnnotationsRerouteLegacyCRUDAPIs),
		}
		require.True(t, hs.rerouteAnnotationsEnabled())
	})
}

func TestLegacyNameMapping(t *testing.T) {
	require.Equal(t, "a-42", legacyIDToName(42))
	id, err := nameToLegacyID("a-42")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)

	_, err = nameToLegacyID("not-annotation")
	require.Error(t, err)
	_, err = nameToLegacyID("")
	require.Error(t, err)
}

func TestHandleK8sAnnotationError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		fallback     string
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "not found maps to 404 with stable message",
			err:          newK8sStatusErr(http.StatusNotFound, "annotations.annotation.grafana.app \"a-1\" not found"),
			fallback:     "ignored",
			expectedCode: http.StatusNotFound,
			expectedMsg:  "Annotation not found",
		},
		{
			name:         "forbidden maps to 403 with stable message",
			err:          newK8sStatusErr(http.StatusForbidden, "denied"),
			fallback:     "ignored",
			expectedCode: http.StatusForbidden,
			expectedMsg:  "Access denied to annotation",
		},
		{
			name:         "bad request passes through k8s message",
			err:          newK8sStatusErr(http.StatusBadRequest, "bad field"),
			fallback:     "ignored",
			expectedCode: http.StatusBadRequest,
			expectedMsg:  "bad field",
		},
		{
			name:         "non-status error maps to 500 with fallback message",
			err:          errString("boom"),
			fallback:     "Failed to save annotation",
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  "Failed to save annotation",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := handleK8sAnnotationError(tc.err, tc.fallback)
			require.NotNil(t, resp)
			assert.Equal(t, tc.expectedCode, resp.Status())

			var body map[string]any
			require.NoError(t, json.Unmarshal(resp.Body(), &body))
			msg, _ := body["message"].(string)
			assert.True(t, strings.Contains(msg, tc.expectedMsg), "got %q, want substring %q", msg, tc.expectedMsg)
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
