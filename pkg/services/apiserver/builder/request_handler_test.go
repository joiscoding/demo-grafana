package builder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/spec3"
)

func TestGenerateOperationNameFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{path: "/search", want: "Search"},
		{path: "/snapshots/create", want: "SnapshotsCreate"},
		{path: "ofrep/v1/evaluate/flags", want: "OfrepEvaluateFlags"},
		{path: "ofrep/v1/evaluate/flags/{flagKey}", want: "OfrepEvaluateFlagsFlagKey"},
		{path: "/namespaces/{namespace}/items", want: "Items"},
		{path: "", want: "Route"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, generateOperationNameFromPath(tt.path))
		})
	}
}

func TestPrefixRouteIDWithK8sVerbIfNotPresent(t *testing.T) {
	t.Parallel()

	require.Equal(t, "getFoo", prefixRouteIDWithK8sVerbIfNotPresent("getFoo", http.MethodGet))
	require.Equal(t, "createMyOperation", prefixRouteIDWithK8sVerbIfNotPresent("MyOperation", http.MethodPost))
}

func TestExtractConsumesFromRequestBody(t *testing.T) {
	t.Parallel()

	require.Nil(t, extractConsumesFromRequestBody(nil))

	consumes := extractConsumesFromRequestBody(&spec3.RequestBody{
		RequestBodyProps: spec3.RequestBodyProps{
			Content: map[string]*spec3.MediaType{
				"application/json": {},
			},
		},
	})
	require.Equal(t, []string{"application/json", "*/*"}, consumes)
}

func TestExtractProducesFromResponses(t *testing.T) {
	t.Parallel()

	require.Nil(t, extractProducesFromResponses(nil))

	produces := extractProducesFromResponses(&spec3.Responses{
		ResponsesProps: spec3.ResponsesProps{
			StatusCodeResponses: map[int]*spec3.Response{
				http.StatusOK: {
					ResponseProps: spec3.ResponseProps{
						Content: map[string]*spec3.MediaType{
							"application/json": {},
						},
					},
				},
			},
		},
	})
	require.ElementsMatch(t, []string{"application/json"}, produces)
}

func TestConvertHandlerToRouteFunction(t *testing.T) {
	t.Parallel()

	var capturedVars map[string]string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedVars = mux.Vars(r)
	})

	routeFn := convertHandlerToRouteFunction(handler)

	req := httptest.NewRequest(http.MethodGet, "/items/abc", nil)
	resp := restful.NewResponse(httptest.NewRecorder())
	restfulReq := restful.NewRequest(req)
	restfulReq.PathParameters()["name"] = "abc"

	routeFn(restfulReq, resp)
	require.Equal(t, map[string]string{"name": "abc"}, capturedVars)
}

func TestAddRouteFromSpec(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)
	ws.Path("/apis/example.grafana.app/v1")

	pathProps := &spec3.PathProps{
		Get: &spec3.Operation{
			OperationProps: spec3.OperationProps{
				OperationId: "getStats",
				Description: "get stats",
			},
		},
	}

	handler := restful.RouteFunction(func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	})

	require.NoError(t, addRouteFromSpec(ws, "stats", pathProps, handler, false))
	require.Len(t, ws.Routes(), 1)
	require.Equal(t, "GET", ws.Routes()[0].Method)
	require.Contains(t, ws.Routes()[0].Path, "/stats")
}

func TestAddRouteFromSpec_Namespaced(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)
	ws.Path("/apis/example.grafana.app/v1")

	pathProps := &spec3.PathProps{
		Post: &spec3.Operation{
			OperationProps: spec3.OperationProps{
				OperationId: "createItem",
				Parameters: []*spec3.Parameter{
					{
						ParameterProps: spec3.ParameterProps{
							Name:        "namespace",
							In:          "path",
							Description: "namespace",
						},
					},
				},
				RequestBody: &spec3.RequestBody{
					RequestBodyProps: spec3.RequestBodyProps{
						Content: map[string]*spec3.MediaType{
							"application/json": {},
						},
					},
				},
				Responses: &spec3.Responses{
					ResponsesProps: spec3.ResponsesProps{
						Default: &spec3.Response{
							ResponseProps: spec3.ResponseProps{
								Content: map[string]*spec3.MediaType{
									"application/json": {},
								},
							},
						},
					},
				},
			},
		},
	}

	handler := restful.RouteFunction(func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusCreated)
	})

	require.NoError(t, addRouteFromSpec(ws, "items", pathProps, handler, true))
	require.Len(t, ws.Routes(), 1)
	require.Equal(t, "POST", ws.Routes()[0].Method)
	require.Contains(t, ws.Routes()[0].Path, "/namespaces/{namespace}/items")
}

func TestAddRouteFromSpec_NilPathProps(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)
	err := addRouteFromSpec(ws, "stats", nil, nil, false)
	require.Error(t, err)
}

func TestAugmentWebServicesWithCustomRoutes(t *testing.T) {
	t.Parallel()

	t.Run("nil container", func(t *testing.T) {
		t.Parallel()
		err := AugmentWebServicesWithCustomRoutes(nil, nil, prometheus.NewRegistry(), nil)
		require.Error(t, err)
	})

	t.Run("adds root route", func(t *testing.T) {
		t.Parallel()
		container := restful.NewContainer()
		gv := schema.GroupVersion{Group: "example.grafana.app", Version: "v1"}

		builder := &mockRouteBuilder{
			gv: gv,
			routes: &APIRoutes{
				Root: []APIRouteHandler{
					{
						Path: "stats",
						Spec: &spec3.PathProps{
							Get: &spec3.Operation{
								OperationProps: spec3.OperationProps{
									OperationId: "getStats",
								},
							},
						},
						Handler: func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusOK)
						},
					},
				},
			},
		}

		require.NoError(t, AugmentWebServicesWithCustomRoutes(container, []APIGroupBuilder{builder}, prometheus.NewRegistry(), nil))

		rootPath := "/apis/" + gv.String()
		var found bool
		for _, ws := range container.RegisteredWebServices() {
			if ws.RootPath() != rootPath {
				continue
			}
			found = true
			require.NotEmpty(t, ws.Routes())
		}
		require.True(t, found)
	})

	t.Run("builder without routes is no-op", func(t *testing.T) {
		t.Parallel()
		container := restful.NewContainer()
		builder := &mockRouteBuilder{
			gv:     schema.GroupVersion{Group: "example.grafana.app", Version: "v1"},
			routes: nil,
		}
		require.NoError(t, AugmentWebServicesWithCustomRoutes(container, []APIGroupBuilder{builder}, prometheus.NewRegistry(), nil))
	})

	t.Run("skips disabled group version", func(t *testing.T) {
		t.Parallel()
		container := restful.NewContainer()
		gv := schema.GroupVersion{Group: "example.grafana.app", Version: "v1"}
		resourceConfig := serverstorage.NewResourceConfig()
		resourceConfig.DisableVersions(gv)

		builder := &mockRouteBuilder{
			gv: gv,
			routes: &APIRoutes{
				Root: []APIRouteHandler{
					{
						Path: "stats",
						Spec: &spec3.PathProps{
							Get: &spec3.Operation{
								OperationProps: spec3.OperationProps{OperationId: "getStats"},
							},
						},
						Handler: func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusOK)
						},
					},
				},
			},
		}

		require.NoError(t, AugmentWebServicesWithCustomRoutes(container, []APIGroupBuilder{builder}, prometheus.NewRegistry(), resourceConfig))
		for _, ws := range container.RegisteredWebServices() {
			require.NotEqual(t, "/apis/"+gv.String(), ws.RootPath())
		}
	})
}

type mockRouteBuilder struct {
	gv     schema.GroupVersion
	routes *APIRoutes
}

func (m *mockRouteBuilder) GetGroupVersion() schema.GroupVersion {
	return m.gv
}

func (m *mockRouteBuilder) GetAPIRoutes(_ schema.GroupVersion) *APIRoutes {
	return m.routes
}

func (m *mockRouteBuilder) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *mockRouteBuilder) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ APIGroupOptions) error {
	return nil
}

func (m *mockRouteBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *mockRouteBuilder) AllowedV0Alpha1Resources() []string { return nil }
