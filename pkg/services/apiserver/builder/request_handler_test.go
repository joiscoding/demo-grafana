package builder

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/spec3"
)

// rhTestBuilder is a minimal APIGroupBuilder used by request_handler tests.
type rhTestBuilder struct {
	gv schema.GroupVersion
}

func (b *rhTestBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }
func (b *rhTestBuilder) InstallSchema(*runtime.Scheme) error  { return nil }
func (b *rhTestBuilder) UpdateAPIGroupInfo(*genericapiserver.APIGroupInfo, APIGroupOptions) error {
	return nil
}
func (b *rhTestBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (b *rhTestBuilder) AllowedV0Alpha1Resources() []string                  { return nil }

// rhTestRouteBuilder also implements APIGroupRouteProvider.
type rhTestRouteBuilder struct {
	rhTestBuilder
	routes *APIRoutes
}

func (b *rhTestRouteBuilder) GetAPIRoutes(_ schema.GroupVersion) *APIRoutes { return b.routes }

func TestConvertHandlerToRouteFunction(t *testing.T) {
	t.Run("populates mux vars from path parameters", func(t *testing.T) {
		var seenName, seenNS string
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			seenName = vars["name"]
			seenNS = vars["namespace"]
			w.WriteHeader(http.StatusTeapot)
		})

		rf := convertHandlerToRouteFunction(h)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rr := httptest.NewRecorder()
		restfulReq := restful.NewRequest(req)
		restfulReq.PathParameters()["name"] = "alice"
		restfulReq.PathParameters()["namespace"] = "default"
		restfulResp := restful.NewResponse(rr)

		rf(restfulReq, restfulResp)

		require.Equal(t, http.StatusTeapot, rr.Code)
		require.Equal(t, "alice", seenName)
		require.Equal(t, "default", seenNS)
	})

	t.Run("no path parameters works", func(t *testing.T) {
		called := false
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			require.Empty(t, mux.Vars(r))
			w.WriteHeader(http.StatusOK)
		})

		rf := convertHandlerToRouteFunction(h)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rr := httptest.NewRecorder()
		rf(restful.NewRequest(req), restful.NewResponse(rr))

		require.True(t, called)
		require.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestExtractConsumesFromRequestBody(t *testing.T) {
	require.Nil(t, extractConsumesFromRequestBody(nil))
	require.Nil(t, extractConsumesFromRequestBody(&spec3.RequestBody{}))

	rb := &spec3.RequestBody{
		RequestBodyProps: spec3.RequestBodyProps{
			Content: map[string]*spec3.MediaType{
				"application/json": {},
				"application/xml":  {},
			},
		},
	}
	got := extractConsumesFromRequestBody(rb)
	sort.Strings(got)
	require.Equal(t, []string{"*/*", "application/json", "application/xml"}, got)
}

func TestExtractProducesFromResponses(t *testing.T) {
	require.Nil(t, extractProducesFromResponses(nil))
	require.Nil(t, extractProducesFromResponses(&spec3.Responses{}))

	resps := &spec3.Responses{
		ResponsesProps: spec3.ResponsesProps{
			Default: &spec3.Response{
				ResponseProps: spec3.ResponseProps{
					Content: map[string]*spec3.MediaType{
						"application/yaml": {},
					},
				},
			},
			StatusCodeResponses: map[int]*spec3.Response{
				200: {
					ResponseProps: spec3.ResponseProps{
						Content: map[string]*spec3.MediaType{
							"application/json": {},
							"application/yaml": {},
						},
					},
				},
				500: nil,
				404: {
					ResponseProps: spec3.ResponseProps{Content: nil},
				},
			},
		},
	}
	got := extractProducesFromResponses(resps)
	sort.Strings(got)
	require.Equal(t, []string{"application/json", "application/yaml"}, got)
}

func TestPrefixRouteIDWithK8sVerbIfNotPresent(t *testing.T) {
	require.Equal(t, "getStats", prefixRouteIDWithK8sVerbIfNotPresent("getStats", http.MethodGet))
	require.Equal(t, "createFoo", prefixRouteIDWithK8sVerbIfNotPresent("Foo", http.MethodPost))
	require.Equal(t, "replaceBar", prefixRouteIDWithK8sVerbIfNotPresent("Bar", http.MethodPut))
	require.Equal(t, "patchBaz", prefixRouteIDWithK8sVerbIfNotPresent("Baz", http.MethodPatch))
	require.Equal(t, "deleteQux", prefixRouteIDWithK8sVerbIfNotPresent("Qux", http.MethodDelete))
	require.Equal(t, "listThings", prefixRouteIDWithK8sVerbIfNotPresent("listThings", http.MethodGet))
}

func TestGenerateOperationNameFromPath(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"/search", "Search"},
		{"/snapshots/create", "SnapshotsCreate"},
		{"ofrep/v1/evaluate/flags", "OfrepEvaluateFlags"},
		{"ofrep/v1/evaluate/flags/{flagKey}", "OfrepEvaluateFlagsFlagKey"},
		{"/apis/foo/v0alpha1/namespaces/{namespace}/{name}/things", "FooThings"},
		{"/", "Route"},
	}
	for _, c := range cases {
		require.Equal(t, c.out, generateOperationNameFromPath(c.in), c.in)
	}
}

func TestAddRouteFromSpec(t *testing.T) {
	t.Run("nil pathProps returns error", func(t *testing.T) {
		ws := new(restful.WebService).Path("/apis/x.grafana.app/v1")
		err := addRouteFromSpec(ws, "foo", nil, func(*restful.Request, *restful.Response) {}, false)
		require.Error(t, err)
	})

	t.Run("registers methods using OpenAPI spec", func(t *testing.T) {
		ws := new(restful.WebService).Path("/apis/x.grafana.app/v1")
		props := &spec3.PathProps{
			Get: &spec3.Operation{
				OperationProps: spec3.OperationProps{
					OperationId: "getFoo",
					Description: "get foo",
					Parameters: []*spec3.Parameter{
						{ParameterProps: spec3.ParameterProps{Name: "namespace", In: "path"}},
						{ParameterProps: spec3.ParameterProps{Name: "q", In: "query", Description: "query"}},
						{ParameterProps: spec3.ParameterProps{Name: "X-Foo", In: "header"}},
					},
					Responses: &spec3.Responses{
						ResponsesProps: spec3.ResponsesProps{
							StatusCodeResponses: map[int]*spec3.Response{
								200: {ResponseProps: spec3.ResponseProps{Content: map[string]*spec3.MediaType{"application/json": {}}}},
							},
						},
					},
				},
			},
			Post: &spec3.Operation{
				OperationProps: spec3.OperationProps{
					RequestBody: &spec3.RequestBody{
						RequestBodyProps: spec3.RequestBodyProps{
							Content: map[string]*spec3.MediaType{"application/json": {}},
						},
					},
				},
			},
			Put:    &spec3.Operation{},
			Patch:  &spec3.Operation{},
			Delete: &spec3.Operation{},
		}

		err := addRouteFromSpec(ws, "foo", props, func(*restful.Request, *restful.Response) {}, true)
		require.NoError(t, err)

		routes := ws.Routes()
		require.Len(t, routes, 5)

		methods := map[string]bool{}
		for _, r := range routes {
			methods[r.Method] = true
			require.Contains(t, r.Path, "/namespaces/{namespace}/foo")
		}
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			require.True(t, methods[m], m)
		}
	})

	t.Run("non-namespaced path", func(t *testing.T) {
		ws := new(restful.WebService).Path("/apis/x.grafana.app/v1")
		props := &spec3.PathProps{Get: &spec3.Operation{}}
		err := addRouteFromSpec(ws, "search", props, func(*restful.Request, *restful.Response) {}, false)
		require.NoError(t, err)
		require.Equal(t, "/apis/x.grafana.app/v1/search", ws.Routes()[0].Path)
	})
}

func TestAugmentWebServicesWithCustomRoutes(t *testing.T) {
	gv := schema.GroupVersion{Group: "x.grafana.app", Version: "v1"}

	makeRoutes := func() *APIRoutes {
		return &APIRoutes{
			Root: []APIRouteHandler{
				{
					Path: "search",
					Spec: &spec3.PathProps{Get: &spec3.Operation{OperationProps: spec3.OperationProps{OperationId: "search"}}},
					Handler: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					},
				},
			},
			Namespace: []APIRouteHandler{
				{
					Path: "stats",
					Spec: &spec3.PathProps{Get: &spec3.Operation{OperationProps: spec3.OperationProps{OperationId: "stats"}}},
					Handler: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					},
				},
			},
		}
	}

	t.Run("nil container returns error", func(t *testing.T) {
		err := AugmentWebServicesWithCustomRoutes(nil, nil, nil, nil)
		require.Error(t, err)
	})

	t.Run("no builders is a no-op", func(t *testing.T) {
		c := restful.NewContainer()
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, nil, nil, nil))
	})

	t.Run("builder without route provider is skipped", func(t *testing.T) {
		c := restful.NewContainer()
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{&rhTestBuilder{gv: gv}}, nil, nil))
	})

	t.Run("adds routes by creating a new WebService", func(t *testing.T) {
		c := restful.NewContainer()
		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: makeRoutes()}
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, nil))

		var ws *restful.WebService
		for _, w := range c.RegisteredWebServices() {
			if w.RootPath() == "/apis/x.grafana.app/v1" {
				ws = w
				break
			}
		}
		require.NotNil(t, ws)
		paths := map[string]bool{}
		for _, r := range ws.Routes() {
			paths[r.Path] = true
		}
		require.True(t, paths["/apis/x.grafana.app/v1/search"])
		require.True(t, paths["/apis/x.grafana.app/v1/namespaces/{namespace}/stats"])
	})

	t.Run("reuses existing WebService when one is already registered", func(t *testing.T) {
		c := restful.NewContainer()
		existing := new(restful.WebService).Path("/apis/x.grafana.app/v1")
		c.Add(existing)

		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: makeRoutes()}
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, nil))
		require.GreaterOrEqual(t, len(existing.Routes()), 2)
	})

	t.Run("apiResourceConfig disables routes for disabled gv", func(t *testing.T) {
		c := restful.NewContainer()
		cfg := serverstorage.NewResourceConfig()
		cfg.DisableVersions(gv)

		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: makeRoutes()}
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, cfg))
		for _, w := range c.RegisteredWebServices() {
			require.NotEqual(t, "/apis/x.grafana.app/v1", w.RootPath())
		}
	})

	t.Run("nil routes from provider is skipped", func(t *testing.T) {
		c := restful.NewContainer()
		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: nil}
		require.NoError(t, AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, nil))
		for _, w := range c.RegisteredWebServices() {
			require.NotEqual(t, "/apis/x.grafana.app/v1", w.RootPath())
		}
	})

	t.Run("addRouteFromSpec error in root routes is surfaced", func(t *testing.T) {
		c := restful.NewContainer()
		bad := &APIRoutes{Root: []APIRouteHandler{{Path: "bad", Spec: nil, Handler: func(http.ResponseWriter, *http.Request) {}}}}
		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: bad}
		err := AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, nil)
		require.Error(t, err)
	})

	t.Run("addRouteFromSpec error in namespace routes is surfaced", func(t *testing.T) {
		c := restful.NewContainer()
		bad := &APIRoutes{Namespace: []APIRouteHandler{{Path: "bad", Spec: nil, Handler: func(http.ResponseWriter, *http.Request) {}}}}
		b := &rhTestRouteBuilder{rhTestBuilder: rhTestBuilder{gv: gv}, routes: bad}
		err := AugmentWebServicesWithCustomRoutes(c, []APIGroupBuilder{b}, nil, nil)
		require.Error(t, err)
	})
}

// allowRegisteringResourceByInfo lives in helper.go but is unexported, so we
// cover it from this internal test file.
func TestAllowRegisteringResourceByInfo(t *testing.T) {
	require.True(t, allowRegisteringResourceByInfo([]string{AllResourcesAllowed}, "foos"))
	require.True(t, allowRegisteringResourceByInfo([]string{"foos"}, "foos/sub"))
	require.False(t, allowRegisteringResourceByInfo([]string{"bars"}, "foos"))
	require.False(t, allowRegisteringResourceByInfo(nil, "foos"))
}
