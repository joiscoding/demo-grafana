package appinstaller

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/grafana/grafana-app-sdk/app"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/services/apiserver/auth/authorizer/storewrapper"
)

// extendedMockAppInstaller is a fully-featured mock that overrides every
// method of appsdkapiserver.AppInstaller. The existing mockAppInstaller in
// installer_test.go embeds the interface and only overrides GroupVersions —
// any other method call would panic. The function fields below are pluggable;
// nil-valued ones use sensible no-op defaults.
type extendedMockAppInstaller struct {
	manifestData    *app.ManifestData
	groupVersions   []schema.GroupVersion
	addToScheme     func(*runtime.Scheme) error
	getOpenAPIDefs  func(common.ReferenceCallback) map[string]common.OpenAPIDefinition
	installAPIs     func(appsdkapiserver.GenericAPIServer, generic.RESTOptionsGetter) error
	admissionPlugin admission.Factory
	initializeApp   func(clientrest.Config) error
	app             app.App
	appErr          error
}

func (m *extendedMockAppInstaller) AddToScheme(s *runtime.Scheme) error {
	if m.addToScheme != nil {
		return m.addToScheme(s)
	}
	return nil
}

func (m *extendedMockAppInstaller) GetOpenAPIDefinitions(cb common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	if m.getOpenAPIDefs != nil {
		return m.getOpenAPIDefs(cb)
	}
	return nil
}

func (m *extendedMockAppInstaller) InstallAPIs(s appsdkapiserver.GenericAPIServer, og generic.RESTOptionsGetter) error {
	if m.installAPIs != nil {
		return m.installAPIs(s, og)
	}
	return nil
}

func (m *extendedMockAppInstaller) AdmissionPlugin() admission.Factory {
	return m.admissionPlugin
}

func (m *extendedMockAppInstaller) InitializeApp(c clientrest.Config) error {
	if m.initializeApp != nil {
		return m.initializeApp(c)
	}
	return nil
}

func (m *extendedMockAppInstaller) App() (app.App, error) {
	return m.app, m.appErr
}

func (m *extendedMockAppInstaller) GroupVersions() []schema.GroupVersion {
	return m.groupVersions
}

func (m *extendedMockAppInstaller) ManifestData() *app.ManifestData {
	return m.manifestData
}

// extendedWithAuth wraps extendedMockAppInstaller and implements AuthorizerProvider.
type extendedWithAuth struct {
	*extendedMockAppInstaller
	authz authorizer.Authorizer
}

func (e *extendedWithAuth) GetAuthorizer() authorizer.Authorizer { return e.authz }

// Add ManifestData support to the original mockAppInstaller so it can be used
// in tests that exercise paths needing ManifestData.
func (m *mockAppInstaller) ManifestData() *app.ManifestData {
	return m.manifestData
}

// fakeApp implements just enough of app.App for createPostStartHook tests:
// only Runner() is invoked by the code under test.
type fakeApp struct {
	app.App
	runner app.Runnable
}

func (f *fakeApp) Runner() app.Runnable { return f.runner }

type fakeRunnable struct {
	err  error
	done chan struct{}
}

func (f *fakeRunnable) Run(ctx context.Context) error {
	if f.done != nil {
		close(f.done)
	}
	return f.err
}

// Fake create/update strategies. They embed the interfaces so only the
// methods used by the code under test (NamespaceScoped) need to be defined;
// any accidental call to other methods would surface as a nil-pointer panic.
type fakeCreateStrategy struct {
	rest.RESTCreateStrategy
	namespaced bool
}

func (f *fakeCreateStrategy) NamespaceScoped() bool { return f.namespaced }

type fakeUpdateStrategy struct {
	rest.RESTUpdateStrategy
	namespaced bool
}

func (f *fakeUpdateStrategy) NamespaceScoped() bool { return f.namespaced }

// nsAuthInstaller implements NamespaceScopedStorageAuthorizerProvider.
type nsAuthInstaller struct {
	*extendedMockAppInstaller
	provide func(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer
}

func (n *nsAuthInstaller) GetNamespaceScopedStorageAuthorizer(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
	if n.provide != nil {
		return n.provide(gr)
	}
	return nil
}

// clusterAuthInstaller implements ClusterScopedStorageAuthorizerProvider.
type clusterAuthInstaller struct {
	*extendedMockAppInstaller
	provide func(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer
}

func (c *clusterAuthInstaller) GetClusterScopedStorageAuthorizer(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
	if c.provide != nil {
		return c.provide(gr)
	}
	return nil
}

// fakeGenericAPIServer implements appsdkapiserver.GenericAPIServer.
type fakeGenericAPIServer struct {
	installed  *genericapiserver.APIGroupInfo
	installErr error
	webSvcs    []*restful.WebService
}

func (f *fakeGenericAPIServer) InstallAPIGroup(info *genericapiserver.APIGroupInfo) error {
	f.installed = info
	return f.installErr
}

func (f *fakeGenericAPIServer) RegisteredWebServices() []*restful.WebService {
	return f.webSvcs
}
