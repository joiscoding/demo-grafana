package request_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	k8srequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/apiserver/endpoints/request"
	"github.com/grafana/grafana/pkg/setting"
)

func TestNamespaceMapper(t *testing.T) {
	tests := []struct {
		name     string
		cfg      string
		orgId    int64
		expected string
	}{
		{
			name:     "default namespace",
			orgId:    1,
			expected: "default",
		},
		{
			name:     "with org",
			orgId:    123,
			expected: "org-123",
		},
		// an invalid use-case, but just documenting that it's handled as stacks-0
		// this currently prevents the need to have the Mapper return (mapped, err) instead of just mapped.
		// err checking is avoided for now to keep the usage fluent
		{
			name:     "with stackId",
			cfg:      "abc",
			orgId:    123,        // ignored
			expected: "stacks-0", // we parse to int and default to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := request.GetNamespaceMapper(&setting.Cfg{StackID: tt.cfg})
			require.Equal(t, tt.expected, mapper(tt.orgId))
		})
	}
}

func TestGetNamespaceMapper_NilCfg(t *testing.T) {
	mapper := request.GetNamespaceMapper(nil)
	require.Equal(t, "default", mapper(1))
	require.Equal(t, "org-7", mapper(7))
}

func TestGetNamespaceMapper_ValidStackID(t *testing.T) {
	mapper := request.GetNamespaceMapper(&setting.Cfg{StackID: "42"})
	require.Equal(t, "stacks-42", mapper(1))
	require.Equal(t, "stacks-42", mapper(99))
}

func TestNamespaceInfoFrom(t *testing.T) {
	t.Run("default namespace parses to org 1", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "default")
		info, err := request.NamespaceInfoFrom(ctx, true)
		require.NoError(t, err)
		require.Equal(t, int64(1), info.OrgID)
	})

	t.Run("org-N namespace parses", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "org-12")
		info, err := request.NamespaceInfoFrom(ctx, true)
		require.NoError(t, err)
		require.Equal(t, int64(12), info.OrgID)
	})

	t.Run("empty namespace requireOrgID returns error", func(t *testing.T) {
		ctx := context.Background()
		info, err := request.NamespaceInfoFrom(ctx, true)
		require.Error(t, err)
		require.Less(t, info.OrgID, int64(1))
	})

	t.Run("empty namespace not required is no error", func(t *testing.T) {
		ctx := context.Background()
		_, err := request.NamespaceInfoFrom(ctx, false)
		require.NoError(t, err)
	})

	t.Run("invalid namespace returns parse error", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "org-bad")
		_, err := request.NamespaceInfoFrom(ctx, true)
		require.Error(t, err)
	})
}

func TestOrgIDForList(t *testing.T) {
	t.Run("namespace set returns parsed org", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "org-9")
		id, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(9), id)
	})

	t.Run("default namespace returns 1", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "default")
		id, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(1), id)
	})

	t.Run("no namespace but requester present returns user org", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 5})
		id, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(5), id)
	})

	t.Run("no namespace and no requester returns -1 and error", func(t *testing.T) {
		id, err := request.OrgIDForList(context.Background())
		require.Error(t, err)
		require.Equal(t, int64(-1), id)
	})
}
