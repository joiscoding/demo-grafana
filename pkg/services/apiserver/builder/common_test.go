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

func TestGetGroupVersions(t *testing.T) {
	t.Parallel()

	gv := schema.GroupVersion{Group: "example.grafana.app", Version: "v1"}

	t.Run("single version provider", func(t *testing.T) {
		t.Parallel()
		b := &mockGroupVersionProvider{gv: gv}
		require.Equal(t, []schema.GroupVersion{gv}, builder.GetGroupVersions(b))
	})

	t.Run("multi version provider", func(t *testing.T) {
		t.Parallel()
		gvs := []schema.GroupVersion{
			{Group: "example.grafana.app", Version: "v1"},
			{Group: "example.grafana.app", Version: "v1beta1"},
		}
		b := &mockGroupVersionsProvider{gvs: gvs}
		require.Equal(t, gvs, builder.GetGroupVersions(b))
	})
}

type mockGroupVersionProvider struct {
	gv schema.GroupVersion
}

func (m *mockGroupVersionProvider) GetGroupVersion() schema.GroupVersion {
	return m.gv
}

func (m *mockGroupVersionProvider) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *mockGroupVersionProvider) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ builder.APIGroupOptions) error {
	return nil
}

func (m *mockGroupVersionProvider) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *mockGroupVersionProvider) AllowedV0Alpha1Resources() []string { return nil }

type mockGroupVersionsProvider struct {
	gvs []schema.GroupVersion
}

func (m *mockGroupVersionsProvider) GetGroupVersions() []schema.GroupVersion {
	return m.gvs
}

func (m *mockGroupVersionsProvider) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *mockGroupVersionsProvider) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ builder.APIGroupOptions) error {
	return nil
}

func (m *mockGroupVersionsProvider) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *mockGroupVersionsProvider) AllowedV0Alpha1Resources() []string { return nil }
