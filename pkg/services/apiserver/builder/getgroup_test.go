package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"
)

func TestGetGroup(t *testing.T) {
	t.Parallel()

	gv := schema.GroupVersion{Group: "example.grafana.app", Version: "v1"}

	t.Run("version provider", func(t *testing.T) {
		t.Parallel()
		group, err := getGroup(&getGroupMockVersionProvider{gv: gv})
		require.NoError(t, err)
		require.Equal(t, "example.grafana.app", group)
	})

	t.Run("versions provider", func(t *testing.T) {
		t.Parallel()
		group, err := getGroup(&getGroupMockVersionsProvider{
			gvs: []schema.GroupVersion{gv},
		})
		require.NoError(t, err)
		require.Equal(t, "example.grafana.app", group)
	})

	t.Run("versions provider with no versions", func(t *testing.T) {
		t.Parallel()
		_, err := getGroup(&getGroupMockVersionsProvider{gvs: nil})
		require.Error(t, err)
	})

	t.Run("builder without version info", func(t *testing.T) {
		t.Parallel()
		_, err := getGroup(&getGroupMockBuilderOnly{})
		require.Error(t, err)
	})
}

type getGroupMockVersionProvider struct {
	gv schema.GroupVersion
}

func (m *getGroupMockVersionProvider) GetGroupVersion() schema.GroupVersion {
	return m.gv
}

func (m *getGroupMockVersionProvider) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *getGroupMockVersionProvider) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ APIGroupOptions) error {
	return nil
}

func (m *getGroupMockVersionProvider) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *getGroupMockVersionProvider) AllowedV0Alpha1Resources() []string { return nil }

type getGroupMockVersionsProvider struct {
	gvs []schema.GroupVersion
}

func (m *getGroupMockVersionsProvider) GetGroupVersions() []schema.GroupVersion {
	return m.gvs
}

func (m *getGroupMockVersionsProvider) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *getGroupMockVersionsProvider) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ APIGroupOptions) error {
	return nil
}

func (m *getGroupMockVersionsProvider) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *getGroupMockVersionsProvider) AllowedV0Alpha1Resources() []string { return nil }

type getGroupMockBuilderOnly struct{}

func (m *getGroupMockBuilderOnly) InstallSchema(_ *runtime.Scheme) error { return nil }

func (m *getGroupMockBuilderOnly) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ APIGroupOptions) error {
	return nil
}

func (m *getGroupMockBuilderOnly) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }

func (m *getGroupMockBuilderOnly) AllowedV0Alpha1Resources() []string { return nil }
