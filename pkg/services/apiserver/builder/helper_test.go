package builder_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/audit"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/apiserver/endpoints/filters"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/storage/legacysql/dualwrite"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
)

func TestAddPostStartHooks(t *testing.T) {
	tests := []struct {
		name      string
		builders  []builder.APIGroupBuilder
		wantErr   bool
		wantHooks []string
	}{
		{
			name:     "no builders",
			builders: []builder.APIGroupBuilder{},
			wantErr:  false,
		},
		{
			name: "builder without post start hooks",
			builders: []builder.APIGroupBuilder{
				&mockAPIGroupPostStartHookProvider{},
			},
			wantErr: false,
		},
		{
			name: "builder that does not implement APIGroupPostStartHookProvider",
			builders: []builder.APIGroupBuilder{
				&plainBuilder{gv: schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}},
			},
			wantErr: false,
		},
		{
			name: "builder with post start hooks",
			builders: []builder.APIGroupBuilder{
				&mockAPIGroupPostStartHookProvider{
					hooks: map[string]server.PostStartHookFunc{
						"test-hook": func(server.PostStartHookContext) error { return nil },
					},
				},
			},
			wantErr:   false,
			wantHooks: []string{"test-hook"},
		},
		{
			name: "builder with post start hook provider error",
			builders: []builder.APIGroupBuilder{
				&mockAPIGroupPostStartHookProvider{
					hooks: map[string]server.PostStartHookFunc{},
					err:   errors.New("hook provider error"),
				},
			},
			wantErr: true,
		},
		{
			name: "two builders registering the same hook surface AddPostStartHook error",
			builders: []builder.APIGroupBuilder{
				&mockAPIGroupPostStartHookProvider{hooks: map[string]server.PostStartHookFunc{
					"dup-hook": func(server.PostStartHookContext) error { return nil },
				}},
				&mockAPIGroupPostStartHookProvider{hooks: map[string]server.PostStartHookFunc{
					"dup-hook": func(server.PostStartHookContext) error { return nil },
				}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := builder.ProvideScheme()
			codecs := builder.ProvideCodecFactory(scheme)
			config := server.NewRecommendedConfig(codecs)
			err := builder.AddPostStartHooks(config, tt.builders)
			if tt.wantErr {
				require.Error(t, err)
			}

			if len(tt.wantHooks) > 0 {
				for _, hookName := range tt.wantHooks {
					_, ok := config.PostStartHooks[hookName]
					require.True(t, ok)
				}
			}
		})
	}
}

func TestRedirection(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(r.URL.Path))
		require.NoError(t, err)
	})
	handler := filters.WithPathRewriters(mockHandler, builder.PathRewriters)
	tests := []struct {
		name   string
		url    string
		expect string
	}{
		{
			name:   "query connections",
			url:    "/apis/query.grafana.app/v0alpha1/namespaces/default/connections",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/connections",
		}, {
			name:   "query (with name hack)",
			url:    "/apis/query.grafana.app/v0alpha1/namespaces/default/query",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/name",
		}, {
			name:   "query sqlschemas at root",
			url:    "/apis/query.grafana.app/v0alpha1/namespaces/default/sqlschemas",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/sqlschemas",
		}, {
			name:   "query sqlschemas as subresource",
			url:    "/apis/query.grafana.app/v0alpha1/namespaces/default/query/sqlschemas",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/sqlschemas",
		}, {
			name:   "name hack in datasource service",
			url:    "/apis/query.grafana.app/v0alpha1/namespaces/default/query",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/name", // hack :(
		}, {
			name:   "datasource connections",
			url:    "/apis/datasource.grafana.app/v0alpha1/namespaces/default/connections",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/connections",
		}, {
			name:   "datasource query",
			url:    "/apis/datasource.grafana.app/v0alpha1/namespaces/default/query",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/name",
		}, {
			name:   "datasource sqlschemas migrated and rewritten",
			url:    "/apis/datasource.grafana.app/v0alpha1/namespaces/default/sqlschemas",
			expect: "/apis/query.grafana.app/v0alpha1/namespaces/default/query/sqlschemas",
		}, {
			name:   "scope find rewritten to subresource",
			url:    "/apis/scope.grafana.app/v0alpha1/namespaces/default/find/something",
			expect: "/apis/scope.grafana.app/v0alpha1/namespaces/default/something/name",
		}, {
			name:   "queryconvert appends name segment",
			url:    "/apis/x.grafana.app/v0alpha1/namespaces/default/queryconvert",
			expect: "/apis/x.grafana.app/v0alpha1/namespaces/default/queryconvert/name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, tt.expect, rr.Body.String())
		})
	}
}

var _ builder.APIGroupBuilder = &mockAPIGroupPostStartHookProvider{}
var _ builder.APIGroupPostStartHookProvider = &mockAPIGroupPostStartHookProvider{}

type mockAPIGroupPostStartHookProvider struct {
	hooks map[string]server.PostStartHookFunc
	err   error
}

func (m *mockAPIGroupPostStartHookProvider) GetPostStartHooks() (map[string]server.PostStartHookFunc, error) {
	return m.hooks, m.err
}

func (m *mockAPIGroupPostStartHookProvider) GetGroupVersion() schema.GroupVersion {
	return schema.GroupVersion{}
}

func (m *mockAPIGroupPostStartHookProvider) InstallSchema(scheme *runtime.Scheme) error {
	return nil
}

func (m *mockAPIGroupPostStartHookProvider) AllowedV0Alpha1Resources() []string {
	return nil
}

func (m *mockAPIGroupPostStartHookProvider) UpdateAPIGroupInfo(apiGroupInfo *server.APIGroupInfo, opts builder.APIGroupOptions) error {
	return nil
}

func (m *mockAPIGroupPostStartHookProvider) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return nil
}

func (m *mockAPIGroupPostStartHookProvider) GetAuthorizer() authorizer.Authorizer {
	return nil
}

func TestProvideDefaultBuildHandlerChainFuncFromBuilders(t *testing.T) {
	f := builder.ProvideDefaultBuildHandlerChainFuncFromBuilders()
	require.NotNil(t, f)
}

func TestGetDefaultBuildHandlerChainFunc(t *testing.T) {
	chainFunc := builder.GetDefaultBuildHandlerChainFunc(nil, prometheus.NewRegistry())
	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	cfg := server.NewRecommendedConfig(codecs)
	cfg.FeatureGate = utilfeature.DefaultFeatureGate
	cfg.Authorization.Authorizer = authorizer.AuthorizerFunc(func(_ context.Context, _ authorizer.Attributes) (authorizer.Decision, string, error) {
		return authorizer.DecisionAllow, "", nil
	})
	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chainFunc(delegate, &cfg.Config)
	require.NotNil(t, h)
}

// helperBuilder is a richer test builder used to drive SetupConfig / InstallAPIs.
type helperBuilder struct {
	gv          schema.GroupVersion
	updateErr   error
	updateImpl  func(g *server.APIGroupInfo, opts builder.APIGroupOptions) error
	postHooks   map[string]server.PostStartHookFunc
	postHookErr error
	policy      audit.PolicyRuleEvaluator
	allowedV0   []string
}

func (b *helperBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }
func (b *helperBuilder) InstallSchema(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(b.gv, &metav1.Status{})
	metav1.AddToGroupVersion(scheme, b.gv)
	return nil
}
func (b *helperBuilder) UpdateAPIGroupInfo(g *server.APIGroupInfo, opts builder.APIGroupOptions) error {
	if b.updateErr != nil {
		return b.updateErr
	}
	if b.updateImpl != nil {
		return b.updateImpl(g, opts)
	}
	return nil
}
func (b *helperBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (b *helperBuilder) AllowedV0Alpha1Resources() []string                  { return b.allowedV0 }

func (b *helperBuilder) GetPostStartHooks() (map[string]server.PostStartHookFunc, error) {
	if b.postHooks == nil && b.postHookErr == nil {
		return nil, nil
	}
	return b.postHooks, b.postHookErr
}

func (b *helperBuilder) GetPolicyRuleEvaluator() audit.PolicyRuleEvaluator { return b.policy }

// plainBuilder implements only the bare APIGroupBuilder + APIGroupVersionProvider
// interfaces. It is used to exercise the branch in AddPostStartHooks where a
// builder does not implement APIGroupPostStartHookProvider.
type plainBuilder struct {
	gv schema.GroupVersion
}

func (b *plainBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }
func (b *plainBuilder) InstallSchema(*runtime.Scheme) error  { return nil }
func (b *plainBuilder) UpdateAPIGroupInfo(*server.APIGroupInfo, builder.APIGroupOptions) error {
	return nil
}
func (b *plainBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (b *plainBuilder) AllowedV0Alpha1Resources() []string                  { return nil }

// noVersionsBuilder doesn't implement APIGroupVersionProvider — used to test
// SetupConfig's error path on builders with no group versions.
type noVersionsBuilder struct{}

func (b *noVersionsBuilder) GetGroupVersions() []schema.GroupVersion { return nil }
func (b *noVersionsBuilder) InstallSchema(*runtime.Scheme) error     { return nil }
func (b *noVersionsBuilder) UpdateAPIGroupInfo(*server.APIGroupInfo, builder.APIGroupOptions) error {
	return nil
}
func (b *noVersionsBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (b *noVersionsBuilder) AllowedV0Alpha1Resources() []string                  { return nil }

// stubPolicyEval is a no-op evaluator used by EvaluatorPolicyRuleFromBuilders tests.
type stubPolicyEval struct{}

func (stubPolicyEval) EvaluatePolicyRule(authorizer.Attributes) audit.RequestAuditConfig {
	return audit.RequestAuditConfig{}
}

func newRecommendedConfigWithDiscovery(t *testing.T) *server.RecommendedConfig {
	t.Helper()
	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	cfg := server.NewRecommendedConfig(codecs)
	cfg.AggregatedDiscoveryGroupManager = aggregated.NewResourceManager("apis")
	return cfg
}

func TestSetupConfig(t *testing.T) {
	t.Run("empty builders", func(t *testing.T) {
		scheme := builder.ProvideScheme()
		cfg := newRecommendedConfigWithDiscovery(t)
		err := builder.SetupConfig(
			scheme,
			cfg,
			nil,
			"v0.0.0",
			builder.GetDefaultBuildHandlerChainFunc,
			nil,
			nil,
			prometheus.NewRegistry(),
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, cfg.OpenAPIConfig)
		require.NotNil(t, cfg.OpenAPIV3Config)
		require.Equal(t, "Grafana API Server", cfg.OpenAPIConfig.Info.Title)
		require.Equal(t, "v0.0.0", cfg.OpenAPIConfig.Info.Version)
	})

	t.Run("registers builder priorities and post hooks", func(t *testing.T) {
		gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v0alpha1"}
		scheme := builder.ProvideScheme()
		b := &helperBuilder{
			gv: gv,
			postHooks: map[string]server.PostStartHookFunc{
				"hello": func(server.PostStartHookContext) error { return nil },
			},
		}
		require.NoError(t, b.InstallSchema(scheme))

		cfg := newRecommendedConfigWithDiscovery(t)
		err := builder.SetupConfig(
			scheme,
			cfg,
			[]builder.APIGroupBuilder{b},
			"v1.2.3",
			builder.GetDefaultBuildHandlerChainFunc,
			[]schema.GroupVersion{gv},
			nil,
			prometheus.NewRegistry(),
			nil,
		)
		require.NoError(t, err)
		_, ok := cfg.PostStartHooks["hello"]
		require.True(t, ok)
	})

	t.Run("builder with no group versions returns error", func(t *testing.T) {
		scheme := builder.ProvideScheme()
		cfg := newRecommendedConfigWithDiscovery(t)
		err := builder.SetupConfig(
			scheme,
			cfg,
			[]builder.APIGroupBuilder{&noVersionsBuilder{}},
			"v1",
			builder.GetDefaultBuildHandlerChainFunc,
			nil,
			nil,
			prometheus.NewRegistry(),
			nil,
		)
		require.Error(t, err)
	})

	t.Run("post start hook provider error", func(t *testing.T) {
		gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}
		scheme := builder.ProvideScheme()
		b := &helperBuilder{
			gv:          gv,
			postHookErr: errors.New("nope"),
		}
		require.NoError(t, b.InstallSchema(scheme))
		cfg := newRecommendedConfigWithDiscovery(t)
		err := builder.SetupConfig(
			scheme,
			cfg,
			[]builder.APIGroupBuilder{b},
			"v1",
			builder.GetDefaultBuildHandlerChainFunc,
			[]schema.GroupVersion{gv},
			nil,
			prometheus.NewRegistry(),
			nil,
		)
		require.Error(t, err)
	})

	t.Run("OperationID transform branches", func(t *testing.T) {
		cfg := newRecommendedConfigWithDiscovery(t)
		err := builder.SetupConfig(
			builder.ProvideScheme(),
			cfg,
			nil,
			"v1",
			builder.GetDefaultBuildHandlerChainFunc,
			nil,
			nil,
			prometheus.NewRegistry(),
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, cfg.OpenAPIV3Config)
		require.NotNil(t, cfg.OpenAPIV3Config.GetOperationIDAndTagsFromRoute)
		fn := cfg.OpenAPIV3Config.GetOperationIDAndTagsFromRoute

		cases := []struct {
			name       string
			route      common.Route
			wantOp     string
			wantTagsAt []string
		}{
			{
				name: "post becomes create and namespaced trimmed",
				route: fakeRoute{
					path:   "/x",
					method: "POST",
					op:     "postNamespacedFoo",
					meta:   map[string]interface{}{},
				},
				wantOp: "createFoo",
			},
			{
				name: "read becomes get",
				route: fakeRoute{
					path: "/x", method: "GET", op: "readNamespacedFoo", meta: map[string]interface{}{},
				},
				wantOp: "getFoo",
			},
			{
				name: "put becomes replace",
				route: fakeRoute{
					path: "/x", method: "PUT", op: "putNamespacedFoo", meta: map[string]interface{}{},
				},
				wantOp: "replaceFoo",
			},
			{
				name: "patch becomes update",
				route: fakeRoute{
					path: "/x", method: "PATCH", op: "patchNamespacedFoo", meta: map[string]interface{}{},
				},
				wantOp: "updateFoo",
			},
			{
				name: "connect extracts sub-path and operation alt",
				route: fakeRoute{
					path:   "/apis/g/v1/namespaces/{namespace}/foos/{name}/logs",
					method: "GET",
					op:     "connectNamespacedFoo",
					meta: map[string]interface{}{
						"x-kubernetes-action":               "connect",
						"x-kubernetes-group-version-kind": metav1.GroupVersionKind{Group: "g", Version: "v1", Kind: "Foo"},
					},
				},
				wantOp:     "getFoo",
				wantTagsAt: []string{"Foo"},
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				op, tags, err := fn(c.route)
				require.NoError(t, err)
				require.Equal(t, c.wantOp, op)
				if c.wantTagsAt != nil {
					require.Equal(t, c.wantTagsAt, tags)
				}
			})
		}
	})
}

type fakeRoute struct {
	path, method, op, description string
	meta                          map[string]interface{}
}

func (r fakeRoute) Method() string                    { return r.method }
func (r fakeRoute) Path() string                      { return r.path }
func (r fakeRoute) OperationName() string             { return r.op }
func (r fakeRoute) Description() string               { return r.description }
func (r fakeRoute) Metadata() map[string]interface{}  { return r.meta }
func (r fakeRoute) Parameters() []common.Parameter    { return nil }
func (r fakeRoute) RequestPayloadSample() interface{} { return nil }
func (r fakeRoute) ResponsePayloadSample() interface{} {
	return nil
}
func (r fakeRoute) StatusCodeResponses() []common.StatusCodeResponse { return nil }
func (r fakeRoute) Consumes() []string                               { return nil }
func (r fakeRoute) Produces() []string                               { return nil }

func TestInstallAPIs(t *testing.T) {
	gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}
	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	metricsRegistry := prometheus.NewRegistry()
	builderMetrics := builder.ProvideBuilderMetrics(metricsRegistry)
	features := featuremgmt.WithFeatures()

	t.Run("no builders is a no-op", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			nil, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})

	t.Run("builder without group versions returns error", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeLegacy
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{&noVersionsBuilder{}}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.Error(t, err)
	})

	t.Run("legacy storage type skips dual write", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeLegacy
		b := &helperBuilder{gv: gv}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})

	t.Run("update API group info error is surfaced", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeLegacy
		b := &helperBuilder{gv: gv, updateErr: errors.New("update failed")}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.Error(t, err)
	})

	t.Run("dual write builder exercises configured modes", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeUnified
		storageOpts.UnifiedStorageConfig = map[string]setting.UnifiedStorageConfig{
			"foos.ex.grafana.app": {DualWriterMode: grafanarest.Mode4},
		}
		mode0Resource := schema.GroupResource{Group: "ex.grafana.app", Resource: "legacy-only"}
		mode4Resource := schema.GroupResource{Group: "ex.grafana.app", Resource: "foos"}
		legacyStub := stubStorage{}
		unifiedStub := stubStorage{}

		b := &helperBuilder{
			gv: gv,
			updateImpl: func(_ *server.APIGroupInfo, opts builder.APIGroupOptions) error {
				require.NotNil(t, opts.DualWriteBuilder)
				// Mode0 (legacy only, not in UnifiedStorageConfig → default 0)
				got, err := opts.DualWriteBuilder(mode0Resource, legacyStub, unifiedStub)
				require.NoError(t, err)
				require.Equal(t, legacyStub, got)
				// Mode4 → unified
				got, err = opts.DualWriteBuilder(mode4Resource, legacyStub, unifiedStub)
				require.NoError(t, err)
				require.Equal(t, unifiedStub, got)
				return nil
			},
		}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})

	t.Run("v0alpha1 resources filtered when feature toggle disabled", func(t *testing.T) {
		v0gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v0alpha1"}
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeLegacy

		b := &helperBuilder{
			gv: v0gv,
			updateImpl: func(g *server.APIGroupInfo, _ builder.APIGroupOptions) error {
				// Populate prioritized versions and resources so the filtering
				// branches in installAPIGroupsForBuilder execute. With no
				// allowed resources, all v0alpha1 entries are removed and the
				// map is left empty (so InstallAPIGroup is skipped).
				g.PrioritizedVersions = []schema.GroupVersion{v0gv}
				g.VersionedResourcesStorageMap = map[string]map[string]registryrest.Storage{
					"v0alpha1": {
						"dropme":  stubStorage{},
						"alsodrop": stubStorage{},
					},
				}
				return nil
			},
		}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})

	t.Run("apiResourceConfig filters versioned resources", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeLegacy
		cfg := serverstorage.NewResourceConfig()
		cfg.DisableVersions(gv)

		b := &helperBuilder{
			gv: gv,
			updateImpl: func(g *server.APIGroupInfo, _ builder.APIGroupOptions) error {
				g.PrioritizedVersions = []schema.GroupVersion{gv}
				g.VersionedResourcesStorageMap = map[string]map[string]registryrest.Storage{
					"v1": {"foos": stubStorage{}},
				}
				return nil
			},
		}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, nil, nil, features, builderMetrics, cfg,
		)
		require.NoError(t, err)
	})

	t.Run("dual write falls through to NewStaticStorage on non-managed mode", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeUnified
		storageOpts.UnifiedStorageConfig = map[string]setting.UnifiedStorageConfig{
			"foos.ex.grafana.app": {DualWriterMode: grafanarest.Mode1},
		}
		gr := schema.GroupResource{Group: "ex.grafana.app", Resource: "foos"}
		// dualWriteService is set so the LogStorageModeComparison branch is taken,
		// but ShouldManage returns false so the function falls through to the
		// switch statement which hits the default case (Mode1).
		dws := &mockDualWriteService{
			shouldManage: func(_ schema.GroupResource) bool { return false },
		}
		b := &helperBuilder{
			gv: gv,
			updateImpl: func(_ *server.APIGroupInfo, opts builder.APIGroupOptions) error {
				got, err := opts.DualWriteBuilder(gr, stubStorage{}, stubStorage{})
				require.NoError(t, err)
				require.NotNil(t, got)
				return nil
			},
		}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, dws, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})

	t.Run("dual write uses dualWriteService when ShouldManage", func(t *testing.T) {
		storageOpts := options.NewStorageOptions()
		storageOpts.StorageType = options.StorageTypeUnified
		expectedStorage := stubStorage{}
		dws := &mockDualWriteService{
			shouldManage: func(_ schema.GroupResource) bool { return true },
			newStorage: func(_ schema.GroupResource, _, _ grafanarest.Storage) (grafanarest.Storage, error) {
				return expectedStorage, nil
			},
		}
		gr := schema.GroupResource{Group: "ex.grafana.app", Resource: "foos"}
		b := &helperBuilder{
			gv: gv,
			updateImpl: func(_ *server.APIGroupInfo, opts builder.APIGroupOptions) error {
				got, err := opts.DualWriteBuilder(gr, stubStorage{}, stubStorage{})
				require.NoError(t, err)
				require.Equal(t, expectedStorage, got)
				return nil
			},
		}
		err := builder.InstallAPIs(
			scheme, codecs, nil, nil,
			[]builder.APIGroupBuilder{b}, storageOpts, metricsRegistry, dws, nil, features, builderMetrics, nil,
		)
		require.NoError(t, err)
	})
}

// stubStorage is a no-op grafanarest.Storage implementation used in tests.
type stubStorage struct {
	grafanarest.Storage
}

type mockDualWriteService struct {
	dualwrite.Service
	shouldManage func(schema.GroupResource) bool
	newStorage   func(schema.GroupResource, grafanarest.Storage, grafanarest.Storage) (grafanarest.Storage, error)
}

func (m *mockDualWriteService) ShouldManage(gr schema.GroupResource) bool {
	if m.shouldManage == nil {
		return false
	}
	return m.shouldManage(gr)
}
func (m *mockDualWriteService) NewStorage(gr schema.GroupResource, legacy, unified grafanarest.Storage) (grafanarest.Storage, error) {
	return m.newStorage(gr, legacy, unified)
}
func (m *mockDualWriteService) LogStorageModeComparison(_ schema.GroupResource, _ grafanarest.DualWriterMode) {
}

func TestEvaluatorPolicyRuleFromBuilders(t *testing.T) {
	gv := schema.GroupVersion{Group: "ex.grafana.app", Version: "v1"}
	emptyGv := schema.GroupVersion{}

	// builder that's not an auditor → ignored
	nonAuditor := &mockAPIGroupPostStartHookProvider{}

	// auditor returning nil → ignored
	nilAuditor := &helperBuilder{gv: gv, policy: nil}

	// auditor returning real eval → included
	withEval := &helperBuilder{gv: gv, policy: stubPolicyEval{}}

	// auditor with empty gv → skipped
	emptyGvBuilder := &helperBuilder{gv: emptyGv, policy: stubPolicyEval{}}

	got := builder.EvaluatorPolicyRuleFromBuilders([]builder.APIGroupBuilder{
		nonAuditor, nilAuditor, withEval, emptyGvBuilder,
	})

	require.Contains(t, got, gv)
	require.NotContains(t, got, emptyGv)
	require.Len(t, got, 1)

}
