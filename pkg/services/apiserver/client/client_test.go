package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	claims "github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/services/search/sort"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/usertest"
	"github.com/grafana/grafana/pkg/storage/unified/resourcepb"
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
}

// fakeIndexClient is a minimal stub for resourcepb.ResourceIndexClient.
type fakeIndexClient struct {
	searchReq  *resourcepb.ResourceSearchRequest
	searchResp *resourcepb.ResourceSearchResponse
	searchErr  error
	statsReq   *resourcepb.ResourceStatsRequest
	statsResp  *resourcepb.ResourceStatsResponse
	statsErr   error
}

func (f *fakeIndexClient) Search(ctx context.Context, in *resourcepb.ResourceSearchRequest, _ ...grpc.CallOption) (*resourcepb.ResourceSearchResponse, error) {
	f.searchReq = in
	return f.searchResp, f.searchErr
}

func (f *fakeIndexClient) GetStats(ctx context.Context, in *resourcepb.ResourceStatsRequest, _ ...grpc.CallOption) (*resourcepb.ResourceStatsResponse, error) {
	f.statsReq = in
	return f.statsResp, f.statsErr
}

func (f *fakeIndexClient) RebuildIndexes(ctx context.Context, in *resourcepb.RebuildIndexesRequest, _ ...grpc.CallOption) (*resourcepb.RebuildIndexesResponse, error) {
	return nil, nil
}

func TestNewK8sHandler(t *testing.T) {
	h := NewK8sHandler(nil, claims.OrgNamespaceFormatter, schema.GroupVersionResource{Group: "g", Version: "v", Resource: "r"},
		func(context.Context) (*rest.Config, error) { return nil, nil }, nil, nil, nil, sort.Service{}, nil)
	require.NotNil(t, h)
	require.Equal(t, "default", h.GetNamespace(1))
}

func TestGetNamespace(t *testing.T) {
	h := &k8sHandler{namespacer: claims.OrgNamespaceFormatter}
	require.Equal(t, "default", h.GetNamespace(1))
	require.Equal(t, "org-2", h.GetNamespace(2))
}

// newTestHandler builds a k8sHandler whose underlying dynamic client talks to
// the given http test server.
func newTestHandler(t *testing.T, srvURL string) *k8sHandler {
	t.Helper()
	return &k8sHandler{
		namespacer: claims.OrgNamespaceFormatter,
		gvr:        schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"},
		restConfig: func(context.Context) (*rest.Config, error) {
			return &rest.Config{Host: srvURL}, nil
		},
	}
}

// newWidget produces a JSON body for a fake "Widget" custom resource.
func newWidget(name string) []byte {
	body, _ := json.Marshal(map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]any{"key": "value"},
	})
	return body
}

func dynamicTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			// Distinguish list (collection) from individual get by trailing segment.
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			last := parts[len(parts)-1]
			if last == "widgets" {
				list, _ := json.Marshal(map[string]any{
					"apiVersion": "example.com/v1",
					"kind":       "WidgetList",
					"metadata":   map[string]any{"resourceVersion": "1"},
					"items":      []any{},
				})
				_, _ = w.Write(list)
				return
			}
			_, _ = w.Write(newWidget(last))
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			body, _ := io.ReadAll(r.Body)
			if len(body) == 0 {
				body = newWidget("created")
			}
			_, _ = w.Write(body)
		case http.MethodDelete:
			status, _ := json.Marshal(map[string]any{
				"kind":       "Status",
				"apiVersion": "v1",
				"status":     "Success",
			})
			_, _ = w.Write(status)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestK8sHandlerCRUD(t *testing.T) {
	srv := dynamicTestServer(t)
	h := newTestHandler(t, srv.URL)
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		got, err := h.Get(ctx, "foo", 1, v1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, "foo", got.GetName())
	})

	t.Run("Create success", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetUnstructuredContent(map[string]any{
			"apiVersion": "example.com/v1",
			"kind":       "Widget",
			"metadata":   map[string]any{"name": "bar", "namespace": "default"},
		})
		created, err := h.Create(ctx, obj, 1, v1.CreateOptions{})
		require.NoError(t, err)
		require.Equal(t, "bar", created.GetName())
	})

	t.Run("Update success", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetUnstructuredContent(map[string]any{
			"apiVersion": "example.com/v1",
			"kind":       "Widget",
			"metadata":   map[string]any{"name": "baz", "namespace": "default"},
		})
		updated, err := h.Update(ctx, obj, 1, v1.UpdateOptions{})
		require.NoError(t, err)
		require.Equal(t, "baz", updated.GetName())
	})

	t.Run("Delete success", func(t *testing.T) {
		require.NoError(t, h.Delete(ctx, "foo", 1, v1.DeleteOptions{}))
	})

	t.Run("DeleteCollection success", func(t *testing.T) {
		require.NoError(t, h.DeleteCollection(ctx, 1))
	})

	t.Run("List success", func(t *testing.T) {
		list, err := h.List(ctx, 1, v1.ListOptions{})
		require.NoError(t, err)
		require.NotNil(t, list)
		require.Empty(t, list.Items)
	})
}

func TestK8sHandlerRestConfigError(t *testing.T) {
	expected := errors.New("no config")
	h := &k8sHandler{
		namespacer: claims.OrgNamespaceFormatter,
		gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "r"},
		restConfig: func(context.Context) (*rest.Config, error) { return nil, expected },
	}
	ctx := context.Background()

	_, err := h.Get(ctx, "x", 1, v1.GetOptions{})
	require.ErrorIs(t, err, expected)

	_, err = h.Create(ctx, &unstructured.Unstructured{}, 1, v1.CreateOptions{})
	require.ErrorIs(t, err, expected)

	_, err = h.Update(ctx, &unstructured.Unstructured{}, 1, v1.UpdateOptions{})
	require.ErrorIs(t, err, expected)

	require.ErrorIs(t, h.Delete(ctx, "x", 1, v1.DeleteOptions{}), expected)
	require.ErrorIs(t, h.DeleteCollection(ctx, 1), expected)

	_, err = h.List(ctx, 1, v1.ListOptions{})
	require.ErrorIs(t, err, expected)
}

func TestK8sHandlerGetClientNewForConfigError(t *testing.T) {
	// dynamic.NewForConfig rejects configs with negative QPS / Burst combos.
	h := &k8sHandler{
		namespacer: claims.OrgNamespaceFormatter,
		gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "r"},
		restConfig: func(context.Context) (*rest.Config, error) {
			return &rest.Config{Host: "://bad-host"}, nil
		},
	}
	_, err := h.Get(context.Background(), "x", 1, v1.GetOptions{})
	require.Error(t, err)
}

func TestSearch(t *testing.T) {
	t.Run("populates default options and key", func(t *testing.T) {
		fake := &fakeIndexClient{
			searchResp: &resourcepb.ResourceSearchResponse{TotalHits: 7},
		}
		h := &k8sHandler{
			namespacer: claims.OrgNamespaceFormatter,
			gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "widgets"},
			searcher:   fake,
		}
		resp, err := h.Search(context.Background(), 1, &resourcepb.ResourceSearchRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(7), resp.TotalHits)
		require.NotNil(t, fake.searchReq.Options)
		require.NotNil(t, fake.searchReq.Options.Key)
		require.Equal(t, "default", fake.searchReq.Options.Key.Namespace)
		require.Equal(t, "g", fake.searchReq.Options.Key.Group)
		require.Equal(t, "widgets", fake.searchReq.Options.Key.Resource)
	})

	t.Run("preserves caller-provided key", func(t *testing.T) {
		fake := &fakeIndexClient{searchResp: &resourcepb.ResourceSearchResponse{}}
		h := &k8sHandler{
			namespacer: claims.OrgNamespaceFormatter,
			gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "widgets"},
			searcher:   fake,
		}
		req := &resourcepb.ResourceSearchRequest{
			Options: &resourcepb.ListOptions{
				Key: &resourcepb.ResourceKey{Namespace: "custom", Group: "x", Resource: "y"},
			},
		}
		_, err := h.Search(context.Background(), 1, req)
		require.NoError(t, err)
		require.Equal(t, "custom", fake.searchReq.Options.Key.Namespace)
	})

	t.Run("propagates error", func(t *testing.T) {
		fake := &fakeIndexClient{searchErr: errors.New("boom")}
		h := &k8sHandler{
			namespacer: claims.OrgNamespaceFormatter,
			gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "widgets"},
			searcher:   fake,
		}
		_, err := h.Search(context.Background(), 1, &resourcepb.ResourceSearchRequest{})
		require.Error(t, err)
	})
}

func TestGetStats(t *testing.T) {
	fake := &fakeIndexClient{statsResp: &resourcepb.ResourceStatsResponse{}}
	h := &k8sHandler{
		namespacer: claims.OrgNamespaceFormatter,
		gvr:        schema.GroupVersionResource{Group: "g", Version: "v", Resource: "widgets"},
		searcher:   fake,
	}
	_, err := h.GetStats(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "default", fake.statsReq.Namespace)
	require.Equal(t, []string{"g/widgets"}, fake.statsReq.Kinds)
}

func TestGetUsersFromMetaExtra(t *testing.T) {
	t.Run("bare meta short-circuits", func(t *testing.T) {
		userSvc := usertest.NewUserServiceFake()
		h := &k8sHandler{userService: userSvc}
		result, err := h.GetUsersFromMeta(context.Background(), []string{"bare"})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("user service error returns empty map and nil error", func(t *testing.T) {
		userSvc := usertest.NewUserServiceFake()
		userSvc.ExpectedError = errors.New("db down")
		h := &k8sHandler{userService: userSvc}
		result, err := h.GetUsersFromMeta(context.Background(), []string{"user:1"})
		require.NoError(t, err)
		require.Empty(t, result)
	})
}
