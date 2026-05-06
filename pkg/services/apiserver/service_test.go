package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	dskitservices "github.com/grafana/dskit/services"
	appsdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/server"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	datasourceV0 "github.com/grafana/grafana/pkg/apis/datasource/v0alpha1"
	"github.com/grafana/grafana/pkg/apiserver/auditing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/registry/apis/datasource"
	"github.com/grafana/grafana/pkg/services/apiserver/aggregatorrunner"
	"github.com/grafana/grafana/pkg/services/apiserver/auth/authorizer"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	grafanaapiserveroptions "github.com/grafana/grafana/pkg/services/apiserver/options"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/pluginstore"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web"
)

func Test_useNamespaceFromPath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		expNs string
	}{
		{
			name:  "no namespace in path",
			path:  "/apis/folder.grafana.app/",
			expNs: "",
		},
		{
			name:  "namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/stacks-11/folders",
			expNs: "stacks-11",
		},
		{
			name:  "invalid namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/invalid/folders",
			expNs: "invalid",
		},
		{
			name:  "org namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/org-123/folders",
			expNs: "org-123",
		},
		{
			name:  "default namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/default/folders",
			expNs: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &user.SignedInUser{}
			useNamespaceFromPath(tt.path, user)
			if user.Namespace != tt.expNs {
				require.Equal(t, tt.expNs, user.Namespace, "expected namespace to be %s, got %s", tt.expNs, user.Namespace)
			}
		})
	}
}

func TestApplyGrafanaConfig(t *testing.T) {
	t.Run("maps grafana settings onto api server options", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Raw.Section("log").Key("level").SetValue("debug")
		cfg.Raw.Section("grafana-apiserver").Key("runtime_config").SetValue("all/all=false")
		cfg.Raw.Section("grafana-apiserver").Key("request_timeout").SetValue("15s")
		cfg.Raw.Section("grafana-apiserver").Key("etcd_servers").SetValue("https://etcd-a:2379,https://etcd-b:2379")
		cfg.Raw.Section("grafana-apiserver").Key("storage_type").SetValue(string(grafanaapiserveroptions.StorageTypeFile))
		cfg.Raw.Section("grafana-apiserver").Key("storage_path").SetValue("/tmp/apiserver")
		cfg.Raw.Section("grafana-apiserver").Key("address").SetValue("unified:10000")
		cfg.Raw.Section("grafana-apiserver").Key("blob_url").SetValue("file:///tmp/blobs")
		cfg.Raw.Section("grafana-apiserver").Key("blob_threshold_bytes").SetValue("1024")

		opts := grafanaapiserveroptions.NewOptions(nil)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerEnsureKubectlAccess), opts)

		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", opts.RecommendedOptions.SecureServing.BindAddress.String())
		require.Equal(t, 3100, opts.RecommendedOptions.SecureServing.BindPort)
		require.Equal(t, []string{"https://etcd-a:2379", "https://etcd-b:2379"}, opts.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList)
		require.True(t, opts.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional)
		require.True(t, opts.RecommendedOptions.Authorization.RemoteKubeConfigFileOptional)
		require.Nil(t, opts.RecommendedOptions.Admission)
		require.Nil(t, opts.RecommendedOptions.CoreAPI)
		require.Equal(t, grafanaapiserveroptions.StorageTypeFile, opts.StorageOptions.StorageType)
		require.Equal(t, "/tmp/apiserver", opts.StorageOptions.DataPath)
		require.Equal(t, "unified:10000", opts.StorageOptions.Address)
		require.Equal(t, "file:///tmp/blobs", opts.StorageOptions.BlobStoreURL)
		require.Equal(t, 1024, opts.StorageOptions.BlobThresholdBytes)
		require.True(t, opts.ExtraOptions.DevMode)
		require.Equal(t, "127.0.0.1:3100", opts.ExtraOptions.ExternalAddress)
		require.Equal(t, "https://grafana.example.test/", opts.ExtraOptions.APIURL)
		require.Equal(t, 7, opts.ExtraOptions.Verbosity)
		require.Equal(t, 15*time.Second, opts.ExtraOptions.RequestTimeout)
	})

	t.Run("uses dev serving defaults", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Env = setting.Dev
		cfg.HTTPAddr = "127.0.0.1"
		cfg.HTTPPort = "not-a-port"

		opts := grafanaapiserveroptions.NewOptions(nil)
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), opts)

		require.NoError(t, err)
		require.Equal(t, "0.0.0.0", opts.RecommendedOptions.SecureServing.BindAddress.String())
		require.Equal(t, 6443, opts.RecommendedOptions.SecureServing.BindPort)
		require.Equal(t, "127.0.0.1:6443", opts.ExtraOptions.ExternalAddress)
		require.Equal(t, "https://0.0.0.0:6443", opts.ExtraOptions.APIURL)
		require.False(t, opts.ExtraOptions.DevMode)
	})

	t.Run("returns invalid ip errors", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.HTTPAddr = "not an ip"
		err := applyGrafanaConfig(cfg, featuremgmt.WithFeatures(), grafanaapiserveroptions.NewOptions(nil))
		require.ErrorContains(t, err, "invalid IP address")
	})
}

func TestEventualRestConfigProvider(t *testing.T) {
	t.Run("waits for readiness before delegating", func(t *testing.T) {
		provider := ProvideEventualRestConfigProvider()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		restConfig, err := provider.GetRestConfig(ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, restConfig)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		reqCtx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(reqCtx)
		require.Nil(t, provider.GetDirectRestConfig(&contextmodel.ReqContext{Context: &web.Context{Req: req}}))

		recorder := httptest.NewRecorder()
		provider.DirectlyServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("delegates after readiness", func(t *testing.T) {
		restConfig := &clientrest.Config{Host: "https://apiserver.example.test"}
		directConfig := &clientrest.Config{Host: "direct"}
		fake := &fakeDirectRestConfigProvider{restConfig: restConfig, directConfig: directConfig}
		provider := &eventualRestConfigProvider{ready: make(chan struct{}), cfg: fake}
		close(provider.ready)

		got, err := provider.GetRestConfig(context.Background())
		require.NoError(t, err)
		require.Same(t, restConfig, got)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		require.Same(t, directConfig, provider.GetDirectRestConfig(&contextmodel.ReqContext{Context: &web.Context{Req: req}}))

		recorder := httptest.NewRecorder()
		provider.DirectlyServeHTTP(recorder, req)
		require.Equal(t, http.StatusAccepted, recorder.Code)
	})
}

func TestLazyClientGeneratorReturnsRestConfigError(t *testing.T) {
	expectedErr := errors.New("rest config failed")
	generator := ProvideClientGenerator(RestConfigProviderFunc(func(context.Context) (*clientrest.Config, error) {
		return nil, expectedErr
	}))

	client, err := generator.ClientFor(appsdkresource.Kind{})
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, client)

	client, err = generator.ClientFor(appsdkresource.Kind{})
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, client)
}

func TestServiceRegistrationAndDirectHelpers(t *testing.T) {
	t.Run("provide service wires routes and eventual rest config provider", func(t *testing.T) {
		rr := routing.NewRouteRegister()
		eventualProvider := ProvideEventualRestConfigProvider()

		s, err := ProvideService(
			testCfg(t),
			featuremgmt.WithFeatures(),
			rr,
			tracing.NewNoopTracerService(),
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			WithoutRestConfig,
			builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
			eventualProvider,
			prometheus.NewRegistry(),
			nil,
			nil,
			nil,
			auditing.ProvideNoopBackend(),
			auditing.ProvideNoopPolicyRuleProvider(),
		)
		require.NoError(t, err)
		require.NotNil(t, s)

		router := &capturingRouter{}
		rr.Register(router)
		require.Contains(t, router.routes, "/apis/features.grafana.app/v0alpha1/*")
		require.Contains(t, router.routes, "/apis/")
		require.Contains(t, router.routes, "/apis/*")
		require.Contains(t, router.routes, "/livez/*")
		require.Contains(t, router.routes, "/readyz/*")
		require.Contains(t, router.routes, "/healthz/*")
		require.Contains(t, router.routes, "/openapi/*")
		require.Contains(t, router.routes, "/version/*")

		// Before the dskit service is running, the proxy wrapper reports startup failures.
		startupReq := httptest.NewRequest(http.MethodGet, "/", nil)
		startupReq.URL.Path = ""
		startupCtx, startupCancel := context.WithCancel(startupReq.Context())
		startupCancel()
		startupReq = startupReq.WithContext(startupCtx)
		resp := invokeReqHandler(t, router.routes["/apis/features.grafana.app/v0alpha1/*"], s, startupReq, nil, false)
		require.Equal(t, http.StatusInternalServerError, resp.Status())

		running := runningTestService(t)
		s.NamedService = running.NamedService

		// A running apiserver without a handler returns 404 from the proxy wrapper.
		notFoundReq := httptest.NewRequest(http.MethodGet, "/", nil)
		notFoundReq.URL.Path = ""
		resp = invokeReqHandler(t, router.routes["/apis/features.grafana.app/v0alpha1/*"], s, notFoundReq, nil, false)
		require.Equal(t, http.StatusNotFound, resp.Status())

		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/apis/folder.grafana.app/v1alpha1/namespaces/org-22/folders", r.URL.Path)
			requester, err := identity.GetRequester(r.Context())
			require.NoError(t, err)
			require.Equal(t, "org-22", requester.GetNamespace())
			w.WriteHeader(http.StatusTeapot)
		})
		req := httptest.NewRequest(http.MethodGet, "/apis/folder.grafana.app/v1alpha1/namespaces/org-22/folders", nil)
		user := &user.SignedInUser{}
		resp = invokeReqHandler(t, router.routes["/apis/features.grafana.app/v0alpha1/*"], s, req, user, false)
		require.Equal(t, http.StatusTeapot, resp.Status())
		require.Equal(t, "org-22", user.Namespace)
		require.Equal(t, int64(22), user.OrgID)

		s.restConfig = &clientrest.Config{Host: "https://loopback.example.test"}
		got, err := eventualProvider.GetRestConfig(context.Background())
		require.NoError(t, err)
		require.Same(t, s.restConfig, got)
	})

	t.Run("get rest config returns loopback config for running service", func(t *testing.T) {
		s := runningTestService(t)
		s.restConfig = &clientrest.Config{Host: "https://loopback.example.test"}

		got, err := s.GetRestConfig(context.Background())

		require.NoError(t, err)
		require.Same(t, s.restConfig, got)
	})

	t.Run("get rest config returns await running errors", func(t *testing.T) {
		s := &service{}
		s.NamedService = dskitservices.NewBasicService(nil, nil, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got, err := s.GetRestConfig(ctx)

		require.Error(t, err)
		require.Nil(t, got)
	})

	t.Run("run returns startup errors", func(t *testing.T) {
		expectedErr := errors.New("start failed")
		s := &service{}
		s.NamedService = dskitservices.NewBasicService(func(context.Context) error {
			return expectedErr
		}, nil, nil)

		err := s.Run(context.Background())

		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("register api stores builder and delegates http route registration", func(t *testing.T) {
		rr := routing.NewRouteRegister()
		s := &service{rr: rr}
		builder := &fakeHTTPRouteBuilder{}

		s.RegisterAPI(builder)

		require.Len(t, s.builders, 1)
		require.Same(t, builder, s.builders[0])
		require.True(t, builder.registered)
	})

	t.Run("register app installer appends installer", func(t *testing.T) {
		s := &service{}

		s.RegisterAppInstaller(nil)

		require.Len(t, s.appInstallers, 1)
		require.Nil(t, s.appInstallers[0])
	})

	t.Run("is never disabled", func(t *testing.T) {
		require.False(t, (&service{}).IsDisabled())
	})

	t.Run("direct rest config injects signed in user into handler context", func(t *testing.T) {
		s := runningTestService(t)
		signedInUser := &user.SignedInUser{Login: "viewer"}
		var requester identity.Requester
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requester, _ = identity.GetRequester(r.Context())
			w.WriteHeader(http.StatusCreated)
		})

		req := httptest.NewRequest(http.MethodGet, "/apis/example", nil)
		resp, err := s.GetDirectRestConfig(&contextmodel.ReqContext{
			Context:      &web.Context{Req: req},
			SignedInUser: signedInUser,
		}).Transport.RoundTrip(req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Same(t, signedInUser, requester)
	})

	t.Run("direct serve waits for running service", func(t *testing.T) {
		s := runningTestService(t)
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		recorder := httptest.NewRecorder()
		s.DirectlyServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("direct helpers return when service is not running", func(t *testing.T) {
		s := &service{}
		s.NamedService = dskitservices.NewBasicService(nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)

		resp, err := s.GetDirectRestConfig(&contextmodel.ReqContext{
			Context:      &web.Context{Req: req},
			SignedInUser: &user.SignedInUser{},
		}).Transport.RoundTrip(req)
		require.Error(t, err)
		require.Nil(t, resp)

		recorder := httptest.NewRecorder()
		s.DirectlyServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("running returns stopped channel errors and exits on context cancellation", func(t *testing.T) {
		expectedErr := errors.New("server stopped")
		s := &service{stoppedCh: make(chan error, 1)}
		s.stoppedCh <- expectedErr
		require.ErrorIs(t, s.running(context.Background()), expectedErr)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		require.NoError(t, s.running(ctx))
	})
}

func TestEnsureKubeConfig(t *testing.T) {
	dir := t.TempDir()
	err := ensureKubeConfig(&clientrest.Config{Host: "https://apiserver.example.test"}, dir)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "grafana.kubeconfig"))
}

func TestPluginContextProvider(t *testing.T) {
	t.Run("returns plugin not found when no matching datasource plugin exists", func(t *testing.T) {
		provider := &pluginContextProvider{
			pluginStore: &fakePluginStore{plugins: []pluginstore.Plugin{
				{JSONData: plugins.JSONData{ID: "other-plugin"}},
			}},
			datasources: &fakeScopedDatasourceProvider{},
		}

		pluginCtx, err := provider.GetPluginContext(context.Background(), "test-plugin", "uid")
		require.ErrorContains(t, err, "plugin not found")
		require.Empty(t, pluginCtx)
	})

	t.Run("returns datasource settings errors", func(t *testing.T) {
		expectedErr := errors.New("settings failed")
		provider := &pluginContextProvider{
			pluginStore: &fakePluginStore{plugins: []pluginstore.Plugin{
				{JSONData: plugins.JSONData{ID: "test-plugin"}},
			}},
			datasources:     &fakeScopedDatasourceProvider{provider: &fakeDatasourceProvider{err: expectedErr}},
			contextProvider: &fakePluginContextWrapper{},
		}

		pluginCtx, err := provider.GetPluginContext(context.Background(), "test-plugin", "uid")
		require.ErrorIs(t, err, expectedErr)
		require.Empty(t, pluginCtx)
	})

	t.Run("wraps datasource settings into plugin context", func(t *testing.T) {
		settings := &backend.DataSourceInstanceSettings{UID: "uid"}
		expectedCtx := backend.PluginContext{DataSourceInstanceSettings: settings}
		provider := &pluginContextProvider{
			pluginStore: &fakePluginStore{plugins: []pluginstore.Plugin{
				{JSONData: plugins.JSONData{ID: "test-plugin"}},
			}},
			datasources:     &fakeScopedDatasourceProvider{provider: &fakeDatasourceProvider{settings: settings}},
			contextProvider: &fakePluginContextWrapper{pluginCtx: expectedCtx},
		}

		pluginCtx, err := provider.GetPluginContext(context.Background(), "test-plugin", "uid")
		require.NoError(t, err)
		require.Equal(t, expectedCtx, pluginCtx)
	})
}

func TestServiceStartWithoutRegisteredAPIs(t *testing.T) {
	t.Run("starts core server", func(t *testing.T) {
		s := newStartableTestService(t, featuremgmt.WithFeatures())

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := s.start(ctx)
		require.NoError(t, err)
		require.NotNil(t, s.options)
		require.NotNil(t, s.handler)
		require.NotNil(t, s.restConfig)
	})

	t.Run("handles disabled kubernetes aggregator and writes dev kubeconfig", func(t *testing.T) {
		s := newStartableTestService(t, featuremgmt.WithFeatures(
			featuremgmt.FlagGrafanaAPIServerEnsureKubectlAccess,
			featuremgmt.FlagKubernetesAggregator,
		))
		s.aggregatorRunner = fakeAggregatorRunner{}

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := s.start(ctx)
		require.NoError(t, err)
		require.NotNil(t, s.handler)
		require.FileExists(t, filepath.Join(s.options.StorageOptions.DataPath, "grafana.kubeconfig"))
	})
}

func TestServiceStartBuilderValidation(t *testing.T) {
	t.Run("returns an error when a builder has no group versions", func(t *testing.T) {
		s := newStartableTestService(t, featuremgmt.WithFeatures())
		s.builders = []builder.APIGroupBuilder{&fakeStartBuilder{}}

		err := s.start(context.Background())
		require.ErrorContains(t, err, "no group versions found")
	})

	t.Run("returns install schema errors", func(t *testing.T) {
		expectedErr := errors.New("install schema failed")
		s := newStartableTestService(t, featuremgmt.WithFeatures())
		s.builders = []builder.APIGroupBuilder{&fakeStartBuilder{
			groupVersions: []schema.GroupVersion{{Group: "test.grafana.app", Version: "v1"}},
			installErr:    expectedErr,
		}}

		err := s.start(context.Background())
		require.ErrorIs(t, err, expectedErr)
	})
}

func TestServiceStartAggregatorErrors(t *testing.T) {
	t.Run("returns kubernetes aggregator configure errors", func(t *testing.T) {
		expectedErr := errors.New("configure failed")
		s := newStartableTestService(t, featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator))
		s.aggregatorRunner = fakeAggregatorRunner{configureErr: expectedErr}

		err := s.start(context.Background())
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("returns kubernetes aggregator run errors", func(t *testing.T) {
		expectedErr := errors.New("run failed")
		s := newStartableTestService(t, featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator))
		s.aggregatorRunner = fakeAggregatorRunner{
			configureServer: &server.GenericAPIServer{},
			runErr:          expectedErr,
		}

		err := s.start(context.Background())
		require.ErrorIs(t, err, expectedErr)
	})
}

func testCfg(t *testing.T) *setting.Cfg {
	t.Helper()

	cfg := setting.NewCfg()
	cfg.Env = setting.Prod
	cfg.HTTPAddr = "127.0.0.1"
	cfg.HTTPPort = "3100"
	cfg.AppURL = "https://grafana.example.test/"
	cfg.DataPath = t.TempDir()
	cfg.BuildVersion = "test-version"
	cfg.BuildCommit = "test-commit"
	cfg.BuildBranch = "test-branch"
	return cfg
}

func newStartableTestService(t *testing.T, features featuremgmt.FeatureToggles) *service {
	t.Helper()

	scheme := builder.ProvideScheme()
	return &service{
		scheme:                            scheme,
		codecs:                            builder.ProvideCodecFactory(scheme),
		cfg:                               testCfg(t),
		features:                          features,
		log:                               log.New("test.apiserver"),
		tracing:                           tracing.NewNoopTracerService(),
		metrics:                           prometheus.NewRegistry(),
		authorizer:                        authorizer.NewGrafanaBuiltInSTAuthorizer(),
		restConfigProvider:                WithoutRestConfig,
		buildHandlerChainFuncFromBuilders: builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		stoppedCh:                         make(chan error, 1),
		auditBackend:                      auditing.ProvideNoopBackend(),
		auditPolicyRuleProvider:           auditing.ProvideNoopPolicyRuleProvider(),
	}
}

func runningTestService(t *testing.T) *service {
	t.Helper()

	s := &service{}
	s.NamedService = dskitservices.NewBasicService(nil, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}, nil)
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))
	t.Cleanup(func() {
		s.StopAsync()
	})
	return s
}

type capturedRoute struct {
	method   string
	handlers []web.Handler
}

type capturingRouter struct {
	routes map[string]capturedRoute
}

func (r *capturingRouter) Handle(method, pattern string, handlers []web.Handler) {
	if r.routes == nil {
		r.routes = make(map[string]capturedRoute)
	}
	r.routes[pattern] = capturedRoute{method: method, handlers: handlers}
}

func (r *capturingRouter) Get(pattern string, handlers ...web.Handler) {
	r.Handle(http.MethodGet, pattern, handlers)
}

func invokeReqHandler(t *testing.T, route capturedRoute, _ *service, req *http.Request, signedInUser *user.SignedInUser, signedIn bool) web.ResponseWriter {
	t.Helper()

	require.NotEmpty(t, route.handlers)
	handler, ok := route.handlers[len(route.handlers)-1].(func(*contextmodel.ReqContext))
	require.True(t, ok)

	recorder := httptest.NewRecorder()
	resp := web.NewResponseWriter(req.Method, recorder)
	handler(&contextmodel.ReqContext{
		Context:      &web.Context{Req: req, Resp: resp},
		SignedInUser: signedInUser,
		IsSignedIn:   signedIn,
	})
	return resp
}

type fakeDirectRestConfigProvider struct {
	restConfig   *clientrest.Config
	directConfig *clientrest.Config
}

func (f *fakeDirectRestConfigProvider) GetRestConfig(context.Context) (*clientrest.Config, error) {
	return f.restConfig, nil
}

func (f *fakeDirectRestConfigProvider) GetDirectRestConfig(*contextmodel.ReqContext) *clientrest.Config {
	return f.directConfig
}

func (f *fakeDirectRestConfigProvider) DirectlyServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

type fakeHTTPRouteBuilder struct {
	registered bool
}

func (f *fakeHTTPRouteBuilder) InstallSchema(*runtime.Scheme) error {
	return nil
}

func (f *fakeHTTPRouteBuilder) UpdateAPIGroupInfo(*server.APIGroupInfo, builder.APIGroupOptions) error {
	return nil
}

func (f *fakeHTTPRouteBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return nil
}

func (f *fakeHTTPRouteBuilder) AllowedV0Alpha1Resources() []string {
	return nil
}

func (f *fakeHTTPRouteBuilder) RegisterHTTPRoutes(routing.RouteRegister) {
	f.registered = true
}

type fakeStartBuilder struct {
	groupVersions []schema.GroupVersion
	installErr    error
}

func (f *fakeStartBuilder) GetGroupVersions() []schema.GroupVersion {
	return f.groupVersions
}

func (f *fakeStartBuilder) InstallSchema(*runtime.Scheme) error {
	return f.installErr
}

func (f *fakeStartBuilder) UpdateAPIGroupInfo(*server.APIGroupInfo, builder.APIGroupOptions) error {
	return nil
}

func (f *fakeStartBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return nil
}

func (f *fakeStartBuilder) AllowedV0Alpha1Resources() []string {
	return nil
}

type fakeAggregatorRunner struct {
	configureServer *server.GenericAPIServer
	configureErr    error
	runErr          error
}

func (f fakeAggregatorRunner) Configure(
	*grafanaapiserveroptions.Options,
	*server.RecommendedConfig,
	*aggregatorrunner.ExtraConfig,
	server.DelegationTarget,
	*runtime.Scheme,
	[]builder.APIGroupBuilder,
) (*server.GenericAPIServer, error) {
	return f.configureServer, f.configureErr
}

func (f fakeAggregatorRunner) Run(
	context.Context,
	*grafanaapiserveroptions.RoundTripperFunc,
	chan error,
) (*server.GenericAPIServer, error) {
	return nil, f.runErr
}

type fakePluginStore struct {
	plugins []pluginstore.Plugin
}

func (f *fakePluginStore) Plugin(ctx context.Context, pluginID string) (pluginstore.Plugin, bool) {
	for _, plugin := range f.plugins {
		if plugin.ID == pluginID {
			return plugin, true
		}
	}
	return pluginstore.Plugin{}, false
}

func (f *fakePluginStore) Plugins(context.Context, ...plugins.Type) []pluginstore.Plugin {
	return f.plugins
}

type fakeScopedDatasourceProvider struct {
	provider datasource.PluginDatasourceProvider
}

func (f *fakeScopedDatasourceProvider) GetDatasourceProvider(plugins.JSONData) datasource.PluginDatasourceProvider {
	return f.provider
}

type fakeDatasourceProvider struct {
	settings *backend.DataSourceInstanceSettings
	err      error
}

func (f *fakeDatasourceProvider) GetInstanceSettings(context.Context, string) (*backend.DataSourceInstanceSettings, error) {
	return f.settings, f.err
}

func (f *fakeDatasourceProvider) GetDataSource(context.Context, string) (*datasourceV0.DataSource, error) {
	return nil, nil
}

func (f *fakeDatasourceProvider) ListDataSources(context.Context) (*datasourceV0.DataSourceList, error) {
	return nil, nil
}

func (f *fakeDatasourceProvider) CreateDataSource(context.Context, *datasourceV0.DataSource) (*datasourceV0.DataSource, error) {
	return nil, nil
}

func (f *fakeDatasourceProvider) UpdateDataSource(context.Context, *datasourceV0.DataSource) (*datasourceV0.DataSource, error) {
	return nil, nil
}

func (f *fakeDatasourceProvider) DeleteDataSource(context.Context, string) error {
	return nil
}

type fakePluginContextWrapper struct {
	pluginCtx backend.PluginContext
	err       error
}

func (f *fakePluginContextWrapper) PluginContextForDataSource(context.Context, *backend.DataSourceInstanceSettings) (backend.PluginContext, error) {
	return f.pluginCtx, f.err
}
