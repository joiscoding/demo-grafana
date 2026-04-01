package folder

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/folder"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestIntegrationFolderParity(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	for _, mode := range []grafanarest.DualWriterMode{
		grafanarest.Mode0,
	} {
		t.Run(fmt.Sprintf("folder parity (mode:%d)", mode), func(t *testing.T) {
			helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
				DisableAnonymous:      true,
				DisableDataMigrations: true,
				AppModeProduction:     true,
				APIServerStorageType:  "unified",
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					"folders.folder.grafana.app": {
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
				legacyPayload := `{"title": "Parity Legacy Folder"}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/folders",
					Body:   []byte(legacyPayload),
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyCreate.Response.StatusCode, "legacy folder create should succeed")
				uid := legacyCreate.Result.UID
				require.NotEmpty(t, uid)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err, "k8s get should find legacy-created folder")
				require.Equal(t, uid, found.GetName())

				spec, ok := found.Object["spec"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "Parity Legacy Folder", spec["title"])

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("k8s create visible via legacy", func(t *testing.T) {
				obj := &unstructured.Unstructured{
					Object: map[string]any{
						"spec": map[string]any{
							"title":       "Parity K8s Folder",
							"description": "Created via K8s API",
						},
					},
				}
				obj.SetGenerateName("parity-")
				obj.SetAPIVersion(gvr.GroupVersion().String())
				obj.SetKind("Folder")

				created, err := client.Resource.Create(ctx, obj, metav1.CreateOptions{})
				require.NoError(t, err)
				uid := created.GetName()
				require.NotEmpty(t, uid)

				legacyGet := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/folders/" + uid,
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyGet.Response.StatusCode, "legacy get should find k8s-created folder")
				require.Equal(t, uid, legacyGet.Result.UID)
				require.Equal(t, "Parity K8s Folder", legacyGet.Result.Title)

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("legacy update reflected in k8s", func(t *testing.T) {
				createPayload := `{"title": "Folder Before Update"}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/folders",
					Body:   []byte(createPayload),
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyCreate.Response.StatusCode)
				uid := legacyCreate.Result.UID
				version := legacyCreate.Result.Version

				updatePayload := fmt.Sprintf(`{"title": "Folder After Update", "version": %d, "overwrite": true}`, version)
				legacyUpdate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPut,
					Path:   "/api/folders/" + uid,
					Body:   []byte(updatePayload),
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyUpdate.Response.StatusCode)
				require.Equal(t, "Folder After Update", legacyUpdate.Result.Title)

				found, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.NoError(t, err)
				spec := found.Object["spec"].(map[string]any)
				require.Equal(t, "Folder After Update", spec["title"])

				err = client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
				require.NoError(t, err)
			})

			t.Run("legacy delete removes from k8s", func(t *testing.T) {
				createPayload := `{"title": "Folder To Delete"}`
				legacyCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/folders",
					Body:   []byte(createPayload),
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyCreate.Response.StatusCode)
				uid := legacyCreate.Result.UID

				deleteRsp := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodDelete,
					Path:   "/api/folders/" + uid,
				}, &map[string]any{})
				require.Equal(t, http.StatusOK, deleteRsp.Response.StatusCode)

				_, err := client.Resource.Get(ctx, uid, metav1.GetOptions{})
				require.Error(t, err, "k8s get should fail after legacy delete")
			})

			t.Run("k8s list includes legacy-created folders", func(t *testing.T) {
				uids := make([]string, 3)
				for i := 0; i < 3; i++ {
					payload := fmt.Sprintf(`{"title": "ListParity-%d"}`, i)
					rsp := apis.DoRequest(helper, apis.RequestParams{
						User:   client.Args.User,
						Method: http.MethodPost,
						Path:   "/api/folders",
						Body:   []byte(payload),
					}, &folder.Folder{})
					require.Equal(t, http.StatusOK, rsp.Response.StatusCode)
					uids[i] = rsp.Result.UID
				}

				k8sList, err := client.Resource.List(ctx, metav1.ListOptions{})
				require.NoError(t, err)

				k8sNames := map[string]bool{}
				for _, item := range k8sList.Items {
					k8sNames[item.GetName()] = true
				}
				for _, uid := range uids {
					require.True(t, k8sNames[uid], "legacy folder %s should appear in k8s list", uid)
				}

				for _, uid := range uids {
					err := client.Resource.Delete(ctx, uid, metav1.DeleteOptions{})
					require.NoError(t, err)
				}
			})

			t.Run("nested folder parity: parent created via legacy, child via k8s", func(t *testing.T) {
				parentPayload := `{"title": "Parity Parent"}`
				parentCreate := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodPost,
					Path:   "/api/folders",
					Body:   []byte(parentPayload),
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, parentCreate.Response.StatusCode)
				parentUID := parentCreate.Result.UID

				childObj := &unstructured.Unstructured{
					Object: map[string]any{
						"metadata": map[string]any{
							"annotations": map[string]any{
								"grafana.app/folder": parentUID,
							},
						},
						"spec": map[string]any{
							"title":       "Parity Child",
							"description": "Child folder via k8s",
						},
					},
				}
				childObj.SetGenerateName("child-")
				childObj.SetAPIVersion(gvr.GroupVersion().String())
				childObj.SetKind("Folder")

				createdChild, err := client.Resource.Create(ctx, childObj, metav1.CreateOptions{})
				require.NoError(t, err)
				childUID := createdChild.GetName()

				legacyGet := apis.DoRequest(helper, apis.RequestParams{
					User:   client.Args.User,
					Method: http.MethodGet,
					Path:   "/api/folders/" + childUID,
				}, &folder.Folder{})
				require.Equal(t, http.StatusOK, legacyGet.Response.StatusCode)
				require.Equal(t, "Parity Child", legacyGet.Result.Title)

				_ = client.Resource.Delete(ctx, childUID, metav1.DeleteOptions{})
				_ = client.Resource.Delete(ctx, parentUID, metav1.DeleteOptions{})
			})
		})
	}
}
