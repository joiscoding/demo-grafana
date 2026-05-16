package appinstaller

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/legacysql/dualwrite"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewDualWriter(t *testing.T) {
	gr := schema.GroupResource{Group: "playlist.grafana.app", Resource: "playlists"}
	key := gr.String()

	legacy := &grafanarest.MockStorage{}
	unified := &grafanarest.MockStorage{}
	metrics := builder.ProvideBuilderMetrics(prometheus.NewRegistry())

	tests := []struct {
		name              string
		storageOpts       *options.StorageOptions
		dualWriteService  dualwrite.Service
		wantSameAsLegacy  bool
		wantSameAsUnified bool
		setupMocks        func(t *testing.T, svc *dualwrite.MockService)
	}{
		{
			name:             "defaults to legacy storage when config is missing",
			storageOpts:      &options.StorageOptions{},
			wantSameAsLegacy: true,
		},
		{
			name: "mode0 returns legacy storage",
			storageOpts: &options.StorageOptions{
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					key: {DualWriterMode: grafanarest.Mode0},
				},
			},
			wantSameAsLegacy: true,
		},
		{
			name: "mode4 returns unified storage",
			storageOpts: &options.StorageOptions{
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					key: {DualWriterMode: grafanarest.Mode4},
				},
			},
			wantSameAsUnified: true,
		},
		{
			name: "mode5 returns unified storage",
			storageOpts: &options.StorageOptions{
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					key: {DualWriterMode: grafanarest.Mode5},
				},
			},
			wantSameAsUnified: true,
		},
		{
			name: "mode2 returns static dual writer storage",
			storageOpts: &options.StorageOptions{
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					key: {DualWriterMode: grafanarest.Mode2},
				},
			},
		},
		{
			name: "managed resource delegates to dual write service",
			setupMocks: func(t *testing.T, svc *dualwrite.MockService) {
				t.Helper()
				managed := &grafanarest.MockStorage{}
				svc.On("ShouldManage", gr).Return(true)
				svc.On("NewStorage", gr, legacy, unified).Return(managed, nil)
				svc.On("LogStorageModeComparison", gr, grafanarest.DualWriterMode(0)).Maybe()
			},
			storageOpts: &options.StorageOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dualWriteService dualwrite.Service
			if tt.setupMocks != nil {
				svc := dualwrite.NewMockService(t)
				tt.setupMocks(t, svc)
				dualWriteService = svc
			}

			got, err := NewDualWriter(gr, tt.storageOpts, legacy, unified, dualWriteService, metrics)
			require.NoError(t, err)
			require.NotNil(t, got)

			switch {
			case tt.wantSameAsLegacy:
				require.Same(t, legacy, got)
			case tt.wantSameAsUnified:
				require.Same(t, unified, got)
			case tt.setupMocks != nil:
				require.NotSame(t, legacy, got)
				require.NotSame(t, unified, got)
			default:
				require.NotSame(t, legacy, got)
				require.NotSame(t, unified, got)
			}
		})
	}
}

func TestNewDualWriter_recordsMetricsAndLogsComparison(t *testing.T) {
	gr := schema.GroupResource{Group: "playlist.grafana.app", Resource: "playlists"}
	key := gr.String()
	legacy := &grafanarest.MockStorage{}
	unified := &grafanarest.MockStorage{}
	metrics := builder.ProvideBuilderMetrics(prometheus.NewRegistry())

	svc := dualwrite.NewMockService(t)
	svc.On("ShouldManage", gr).Return(false)
	svc.On("LogStorageModeComparison", gr, grafanarest.Mode3).Once()

	storageOpts := &options.StorageOptions{
		UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			key: {DualWriterMode: grafanarest.Mode3},
		},
	}

	got, err := NewDualWriter(gr, storageOpts, legacy, unified, svc, metrics)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotSame(t, legacy, got)
	require.NotSame(t, unified, got)

	svc.AssertExpectations(t)
}
