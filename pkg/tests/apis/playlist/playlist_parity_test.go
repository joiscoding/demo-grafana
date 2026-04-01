package playlist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/playlist"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestIntegrationPlaylistParity(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	for _, mode := range []grafanarest.DualWriterMode{
		grafanarest.Mode0,
	} {
		t.Run(fmt.Sprintf("playlist parity (mode:%d)", mode), func(t *testing.T) {
			helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
				DisableAnonymous:      true,
				DisableDataMigrations: true,
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					RESOURCEGROUP: {
						DualWriterMode: mode,
					},
				},
			})

			ctx := context.Background()
			client := helper.GetResourceClient(apis.ResourceClientArgs{
				User: helper.Org1.Editor,
				GVR:  gvr,
			})

			t.Run("legacy create visible via k8s", func(t *testing.T) {
				legacyPayload := `{
					"name": "Parity Legacy",
					"interval": "30s",
					"items": [
						{"type": "dashboard_by_tag", "value": "parity-tag"}
					]
				}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/playlists",
					Body:   []byte(legacyPayload),
				}, &playlist.Playlist{})
				require.Equal(t, 200, legacyCreate.Response.StatusCode, "legacy playlist create should succeed")
				uid := legacyCreate.Result.UID
				require.NotEmpty(t, uid)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err, "k8s get should find legacy-created playlist")
				require.Equal(t, uid, found.GetName())

				spec, ok := found.Object["spec"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "Parity Legacy", spec["title"])
				require.Equal(t, "30s", spec["interval"])

				items, ok := spec["items"].([]any)
				require.True(t, ok)
				require.Len(t, items, 1)
				item0 := items[0].(map[string]any)
				require.Equal(t, "dashboard_by_tag", item0["type"])
				require.Equal(t, "parity-tag", item0["value"])

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("k8s create visible via legacy", func(t *testing.T) {
				obj := &unstructured.Unstructured{
					Object: map[string]any{
						"spec": map[string]any{
							"title":    "Parity K8s",
							"interval": "45s",
							"items": []any{
								map[string]any{"type": "dashboard_by_uid", "value": "abc123"},
							},
						},
					},
				}
				obj.SetGenerateName("parity-")
				obj.SetAPIVersion(gvr.GroupVersion().String())
				obj.SetKind("Playlist")

				created, err := client.Resource.Create(ctx, obj, metav1.CreateOptions{})
				require.NoError(t, err)
				uid := created.GetName()
				require.NotEmpty(t, uid)

				legacyGet := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/playlists/" + uid,
				}, &playlist.PlaylistDTO{})
				require.Equal(t, 200, legacyGet.Response.StatusCode, "legacy get should find k8s-created playlist")
				require.Equal(t, uid, legacyGet.Result.Uid)
				require.Equal(t, "Parity K8s", legacyGet.Result.Name)
				require.Equal(t, "45s", legacyGet.Result.Interval)
				require.Len(t, legacyGet.Result.Items, 1)
				require.Equal(t, "dashboard_by_uid", legacyGet.Result.Items[0].Type)
				require.Equal(t, "abc123", legacyGet.Result.Items[0].Value)

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("legacy update reflected in k8s", func(t *testing.T) {
				createPayload := `{
					"name": "Update Test",
					"interval": "10s",
					"items": [{"type": "dashboard_by_tag", "value": "initial"}]
				}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/playlists",
					Body:   []byte(createPayload),
				}, &playlist.Playlist{})
				require.Equal(t, 200, legacyCreate.Response.StatusCode)
				uid := legacyCreate.Result.UID

				updatePayload := `{
					"name": "Updated Title",
					"interval": "5m",
					"items": [
						{"type": "dashboard_by_tag", "value": "updated"},
						{"type": "dashboard_by_uid", "value": "def456"}
					]
				}`
				legacyUpdate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPut,
					Path:   "/api/playlists/" + uid,
					Body:   []byte(updatePayload),
				}, &playlist.PlaylistDTO{})
				require.Equal(t, 200, legacyUpdate.Response.StatusCode)
				require.Equal(t, "Updated Title", legacyUpdate.Result.Name)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err)
				spec := found.Object["spec"].(map[string]any)
				require.Equal(t, "Updated Title", spec["title"])
				require.Equal(t, "5m", spec["interval"])

				items, _ := json.Marshal(spec["items"])
				require.Contains(t, string(items), "updated")
				require.Contains(t, string(items), "def456")

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("legacy delete removes from k8s", func(t *testing.T) {
				createPayload := `{
					"name": "Delete Test",
					"interval": "10s",
					"items": [{"type": "dashboard_by_tag", "value": "todelete"}]
				}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/playlists",
					Body:   []byte(createPayload),
				}, &playlist.Playlist{})
				require.Equal(t, 200, legacyCreate.Response.StatusCode)
				uid := legacyCreate.Result.UID

				deleteRsp := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodDelete,
					Path:   "/api/playlists/" + uid,
				}, &struct{}{})
				require.Equal(t, 200, deleteRsp.Response.StatusCode)

				_, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				statusError := helper.AsStatusError(err)
				require.Equal(t, metav1.StatusReasonNotFound, statusError.Status().Reason)
			})

			t.Run("k8s list matches legacy search", func(t *testing.T) {
				for i := 0; i < 3; i++ {
					payload := fmt.Sprintf(`{"name":"ListTest-%d","interval":"10s","items":[{"type":"dashboard_by_tag","value":"list"}]}`, i)
					rsp := apis.DoRequest(helper, apis.RequestParams{
						User:   client.Args.User,
						Method: http.MethodPost,
						Path:   "/api/playlists",
						Body:   []byte(payload),
					}, &playlist.Playlist{})
					require.Equal(t, 200, rsp.Response.StatusCode)
				}

				k8sList, err := client.Resource.List(ctx, metav1.ListOptions{})
				require.NoError(t, err)

				legacyList := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/playlists",
				}, &playlist.Playlists{})
				require.Equal(t, 200, legacyList.Response.StatusCode)

				require.Equal(t, len(k8sList.Items), len(*legacyList.Result),
					"k8s list and legacy search should return same count")

				k8sNames := map[string]bool{}
				for _, item := range k8sList.Items {
					k8sNames[item.GetName()] = true
				}
				for _, p := range *legacyList.Result {
					require.True(t, k8sNames[p.UID], "legacy playlist %s should appear in k8s list", p.UID)
				}

				for _, item := range k8sList.Items {
					err := client.Resource.Delete(ctx, item.GetName(), metav1.DeleteOptions{})
					require.NoError(t, err)
				}
			})
		})
	}
}
