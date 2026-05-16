package apiserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientrest "k8s.io/client-go/rest"

	"github.com/grafana/grafana/pkg/services/user"
)

func TestMaxRequestBodyBytes(t *testing.T) {
	assert.Equal(t, 16*1024*1024, MaxRequestBodyBytes)
}

func Test_service_IsDisabled(t *testing.T) {
	t.Parallel()
	s := &service{}
	assert.False(t, s.IsDisabled())
}

func Test_useNamespaceFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expNs    string
		expOrgID int64
	}{
		{
			name:  "no namespace in path",
			path:  "/apis/folder.grafana.app/",
			expNs: "",
		},
		{
			name:     "stack namespace in path",
			path:     "/apis/folder.grafana.app/v1alpha1/namespaces/stacks-11/folders",
			expNs:    "stacks-11",
			expOrgID: 1,
		},
		{
			name:  "invalid namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/invalid/folders",
			expNs: "invalid",
		},
		{
			name:  "malformed org namespace is ignored",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/org-notanumber/folders",
			expNs: "",
		},
		{
			name:     "org namespace in path",
			path:     "/apis/folder.grafana.app/v1alpha1/namespaces/org-123/folders",
			expNs:    "org-123",
			expOrgID: 123,
		},
		{
			name:  "default namespace in path",
			path:  "/apis/folder.grafana.app/v1alpha1/namespaces/default/folders",
			expNs: "default",
		},
		{
			name:  "path without apis prefix",
			path:  "/livez",
			expNs: "",
		},
		{
			name:  "path too short for namespace",
			path:  "/apis/",
			expNs: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &user.SignedInUser{}
			useNamespaceFromPath(tt.path, u)
			assert.Equal(t, tt.expNs, u.Namespace)
			if tt.expOrgID != 0 {
				assert.Equal(t, tt.expOrgID, u.OrgID)
			}
		})
	}
}

func Test_ensureKubeConfig(t *testing.T) {
	dir := t.TempDir()
	restConfig := &clientrest.Config{
		Host:        "https://127.0.0.1:6443",
		BearerToken: "test-token",
	}

	err := ensureKubeConfig(restConfig, dir)
	require.NoError(t, err)

	kubeconfigPath := filepath.Join(dir, "grafana.kubeconfig")
	_, err = os.Stat(kubeconfigPath)
	require.NoError(t, err)

	content, err := os.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "https://127.0.0.1:6443")
}
