package appinstaller

import (
	"testing"

	appsdkapiserver "github.com/grafana/grafana-app-sdk/k8s/apiserver"
	"github.com/grafana/grafana-app-sdk/app"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewAPIResourceConfig(t *testing.T) {
	tests := []struct {
		name          string
		installers    []appsdkapiserver.AppInstaller
		enabledGVs    []schema.GroupVersion
		disabledGVs   []schema.GroupVersion
	}{
		{
			name:       "empty installers",
			installers: nil,
		},
		{
			name: "single installer one served version",
			installers: []appsdkapiserver.AppInstaller{
				&mockAppInstaller{manifestData: &app.ManifestData{
					Group: "g.example.com",
					Versions: []app.ManifestVersion{
						{Name: "v1", Served: true},
					},
				}},
			},
			enabledGVs: []schema.GroupVersion{{Group: "g.example.com", Version: "v1"}},
		},
		{
			name: "served and not served versions",
			installers: []appsdkapiserver.AppInstaller{
				&mockAppInstaller{manifestData: &app.ManifestData{
					Group: "g.example.com",
					Versions: []app.ManifestVersion{
						{Name: "v1", Served: true},
						{Name: "v1alpha1", Served: false},
						{Name: "v2", Served: true},
					},
				}},
				&mockAppInstaller{manifestData: &app.ManifestData{
					Group: "other.example.com",
					Versions: []app.ManifestVersion{
						{Name: "v1beta1", Served: false},
					},
				}},
			},
			enabledGVs: []schema.GroupVersion{
				{Group: "g.example.com", Version: "v1"},
				{Group: "g.example.com", Version: "v2"},
			},
			disabledGVs: []schema.GroupVersion{
				{Group: "g.example.com", Version: "v1alpha1"},
				{Group: "other.example.com", Version: "v1beta1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAPIResourceConfig(tt.installers)
			require.NotNil(t, cfg)
			for _, gv := range tt.enabledGVs {
				require.True(t, cfg.ResourceEnabled(gv.WithResource("any")),
					"expected version %s to be enabled", gv)
			}
			for _, gv := range tt.disabledGVs {
				require.False(t, cfg.ResourceEnabled(gv.WithResource("any")),
					"expected version %s to be disabled", gv)
			}
		})
	}
}
