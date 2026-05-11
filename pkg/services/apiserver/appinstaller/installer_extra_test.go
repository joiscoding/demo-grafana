package appinstaller

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/grafana/grafana-app-sdk/app"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/common"
	openapi "k8s.io/kube-openapi/pkg/validation/spec"
)

func TestAddToScheme(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		gvs, err := AddToScheme(nil, runtime.NewScheme())
		require.NoError(t, err)
		require.Empty(t, gvs)
	})

	t.Run("aggregates group versions", func(t *testing.T) {
		var called bool
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{
				addToScheme:   func(*runtime.Scheme) error { called = true; return nil },
				groupVersions: []schema.GroupVersion{{Group: "a", Version: "v1"}, {Group: "a", Version: "v2"}},
			},
			&extendedMockAppInstaller{
				groupVersions: []schema.GroupVersion{{Group: "b", Version: "v1"}},
			},
		}
		gvs, err := AddToScheme(installers, runtime.NewScheme())
		require.NoError(t, err)
		require.True(t, called)
		require.ElementsMatch(t, []schema.GroupVersion{
			{Group: "a", Version: "v1"}, {Group: "a", Version: "v2"}, {Group: "b", Version: "v1"},
		}, gvs)
	})

	t.Run("propagates AddToScheme error", func(t *testing.T) {
		wantErr := errors.New("boom")
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{addToScheme: func(*runtime.Scheme) error { return wantErr }},
		}
		_, err := AddToScheme(installers, runtime.NewScheme())
		require.Error(t, err)
		require.ErrorIs(t, err, wantErr)
	})
}

// nopAdmission satisfies admission.Interface for tests.
type nopAdmission struct{ name string }

func (n *nopAdmission) Handles(admission.Operation) bool { return false }

func TestRegisterAdmission(t *testing.T) {
	t.Run("no installers without existing admission returns empty chain", func(t *testing.T) {
		got, err := RegisterAdmission(nil, nil)
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	t.Run("nil admission plugin is skipped", func(t *testing.T) {
		got, err := RegisterAdmission(nil, []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{},
		})
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	t.Run("admission factory error is surfaced", func(t *testing.T) {
		boom := errors.New("admit-fail")
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{admissionPlugin: admission.Factory(func(_ io.Reader) (admission.Interface, error) {
				return nil, boom
			})},
		}
		_, err := RegisterAdmission(nil, installers)
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
	})

	t.Run("chains plugins and existing admission", func(t *testing.T) {
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{admissionPlugin: admission.Factory(func(_ io.Reader) (admission.Interface, error) {
				return &nopAdmission{name: "from-installer"}, nil
			})},
		}
		got, err := RegisterAdmission(&nopAdmission{name: "existing"}, installers)
		require.NoError(t, err)
		require.NotNil(t, got)
	})
}

func TestBuildOpenAPIDefGetter(t *testing.T) {
	cb := common.ReferenceCallback(func(_ string) openapi.Ref { return openapi.Ref{} })
	installers := []appsdkapiserver.AppInstaller{
		&extendedMockAppInstaller{
			getOpenAPIDefs: func(common.ReferenceCallback) map[string]common.OpenAPIDefinition {
				return map[string]common.OpenAPIDefinition{
					"foo": {},
				}
			},
		},
		&extendedMockAppInstaller{
			getOpenAPIDefs: func(common.ReferenceCallback) map[string]common.OpenAPIDefinition {
				return map[string]common.OpenAPIDefinition{
					"bar": {},
				}
			},
		},
	}
	getter := BuildOpenAPIDefGetter(installers)
	defs := getter(cb)
	require.Contains(t, defs, "foo")
	require.Contains(t, defs, "bar")
}

func TestRegisterPostStartHooks(t *testing.T) {
	newConfig := func() *genericapiserver.RecommendedConfig {
		return &genericapiserver.RecommendedConfig{
			Config: genericapiserver.Config{
				PostStartHooks:         map[string]genericapiserver.PostStartHookConfigEntry{},
				DisabledPostStartHooks: sets.NewString(),
			},
		}
	}

	t.Run("nil manifest data returns error", func(t *testing.T) {
		installers := []appsdkapiserver.AppInstaller{&extendedMockAppInstaller{}}
		err := RegisterPostStartHooks(installers, newConfig())
		require.Error(t, err)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		md := &app.ManifestData{AppName: "duplicate"}
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{manifestData: md},
			&extendedMockAppInstaller{manifestData: md},
		}
		err := RegisterPostStartHooks(installers, newConfig())
		require.Error(t, err)
	})

	t.Run("happy path registers one hook per installer", func(t *testing.T) {
		cfg := newConfig()
		installers := []appsdkapiserver.AppInstaller{
			&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app-a"}},
			&extendedMockAppInstaller{manifestData: &app.ManifestData{AppName: "app-b"}},
		}
		require.NoError(t, RegisterPostStartHooks(installers, cfg))
		require.Contains(t, cfg.Config.PostStartHooks, "app-a")
		require.Contains(t, cfg.Config.PostStartHooks, "app-b")
	})
}

// runHook builds the hook closure and runs it with a synthetic
// PostStartHookContext.
func runHook(t *testing.T, installer appsdkapiserver.AppInstaller) error {
	t.Helper()
	hook := createPostStartHook(installer)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return hook(genericapiserver.PostStartHookContext{
		Context:              ctx,
		LoopbackClientConfig: &clientrest.Config{},
	})
}

func TestCreatePostStartHook(t *testing.T) {
	t.Run("initialize error returns wrapped error", func(t *testing.T) {
		boom := errors.New("init-fail")
		err := runHook(t, &extendedMockAppInstaller{
			manifestData: &app.ManifestData{AppName: "x"},
			initializeApp: func(clientrest.Config) error {
				return boom
			},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
	})

	t.Run("already-initialized is treated as success", func(t *testing.T) {
		done := make(chan struct{})
		err := runHook(t, &extendedMockAppInstaller{
			manifestData: &app.ManifestData{AppName: "x"},
			initializeApp: func(clientrest.Config) error {
				return appsdkapiserver.ErrAppAlreadyInitialized
			},
			app: &fakeApp{runner: &fakeRunnable{done: done}},
		})
		require.NoError(t, err)
		<-done
	})

	t.Run("app retrieval error", func(t *testing.T) {
		boom := errors.New("app-fail")
		err := runHook(t, &extendedMockAppInstaller{
			manifestData: &app.ManifestData{AppName: "x"},
			appErr:       boom,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
	})

	t.Run("happy path runs runner asynchronously", func(t *testing.T) {
		done := make(chan struct{})
		runner := &fakeRunnable{done: done}
		err := runHook(t, &extendedMockAppInstaller{
			manifestData: &app.ManifestData{AppName: "x"},
			app:          &fakeApp{runner: runner},
		})
		require.NoError(t, err)
		<-done
	})

	t.Run("runner returning error is logged but not surfaced", func(t *testing.T) {
		done := make(chan struct{})
		err := runHook(t, &extendedMockAppInstaller{
			manifestData: &app.ManifestData{AppName: "x"},
			app:          &fakeApp{runner: &fakeRunnable{err: errors.New("runner-fail"), done: done}},
		})
		require.NoError(t, err)
		<-done
	})
}

// Ensure mockAuthorizer satisfies the interface (used elsewhere in package tests)
var _ authorizer.Authorizer = (*mockAuthorizer)(nil)
