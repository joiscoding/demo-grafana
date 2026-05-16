package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana-app-sdk/resource"
	examplev1 "github.com/grafana/grafana/pkg/services/apiserver/builder/runner/testdata/app/pkg/apis/example/v1"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/storage/unified/apistore"
)

func TestNewAppBuilder(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()

	b, err := NewAppBuilder(AppBuilderConfig{
		groupVersion: gv,
		ManagedKinds: map[schema.GroupVersion][]resource.Kind{
			gv: {examplev1.ExampleKind()},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestAppBuilder_SetApp_GetGroupVersion(t *testing.T) {
	gv := schema.GroupVersion{Group: "example.grafana.app", Version: "v1"}
	b := &appBuilder{
		config: AppBuilderConfig{groupVersion: gv},
	}
	require.Equal(t, gv, b.GetGroupVersion())

	b.SetApp(&mockApp{})
	require.Equal(t, &mockApp{}, b.app)
}

func TestAppBuilder_AllowedV0Alpha1Resources(t *testing.T) {
	allowed := []string{"examples"}
	b := &appBuilder{
		config: AppBuilderConfig{AllowedV0Alpha1Resources: allowed},
	}
	require.Equal(t, allowed, b.AllowedV0Alpha1Resources())
}

func TestAppBuilder_GetOpenAPIDefinitions_GetAuthorizer(t *testing.T) {
	openAPIDef := func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
		return nil
	}
	auth := authorizer.AuthorizerFunc(func(_ context.Context, _ authorizer.Attributes) (authorizer.Decision, string, error) {
		return authorizer.DecisionAllow, "", nil
	})

	b := &appBuilder{
		config: AppBuilderConfig{
			OpenAPIDefGetter: openAPIDef,
			Authorizer:       auth,
		},
	}

	require.NotNil(t, b.GetOpenAPIDefinitions())
	require.NotNil(t, b.GetAuthorizer())
}

func TestAppBuilder_InstallSchema(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	selectableKind := resource.Kind{
		Schema: resource.NewSimpleSchema(
			gv.Group,
			gv.Version,
			&examplev1.Example{},
			&examplev1.ExampleList{},
			resource.WithKind("SelectableExample"),
			resource.WithPlural("selectableexamples"),
			resource.WithSelectableFields([]resource.SelectableField{{FieldSelector: "metadata.name"}}),
		),
	}

	tests := []struct {
		name    string
		kinds   []resource.Kind
		wantErr bool
	}{
		{
			name:  "registers example types",
			kinds: []resource.Kind{examplev1.ExampleKind()},
		},
		{
			name:  "registers selectable field conversions",
			kinds: []resource.Kind{examplev1.ExampleKind(), selectableKind},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			b := &appBuilder{
				config: AppBuilderConfig{
					groupVersion: gv,
					ManagedKinds: map[schema.GroupVersion][]resource.Kind{
						gv: tt.kinds,
					},
				},
			}

			err := b.InstallSchema(scheme)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			gvk := gv.WithKind("Example")
			require.True(t, scheme.Recognizes(gvk))

			internalGV := schema.GroupVersion{Group: gv.Group, Version: runtime.APIVersionInternal}
			require.True(t, scheme.Recognizes(internalGV.WithKind("Example")))

			// internal kinds are registered only once across multiple InstallSchema calls
			require.NoError(t, b.InstallSchema(scheme))
		})
	}
}

func TestAppBuilder_UpdateAPIGroupInfo(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	scheme := builder.ProvideScheme()
	schemaBuilder, err := NewAppBuilder(AppBuilderConfig{
		groupVersion: gv,
		ManagedKinds: map[schema.GroupVersion][]resource.Kind{
			gv: {examplev1.ExampleKind()},
		},
	})
	require.NoError(t, err)
	require.NoError(t, schemaBuilder.InstallSchema(scheme))

	optsGetter, err := apistore.NewRESTOptionsGetterMemory(storagebackend.Config{}, nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		config     AppBuilderConfig
		opts       builder.APIGroupOptions
		wantErr    bool
		checkPaths []string
	}{
		{
			name: "creates storage for managed kinds",
			config: AppBuilderConfig{
				groupVersion: gv,
				ManagedKinds: map[schema.GroupVersion][]resource.Kind{
					gv: {examplev1.ExampleKind()},
				},
			},
			opts: builder.APIGroupOptions{
				Scheme:     scheme,
				OptsGetter: optsGetter,
			},
			checkPaths: []string{"examples", "examples/status"},
		},
		{
			name: "uses dual write when legacy storage exists",
			config: AppBuilderConfig{
				groupVersion: gv,
				ManagedKinds: map[schema.GroupVersion][]resource.Kind{
					gv: {examplev1.ExampleKind()},
				},
				LegacyStorageGetter: func(gvr schema.GroupVersionResource) grafanarest.Storage {
					if gvr.Resource == "examples" {
						return grafanarest.NewMockStorage(t)
					}
					return nil
				},
			},
			opts: builder.APIGroupOptions{
				Scheme:     scheme,
				OptsGetter: optsGetter,
				DualWriteBuilder: func(gr schema.GroupResource, legacy, unified grafanarest.Storage) (grafanarest.Storage, error) {
					require.NotNil(t, legacy)
					require.NotNil(t, unified)
					return unified, nil
				},
			},
			checkPaths: []string{"examples"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewAppBuilder(tt.config)
			require.NoError(t, err)

			apiGroupInfo := &genericapiserver.APIGroupInfo{
				VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
			}
			err = b.UpdateAPIGroupInfo(apiGroupInfo, tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			versionStorage := apiGroupInfo.VersionedResourcesStorageMap[gv.Version]
			require.NotNil(t, versionStorage)
			for _, path := range tt.checkPaths {
				require.NotNil(t, versionStorage[path], "expected storage at %q", path)
			}
		})
	}
}
