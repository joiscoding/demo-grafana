package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"
)

type bareBuilder struct{}

func (bareBuilder) InstallSchema(_ *runtime.Scheme) error { return nil }
func (bareBuilder) UpdateAPIGroupInfo(_ *genericapiserver.APIGroupInfo, _ APIGroupOptions) error {
	return nil
}
func (bareBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions { return nil }
func (bareBuilder) AllowedV0Alpha1Resources() []string                  { return nil }

type singleProvider struct {
	bareBuilder
	gv schema.GroupVersion
}

func (s singleProvider) GetGroupVersion() schema.GroupVersion { return s.gv }

type multiProvider struct {
	bareBuilder
	gvs []schema.GroupVersion
}

func (m multiProvider) GetGroupVersions() []schema.GroupVersion { return m.gvs }

func TestGetGroup(t *testing.T) {
	t.Run("from APIGroupVersionProvider", func(t *testing.T) {
		got, err := getGroup(singleProvider{gv: schema.GroupVersion{Group: "g1", Version: "v1"}})
		require.NoError(t, err)
		require.Equal(t, "g1", got)
	})

	t.Run("from APIGroupVersionsProvider", func(t *testing.T) {
		got, err := getGroup(multiProvider{gvs: []schema.GroupVersion{{Group: "g2", Version: "v1"}, {Group: "g2", Version: "v2"}}})
		require.NoError(t, err)
		require.Equal(t, "g2", got)
	})

	t.Run("multi provider with no versions returns error", func(t *testing.T) {
		_, err := getGroup(multiProvider{gvs: nil})
		require.Error(t, err)
	})

	t.Run("builder without provider returns error", func(t *testing.T) {
		_, err := getGroup(bareBuilder{})
		require.Error(t, err)
	})
}
