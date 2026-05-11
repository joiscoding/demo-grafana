package appinstaller

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"

	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/options"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/legacysql/dualwrite"
)

// fakeStorage is a minimal rest.Storage used to satisfy grafanarest.Storage.
type fakeStorage struct {
	name string
}

func (s *fakeStorage) New() runtime.Object                 { return nil }
func (s *fakeStorage) Destroy()                            {}
func (s *fakeStorage) Get(ctx context.Context, name string, _ interface{}) (runtime.Object, error) {
	return nil, nil
}

// Ensure fakeStorage satisfies rest.Storage at compile time.
var _ rest.Storage = (*fakeStorage)(nil)

// dummyStorage satisfies grafanarest.Storage but does as little as possible.
// We need this so the NewDualWriter happy path can pass legacy/storage.
type dummyStorage struct {
	grafanarest.Storage
	name string
}

func (d *dummyStorage) Destroy() {}

// shouldManageFalseService wraps the static service to keep ShouldManage=false
// behavior (which the static service already provides).
func newStaticTestService(t *testing.T) dualwrite.Service {
	return dualwrite.ProvideStaticServiceForTests(&setting.Cfg{})
}

func TestNewDualWriter(t *testing.T) {
	legacy := &dummyStorage{name: "legacy"}
	uni := &dummyStorage{name: "unified"}
	gr := schema.GroupResource{Group: "g.example.com", Resource: "widgets"}
	bm := builder.ProvideBuilderMetrics(prometheus.NewRegistry())

	t.Run("dualwrite service manages: delegates to NewStorage", func(t *testing.T) {
		svc := dualwrite.ProvideTestService() // ShouldManage=true, NewStorage returns "not implemented"
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{}}
		_, err := NewDualWriter(gr, opts, legacy, uni, svc, bm)
		require.Error(t, err)
	})

	t.Run("no service, no config defaults to mode0 -> legacy", func(t *testing.T) {
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{}}
		got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
		require.NoError(t, err)
		require.Same(t, legacy, got)
	})

	t.Run("config mode4 returns unified storage", func(t *testing.T) {
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			gr.String(): {DualWriterMode: grafanarest.Mode4},
		}}
		got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
		require.NoError(t, err)
		require.Same(t, uni, got)
	})

	t.Run("config mode5 returns unified storage", func(t *testing.T) {
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			gr.String(): {DualWriterMode: grafanarest.Mode5},
		}}
		got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
		require.NoError(t, err)
		require.Same(t, uni, got)
	})

	t.Run("config mode0 returns legacy", func(t *testing.T) {
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			gr.String(): {DualWriterMode: grafanarest.Mode0},
		}}
		got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
		require.NoError(t, err)
		require.Same(t, legacy, got)
	})

	t.Run("config mode2 falls through to NewStaticStorage", func(t *testing.T) {
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			gr.String(): {DualWriterMode: grafanarest.Mode2},
		}}
		got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
		require.NoError(t, err)
		// NewStaticStorage returns a *dualWriter wrapper, not legacy or unified directly.
		require.NotNil(t, got)
		require.NotSame(t, legacy, got)
		require.NotSame(t, uni, got)
	})

	t.Run("non-managing service logs comparison and continues", func(t *testing.T) {
		svc := newStaticTestService(t)
		opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
			gr.String(): {DualWriterMode: grafanarest.Mode0},
		}}
		got, err := NewDualWriter(gr, opts, legacy, uni, svc, bm)
		require.NoError(t, err)
		require.Same(t, legacy, got)
	})
}

// Ensure that a non-found resource configuration does not panic and uses
// the zero-value DualWriterMode (Mode0 -> legacy).
func TestNewDualWriter_NoResourceConfigInMap(t *testing.T) {
	legacy := &dummyStorage{name: "legacy"}
	uni := &dummyStorage{name: "unified"}
	gr := schema.GroupResource{Group: "g.example.com", Resource: "doesnotexist"}
	bm := builder.ProvideBuilderMetrics(prometheus.NewRegistry())
	opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
		"other.example.com": {DualWriterMode: grafanarest.Mode2},
	}}
	got, err := NewDualWriter(gr, opts, legacy, uni, nil, bm)
	require.NoError(t, err)
	require.Same(t, legacy, got)
}

// sanity: errors from underlying components are surfaced
func TestNewDualWriter_ErrorSurfaceFromService(t *testing.T) {
	bm := builder.ProvideBuilderMetrics(prometheus.NewRegistry())
	svc := dualwrite.ProvideTestService()
	opts := &options.StorageOptions{UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{}}
	_, err := NewDualWriter(schema.GroupResource{Group: "g", Resource: "r"}, opts, &dummyStorage{}, &dummyStorage{}, svc, bm)
	require.Error(t, err)
	require.True(t, errors.Is(err, err)) // sanity check: error compares to itself
}
