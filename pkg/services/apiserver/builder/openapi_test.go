package builder

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/spec3"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

func TestOpenAPI_GetPathOperations(t *testing.T) {
	testCases := []struct {
		name    string
		input   *spec3.Path
		expect  []string // the methods we should see
		exclude []string // the methods we should never see
	}{
		{
			name: "some operations",
			input: &spec3.Path{
				PathProps: spec3.PathProps{
					Get:    &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "get"}},
					Post:   &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "post"}},
					Delete: &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "delete"}},
				},
			},
			expect:  []string{"GET", "POST", "DELETE"},
			exclude: []string{"PUT", "PATCH", "OPTIONS", "HEAD", "TRACE"},
		},
		{
			name: "all operations",
			input: &spec3.Path{
				PathProps: spec3.PathProps{
					Get:     &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "get"}},
					Post:    &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "post"}},
					Delete:  &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "delete"}},
					Put:     &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "put"}},
					Patch:   &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "patch"}},
					Options: &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "options"}},
					Head:    &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "head"}},
					Trace:   &spec3.Operation{OperationProps: spec3.OperationProps{Summary: "trace"}},
				},
			},
			expect:  []string{"GET", "POST", "DELETE", "PUT", "PATCH", "OPTIONS", "HEAD", "TRACE"},
			exclude: []string{},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			expect := make(map[string]bool)
			for _, k := range tt.expect {
				expect[k] = true
			}

			for k, op := range GetPathOperations(tt.input) {
				require.NotNil(t, op)
				require.Equal(t, strings.ToLower(k), op.Summary)

				if !expect[k] {
					if slices.Contains(tt.expect, k) {
						require.Fail(t, "method returned multiple times", k)
					} else {
						require.Fail(t, "unexpected method", k)
					}
				}
				delete(expect, k)
				require.NotContains(t, tt.exclude, k, "exclude")
			}

			if len(expect) > 0 {
				require.Fail(t, "missing expected method", expect)
			}
		})
	}
}

// openapiBuilder is a minimal APIGroupBuilder used in openapi tests.
type openapiBuilder struct {
	gv     schema.GroupVersion
	defs   openapicommon.GetOpenAPIDefinitions
	routes *APIRoutes
	pp     func(*spec3.OpenAPI) (*spec3.OpenAPI, error)
}

func (b *openapiBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }
func (b *openapiBuilder) InstallSchema(*runtime.Scheme) error  { return nil }
func (b *openapiBuilder) UpdateAPIGroupInfo(*genericapiserver.APIGroupInfo, APIGroupOptions) error {
	return nil
}
func (b *openapiBuilder) GetOpenAPIDefinitions() openapicommon.GetOpenAPIDefinitions { return b.defs }
func (b *openapiBuilder) AllowedV0Alpha1Resources() []string                         { return nil }
func (b *openapiBuilder) GetAPIRoutes(_ schema.GroupVersion) *APIRoutes              { return b.routes }
func (b *openapiBuilder) PostProcessOpenAPI(s *spec3.OpenAPI) (*spec3.OpenAPI, error) {
	if b.pp == nil {
		return s, nil
	}
	return b.pp(s)
}

// pureBuilder doesn't implement APIGroupRouteProvider or OpenAPIPostProcessor.
type pureBuilder struct {
	gv   schema.GroupVersion
	defs openapicommon.GetOpenAPIDefinitions
}

func (b *pureBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }
func (b *pureBuilder) InstallSchema(*runtime.Scheme) error  { return nil }
func (b *pureBuilder) UpdateAPIGroupInfo(*genericapiserver.APIGroupInfo, APIGroupOptions) error {
	return nil
}
func (b *pureBuilder) GetOpenAPIDefinitions() openapicommon.GetOpenAPIDefinitions { return b.defs }
func (b *pureBuilder) AllowedV0Alpha1Resources() []string                         { return nil }

func TestGetOpenAPIDefinitionsBuilder(t *testing.T) {
	gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}

	// builder that contributes a custom definition
	const customKey = "ex.grafana.app/v1.Foo"
	bWithDefs := &pureBuilder{gv: gv, defs: func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		return map[string]openapicommon.OpenAPIDefinition{customKey: {}}
	}}
	// builder that returns nil from GetOpenAPIDefinitions to exercise the nil branch
	bNoDefs := &pureBuilder{gv: gv, defs: nil}

	// additional getters: include a nil one to exercise the nil branch
	const addKey = "additional/Bar"
	addGetter := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		return map[string]openapicommon.OpenAPIDefinition{addKey: {}}
	}

	get := GetOpenAPIDefinitions([]APIGroupBuilder{bWithDefs, bNoDefs}, addGetter, nil)
	defs := get(func(s string) spec.Ref { return spec.Ref{} })

	_, hasCustom := defs[customKey]
	_, hasAdditional := defs[addKey]
	require.True(t, hasCustom, "expected custom key from builder")
	require.True(t, hasAdditional, "expected key from additional getter")
	require.NotEmpty(t, defs, "expected baseline definitions to be present")
}

func TestIsAllRoute(t *testing.T) {
	prefix := "/apis/x.grafana.app/v1/"
	paths := map[string]*spec3.Path{
		prefix + "namespaces/{namespace}/foos": {},
		prefix + "foos":                        {},
		prefix + "cluster-only":                {},
	}

	require.True(t, isAllRoute(prefix, prefix+"foos", paths))
	require.False(t, isAllRoute(prefix, prefix+"cluster-only", paths))
	require.False(t, isAllRoute(prefix, prefix+"namespaces/{namespace}/foos", paths))
}

func TestAddBuilderRoutes(t *testing.T) {
	gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}
	other := schema.GroupVersion{Group: "other.grafana.app", Version: "v1"}

	t.Run("adds root and namespace routes for matching gv", func(t *testing.T) {
		b := &openapiBuilder{
			gv: gv,
			routes: &APIRoutes{
				Root:      []APIRouteHandler{{Path: "search", Spec: &spec3.PathProps{}}},
				Namespace: []APIRouteHandler{{Path: "stats", Spec: &spec3.PathProps{}}},
			},
		}
		s := &spec3.OpenAPI{Paths: &spec3.Paths{Paths: map[string]*spec3.Path{}}}
		out, err := addBuilderRoutes(gv, s, []APIGroupBuilder{b}, nil)
		require.NoError(t, err)
		require.NotNil(t, out.Paths.Paths["/apis/ex.grafana.app/v1/search"])
		require.NotNil(t, out.Paths.Paths["/apis/ex.grafana.app/v1/namespaces/{namespace}/stats"])
	})

	t.Run("skips builders for non-matching gv", func(t *testing.T) {
		b := &openapiBuilder{gv: other, routes: &APIRoutes{Root: []APIRouteHandler{{Path: "search", Spec: &spec3.PathProps{}}}}}
		s := &spec3.OpenAPI{Paths: &spec3.Paths{Paths: map[string]*spec3.Path{
			"/apis/ex.grafana.app/v1/foo": {},
		}}}
		out, err := addBuilderRoutes(gv, s, []APIGroupBuilder{b}, nil)
		require.NoError(t, err)
		// the foo path remains
		_, ok := out.Paths.Paths["/apis/ex.grafana.app/v1/foo"]
		require.True(t, ok)
	})

	t.Run("post processor is invoked and can return error", func(t *testing.T) {
		b := &openapiBuilder{
			gv:     gv,
			routes: &APIRoutes{},
			pp: func(s *spec3.OpenAPI) (*spec3.OpenAPI, error) {
				return nil, errors.New("boom")
			},
		}
		_, err := addBuilderRoutes(gv, &spec3.OpenAPI{Paths: &spec3.Paths{Paths: map[string]*spec3.Path{}}}, []APIGroupBuilder{b}, nil)
		require.Error(t, err)
	})

	t.Run("filters disabled groups from openapi", func(t *testing.T) {
		cfg := serverstorage.NewResourceConfig()
		cfg.DisableVersions(gv)
		s := &spec3.OpenAPI{Paths: &spec3.Paths{Paths: map[string]*spec3.Path{
			"/apis/ex.grafana.app/v1/foo":   {},
			"/apis/other.grafana.app/v1/x":  {},
			"/apis/ex.grafana.app/v1/":      {},
		}}}
		out, err := addBuilderRoutes(gv, s, nil, cfg)
		require.NoError(t, err)
		_, present := out.Paths.Paths["/apis/ex.grafana.app/v1/foo"]
		require.False(t, present, "disabled group path should be removed")
		_, otherPresent := out.Paths.Paths["/apis/other.grafana.app/v1/x"]
		require.True(t, otherPresent)
	})
}

func TestGetOpenAPIPostProcessor(t *testing.T) {
	gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}
	prefix := "/apis/" + gv.String() + "/"

	t.Run("nil paths returns original spec", func(t *testing.T) {
		p := getOpenAPIPostProcessor("1.0", nil, []schema.GroupVersion{gv}, nil)
		in := &spec3.OpenAPI{}
		out, err := p(in)
		require.NoError(t, err)
		require.Same(t, in, out)
	})

	t.Run("no matching prefix returns original spec", func(t *testing.T) {
		p := getOpenAPIPostProcessor("1.0", nil, []schema.GroupVersion{gv}, nil)
		in := &spec3.OpenAPI{Paths: &spec3.Paths{Paths: map[string]*spec3.Path{"/apis/other/v1/": {}}}}
		out, err := p(in)
		require.NoError(t, err)
		require.Same(t, in, out)
	})

	t.Run("processes spec for matching gv", func(t *testing.T) {
		// Build a spec that exercises most branches.
		discoveryGet := &spec3.Operation{}
		discovery := &spec3.Path{PathProps: spec3.PathProps{Get: discoveryGet}}

		deleteOp := &spec3.Operation{}
		deleteOp.Extensions = spec.Extensions{}
		deleteOp.Extensions.Add("x-kubernetes-action", "delete")
		deleteOp.RequestBody = &spec3.RequestBody{
			RequestBodyProps: spec3.RequestBodyProps{
				Content: map[string]*spec3.MediaType{
					"*/*": {},
				},
			},
		}

		putOp := &spec3.Operation{}
		putOp.RequestBody = &spec3.RequestBody{
			RequestBodyProps: spec3.RequestBodyProps{
				Content: map[string]*spec3.MediaType{"*/*": {}},
			},
		}

		fooPath := &spec3.Path{PathProps: spec3.PathProps{Delete: deleteOp, Put: putOp}}
		// "for all namespaces" path that has a namespaced counterpart — should be removed
		allNS := &spec3.Path{}
		nsPath := &spec3.Path{}
		// watch route — should be removed
		watchPath := &spec3.Path{}

		// connect sub-resource that should inherit parent tags
		parentOp := &spec3.Operation{OperationProps: spec3.OperationProps{Tags: []string{"FooKind"}}}
		parent := &spec3.Path{PathProps: spec3.PathProps{Get: parentOp}}
		connectOp := &spec3.Operation{}
		connectOp.Extensions = spec.Extensions{}
		connectOp.Extensions.Add("x-kubernetes-action", "connect")
		connectPath := &spec3.Path{PathProps: spec3.PathProps{Get: connectOp}}

		// protobuf response to be removed
		respWithProto := &spec3.Operation{
			OperationProps: spec3.OperationProps{
				Responses: &spec3.Responses{
					ResponsesProps: spec3.ResponsesProps{
						StatusCodeResponses: map[int]*spec3.Response{
							200: {ResponseProps: spec3.ResponseProps{Content: map[string]*spec3.MediaType{
								"application/json":                       {},
								"application/vnd.kubernetes.protobuf":    {},
								"application/vnd.kubernetes.protobuf;stream=watch": {},
							}}},
						},
						Default: &spec3.Response{ResponseProps: spec3.ResponseProps{Content: map[string]*spec3.MediaType{
							"application/json":                    {},
							"application/vnd.kubernetes.protobuf": {},
						}}},
					},
				},
				RequestBody: &spec3.RequestBody{
					RequestBodyProps: spec3.RequestBodyProps{
						Content: map[string]*spec3.MediaType{
							"application/json":                    {},
							"application/vnd.kubernetes.protobuf": {},
						},
					},
				},
			},
		}
		protoPath := &spec3.Path{PathProps: spec3.PathProps{Get: respWithProto}}

		paths := map[string]*spec3.Path{
			prefix:                                                  discovery,
			prefix + "foos":                                         fooPath,
			prefix + "foos2":                                        allNS,
			prefix + "namespaces/{namespace}/foos2":                 nsPath,
			prefix + "watch/foos":                                   watchPath,
			prefix + "namespaces/{namespace}/foos/{name}":      parent,
			prefix + "namespaces/{namespace}/foos/{name}/logs": connectPath,
			prefix + "withproto":                                    protoPath,
		}

		// Schemas to exercise extension manipulation
		schemas := map[string]*spec.Schema{
			"io.k8s.apimachinery.pkg.apis.meta.v1.Foo": {
				VendorExtensible: spec.VendorExtensible{Extensions: spec.Extensions{"x-kubernetes-group-version-kind": []any{}}},
			},
			"ex.grafana.app.v1.Foo": {
				VendorExtensible: spec.VendorExtensible{Extensions: spec.Extensions{"x-kubernetes-group-version-kind": []any{
					map[string]any{"group": gv.Group, "version": "v1", "kind": "Foo"},
					map[string]any{"group": gv.Group, "version": "__internal", "kind": "Foo"},
					map[string]any{"group": "other", "version": "v1", "kind": "Foo"},
				}}},
			},
			"no-extensions": {},
		}

		in := &spec3.OpenAPI{
			Version:    "3.0.0",
			Paths:      &spec3.Paths{Paths: paths},
			Components: &spec3.Components{Schemas: schemas},
		}

		// builder providing additional root route via addBuilderRoutes
		b := &openapiBuilder{
			gv: gv,
			routes: &APIRoutes{
				Root: []APIRouteHandler{{Path: "extra", Spec: &spec3.PathProps{Get: &spec3.Operation{
					OperationProps: spec3.OperationProps{
						RequestBody: &spec3.RequestBody{
							RequestBodyProps: spec3.RequestBodyProps{
								Content: map[string]*spec3.MediaType{
									"application/vnd.kubernetes.protobuf": {},
									"application/json":                    {},
								},
							},
						},
					},
				}}}},
			},
		}

		p := getOpenAPIPostProcessor("1.2.3", []APIGroupBuilder{b}, []schema.GroupVersion{gv}, nil)
		out, err := p(in)
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Equal(t, "1.2.3", out.Info.Version)
		require.Equal(t, gv.String(), out.Info.Title)

		// watch path removed
		_, hasWatch := out.Paths.Paths[prefix+"watch/foos"]
		require.False(t, hasWatch)
		// all-NS path removed because namespaced counterpart exists
		_, hasAllNS := out.Paths.Paths[prefix+"foos2"]
		require.False(t, hasAllNS)
		// namespaced one still present
		_, hasNS := out.Paths.Paths[prefix+"namespaces/{namespace}/foos2"]
		require.True(t, hasNS)
		// delete op request body cleared
		require.Nil(t, out.Paths.Paths[prefix+"foos"].Delete.RequestBody)
		// */* expanded for put
		putContent := out.Paths.Paths[prefix+"foos"].Put.RequestBody.Content
		require.Contains(t, putContent, "application/json")
		require.Contains(t, putContent, "application/yaml")
		// discovery tags rewritten
		require.Equal(t, []string{"API Discovery"}, out.Paths.Paths[prefix].Get.Tags)
		// connect sub inherits parent tags
		require.Equal(t, []string{"FooKind"}, out.Paths.Paths[prefix+"namespaces/{namespace}/foos/{name}/logs"].Get.Tags)
		// meta schema extension stripped
		_, hasMetaExt := schemas["io.k8s.apimachinery.pkg.apis.meta.v1.Foo"].Extensions["x-kubernetes-group-version-kind"]
		require.False(t, hasMetaExt)
		// for ex.grafana.app schema, internal/other-group entries filtered
		gvks := schemas["ex.grafana.app.v1.Foo"].Extensions["x-kubernetes-group-version-kind"].([]map[string]any)
		require.Len(t, gvks, 1)
		require.Equal(t, "v1", gvks[0]["version"])
		// extra builder route is registered and its protobuf content type stripped
		extra, ok := out.Paths.Paths[prefix+"extra"]
		require.True(t, ok)
		require.NotContains(t, extra.Get.RequestBody.Content, "application/vnd.kubernetes.protobuf")
		// protobuf removed from existing path
		protoOut := out.Paths.Paths[prefix+"withproto"].Get
		require.NotContains(t, protoOut.RequestBody.Content, "application/vnd.kubernetes.protobuf")
		require.NotContains(t, protoOut.Responses.StatusCodeResponses[200].Content, "application/vnd.kubernetes.protobuf")
		require.NotContains(t, protoOut.Responses.StatusCodeResponses[200].Content, "application/vnd.kubernetes.protobuf;stream=watch")
		require.NotContains(t, protoOut.Responses.Default.Content, "application/vnd.kubernetes.protobuf")
	})

	t.Run("addBuilderRoutes error is propagated", func(t *testing.T) {
		b := &openapiBuilder{
			gv:     gv,
			routes: &APIRoutes{},
			pp: func(*spec3.OpenAPI) (*spec3.OpenAPI, error) {
				return nil, errors.New("boom")
			},
		}
		p := getOpenAPIPostProcessor("1", []APIGroupBuilder{b}, []schema.GroupVersion{gv}, nil)
		in := &spec3.OpenAPI{
			Paths:      &spec3.Paths{Paths: map[string]*spec3.Path{prefix: {}}},
			Components: &spec3.Components{},
		}
		_, err := p(in)
		require.Error(t, err)
	})
}
