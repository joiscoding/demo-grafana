package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

type minimalBuilder struct{}

func (minimalBuilder) InstallSchema(_ *runtime.Scheme) error { return nil }
func (minimalBuilder) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ builder.APIGroupOptions) error {
	return nil
}
func (minimalBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (minimalBuilder) AllowedV0Alpha1Resources() []string                  { return nil }

type singleVersionBuilder struct {
	minimalBuilder
	gv schema.GroupVersion
}

func (s singleVersionBuilder) GetGroupVersion() schema.GroupVersion { return s.gv }

type multiVersionBuilder struct {
	minimalBuilder
	gvs []schema.GroupVersion
}

func (m multiVersionBuilder) GetGroupVersions() []schema.GroupVersion { return m.gvs }

func TestGetGroupVersions(t *testing.T) {
	t.Run("single version provider", func(t *testing.T) {
		gv := schema.GroupVersion{Group: "g", Version: "v1"}
		got := builder.GetGroupVersions(singleVersionBuilder{gv: gv})
		require.Equal(t, []schema.GroupVersion{gv}, got)
	})

	t.Run("multi version provider", func(t *testing.T) {
		gvs := []schema.GroupVersion{
			{Group: "g", Version: "v1"},
			{Group: "g", Version: "v2"},
		}
		got := builder.GetGroupVersions(multiVersionBuilder{gvs: gvs})
		require.Equal(t, gvs, got)
	})

	t.Run("panics when builder does not implement provider", func(t *testing.T) {
		require.Panics(t, func() {
			builder.GetGroupVersions(minimalBuilder{})
		})
	})
}

func TestCommonStructsConstruction(t *testing.T) {
	opts := builder.APIGroupOptions{}
	require.NotNil(t, &opts)

	routes := builder.APIRoutes{
		Root:      []builder.APIRouteHandler{{Path: "/r"}},
		Namespace: []builder.APIRouteHandler{{Path: "/n"}},
	}
	require.Len(t, routes.Root, 1)
	require.Len(t, routes.Namespace, 1)

	require.Equal(t, "*", builder.AllResourcesAllowed)
}
