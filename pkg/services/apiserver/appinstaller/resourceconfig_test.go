package appinstaller

import (
	"testing"

	"github.com/grafana/grafana-app-sdk/app"
	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewAPIResourceConfig(t *testing.T) {
	tests := []struct {
		name            string
		installers      []appsdkapiserver.AppInstaller
		enabledVersions []schema.GroupVersion
		disabledVersions []schema.GroupVersion
	}{
		{
			name:       "empty installers",
			installers: nil,
		},
		{
			name: "single installer with served and unserved versions",
			installers: []appsdkapiserver.AppInstaller{
				&mockInstallerWithManifest{
					manifest: &app.ManifestData{
						Group: "test.example.com",
						Versions: []app.ManifestVersion{
							{Name: "v1", Served: true},
							{Name: "v2", Served: true},
							{Name: "v0alpha1", Served: false},
						},
					},
				},
			},
			enabledVersions: []schema.GroupVersion{
				{Group: "test.example.com", Version: "v1"},
				{Group: "test.example.com", Version: "v2"},
			},
			disabledVersions: []schema.GroupVersion{
				{Group: "test.example.com", Version: "v0alpha1"},
			},
		},
		{
			name: "multiple installers",
			installers: []appsdkapiserver.AppInstaller{
				&mockInstallerWithManifest{
					manifest: &app.ManifestData{
						Group: "first.example.com",
						Versions: []app.ManifestVersion{
							{Name: "v1", Served: true},
						},
					},
				},
				&mockInstallerWithManifest{
					manifest: &app.ManifestData{
						Group: "second.example.com",
						Versions: []app.ManifestVersion{
							{Name: "v1", Served: false},
						},
					},
				},
			},
			enabledVersions: []schema.GroupVersion{
				{Group: "first.example.com", Version: "v1"},
			},
			disabledVersions: []schema.GroupVersion{
				{Group: "second.example.com", Version: "v1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAPIResourceConfig(tt.installers)
			require.NotNil(t, cfg)

			for _, gv := range tt.enabledVersions {
				enabled, ok := cfg.GroupVersionConfigs[gv]
				require.True(t, ok, "expected version %s to be configured", gv)
				require.True(t, enabled, "expected version %s to be enabled", gv)
			}
			for _, gv := range tt.disabledVersions {
				enabled, ok := cfg.GroupVersionConfigs[gv]
				require.True(t, ok, "expected version %s to be configured", gv)
				require.False(t, enabled, "expected version %s to be disabled", gv)
			}
		})
	}
}

type mockInstallerWithManifest struct {
	appsdkapiserver.AppInstaller
	manifest *app.ManifestData
}

func (m *mockInstallerWithManifest) ManifestData() *app.ManifestData {
	return m.manifest
}
