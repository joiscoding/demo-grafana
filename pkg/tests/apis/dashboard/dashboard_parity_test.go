package dashboards

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	dashboardV1 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1beta1"
	"github.com/grafana/grafana/pkg/api/dtos"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestIntegrationDashboardParity(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	gvr := dashboardV1.DashboardResourceInfo.GroupVersionResource()

	for _, mode := range []grafanarest.DualWriterMode{
		grafanarest.Mode0,
	} {
		t.Run(fmt.Sprintf("dashboard parity (mode:%d)", mode), func(t *testing.T) {
			helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
				DisableAnonymous:      true,
				DisableDataMigrations: true,
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					"dashboards.dashboard.grafana.app": {
						DualWriterMode: mode,
					},
				},
			})

			ctx := context.Background()
			client := helper.GetResourceClient(apis.ResourceClientArgs{
				User: helper.Org1.Admin,
				GVR:  gvr,
			})

			t.Run("legacy create visible via k8s", func(t *testing.T) {
				legacyPayload := `{
					"dashboard": {
						"title": "Parity Legacy Dashboard",
						"tags": ["parity", "test"],
						"schemaVersion": 42
					},
					"overwrite": false
				}`
				var createRsp map[string]any
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/dashboards/db",
					Body:   []byte(legacyPayload),
				}, &createRsp)
				require.Equal(t, http.StatusOK, legacyCreate.Response.StatusCode, "legacy dashboard create should succeed")

				uid, ok := createRsp["uid"].(string)
				require.True(t, ok, "response should contain uid")
				require.NotEmpty(t, uid)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err, "k8s get should find legacy-created dashboard")
				require.Equal(t, uid, found.GetName())

				spec, ok := found.Object["spec"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "Parity Legacy Dashboard", spec["title"])

				zeroInt64 := int64(0)
				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{GracePeriodSeconds: &zeroInt64})
				require.NoError(t, err)
			})

			t.Run("k8s create visible via legacy", func(t *testing.T) {
				obj := &unstructured.Unstructured{
					Object: map[string]any{
						"spec": map[string]any{
							"title":         "Parity K8s Dashboard",
							"schemaVersion": 42,
						},
					},
				}
				obj.SetGenerateName("parity-")
				obj.SetAPIVersion(gvr.GroupVersion().String())
				obj.SetKind("Dashboard")

				created, err := client.Resource.Create(ctx, obj, metav1.CreateOptions{})
				require.NoError(t, err)
				uid := created.GetName()
				require.NotEmpty(t, uid)

				legacyGet := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/dashboards/uid/" + uid,
				}, &dtos.DashboardFullWithMeta{})
				require.Equal(t, http.StatusOK, legacyGet.Response.StatusCode, "legacy get should find k8s-created dashboard")
				require.Equal(t, "Parity K8s Dashboard", legacyGet.Result.Dashboard.Get("title").MustString(""))

				zeroInt64 := int64(0)
				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{GracePeriodSeconds: &zeroInt64})
				require.NoError(t, err)
			})

			t.Run("legacy update reflected in k8s", func(t *testing.T) {
				createPayload := `{
					"dashboard": {
						"title": "Update Parity",
						"uid": "parity-update-test",
						"schemaVersion": 42
					},
					"overwrite": true
				}`
				var createRsp map[string]any
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/dashboards/db",
					Body:   []byte(createPayload),
				}, &createRsp)
				require.Equal(t, http.StatusOK, legacyCreate.Response.StatusCode)

				uid := createRsp["uid"].(string)

				updatePayload := `{
					"dashboard": {
						"title": "Updated Parity Title",
						"uid": "` + uid + `",
						"schemaVersion": 42,
						"version": 1
					},
					"overwrite": true
				}`
				var updateRsp map[string]any
				legacyUpdate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/dashboards/db",
					Body:   []byte(updatePayload),
				}, &updateRsp)
				require.Equal(t, http.StatusOK, legacyUpdate.Response.StatusCode)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err)
				spec := found.Object["spec"].(map[string]any)
				require.Equal(t, "Updated Parity Title", spec["title"])

				zeroInt64 := int64(0)
				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{GracePeriodSeconds: &zeroInt64})
				require.NoError(t, err)
			})

			t.Run("legacy delete removes from k8s", func(t *testing.T) {
				createPayload := `{
					"dashboard": {
						"title": "Delete Parity",
						"uid": "parity-delete-test",
						"schemaVersion": 42
					},
					"overwrite": true
				}`
				var createRsp map[string]any
				apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/dashboards/db",
					Body:   []byte(createPayload),
				}, &createRsp)

				uid := createRsp["uid"].(string)

				deleteRsp := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodDelete,
					Path:   "/api/dashboards/uid/" + uid,
				}, &struct{}{})
				require.Equal(t, http.StatusOK, deleteRsp.Response.StatusCode)

				_, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				statusError := helper.AsStatusError(err)
				require.Equal(t, metav1.StatusReasonNotFound, statusError.Status().Reason)
			})

			t.Run("k8s list matches legacy search for tagged dashboards", func(t *testing.T) {
				for i := 0; i < 3; i++ {
					payload := fmt.Sprintf(`{
						"dashboard": {
							"title": "ParityList-%d",
							"tags": ["parity-list"],
							"schemaVersion": 42
						},
						"overwrite": false
					}`, i)
					var createRsp map[string]any
					rsp := apis.DoRequest(helper, apis.RequestParams{
						User:   client.Args.User,
						Method: http.MethodPost,
						Path:   "/api/dashboards/db",
						Body:   []byte(payload),
					}, &createRsp)
					require.Equal(t, http.StatusOK, rsp.Response.StatusCode)
				}

				k8sList, err := client.Resource.List(ctx, metav1.ListOptions{})
				require.NoError(t, err)

				var searchResult []json.RawMessage
				legacySearch := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/search?tag=parity-list",
				}, &searchResult)
				require.Equal(t, http.StatusOK, legacySearch.Response.StatusCode)
				require.Len(t, searchResult, 3, "legacy search should find 3 tagged dashboards")

				require.GreaterOrEqual(t, len(k8sList.Items), 3, "k8s list should contain at least the tagged dashboards")

				zeroInt64 := int64(0)
				for _, item := range k8sList.Items {
					_ = client.Resource.Delete(ctx, item.GetName(), metav1.DeleteOptions{GracePeriodSeconds: &zeroInt64})
				}
			})
		})
	}
}
