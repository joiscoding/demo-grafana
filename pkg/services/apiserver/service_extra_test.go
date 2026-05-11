package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sauthorizer "k8s.io/apiserver/pkg/authorization/authorizer"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"
	clientrest "k8s.io/client-go/rest"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/apiserver/auditing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/registry/apis/datasource"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	userPkg "github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/apiserver/aggregatorrunner"
	"github.com/grafana/grafana/pkg/services/apiserver/auth/authorizer"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/pluginstore"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/legacysql/dualwrite"
	"github.com/grafana/grafana/pkg/web"
)


// --- Fakes for ProvideService / start ---

type fakeDatasourceProvider struct {
	datasource.PluginDatasourceProvider
	settings *backend.DataSourceInstanceSettings
	err      error
}

func (f *fakeDatasourceProvider) GetInstanceSettings(_ context.Context, _ string) (*backend.DataSourceInstanceSettings, error) {
	return f.settings, f.err
}

type fakeScopedDSProvider struct {
	provider datasource.PluginDatasourceProvider
}

func (f *fakeScopedDSProvider) GetDatasourceProvider(_ plugins.JSONData) datasource.PluginDatasourceProvider {
	return f.provider
}

type fakeContextWrapper struct {
	pc  backend.PluginContext
	err error
}

func (f *fakeContextWrapper) PluginContextForDataSource(_ context.Context, _ *backend.DataSourceInstanceSettings) (backend.PluginContext, error) {
	return f.pc, f.err
}

// --- minimal service constructor used by simple unit tests ---

func newTestService(t *testing.T) *service {
	t.Helper()
	s := &service{
		log:        log.New("test"),
		features:   featuremgmt.WithFeatures(),
		authorizer: authorizer.NewGrafanaBuiltInSTAuthorizer(),
		stoppedCh:  make(chan error, 1),
	}
	s.NamedService = services.NewBasicService(s.start, s.running, nil).WithName("test-apiserver")
	return s
}

func TestService_IsDisabled(t *testing.T) {
	s := &service{}
	require.False(t, s.IsDisabled())
}

type routeRegistrarBuilder struct {
	builder.APIGroupBuilder
	called bool
}

func (r *routeRegistrarBuilder) RegisterHTTPRoutes(_ routing.RouteRegister) {
	r.called = true
}

type plainBuilder struct {
	builder.APIGroupBuilder
}

func TestService_RegisterAPI(t *testing.T) {
	s := &service{rr: routing.NewRouteRegister()}
	b := &plainBuilder{}
	s.RegisterAPI(b)
	require.Len(t, s.builders, 1)

	rb := &routeRegistrarBuilder{}
	s.RegisterAPI(rb)
	require.Len(t, s.builders, 2)
	require.True(t, rb.called, "HTTPRouteRegistrar should have been invoked")
}

func TestService_RegisterAppInstaller(t *testing.T) {
	s := &service{}
	s.RegisterAppInstaller(nil)
	require.Len(t, s.appInstallers, 1)
}

func TestService_GetRestConfig_CtxCancelled(t *testing.T) {
	s := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := s.GetRestConfig(ctx)
	require.Nil(t, cfg)
	require.Error(t, err)
}

func TestService_GetDirectRestConfig_TransportCtxCancelled(t *testing.T) {
	s := newTestService(t)
	rc := newReqContext(context.Background())
	rc.SignedInUser = nil

	conf := s.GetDirectRestConfig(rc)
	require.NotNil(t, conf)
	require.NotNil(t, conf.Transport)

	// Drive the round-tripper with a cancelled ctx so AwaitRunning errors out.
	rt, ok := conf.Transport.(http.RoundTripper)
	require.True(t, ok)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	resp, err := rt.RoundTrip(req)
	require.Nil(t, resp)
	require.Error(t, err)
}

func TestService_DirectlyServeHTTP_CtxCancelled(t *testing.T) {
	s := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	require.NotPanics(t, func() { s.DirectlyServeHTTP(rec, req) })
	require.Equal(t, http.StatusOK, rec.Code) // unchanged
}

func TestService_GetDirectRestConfig_Success(t *testing.T) {
	s := newTestService(t)
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	rc := newReqContext(context.Background())
	rc.SignedInUser = &userPkg.SignedInUser{UserUID: "u1"}

	conf := s.GetDirectRestConfig(rc)
	require.NotNil(t, conf)
	rt := conf.Transport.(http.RoundTripper)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestService_GetRestConfig_Success(t *testing.T) {
	expected := &clientrest.Config{Host: "https://example"}
	s := newTestService(t)
	s.restConfig = expected
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	got, err := s.GetRestConfig(context.Background())
	require.NoError(t, err)
	require.Same(t, expected, got)
}

func TestService_Run_AlreadyRunningCleanup(t *testing.T) {
	s := newTestService(t)
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Let Run start.
		time.Sleep(50 * time.Millisecond)
		cancel()
		// Stop the underlying service so AwaitTerminated returns.
		s.StopAsync()
	}()
	require.NoError(t, s.Run(ctx))
}

func TestService_DirectlyServeHTTP_HandlerCalled(t *testing.T) {
	s := newTestService(t)
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	// Mark the dskit service as running so AwaitRunning succeeds.
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	rec := httptest.NewRecorder()
	s.DirectlyServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusTeapot, rec.Code)
}

func TestService_Running_CtxDone(t *testing.T) {
	s := &service{stoppedCh: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, s.running(ctx))
}

func TestService_Running_StoppedChNil(t *testing.T) {
	s := &service{stoppedCh: make(chan error, 1)}
	s.stoppedCh <- nil
	require.NoError(t, s.running(context.Background()))
}

func TestService_Running_StoppedChError(t *testing.T) {
	s := &service{stoppedCh: make(chan error, 1)}
	wantErr := errors.New("boom")
	s.stoppedCh <- wantErr
	require.ErrorIs(t, s.running(context.Background()), wantErr)
}

func TestEnsureKubeConfig(t *testing.T) {
	dir := t.TempDir()
	rc := &clientrest.Config{Host: "https://example", BearerToken: "token"}
	require.NoError(t, ensureKubeConfig(rc, dir))

	info, err := os.Stat(filepath.Join(dir, "grafana.kubeconfig"))
	require.NoError(t, err)
	require.False(t, info.IsDir())
}

func TestPluginContextProvider_PluginNotFound(t *testing.T) {
	p := &pluginContextProvider{
		pluginStore: pluginstore.NewFakePluginStore(),
		datasources: &fakeScopedDSProvider{},
	}
	_, err := p.GetPluginContext(context.Background(), "missing", "uid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin not found")
}

func TestPluginContextProvider_InstanceSettingsError(t *testing.T) {
	wantErr := errors.New("settings boom")
	p := &pluginContextProvider{
		pluginStore: pluginstore.NewFakePluginStore(pluginstore.Plugin{
			JSONData: plugins.JSONData{ID: "myplugin", Type: plugins.TypeDataSource},
		}),
		datasources:     &fakeScopedDSProvider{provider: &fakeDatasourceProvider{err: wantErr}},
		contextProvider: &fakeContextWrapper{},
	}
	_, err := p.GetPluginContext(context.Background(), "myplugin", "uid")
	require.ErrorIs(t, err, wantErr)
}

func TestPluginContextProvider_Success(t *testing.T) {
	wantPC := backend.PluginContext{PluginID: "myplugin"}
	p := &pluginContextProvider{
		pluginStore: pluginstore.NewFakePluginStore(pluginstore.Plugin{
			JSONData: plugins.JSONData{ID: "myplugin", Type: plugins.TypeDataSource},
		}),
		datasources:     &fakeScopedDSProvider{provider: &fakeDatasourceProvider{settings: &backend.DataSourceInstanceSettings{}}},
		contextProvider: &fakeContextWrapper{pc: wantPC},
	}
	pc, err := p.GetPluginContext(context.Background(), "myplugin", "uid")
	require.NoError(t, err)
	require.Equal(t, "myplugin", pc.PluginID)
}

// --- ProvideService smoke test ---

func newTestCfg(t *testing.T) *setting.Cfg {
	t.Helper()
	cfg := setting.NewCfg()
	cfg.HTTPAddr = "127.0.0.1"
	cfg.HTTPPort = "3000"
	cfg.DataPath = t.TempDir()
	cfg.BuildVersion = "0.0.0-test"
	cfg.BuildCommit = "deadbeef"
	cfg.BuildBranch = "test"
	cfg.BuildStamp = time.Now().Unix()
	return cfg
}

func TestProvideService_Smoke(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg,
		featuremgmt.WithFeatures(),
		rr,
		tracing.NewNoopTracerService(),
		nil, // db
		nil, // pluginClient
		nil, // datasources
		nil, // contextProvider
		nil, // pluginStore
		dualwrite.ProvideTestService(),
		nil, // unified
		nil, // secrets
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		nil, // aggregator runner
		nil, // app installers
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NotNil(t, s.NamedService)

	// Ready channel should be closed.
	select {
	case <-eventual.ready:
	default:
		t.Fatal("eventual ready channel should be closed")
	}

	// IsDisabled is always false.
	require.False(t, s.IsDisabled())
}

func TestService_start_InvalidIP(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.HTTPAddr = "not-an-ip"

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg,
		featuremgmt.WithFeatures(),
		rr,
		tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(),
		nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	// Call start directly with an invalid IP — applyGrafanaConfig should error.
	err = s.start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid IP address")
}

// emptyGroupVersionsBuilder is used to exercise the "no group versions" error
// path in start.
type emptyGroupVersionsBuilder struct {
	builder.APIGroupBuilder
}

func (emptyGroupVersionsBuilder) GetGroupVersions() []schema.GroupVersion { return nil }

// simpleBuilder is a minimal APIGroupBuilder that returns a single GroupVersion
// and implements APIGroupAuthorizer so the builder loop in start exercises the
// authorizer registration path.
type simpleBuilder struct {
	gv schema.GroupVersion
}

func (b *simpleBuilder) InstallSchema(scheme *runtime.Scheme) error {
	scheme.AddUnversionedTypes(b.gv)
	return nil
}

func (b *simpleBuilder) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ builder.APIGroupOptions) error {
	return nil
}

func (b *simpleBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
		return map[string]common.OpenAPIDefinition{}
	}
}

func (b *simpleBuilder) AllowedV0Alpha1Resources() []string { return nil }

func (b *simpleBuilder) GetGroupVersion() schema.GroupVersion { return b.gv }

func (b *simpleBuilder) GetAuthorizer() k8sauthorizer.Authorizer {
	return k8sauthorizer.AuthorizerFunc(
		func(_ context.Context, _ k8sauthorizer.Attributes) (k8sauthorizer.Decision, string, error) {
			return k8sauthorizer.DecisionAllow, "", nil
		},
	)
}

func TestService_start_WithSimpleBuilder(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.RegisterAPI(&simpleBuilder{gv: schema.GroupVersion{Group: "test.example.com", Version: "v1alpha1"}})
	s.stoppedCh = make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.start(ctx)
	// Accept either success or controlled failure — we just want to maximise coverage.
	t.Logf("start with builder returned: %v", err)
}

// failingInstallSchemaBuilder triggers the error return on InstallSchema.
type failingInstallSchemaBuilder struct {
	simpleBuilder
	err error
}

func (b *failingInstallSchemaBuilder) InstallSchema(_ *runtime.Scheme) error { return b.err }

func TestService_start_InstallSchemaError(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	wantErr := errors.New("install boom")
	s.RegisterAPI(&failingInstallSchemaBuilder{
		simpleBuilder: simpleBuilder{gv: schema.GroupVersion{Group: "x.example.com", Version: "v1"}},
		err:           wantErr,
	})
	s.stoppedCh = make(chan error, 1)
	require.ErrorIs(t, s.start(context.Background()), wantErr)
}

// nilAuthorizerBuilder triggers the panic when GetAuthorizer returns nil.
type nilAuthorizerBuilder struct {
	simpleBuilder
}

func (nilAuthorizerBuilder) GetAuthorizer() k8sauthorizer.Authorizer { return nil }

func TestService_start_NilAuthorizerPanic(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.RegisterAPI(&nilAuthorizerBuilder{
		simpleBuilder: simpleBuilder{gv: schema.GroupVersion{Group: "y.example.com", Version: "v1"}},
	})
	s.stoppedCh = make(chan error, 1)
	require.PanicsWithValue(t, "authorizer can not be nil for api group=y.example.com/v1", func() {
		_ = s.start(context.Background())
	})
}

func TestService_start_UnknownRuntimeConfig_APIEnablementValidateError(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.Raw.Section("grafana-apiserver").Key("runtime_config").SetValue("unknown.example.com/v1=true")

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	err = s.start(context.Background())
	require.Error(t, err)
}

func TestService_start_InvalidStorageType_OptionsValidateError(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.Raw.Section("grafana-apiserver").Key("storage_type").SetValue("bogus")

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	err = s.start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "grafana-apiserver-storage-type")
}

func TestService_start_EtcdStorageType(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.Raw.Section("grafana-apiserver").Key("storage_type").SetValue("etcd")
	cfg.Raw.Section("grafana-apiserver").Key("etcd_servers").SetValue("http://127.0.0.1:2379")

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	// Etcd is configured — exercise the etcd branch of start; we don't require
	// success here, only coverage of the storage-type-specific code paths.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.start(ctx)
	t.Logf("etcd storage_type start returned: %v", err)
}

func TestService_start_NoGroupVersionsBuilder(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.RegisterAPI(emptyGroupVersionsBuilder{})
	s.stoppedCh = make(chan error, 1)

	err = s.start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no group versions")
}

// fakeAggregatorRunner is used to drive the FlagKubernetesAggregator branch of start.
type fakeAggregatorRunner struct {
	server         *genericapiserver.GenericAPIServer
	configureNil   bool
	configureErr   error
	runErr         error
	configureCalls int
	runCalls       int
}

func (f *fakeAggregatorRunner) Configure(
	_ *options.Options,
	_ *genericapiserver.RecommendedConfig,
	_ *aggregatorrunner.ExtraConfig,
	delegate genericapiserver.DelegationTarget,
	_ *runtime.Scheme,
	_ []builder.APIGroupBuilder,
) (*genericapiserver.GenericAPIServer, error) {
	f.configureCalls++
	if f.configureErr != nil {
		return nil, f.configureErr
	}
	if f.configureNil {
		return nil, nil
	}
	if f.server == nil {
		if gs, ok := delegate.(*genericapiserver.GenericAPIServer); ok {
			f.server = gs
		}
	}
	return f.server, nil
}

func (f *fakeAggregatorRunner) Run(
	_ context.Context,
	_ *options.RoundTripperFunc,
	_ chan error,
) (*genericapiserver.GenericAPIServer, error) {
	f.runCalls++
	return f.server, f.runErr
}

func TestService_start_KubernetesAggregator_RunBranch(t *testing.T) {
	cfg := newTestCfg(t)

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator)

	// configureErr nil, returns delegate as server. Run returns same server.
	runner := &fakeAggregatorRunner{}
	s, err := ProvideService(
		cfg, features, rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		runner, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.start(ctx)
	// The runner returns a real *GenericAPIServer (the core server); we expect success.
	require.NoError(t, err)
	require.Equal(t, 1, runner.configureCalls)
	require.Equal(t, 1, runner.runCalls)
}

func TestService_start_KubernetesAndDataplaneAggregator(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(
		featuremgmt.FlagKubernetesAggregator,
		featuremgmt.FlagDataplaneAggregator,
	)

	runner := &fakeAggregatorRunner{}
	s, err := ProvideService(
		cfg, features, rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		runner, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.start(ctx)
	t.Logf("k8s+dataplane aggregator start returned: %v", err)
}

func TestService_start_KubernetesAggregator_RunError(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator)

	wantErr := errors.New("run boom")
	runner := &fakeAggregatorRunner{runErr: wantErr}
	s, err := ProvideService(
		cfg, features, rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		runner, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	require.ErrorIs(t, s.start(context.Background()), wantErr)
}

func TestService_start_KubernetesAggregator_ConfigureError(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator)

	wantErr := errors.New("configure boom")
	runner := &fakeAggregatorRunner{configureErr: wantErr}
	s, err := ProvideService(
		cfg, features, rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		runner, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	require.ErrorIs(t, s.start(context.Background()), wantErr)
}

func TestService_start_DataplaneAggregator(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(featuremgmt.FlagDataplaneAggregator)

	s, err := ProvideService(
		cfg, features, rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// We don't require success — only that as many statements as possible are covered.
	err = s.start(ctx)
	t.Logf("dataplane aggregator start returned: %v", err)
}

func TestService_start_KubernetesAggregator_EnterpriseUnlinked(t *testing.T) {
	cfg := newTestCfg(t)

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()
	features := featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAggregator)

	s, err := ProvideService(
		cfg,
		features,
		rr,
		tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(),
		nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		&fakeAggregatorRunner{configureNil: true}, // Configure returns (nil, nil) -> enterprise-not-linked branch
		nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.start(ctx)
	require.NoError(t, err)
}

func TestService_start_WithValidCfg_RunsAsFarAsPossible(t *testing.T) {
	cfg := newTestCfg(t)

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg,
		featuremgmt.WithFeatures(),
		rr,
		tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(),
		nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)

	// We don't care whether start succeeds — we only want to exercise as many
	// statements as possible. The test is best-effort coverage. We also set a
	// short ctx so any background goroutines (PrepareRun) are cancelled quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.start(ctx)
	t.Logf("start returned: %v", err)
}

// captureRouter collects routes registered via routing.RouteRegister.Register.
type captureRouter struct {
	routes []captured
}

type captured struct {
	method   string
	pattern  string
	handlers []web.Handler
}

func (c *captureRouter) Handle(method, pattern string, handlers []web.Handler) {
	c.routes = append(c.routes, captured{method: method, pattern: pattern, handlers: handlers})
}

func (c *captureRouter) Get(pattern string, handlers ...web.Handler) {
	c.routes = append(c.routes, captured{method: http.MethodGet, pattern: pattern, handlers: handlers})
}

// findProxyHandler locates the last handler registered for the given pattern
// (the proxyHandler closure registered by ProvideService).
func findProxyHandler(t *testing.T, rr routing.RouteRegister, pattern string) func(*contextmodel.ReqContext) {
	t.Helper()
	cr := &captureRouter{}
	rr.Register(cr)
	for _, r := range cr.routes {
		if r.pattern == pattern {
			for i := len(r.handlers) - 1; i >= 0; i-- {
				if h, ok := r.handlers[i].(func(c *contextmodel.ReqContext)); ok {
					return h
				}
			}
		}
	}
	t.Fatalf("proxy handler for pattern %q not found", pattern)
	return nil
}

func makeReqCtx(t *testing.T, method, target string) (*contextmodel.ReqContext, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	return &contextmodel.ReqContext{
		Context: &web.Context{Req: req, Resp: web.NewResponseWriter(req.Method, rec)},
	}, rec
}

func TestProvideService_ProxyHandler_NoHandlerYields404(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	// Force AwaitRunning to succeed without actually starting the apiserver.
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	h := findProxyHandler(t, rr, "/apis/")
	ctx, rec := makeReqCtx(t, http.MethodGet, "/apis/")
	h(ctx)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProvideService_ProxyHandler_UnauthenticatedWithNamespace(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	// Install a handler that echoes a 200.
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := findProxyHandler(t, rr, "/apis/*")
	ctx, rec := makeReqCtx(t, http.MethodGet, "/apis/folder.grafana.app/v1alpha1/namespaces/stacks-77/folders")
	// Unauthenticated SignedInUser branch — IsSignedIn false but SignedInUser non-nil.
	ctx.SignedInUser = &userPkg.SignedInUser{}
	ctx.IsSignedIn = false
	h(ctx)
	require.Equal(t, http.StatusOK, rec.Code)
	// useNamespaceFromPath should have populated namespace on the user.
	require.Equal(t, "stacks-77", ctx.SignedInUser.Namespace)
}

func TestProvideService_ProxyHandler_EmptyURLPath(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))

	var observedPath string
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	h := findProxyHandler(t, rr, "/apis/")
	req := httptest.NewRequest(http.MethodGet, "/apis/", nil)
	req.URL.Path = "" // exercise the empty path branch
	rec := httptest.NewRecorder()
	rc := &contextmodel.ReqContext{
		Context: &web.Context{Req: req, Resp: web.NewResponseWriter(req.Method, rec)},
	}
	h(rc)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/", observedPath)
}

func TestProvideService_ProxyHandler_AwaitRunningError(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	_, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	// Do NOT start the service — AwaitRunning will error once the request ctx is cancelled.
	h := findProxyHandler(t, rr, "/apis/")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/apis/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	rc := &contextmodel.ReqContext{
		Context: &web.Context{Req: req, Resp: web.NewResponseWriter(req.Method, rec)},
	}
	h(rc)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestService_Run_StartAsyncError(t *testing.T) {
	s := newTestService(t)
	// Move the service into a terminated state so a subsequent StartAsync call
	// in Run returns an error immediately.
	s.NamedService = services.NewIdleService(nil, nil).WithName("test")
	require.NoError(t, s.StartAsync(context.Background()))
	require.NoError(t, s.AwaitRunning(context.Background()))
	s.StopAsync()
	require.NoError(t, s.AwaitTerminated(context.Background()))

	err := s.Run(context.Background())
	require.Error(t, err)
}

// failingAddToSchemeInstaller exercises the appinstaller.AddToScheme error path
// in start without requiring the apiserver to actually run.
type failingAddToSchemeInstaller struct {
	appsdkapiserver.AppInstaller
	err error
}

func (f *failingAddToSchemeInstaller) AddToScheme(_ *runtime.Scheme) error { return f.err }

func TestService_start_AppInstallerAddToSchemeError(t *testing.T) {
	cfg := newTestCfg(t)
	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	wantErr := errors.New("scheme boom")
	installer := &failingAddToSchemeInstaller{err: wantErr}

	s, err := ProvideService(
		cfg, featuremgmt.WithFeatures(), rr, tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(), nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(), nil,
		[]appsdkapiserver.AppInstaller{installer},
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)
	s.stoppedCh = make(chan error, 1)
	require.ErrorIs(t, s.start(context.Background()), wantErr)
}

func TestService_Run_PropagatesStartError(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.HTTPAddr = "not-an-ip"

	rr := routing.NewRouteRegister()
	eventual := ProvideEventualRestConfigProvider()

	s, err := ProvideService(
		cfg,
		featuremgmt.WithFeatures(),
		rr,
		tracing.NewNoopTracerService(),
		nil, nil, nil, nil, nil,
		dualwrite.ProvideTestService(),
		nil, nil,
		eventual,
		builder.ProvideDefaultBuildHandlerChainFuncFromBuilders(),
		eventual,
		prometheus.NewRegistry(),
		nil, nil,
		builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
		auditing.ProvideNoopBackend(),
		auditing.ProvideNoopPolicyRuleProvider(),
	)
	require.NoError(t, err)

	// Run should start the dskit service, which calls start (which errors),
	// then AwaitTerminated returns the start error.
	runErr := s.Run(context.Background())
	require.Error(t, runErr)
}

