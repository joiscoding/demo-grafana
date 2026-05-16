package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/usertest"
	"github.com/grafana/grafana/pkg/storage/unified/resourcepb"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func TestGetUsersFromMeta(t *testing.T) {
	userSvcTest := usertest.NewUserServiceFake()
	userSvcTest.ExpectedListUsersByIdOrUid = []*user.User{
		{
			ID:  1,
			UID: "uid-value",
		},
		{
			ID:  2,
			UID: "uid-value2",
		},
	}
	client := &k8sHandler{
		userService: userSvcTest,
	}
	t.Run("returns user with valid UID", func(t *testing.T) {
		result, err := client.GetUsersFromMeta(context.Background(), []string{"user:uid-value"})
		require.NoError(t, err)
		require.Equal(t, "uid-value", result["user:uid-value"].UID)
		require.Equal(t, int64(1), result["user:uid-value"].ID)
	})

	t.Run("returns user when id is passed in", func(t *testing.T) {
		result, err := client.GetUsersFromMeta(context.Background(), []string{"user:1"})
		require.NoError(t, err)
		require.Equal(t, "uid-value", result["user:1"].UID)
		require.Equal(t, int64(1), result["user:1"].ID)
	})

	t.Run("returns users when id and uid are passed in", func(t *testing.T) {
		result, err := client.GetUsersFromMeta(context.Background(), []string{"user:1", "user:uid-value2"})
		require.NoError(t, err)
		require.Equal(t, "uid-value", result["user:1"].UID)
		require.Equal(t, int64(1), result["user:1"].ID)
		require.Equal(t, "uid-value2", result["user:uid-value2"].UID)
		require.Equal(t, int64(2), result["user:uid-value2"].ID)
	})

	t.Run("returns empty map for invalid meta format", func(t *testing.T) {
		result, err := client.GetUsersFromMeta(context.Background(), []string{"invalid"})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("returns empty map when user service fails", func(t *testing.T) {
		userSvcTest.ExpectedError = errors.New("list failed")
		t.Cleanup(func() { userSvcTest.ExpectedError = nil })

		result, err := client.GetUsersFromMeta(context.Background(), []string{"user:1"})
		require.NoError(t, err)
		require.Empty(t, result)
	})
}

type mockResourceIndexClient struct {
	resourcepb.ResourceIndexClient

	searchResp *resourcepb.ResourceSearchResponse
	searchErr  error
	statsResp  *resourcepb.ResourceStatsResponse
	statsErr   error
}

func (m *mockResourceIndexClient) Search(ctx context.Context, in *resourcepb.ResourceSearchRequest, opts ...grpc.CallOption) (*resourcepb.ResourceSearchResponse, error) {
	return m.searchResp, m.searchErr
}

func (m *mockResourceIndexClient) GetStats(ctx context.Context, in *resourcepb.ResourceStatsRequest, opts ...grpc.CallOption) (*resourcepb.ResourceStatsResponse, error) {
	return m.statsResp, m.statsErr
}

func (m *mockResourceIndexClient) RebuildIndexes(ctx context.Context, in *resourcepb.RebuildIndexesRequest, opts ...grpc.CallOption) (*resourcepb.RebuildIndexesResponse, error) {
	return nil, errors.New("not implemented")
}

func testGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "dashboard.grafana.app", Version: "v1beta1", Resource: "dashboards"}
}

func testDashboardJSON(name string) []byte {
	return []byte(`{"apiVersion":"dashboard.grafana.app/v1beta1","kind":"Dashboard","metadata":{"name":"` + name + `","namespace":"org-1"}}`)
}

func newTestK8sHandler(t *testing.T, srv *httptest.Server, searcher resourcepb.ResourceIndexClient) *k8sHandler {
	t.Helper()

	return &k8sHandler{
		namespacer: func(orgID int64) string { return "org-1" },
		gvr:        testGVR(),
		restConfig: func(ctx context.Context) (*rest.Config, error) {
			return &rest.Config{Host: srv.URL}, nil
		},
		searcher: searcher,
	}
}

func TestK8sHandler_GetNamespace(t *testing.T) {
	handler := &k8sHandler{
		namespacer: func(orgID int64) string {
			if orgID == 5 {
				return "namespace-5"
			}
			return "other"
		},
	}
	require.Equal(t, "namespace-5", handler.GetNamespace(5))
}

func TestK8sHandler_Search(t *testing.T) {
	searcher := &mockResourceIndexClient{
		searchResp: &resourcepb.ResourceSearchResponse{TotalHits: 3},
	}
	handler := &k8sHandler{
		namespacer: func(orgID int64) string { return "org-1" },
		gvr:        testGVR(),
		searcher:   searcher,
	}

	t.Run("fills default options and namespace", func(t *testing.T) {
		resp, err := handler.Search(context.Background(), 1, &resourcepb.ResourceSearchRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(3), resp.TotalHits)
	})

	t.Run("preserves existing resource key", func(t *testing.T) {
		req := &resourcepb.ResourceSearchRequest{
			Options: &resourcepb.ListOptions{
				Key: &resourcepb.ResourceKey{Namespace: "custom"},
			},
		}
		_, err := handler.Search(context.Background(), 1, req)
		require.NoError(t, err)
		require.Equal(t, "custom", req.Options.Key.Namespace)
	})

	t.Run("returns searcher error", func(t *testing.T) {
		searcher.searchErr = errors.New("search failed")
		_, err := handler.Search(context.Background(), 1, &resourcepb.ResourceSearchRequest{})
		require.Error(t, err)
	})
}

func TestK8sHandler_GetStats(t *testing.T) {
	searcher := &mockResourceIndexClient{
		statsResp: &resourcepb.ResourceStatsResponse{},
	}
	handler := &k8sHandler{
		namespacer: func(orgID int64) string { return "org-1" },
		gvr:        testGVR(),
		searcher:   searcher,
	}

	resp, err := handler.GetStats(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestK8sHandler_DynamicClient(t *testing.T) {
	const dashName = "test-dash"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(testDashboardJSON(dashName))
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		case http.MethodDelete:
			if r.URL.Path == "/apis/dashboard.grafana.app/v1beta1/namespaces/org-1/dashboards" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"kind":"Status","status":"Success"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"kind":"Status","status":"Success"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	handler := newTestK8sHandler(t, srv, nil)
	ctx := context.Background()
	obj := &unstructured.Unstructured{}
	require.NoError(t, obj.UnmarshalJSON(testDashboardJSON(dashName)))

	t.Run("Get", func(t *testing.T) {
		got, err := handler.Get(ctx, dashName, 1, v1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, dashName, got.GetName())
	})

	t.Run("Create", func(t *testing.T) {
		got, err := handler.Create(ctx, obj.DeepCopy(), 1, v1.CreateOptions{})
		require.NoError(t, err)
		require.Equal(t, dashName, got.GetName())
	})

	t.Run("Update", func(t *testing.T) {
		got, err := handler.Update(ctx, obj.DeepCopy(), 1, v1.UpdateOptions{})
		require.NoError(t, err)
		require.Equal(t, dashName, got.GetName())
	})

	t.Run("List", func(t *testing.T) {
		listJSON := []byte(`{"apiVersion":"dashboard.grafana.app/v1beta1","kind":"DashboardList","items":[{"apiVersion":"dashboard.grafana.app/v1beta1","kind":"Dashboard","metadata":{"name":"test-dash","namespace":"org-1"}}]}`)
		listSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(listJSON)
		}))
		defer listSrv.Close()
		listHandler := newTestK8sHandler(t, listSrv, nil)

		list, err := listHandler.List(ctx, 1, v1.ListOptions{})
		require.NoError(t, err)
		require.Len(t, list.Items, 1)
	})

	t.Run("Delete", func(t *testing.T) {
		require.NoError(t, handler.Delete(ctx, dashName, 1, v1.DeleteOptions{}))
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		require.NoError(t, handler.DeleteCollection(ctx, 1))
	})

	t.Run("returns error when rest config fails", func(t *testing.T) {
		errHandler := &k8sHandler{
			namespacer: func(orgID int64) string { return "org-1" },
			gvr:        testGVR(),
			restConfig: func(ctx context.Context) (*rest.Config, error) {
				return nil, errors.New("config error")
			},
		}
		_, err := errHandler.Get(ctx, dashName, 1, v1.GetOptions{})
		require.Error(t, err)
	})
}

func TestMockK8sHandler(t *testing.T) {
	m := &MockK8sHandler{}
	ctx := context.Background()
	obj := &unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"name": "x"}}}

	m.On("GetNamespace", int64(1)).Return("org-1")
	m.On("Get", ctx, "x", int64(1), v1.GetOptions{}, []string(nil)).Return(obj, nil)
	m.On("Create", ctx, obj, int64(1), v1.CreateOptions{}).Return(obj, nil)
	m.On("Update", ctx, obj, int64(1), v1.UpdateOptions{}).Return(obj, nil)
	m.On("Delete", ctx, "x", int64(1), v1.DeleteOptions{}).Return(nil)
	m.On("DeleteCollection", ctx, int64(1)).Return(nil)
	m.On("List", ctx, int64(1), v1.ListOptions{}).Return(&unstructured.UnstructuredList{}, nil)
	m.On("Search", ctx, int64(1), mock.Anything).Return(&resourcepb.ResourceSearchResponse{}, nil)
	m.On("GetStats", ctx, int64(1)).Return(&resourcepb.ResourceStatsResponse{}, nil)
	m.On("GetUsersFromMeta", ctx, []string{"user:1"}).Return(map[string]*user.User{}, nil)

	require.Equal(t, "org-1", m.GetNamespace(1))
	got, err := m.Get(ctx, "x", 1, v1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, obj, got)
	_, err = m.Create(ctx, obj, 1, v1.CreateOptions{})
	require.NoError(t, err)
	_, err = m.Update(ctx, obj, 1, v1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, m.Delete(ctx, "x", 1, v1.DeleteOptions{}))
	require.NoError(t, m.DeleteCollection(ctx, 1))
	_, err = m.List(ctx, 1, v1.ListOptions{})
	require.NoError(t, err)
	_, err = m.Search(ctx, 1, &resourcepb.ResourceSearchRequest{})
	require.NoError(t, err)
	_, err = m.GetStats(ctx, 1)
	require.NoError(t, err)
	_, err = m.GetUsersFromMeta(ctx, []string{"user:1"})
	require.NoError(t, err)
	m.AssertExpectations(t)
}
