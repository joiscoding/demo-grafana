package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-app-sdk/resource"
	examplev1 "github.com/grafana/grafana/pkg/services/apiserver/builder/runner/testdata/app/pkg/apis/example/v1"
)

func TestKindToResourceInfo(t *testing.T) {
	exampleKind := examplev1.ExampleKind()
	clusterKind := resource.Kind{
		Schema: resource.NewSimpleSchema(
			"example.grafana.app",
			"v1",
			&examplev1.Example{},
			&examplev1.ExampleList{},
			resource.WithKind("ClusterExample"),
			resource.WithPlural("clusterexamples"),
			resource.WithScope(resource.ClusterScope),
		),
	}

	tests := []struct {
		name          string
		kind          resource.Kind
		wantCluster   bool
		wantResource  string
		wantSingular  string
		wantKind      string
		wantStorage   string
	}{
		{
			name:         "namespaced kind",
			kind:         exampleKind,
			wantCluster:  false,
			wantResource: "examples",
			wantSingular: "example",
			wantKind:     "Example",
			wantStorage:  "examples",
		},
		{
			name:         "cluster scoped kind",
			kind:         clusterKind,
			wantCluster:  true,
			wantResource: "clusterexamples",
			wantSingular: "clusterexample",
			wantKind:     "ClusterExample",
			wantStorage:  "clusterexamples",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := KindToResourceInfo(tt.kind)

			require.Equal(t, tt.wantCluster, info.IsClusterScoped())
			require.Equal(t, tt.wantResource, info.GetName())
			require.Equal(t, tt.wantSingular, info.GetSingularName())
			require.Equal(t, tt.kind.GroupVersionKind().Kind, info.GroupVersionKind().Kind)
			require.Equal(t, tt.wantKind, info.GroupVersionKind().Kind)
			require.Equal(t, tt.wantStorage, info.StoragePath())

			obj := info.NewFunc()
			require.IsType(t, &examplev1.Example{}, obj)

			list := info.NewListFunc()
			require.IsType(t, &examplev1.ExampleList{}, list)
		})
	}
}
