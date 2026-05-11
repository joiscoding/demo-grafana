package appinstaller

import (
	"context"
	"errors"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/grafana/grafana-app-sdk/app"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	genericrest "k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	"github.com/grafana/grafana/pkg/services/apiserver/auth/authorizer/storewrapper"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	grafanaapiserveroptions "github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/legacysql/dualwrite"
)

func TestGetResourceFromStoragePath(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{path: "widgets", want: "widgets"},
		{path: "widgets/status", want: "widgets"},
		{path: "widgets/scale/sub", want: "widgets"},
		{path: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, err := getResourceFromStoragePath(tc.path)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func newWrapper(installer appsdkapiserver.AppInstaller, inner appsdkapiserver.GenericAPIServer) *serverWrapper {
	return &serverWrapper{
		ctx:              context.Background(),
		GenericAPIServer: inner,
		installer:        installer,
		storageOpts: &grafanaapiserveroptions.StorageOptions{
			UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{},
		},
		dualWriteService: nil,
		builderMetrics:   builder.ProvideBuilderMetrics(prometheus.NewRegistry()),
	}
}

func TestServerWrapper_RegisteredWebServices(t *testing.T) {
	want := []*restful.WebService{{}}
	inner := &fakeGenericAPIServer{webSvcs: want}
	w := newWrapper(&extendedMockAppInstaller{}, inner)
	require.Equal(t, want, w.RegisteredWebServices())
}

func TestConfigureStorage_GenericRegistryStore_Namespaced(t *testing.T) {
	gr := schema.GroupResource{Group: "g", Resource: "widgets"}

	t.Run("no authorizer provider returns store as-is", func(t *testing.T) {
		gs := &genericregistry.Store{
			CreateStrategy: &fakeCreateStrategy{namespaced: true},
			UpdateStrategy: &fakeUpdateStrategy{namespaced: true},
		}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		require.Same(t, gs, got)
		require.NotNil(t, gs.KeyFunc)
		require.NotNil(t, gs.KeyRootFunc)
	})

	t.Run("authorizer provider returning nil returns store as-is", func(t *testing.T) {
		gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: true}}
		ns := &nsAuthInstaller{extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}}
		w := newWrapper(ns, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		require.Same(t, gs, got)
	})

	t.Run("authorizer provider returning value wraps store", func(t *testing.T) {
		gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: true}}
		ns := &nsAuthInstaller{
			extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}},
			provide: func(_ schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
				return &storewrapper.NoopAuthorizer{}
			},
		}
		w := newWrapper(ns, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		_, ok := got.(*storewrapper.Wrapper)
		require.True(t, ok, "expected *storewrapper.Wrapper, got %T", got)
	})

	t.Run("dualWriteSupported wraps update strategy", func(t *testing.T) {
		origUpdate := &fakeUpdateStrategy{namespaced: true}
		gs := &genericregistry.Store{
			CreateStrategy: &fakeCreateStrategy{namespaced: true},
			UpdateStrategy: origUpdate,
		}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		_ = w.configureStorage(gr, true, gs)
		wrapped, ok := gs.UpdateStrategy.(*updateStrategyWrapper)
		require.True(t, ok)
		require.Equal(t, origUpdate, wrapped.RESTUpdateStrategy)
	})
}

func TestConfigureStorage_GenericRegistryStore_Cluster(t *testing.T) {
	gr := schema.GroupResource{Group: "g", Resource: "clusters"}

	t.Run("no cluster authorizer provider uses deny authorizer", func(t *testing.T) {
		gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: false}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		_, ok := got.(*storewrapper.Wrapper)
		require.True(t, ok)
	})

	t.Run("cluster authorizer provider returning nil uses deny", func(t *testing.T) {
		gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: false}}
		c := &clusterAuthInstaller{extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}}
		w := newWrapper(c, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		_, ok := got.(*storewrapper.Wrapper)
		require.True(t, ok)
	})

	t.Run("cluster authorizer provider value used", func(t *testing.T) {
		gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: false}}
		c := &clusterAuthInstaller{
			extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}},
			provide: func(_ schema.GroupResource) storewrapper.ResourceStorageAuthorizer {
				return &storewrapper.NoopAuthorizer{}
			},
		}
		w := newWrapper(c, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		_, ok := got.(*storewrapper.Wrapper)
		require.True(t, ok)
	})

	t.Run("nil CreateStrategy treated as cluster-scoped", func(t *testing.T) {
		gs := &genericregistry.Store{}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, gs)
		_, ok := got.(*storewrapper.Wrapper)
		require.True(t, ok)
	})
}

func TestConfigureStorage_StatusREST(t *testing.T) {
	gr := schema.GroupResource{Group: "g", Resource: "widgets"}

	t.Run("namespaced status sets keys", func(t *testing.T) {
		s := &appsdkapiserver.StatusREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: true}}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
		require.NotNil(t, s.Store.KeyFunc)
		require.NotNil(t, s.Store.KeyRootFunc)
	})

	t.Run("cluster-scoped status without cluster-auth provider logs warning", func(t *testing.T) {
		s := &appsdkapiserver.StatusREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: false}}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
	})

	t.Run("cluster-scoped status with cluster-auth provider", func(t *testing.T) {
		s := &appsdkapiserver.StatusREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: false}}}
		c := &clusterAuthInstaller{extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}}
		w := newWrapper(c, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
	})

	t.Run("nil UpdateStrategy is treated as namespaced", func(t *testing.T) {
		s := &appsdkapiserver.StatusREST{Store: &genericregistry.Store{}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
	})
}

func TestConfigureStorage_SubresourceREST(t *testing.T) {
	gr := schema.GroupResource{Group: "g", Resource: "widgets"}

	t.Run("namespaced subresource sets keys", func(t *testing.T) {
		s := &appsdkapiserver.SubresourceREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: true}}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
		require.NotNil(t, s.Store.KeyFunc)
	})

	t.Run("cluster-scoped subresource without cluster-auth provider logs warning", func(t *testing.T) {
		s := &appsdkapiserver.SubresourceREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: false}}}
		w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
	})

	t.Run("cluster-scoped subresource with cluster-auth provider", func(t *testing.T) {
		s := &appsdkapiserver.SubresourceREST{Store: &genericregistry.Store{UpdateStrategy: &fakeUpdateStrategy{namespaced: false}}}
		c := &clusterAuthInstaller{extendedMockAppInstaller: &extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}}
		w := newWrapper(c, &fakeGenericAPIServer{})
		got := w.configureStorage(gr, false, s)
		require.Same(t, s, got)
	})
}

// dummyRESTStorage implements genericrest.Storage minimally for tests.
type dummyRESTStorage struct{}

func (dummyRESTStorage) New() runtime.Object { return nil }
func (dummyRESTStorage) Destroy()            {}

var _ genericrest.Storage = dummyRESTStorage{}

func TestConfigureStorage_UnknownTypeReturnedAsIs(t *testing.T) {
	gr := schema.GroupResource{Group: "g", Resource: "widgets"}
	in := dummyRESTStorage{}
	w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, &fakeGenericAPIServer{})
	got := w.configureStorage(gr, false, in)
	require.Equal(t, in, got)
}

func TestInstallAPIs(t *testing.T) {
	ctx := context.Background()
	bm := builder.ProvideBuilderMetrics(prometheus.NewRegistry())

	t.Run("happy path calls install per installer", func(t *testing.T) {
		called := 0
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{
				manifestData: &app.ManifestData{AppName: "a"},
				installAPIs: func(_ appsdkapiserver.GenericAPIServer, _ generic.RESTOptionsGetter) error {
					called++
					return nil
				},
			},
			&extendedMockAppInstaller{
				manifestData: &app.ManifestData{AppName: "b"},
				installAPIs: func(_ appsdkapiserver.GenericAPIServer, _ generic.RESTOptionsGetter) error {
					called++
					return nil
				},
			},
		}
		err := InstallAPIs(ctx, installers, nil, nil,
			&grafanaapiserveroptions.StorageOptions{}, dualwrite.ProvideTestService(), bm, nil)
		require.NoError(t, err)
		require.Equal(t, 2, called)
	})

	t.Run("error from install is wrapped", func(t *testing.T) {
		boom := errors.New("install-fail")
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{
				manifestData: &app.ManifestData{AppName: "x"},
				installAPIs: func(_ appsdkapiserver.GenericAPIServer, _ generic.RESTOptionsGetter) error {
					return boom
				},
			},
		}
		err := InstallAPIs(ctx, installers, nil, nil,
			&grafanaapiserveroptions.StorageOptions{}, nil, bm, nil)
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
	})
}

// TestServerWrapper_InstallAPIGroup_DisabledResourceSkipped exercises the
// early-return branch of InstallAPIGroup where the apiResourceConfig disables
// a given resource.
func TestServerWrapper_InstallAPIGroup_DisabledResourceSkipped(t *testing.T) {
	gr := schema.GroupResource{Group: "g.example.com", Resource: "widgets"}
	gv := schema.GroupVersion{Group: gr.Group, Version: "v1"}

	rc := serverstorage.NewResourceConfig()
	rc.DisableVersions(gv)

	inner := &fakeGenericAPIServer{}
	w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app", Group: gr.Group}}, inner)
	w.apiResourceConfig = rc

	info := &genericapiserver.APIGroupInfo{
		VersionedResourcesStorageMap: map[string]map[string]genericrest.Storage{
			"v1": {
				"widgets": dummyRESTStorage{},
			},
		},
	}
	require.NoError(t, w.InstallAPIGroup(info))
	require.NotContains(t, info.VersionedResourcesStorageMap["v1"], "widgets")
	require.NotNil(t, inner.installed)
}

// TestServerWrapper_InstallAPIGroup_HappyPath ensures InstallAPIGroup runs for
// a recognised store type and forwards the modified info to the inner server.
func TestServerWrapper_InstallAPIGroup_HappyPath(t *testing.T) {
	gr := schema.GroupResource{Group: "g.example.com", Resource: "widgets"}

	rc := serverstorage.NewResourceConfig()
	rc.EnableVersions(schema.GroupVersion{Group: gr.Group, Version: "v1"})

	gs := &genericregistry.Store{CreateStrategy: &fakeCreateStrategy{namespaced: true}}
	inner := &fakeGenericAPIServer{}
	w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app", Group: gr.Group}}, inner)
	w.apiResourceConfig = rc

	info := &genericapiserver.APIGroupInfo{
		VersionedResourcesStorageMap: map[string]map[string]genericrest.Storage{
			"v1": {
				"widgets": gs,
			},
		},
	}
	require.NoError(t, w.InstallAPIGroup(info))
	require.NotNil(t, inner.installed)
}

// TestServerWrapper_InstallAPIGroup_InnerError surfaces inner-server errors.
func TestServerWrapper_InstallAPIGroup_InnerError(t *testing.T) {
	inner := &fakeGenericAPIServer{installErr: errors.New("inner-fail")}
	w := newWrapper(&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app"}}, inner)
	info := &genericapiserver.APIGroupInfo{
		VersionedResourcesStorageMap: map[string]map[string]genericrest.Storage{},
	}
	err := w.InstallAPIGroup(info)
	require.Error(t, err)
}
