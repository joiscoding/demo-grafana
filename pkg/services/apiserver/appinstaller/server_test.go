package appinstaller

import (
	"context"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/grafana/grafana-app-sdk/app"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	genericrest "k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	"github.com/grafana/grafana/pkg/services/apiserver/auth/authorizer/storewrapper"
)

func TestGetResourceFromStoragePath(t *testing.T) {
	tests := []struct {
		name        string
		storagePath string
		want        string
		wantErr     bool
	}{
		{
			name:        "resource only",
			storagePath: "widgets",
			want:        "widgets",
		},
		{
			name:        "status subresource",
			storagePath: "widgets/status",
			want:        "widgets",
		},
		{
			name:        "nested subresource",
			storagePath: "widgets/scale",
			want:        "widgets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getResourceFromStoragePath(tt.storagePath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestServerWrapper_configureStorage(t *testing.T) {
	gr := schema.GroupResource{Group: "test.example.com", Resource: "widgets"}
	ctx := context.Background()

	tests := []struct {
		name               string
		installer          appsdkapiserver.AppInstaller
		namespaced         bool
		dualWriteSupported bool
		wantWrapper        bool
		wantUpdateWrapper  bool
	}{
		{
			name:       "namespaced resource without authorizer returns store unchanged",
			installer:  &mockInstallerForServer{manifest: &app.ManifestData{AppName: "test", Group: gr.Group}},
			namespaced: true,
		},
		{
			name: "namespaced resource with authorizer wraps storage",
			installer: &mockInstallerWithNSAuth{
				mockInstallerForServer: mockInstallerForServer{
					manifest: &app.ManifestData{AppName: "test", Group: gr.Group},
				},
				authz: &testStorageAuthorizer{},
			},
			namespaced:  true,
			wantWrapper: true,
		},
		{
			name:       "cluster-scoped resource without authorizer uses deny wrapper",
			installer:  &mockInstallerForServer{manifest: &app.ManifestData{AppName: "test", Group: gr.Group}},
			namespaced: false,
			wantWrapper: true,
		},
		{
			name: "cluster-scoped resource with authorizer wraps storage",
			installer: &mockInstallerWithClusterAuth{
				mockInstallerForServer: mockInstallerForServer{
					manifest: &app.ManifestData{AppName: "test", Group: gr.Group},
				},
				authz: &testStorageAuthorizer{},
			},
			namespaced:  false,
			wantWrapper: true,
		},
		{
			name:               "dual write wraps update strategy for namespaced store",
			installer:          &mockInstallerForServer{manifest: &app.ManifestData{AppName: "test", Group: gr.Group}},
			namespaced:         true,
			dualWriteSupported: true,
			wantUpdateWrapper:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := newTestRESTStrategy(tt.namespaced)
			store := &genericregistry.Store{
				CreateStrategy: strategy,
				UpdateStrategy: strategy,
			}

			sw := &serverWrapper{
				ctx:       ctx,
				installer: tt.installer,
			}

			got := sw.configureStorage(gr, tt.dualWriteSupported, store)

			if tt.wantWrapper {
				_, ok := got.(*storewrapper.Wrapper)
				require.True(t, ok)
				return
			}

			gs, ok := got.(*genericregistry.Store)
			require.True(t, ok)
			if tt.wantUpdateWrapper {
				_, ok := gs.UpdateStrategy.(*updateStrategyWrapper)
				require.True(t, ok)
				return
			}

			require.Same(t, store, gs)
			key, err := gs.KeyFunc(genericapirequest.WithNamespace(ctx, "default"), "item")
			require.NoError(t, err)
			require.Contains(t, key, "/namespace/default/name/item")
		})
	}
}

func TestServerWrapper_configureStorage_clusterScopedKeyFunc(t *testing.T) {
	gr := schema.GroupResource{Group: "test.example.com", Resource: "widgets"}
	strategy := newTestRESTStrategy(false)
	store := &genericregistry.Store{
		CreateStrategy: strategy,
		UpdateStrategy: strategy,
	}

	sw := &serverWrapper{
		ctx: context.Background(),
		installer: &mockInstallerWithClusterAuth{
			mockInstallerForServer: mockInstallerForServer{
				manifest: &app.ManifestData{AppName: "test", Group: gr.Group},
			},
			authz: &testStorageAuthorizer{},
		},
	}

	got := sw.configureStorage(gr, false, store)
	_, ok := got.(*storewrapper.Wrapper)
	require.True(t, ok)

	key, err := store.KeyFunc(context.Background(), "cluster-item")
	require.NoError(t, err)
	require.Contains(t, key, "/resource/widgets/name/cluster-item")
	require.NotContains(t, key, "/namespace/")
}

func TestServerWrapper_InstallAPIGroup_skipsDisabledResources(t *testing.T) {
	gr := schema.GroupResource{Group: "test.example.com", Resource: "widgets"}
	gvr := gr.WithVersion("v1")

	apiGroupInfo := &genericapiserver.APIGroupInfo{
		VersionedResourcesStorageMap: map[string]map[string]genericrest.Storage{
			"v1": {
				"widgets": &noopRESTStorage{},
			},
		},
	}

	resourceConfig := serverstorage.NewResourceConfig()
	resourceConfig.DisableResources(gvr)

	mockServer := &mockGenericAPIServer{}
	sw := &serverWrapper{
		ctx:               context.Background(),
		GenericAPIServer:  mockServer,
		installer:         &mockInstallerForServer{manifest: &app.ManifestData{AppName: "test", Group: gr.Group}},
		apiResourceConfig: resourceConfig,
	}

	err := sw.InstallAPIGroup(apiGroupInfo)
	require.NoError(t, err)
	require.NotNil(t, mockServer.installed)
	_, exists := apiGroupInfo.VersionedResourcesStorageMap["v1"]["widgets"]
	require.False(t, exists)
}

func TestServerWrapper_RegisteredWebServices(t *testing.T) {
	mockServer := &mockGenericAPIServer{webServices: []*restful.WebService{{}}}
	sw := &serverWrapper{GenericAPIServer: mockServer}
	require.Equal(t, mockServer.webServices, sw.RegisteredWebServices())
}

type mockInstallerForServer struct {
	appsdkapiserver.AppInstaller
	manifest *app.ManifestData
}

func (m *mockInstallerForServer) ManifestData() *app.ManifestData {
	return m.manifest
}

type mockInstallerWithNSAuth struct {
	mockInstallerForServer
	authz storewrapper.ResourceStorageAuthorizer
}

func (m *mockInstallerWithNSAuth) GetNamespaceScopedStorageAuthorizer(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
	return m.authz
}

type mockInstallerWithClusterAuth struct {
	mockInstallerForServer
	authz storewrapper.ResourceStorageAuthorizer
}

func (m *mockInstallerWithClusterAuth) GetClusterScopedStorageAuthorizer(gr schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
	return m.authz
}

type mockGenericAPIServer struct {
	appsdkapiserver.GenericAPIServer
	installed   *genericapiserver.APIGroupInfo
	webServices []*restful.WebService
}

func (m *mockGenericAPIServer) InstallAPIGroup(apiGroupInfo *genericapiserver.APIGroupInfo) error {
	m.installed = apiGroupInfo
	return nil
}

func (m *mockGenericAPIServer) RegisteredWebServices() []*restful.WebService {
	return m.webServices
}

type noopRESTStorage struct{}

func (noopRESTStorage) New() runtime.Object { return &metav1.Status{} }
func (noopRESTStorage) Destroy()            {}

type testRESTStrategy struct {
	namespaced bool
	typed      runtime.ObjectTyper
}

func newTestRESTStrategy(namespaced bool) *testRESTStrategy {
	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	return &testRESTStrategy{namespaced: namespaced, typed: scheme}
}

func (t *testRESTStrategy) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	return t.typed.ObjectKinds(obj)
}

func (t *testRESTStrategy) Recognizes(gvk schema.GroupVersionKind) bool {
	return t.typed.Recognizes(gvk)
}

func (t *testRESTStrategy) GenerateName(base string) string {
	return base + "-generated"
}

func (t *testRESTStrategy) NamespaceScoped() bool {
	return t.namespaced
}

func (t *testRESTStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {}

func (t *testRESTStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return nil
}

func (t *testRESTStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (t *testRESTStrategy) Canonicalize(obj runtime.Object) {}

func (t *testRESTStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {}

func (t *testRESTStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return nil
}

func (t *testRESTStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (t *testRESTStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (t *testRESTStrategy) AllowUnconditionalUpdate() bool {
	return false
}

var (
	_ genericrest.RESTCreateStrategy            = (*testRESTStrategy)(nil)
	_ genericrest.RESTUpdateStrategy            = (*testRESTStrategy)(nil)
	_ storewrapper.ResourceStorageAuthorizer = (*testStorageAuthorizer)(nil)
)

type testStorageAuthorizer struct{}

func (t *testStorageAuthorizer) BeforeCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (t *testStorageAuthorizer) BeforeUpdate(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (t *testStorageAuthorizer) BeforeDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (t *testStorageAuthorizer) AfterGet(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (t *testStorageAuthorizer) FilterList(ctx context.Context, list runtime.Object) (runtime.Object, error) {
	return list, nil
}
