package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/grafana/grafana-app-sdk/app"
	"github.com/grafana/grafana-app-sdk/resource"
	examplev1 "github.com/grafana/grafana/pkg/services/apiserver/builder/runner/testdata/app/pkg/apis/example/v1"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestNewAPIGroupRunner(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	registrar := &mockAPIRegistrar{}

	runner, err := NewAPIGroupRunner(RunnerConfig{APIRegistrar: registrar}, newTestProvider(gv, nil))
	require.NoError(t, err)
	require.NotNil(t, runner)
	require.Len(t, registrar.registered, 1)
}

func TestNewAppBuilderGroup(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()

	tests := []struct {
		name        string
		provider    app.Provider
		wantErr     bool
		errContains string
		wantBuilders int
	}{
		{
			name:         "embedded manifest with app builder config",
			provider:     newTestProvider(gv, nil),
			wantBuilders: 1,
		},
		{
			name: "unsupported manifest location",
			provider: &mockProvider{
				manifest: app.Manifest{
					ManifestData: &app.ManifestData{AppName: "example"},
					Location:     app.ManifestLocation{Type: app.ManifestLocationFilePath},
				},
				specificConfig: &AppBuilderConfig{
					ManagedKinds: map[schema.GroupVersion][]resource.Kind{
						gv: {examplev1.ExampleKind()},
					},
				},
			},
			wantErr:     true,
			errContains: "unsupported manifest location type",
		},
		{
			name: "invalid specific config type",
			provider: &mockProvider{
				manifest:       testEmbeddedManifest(),
				specificConfig: "not-app-builder-config",
			},
			wantErr:     true,
			errContains: "not of type *AppBuilderConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := newAppBuilderGroup(RunnerConfig{APIRegistrar: &mockAPIRegistrar{}}, tt.provider)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.Len(t, group.builders, tt.wantBuilders)
		})
	}
}

func TestAPIGroupRunner_Init(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()

	tests := []struct {
		name        string
		config      RunnerConfig
		provider    app.Provider
		wantErr     bool
		errContains string
	}{
		{
			name: "rest config getter error",
			config: RunnerConfig{
				RestConfigGetter: func(context.Context) (*rest.Config, error) {
					return nil, errors.New("rest config error")
				},
			},
			provider:    newTestProvider(gv, nil),
			wantErr:     true,
			errContains: "rest config error",
		},
		{
			name: "new app error",
			config: RunnerConfig{
				RestConfigGetter: func(context.Context) (*rest.Config, error) {
					return &rest.Config{}, nil
				},
			},
			provider: newTestProvider(gv, func(app.Config) (app.App, error) {
				return nil, errors.New("new app error")
			}),
			wantErr:     true,
			errContains: "new app error",
		},
		{
			name: "success sets app on builders",
			config: RunnerConfig{
				RestConfigGetter: func(context.Context) (*rest.Config, error) {
					return &rest.Config{}, nil
				},
			},
			provider: newTestProvider(gv, func(app.Config) (app.App, error) {
				return &mockRunnableApp{runner: &mockRunnable{}}, nil
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			if cfg.APIRegistrar == nil {
				cfg.APIRegistrar = &mockAPIRegistrar{}
			}
			runner, err := NewAPIGroupRunner(cfg, tt.provider)
			require.NoError(t, err)

			err = runner.Init(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, runner.groups[0].app)
			for _, b := range runner.groups[0].builders {
				require.NotNil(t, b.(*appBuilder).app)
			}
		})
	}
}

func TestAPIGroupRunner_GetBuilders(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	runner, err := NewAPIGroupRunner(RunnerConfig{APIRegistrar: &mockAPIRegistrar{}}, newTestProvider(gv, nil))
	require.NoError(t, err)

	builders := runner.GetBuilders()
	require.Len(t, builders, 1)
	require.Equal(t, gv, builders[0].(*appBuilder).GetGroupVersion())
}

func TestAPIGroupRunner_Run(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	runner, err := NewAPIGroupRunner(RunnerConfig{
		RestConfigGetter: func(context.Context) (*rest.Config, error) {
			return &rest.Config{}, nil
		},
		APIRegistrar: &mockAPIRegistrar{},
	}, newTestProvider(gv, func(app.Config) (app.App, error) {
		return &mockRunnableApp{
			runner: &mockRunnable{
				runFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
			},
		}, nil
	}))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	require.NoError(t, runner.Init(context.Background()))

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		cancel()
		require.NoError(t, <-errCh)
	}
}

func TestAppBuilderGroup_setApp(t *testing.T) {
	gv := examplev1.ExampleKind().GroupVersionKind().GroupVersion()
	group, err := newAppBuilderGroup(RunnerConfig{APIRegistrar: &mockAPIRegistrar{}}, newTestProvider(gv, nil))
	require.NoError(t, err)

	appInstance := &mockApp{}
	group.setApp(appInstance)
	require.Equal(t, appInstance, group.app)
	for _, b := range group.builders {
		require.Equal(t, appInstance, b.(*appBuilder).app)
	}
}

func testEmbeddedManifest() app.Manifest {
	return app.NewEmbeddedManifest(app.ManifestData{
		AppName: "example",
		Group:   "example.grafana.app",
	})
}

func newTestProvider(gv schema.GroupVersion, newApp func(app.Config) (app.App, error)) app.Provider {
	if newApp == nil {
		newApp = func(app.Config) (app.App, error) {
			return &mockRunnableApp{runner: &mockRunnable{}}, nil
		}
	}
	return &mockProvider{
		manifest: testEmbeddedManifest(),
		specificConfig: &AppBuilderConfig{
			ManagedKinds: map[schema.GroupVersion][]resource.Kind{
				gv: {examplev1.ExampleKind()},
			},
			CustomConfig: map[string]string{"test": "config"},
		},
		newApp: newApp,
	}
}

type mockProvider struct {
	manifest       app.Manifest
	specificConfig app.SpecificConfig
	newApp         func(app.Config) (app.App, error)
}

func (m *mockProvider) Manifest() app.Manifest {
	return m.manifest
}

func (m *mockProvider) SpecificConfig() app.SpecificConfig {
	return m.specificConfig
}

func (m *mockProvider) NewApp(cfg app.Config) (app.App, error) {
	return m.newApp(cfg)
}

type mockAPIRegistrar struct {
	registered []builder.APIGroupBuilder
}

func (m *mockAPIRegistrar) RegisterAPI(b builder.APIGroupBuilder) {
	m.registered = append(m.registered, b)
}

func (m *mockAPIRegistrar) RegisterAppInstaller(_ appsdkapiserver.AppInstaller) {}

type mockRunnable struct {
	runFunc func(context.Context) error
}

func (m *mockRunnable) Run(ctx context.Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	<-ctx.Done()
	return nil
}

type mockRunnableApp struct {
	mockApp
	runner app.Runnable
}

func (m *mockRunnableApp) Runner() app.Runnable {
	return m.runner
}
